'use client'

import { useMemo } from 'react'
import dynamic from 'next/dynamic'
import type { ComponentType } from 'react'
import type { EChartsOption } from 'echarts'

import { Skeleton } from '@/components/ui/skeleton'
import { formatTimestamp, formatBitsPerSec } from '@/lib/formatters'
import type {
  AggregatedContainerMetricPoint,
  Container,
  ContainerHistory,
  LiveServerContainer,
  MetricRange,
  RawContainerMetricPoint,
} from '@/types'

const ReactECharts = dynamic(
  () => import('echarts-for-react').then((m) => m.default),
  { ssr: false, loading: () => <Skeleton className="h-55 w-full" /> },
) as ComponentType<{
  option: EChartsOption
  style?: React.CSSProperties
  notMerge?: boolean
  lazyUpdate?: boolean
}>

// ---------------------------------------------------------------------------
// Formattery
// ---------------------------------------------------------------------------

function fmtBytes(v: number): string {
  if (!isFinite(v) || v < 0) return '0 B'
  const mb = v / (1024 * 1024)
  if (mb < 1024) return `${mb.toFixed(2)} MB`
  return `${(mb / 1024).toFixed(2)} GB`
}

function fmtBytesPerSec(v: number): string {
  if (!isFinite(v) || v < 0) return '0 B/s'
  const k = 1024
  if (v < k) return `${v.toFixed(0)} B/s`
  if (v < k * k) return `${(v / k).toFixed(2)} KB/s`
  if (v < k * k * k) return `${(v / (k * k)).toFixed(2)} MB/s`
  return `${(v / (k * k * k)).toFixed(2)} GB/s`
}

function fmtBitsPerSec(v: number): string {
  return formatBitsPerSec(v)
}

const CONTAINER_COLORS = [
  '#60a5fa', '#34d399', '#f87171', '#a78bfa',
  '#fb923c', '#38bdf8', '#f472b6', '#facc15',
  '#4ade80', '#c084fc',
]

// ---------------------------------------------------------------------------
// Typy wejścia
// ---------------------------------------------------------------------------

type ContainerChartType = 'cpu' | 'ram' | 'net'

type UnifiedPoint = LiveServerContainer | RawContainerMetricPoint | AggregatedContainerMetricPoint

interface ContainerEntry {
  id: string
  name: string
  project: string
  color: string
  points: UnifiedPoint[]
}

// ---------------------------------------------------------------------------
// Odczyt wartości — różne struktury live vs history
// ---------------------------------------------------------------------------

function getCpu(p: UnifiedPoint): number {
  if ('cpu_avg' in p) return p.cpu_avg
  return (p as LiveServerContainer | RawContainerMetricPoint).cpu
}

function getMem(p: UnifiedPoint): number {
  if ('mem_avg' in p) return p.mem_avg
  return 'mem_used' in p ? p.mem_used : 0
}

function getNetRx(p: UnifiedPoint): number {
  if ('net_rx_avg' in p) return p.net_rx_avg
  if ('net_rx_bytes' in p) return (p as RawContainerMetricPoint).net_rx_bytes
  return (p as LiveServerContainer).net_rx
}

function getNetTx(p: UnifiedPoint): number {
  if ('net_tx_avg' in p) return p.net_tx_avg
  if ('net_tx_bytes' in p) return (p as RawContainerMetricPoint).net_tx_bytes
  return (p as LiveServerContainer).net_tx
}

function getTimeline(entries: ContainerEntry[]): number[] {
  const allTimestamps = new Set<number>()

  for (const entry of entries) {
    for (const point of entry.points) {
      allTimestamps.add(point.timestamp)
    }
  }

  return Array.from(allTimestamps)
    .sort((a, b) => a - b)
    .slice(-60)
}

// ---------------------------------------------------------------------------
// Budowanie opcji wykresu
// ---------------------------------------------------------------------------

function buildOption(
  entries: ContainerEntry[],
  type: ContainerChartType,
  range: MetricRange,
  isLive: boolean,
): EChartsOption {
  const isCpu = type === 'cpu'
  const isNet = type === 'net'

  const tooltipFmt = isCpu ? (v: number) => `${v.toFixed(2)}%` : isNet ? fmtBitsPerSec : fmtBytes
  const yAxisFmt = isCpu ? (v: number) => `${v}%` : isNet ? fmtBitsPerSec : fmtBytes

  const timeline = getTimeline(entries)
  const timestamps = timeline.map((timestamp) =>
    formatTimestamp(timestamp, isLive ? '1m' : range),
  )

  const series: EChartsOption['series'] = []

  for (const entry of entries) {
    const pointByTimestamp = new Map<number, UnifiedPoint>()
    for (const point of entry.points) {
      pointByTimestamp.set(point.timestamp, point)
    }

    if (isCpu) {
      series.push({
        id: `${entry.id}:cpu`,
        name: entry.name,
        type: 'line',
        data: timeline.map((timestamp) => {
          const point = pointByTimestamp.get(timestamp)
          return point ? getCpu(point) : null
        }),
        smooth: true,
        showSymbol: false,
        lineStyle: { width: 2 },
        areaStyle: { opacity: 0.08 },
        color: entry.color,
        emphasis: { disabled: true },
      })
    } else if (!isNet) {
      series.push({
        id: `${entry.id}:ram`,
        name: entry.name,
        type: 'line',
        data: timeline.map((timestamp) => {
          const point = pointByTimestamp.get(timestamp)
          return point ? getMem(point) : null
        }),
        smooth: true,
        showSymbol: false,
        lineStyle: { width: 2 },
        areaStyle: { opacity: 0.08 },
        color: entry.color,
        emphasis: { disabled: true },
      })
    } else {
      series.push(
        {
          id: `${entry.id}:net-rx`,
          name: `${entry.name} ↓`,
          type: 'line',
          data: timeline.map((timestamp) => {
            const point = pointByTimestamp.get(timestamp)
            return point ? getNetRx(point) : null
          }),
          smooth: true,
          showSymbol: false,
          lineStyle: { width: 2, type: 'solid' },
          areaStyle: { opacity: 0.05 },
          color: entry.color,
          emphasis: { disabled: true },
        },
        {
          id: `${entry.id}:net-tx`,
          name: `${entry.name} ↑`,
          type: 'line',
          data: timeline.map((timestamp) => {
            const point = pointByTimestamp.get(timestamp)
            return point ? getNetTx(point) : null
          }),
          smooth: true,
          showSymbol: false,
          lineStyle: { width: 2, type: 'dashed' },
          areaStyle: { opacity: 0 },
          color: entry.color,
          emphasis: { disabled: true },
        },
      )
    }
  }

  const entryById = new Map<string, ContainerEntry>()
  for (const entry of entries) {
    entryById.set(entry.id, entry)
  }

  return {
    backgroundColor: 'transparent',
    animation: false,
    grid: { top: 36, right: 16, bottom: 32, left: 72 },
    tooltip: {
      trigger: 'axis',
      backgroundColor: '#18181b',
      borderColor: '#27272a',
      textStyle: { color: '#e4e4e7', fontSize: 12 },
      formatter: (params: unknown) => {
        const list = params as { seriesName: string; value: unknown; axisValue: string; seriesId: string }[]
        if (!Array.isArray(list) || list.length === 0) return ''

        const getValue = (value: unknown): number =>
          typeof value === 'number' && isFinite(value)
            ? value
            : Number.NEGATIVE_INFINITY

        const projectTotals = new Map<string, { cpu: number; ram: number; netRx: number; netTx: number }>()
        const containerDetails: { name: string; project: string; value: number }[] = []

        for (const item of list) {
          if (typeof item.value !== 'number' || !isFinite(item.value)) continue

          const entryId = item.seriesId?.split(':')[0] ?? ''
          const entry = entryById.get(entryId)
          const project = entry?.project || 'Standalone'

          let valueForTotal = 0
          if (isCpu) valueForTotal = item.value
          else if (!isNet) valueForTotal = item.value
          else {
            const isRx = item.seriesName?.endsWith('↓')
            valueForTotal = item.value
          }

          if (!projectTotals.has(project)) {
            projectTotals.set(project, { cpu: 0, ram: 0, netRx: 0, netTx: 0 })
          }
          const total = projectTotals.get(project)!
          if (isCpu) total.cpu += item.value
          else if (!isNet) total.ram += item.value
          else if (item.seriesName?.endsWith('↓')) total.netRx += item.value
          else total.netTx += item.value

          containerDetails.push({
            name: item.seriesName ?? entryId.slice(0, 12),
            project,
            value: item.value,
          })
        }

        containerDetails.sort((a, b) => b.value - a.value)
        const sortedProjects = Array.from(projectTotals.keys()).sort()

        let html = `<div style="margin-bottom:6px;color:#a1a1aa">${list[0].axisValue}</div>`

        for (const project of sortedProjects) {
          const total = projectTotals.get(project)!
          const containersInProject = containerDetails.filter(c => c.project === project)

          const projectLabel = project || 'Standalone'
          let projectTotal = ''
          if (isCpu) projectTotal = tooltipFmt(total.cpu)
          else if (!isNet) projectTotal = tooltipFmt(total.ram)
          else projectTotal = `${tooltipFmt(total.netRx)} ↓ / ${tooltipFmt(total.netTx)} ↑`

          html += `<div style="margin-top:6px;padding-bottom:2px;border-bottom:1px solid #3f3f46">
            <span style="color:#e4e4e7;font-weight:600">${projectLabel}</span>: <b>${projectTotal}</b>
          </div>`

          for (const c of containersInProject) {
            html += `<div style="margin-left:8px;color:#a1a1aa">${c.name}: <b>${tooltipFmt(c.value)}</b></div>`
          }
        }

        return html
      },
    },
    legend: {
      show: true,
      textStyle: { color: '#a1a1aa', fontSize: 11 },
      top: 6,
      right: 16,
      type: 'scroll',
    },
    xAxis: {
      type: 'category',
      data: timestamps,
      axisLabel: { color: '#71717a', fontSize: 10, interval: 'auto', hideOverlap: true },
      axisLine: { lineStyle: { color: '#3f3f46' } },
      splitLine: { show: false },
      boundaryGap: false,
    },
    yAxis: {
      type: 'value',
      min: 0,
      axisLabel: {
        color: '#71717a',
        fontSize: 10,
        formatter: yAxisFmt,
        width: 64,
        overflow: 'truncate',
      },
      splitLine: { lineStyle: { color: '#27272a' } },
    },
    series,
  }
}

// ---------------------------------------------------------------------------
// Wykres
// ---------------------------------------------------------------------------

function ContainerChart({
  entries,
  type,
  range,
  isLive,
}: {
  entries: ContainerEntry[]
  type: ContainerChartType
  range: MetricRange
  isLive: boolean
}) {
  const option = useMemo(() => buildOption(entries, type, range, isLive), [entries, type, range, isLive])
  return (
    <ReactECharts
      option={option}
      style={{ width: '100%', height: '220px' }}
      notMerge={true}
      lazyUpdate={false}
    />
  )
}

// ---------------------------------------------------------------------------
// Props i eksport
// ---------------------------------------------------------------------------

interface ContainerMetricsSectionProps {
  /** Tryb live: mapa containerId -> punkty SSE */
  containerPoints: Map<string, LiveServerContainer[]>
  /** Lista kontenerów serwera (do pobrania nazwy) */
  containers: Container[]
  /** Tryb historyczny: dane z API */
  historyContainers: ContainerHistory[]
  isLive: boolean
  range: MetricRange
}

export function ContainerMetricsSection({
  containerPoints,
  containers,
  historyContainers,
  isLive,
  range,
}: ContainerMetricsSectionProps) {
  const infoByID = useMemo(() => {
    const map = new Map<string, Container>()
    for (const c of containers) map.set(c.container_id, c)
    return map
  }, [containers])

  const entries: ContainerEntry[] = useMemo(() => {
    if (isLive) {
      return Array.from(containerPoints.entries()).map(([id, pts], idx) => {
        const sliced = pts.slice(-60)
        const containerInfo = infoByID.get(id)
        return {
          id,
          name: containerInfo?.name ?? id.slice(0, 12),
          project: containerInfo?.project ?? '',
          color: CONTAINER_COLORS[idx % CONTAINER_COLORS.length],
          points: sliced,
        }
      })
    }

    return historyContainers.map((c, idx) => ({
      id: c.container_id,
      name: c.name || c.container_id.slice(0, 12),
      project: c.project ?? '',
      color: CONTAINER_COLORS[idx % CONTAINER_COLORS.length],
      points: c.points,
    }))
  }, [isLive, containerPoints, historyContainers, infoByID])

  if (entries.length === 0) return null

  return (
    <div className="space-y-3">
      <h3 className="text-sm font-semibold text-zinc-400 uppercase tracking-wider">
        Kontenery Docker ({entries.length})
      </h3>
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        {(['cpu', 'ram', 'net'] as ContainerChartType[]).map((type) => (
          <div key={type} className="rounded-lg border border-zinc-800 bg-zinc-900/50 p-3">
            <p className="mb-1 px-1 text-xs font-medium text-zinc-400">
              {type === 'cpu' ? 'CPU' : type === 'ram' ? 'RAM' : 'Sieć'}
            </p>
            <ContainerChart entries={entries} type={type} range={range} isLive={isLive} />
          </div>
        ))}
      </div>
    </div>
  )
}