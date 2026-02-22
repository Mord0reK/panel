'use client'

import { useMemo } from 'react'
import ReactECharts from 'echarts-for-react'
import type { EChartsOption } from 'echarts'

import { formatTimestamp } from '@/lib/formatters'
import type { AggregatedHostMetricPoint, MetricRange } from '@/types'

// ---------------------------------------------------------------------------
// Typy wykresu — takie same jak LiveChart
// ---------------------------------------------------------------------------

export type HistoryChartType = 'cpu' | 'ram' | 'disk' | 'disk_percent' | 'net'

interface HistoryChartProps {
  /** Tablica zagregowanych punktów hosta */
  points: AggregatedHostMetricPoint[]
  /** Typ wykresu */
  type: HistoryChartType
  /** Zakres metryk — wpływa na format osi X */
  range: MetricRange
}

// ---------------------------------------------------------------------------
// Konfiguracja per typ wykresu (avg + opcjonalnie min/max area)
// ---------------------------------------------------------------------------

function fmtBytes(v: number): string {
  if (!isFinite(v) || v < 0) return '0 B'
  if (v === 0) return '0 B'
  const k = 1024
  const mb = v / (k * k)
  if (mb < 1024) return `${mb.toFixed(2)} MB`
  const gb = mb / k
  return `${gb.toFixed(2)} GB`
}

function fmtBytesPerSec(v: number): string {
  if (!isFinite(v) || v < 0) return '0 B/s'
  if (v === 0) return '0 B/s'
  const k = 1024
  if (v < k) return `${v.toFixed(0)} B/s`
  if (v < k * k) return `${(v / k).toFixed(2)} KB/s`
  if (v < k * k * k) return `${(v / (k * k)).toFixed(2)} MB/s`
  return `${(v / (k * k * k)).toFixed(2)} GB/s`
}

interface SeriesFieldDef {
  avg: keyof AggregatedHostMetricPoint
  min?: keyof AggregatedHostMetricPoint
  max?: keyof AggregatedHostMetricPoint
  name: string
  color: string
}

interface HistoryChartConfig {
  title: string
  series: SeriesFieldDef[]
  tooltipFormatter: (v: number) => string
  yAxisFormatter?: (v: number) => string
  yAxisLabel: string
  yMin?: number
  yMax?: number
}

const CONFIGS: Record<HistoryChartType, HistoryChartConfig> = {
  cpu: {
    title: 'Wykorzystanie CPU',
    series: [
      {
        avg: 'cpu_avg',
        min: 'cpu_min',
        max: 'cpu_max',
        name: 'Wykorzystanie CPU',
        color: '#60a5fa',
      },
    ],
    tooltipFormatter: (v) => `${v.toFixed(2)}%`,
    yAxisLabel: '%',
    yMin: 0,
  },
  ram: {
    title: 'Wykorzystanie pamięci RAM',
    series: [
      {
        avg: 'mem_used_avg',
        min: 'mem_used_min',
        max: 'mem_used_max',
        name: 'Użyta pamięć',
        color: '#a78bfa',
      },
    ],
    tooltipFormatter: fmtBytes,
    yAxisFormatter: fmtBytes,
    yAxisLabel: '',
    yMin: 0,
  },
  disk: {
    title: 'Przepustowość dysku (I/O)',
    series: [
      {
        avg: 'disk_read_bytes_per_sec_avg',
        name: 'Odczyt',
        color: '#34d399',
      },
      {
        avg: 'disk_write_bytes_per_sec_avg',
        min: 'disk_write_bytes_per_sec_min',
        max: 'disk_write_bytes_per_sec_max',
        name: 'Zapis',
        color: '#f87171',
      },
    ],
    tooltipFormatter: fmtBytesPerSec,
    yAxisLabel: '',
    yMin: 0,
  },
  disk_percent: {
    title: 'Zajętość dysku',
    series: [
      {
        avg: 'disk_used_percent_avg',
        min: 'disk_used_percent_min',
        max: 'disk_used_percent_max',
        name: 'Zajęte',
        color: '#f59e0b',
      },
    ],
    tooltipFormatter: (v) => `${v.toFixed(1)}%`,
    yAxisFormatter: (v) => `${v}%`,
    yAxisLabel: '%',
    yMin: 0,
    yMax: 100,
  },
  net: {
    title: 'Przepustowość sieci (net I/O)',
    series: [
      {
        avg: 'net_rx_bytes_per_sec_avg',
        min: 'net_rx_bytes_per_sec_min',
        max: 'net_rx_bytes_per_sec_max',
        name: '↓ Odebrane (RX)',
        color: '#38bdf8',
      },
      {
        avg: 'net_tx_bytes_per_sec_avg',
        min: 'net_tx_bytes_per_sec_min',
        max: 'net_tx_bytes_per_sec_max',
        name: '↑ Wysłane (TX)',
        color: '#fb923c',
      },
    ],
    tooltipFormatter: fmtBytesPerSec,
    yAxisLabel: '',
    yMin: 0,
  },
}

// ---------------------------------------------------------------------------
// Komponent
// ---------------------------------------------------------------------------

export function HistoryChart({ points, type, range }: HistoryChartProps) {
  const config = CONFIGS[type]

  const option = useMemo<EChartsOption>(() => {
    const timestamps = points.map((p) => formatTimestamp(p.timestamp, range))
    const showLegend = config.series.length > 1

    // Budowanie serii — avg linia + opcjonalny min/max area fill
    const echartsSeries: EChartsOption['series'] = []
    for (const def of config.series) {
      // Linia avg
      echartsSeries.push({
        name: def.name,
        type: 'line',
        data: points.map((p) => p[def.avg] as number),
        smooth: true,
        showSymbol: false,
        lineStyle: { width: 2 },
        color: def.color,
        z: 2,
      })

      // Min/max area fill (band)
      if (def.min && def.max) {
        // Max (górna granica)
        echartsSeries.push({
          name: `${def.name} max`,
          type: 'line',
          data: points.map((p) => p[def.max!] as number),
          smooth: true,
          showSymbol: false,
          lineStyle: { width: 0 },
          areaStyle: { opacity: 0.12, color: def.color },
          stack: `band-${def.name}`,
          z: 1,
          silent: true,
        })
        // Min (dolna granica — odejmowana od max przez stack)
        echartsSeries.push({
          name: `${def.name} min`,
          type: 'line',
          data: points.map((p) => p[def.min!] as number),
          smooth: true,
          showSymbol: false,
          lineStyle: { width: 0 },
          areaStyle: { opacity: 0 },
          stack: `band-${def.name}`,
          z: 1,
          silent: true,
        })
      }
    }

    return {
      backgroundColor: 'transparent',
      grid: { top: 36, right: 16, bottom: 28, left: 56 },
      title: {
        text: config.title,
        textStyle: { color: '#e4e4e7', fontSize: 14, fontWeight: 500 },
        left: 8,
        top: 4,
      },
      tooltip: {
        trigger: 'axis',
        backgroundColor: '#18181b',
        borderColor: '#27272a',
        textStyle: { color: '#e4e4e7', fontSize: 12 },
        formatter: (params: unknown) => {
          const list = params as {
            seriesName: string
            value: number
            axisValue: string
          }[]
          if (!Array.isArray(list) || list.length === 0) return ''
          const visible = list.filter(
            (item) =>
              !item.seriesName.endsWith(' min') &&
              !item.seriesName.endsWith(' max'),
          )
          visible.sort((a, b) => b.value - a.value)
          let html = `<div style="margin-bottom:4px">${visible[0]?.axisValue ?? ''}</div>`
          for (const item of visible) {
            html += `<div>${item.seriesName}: <b>${config.tooltipFormatter(item.value)}</b></div>`
          }
          return html
        },
      },
      legend: showLegend
        ? {
            show: true,
            data: config.series.map((s) => s.name),
            textStyle: { color: '#a1a1aa', fontSize: 11 },
            top: 4,
            right: 16,
          }
        : { show: false },
      xAxis: {
        type: 'category',
        data: timestamps,
        axisLabel: { color: '#71717a', fontSize: 10 },
        axisLine: { lineStyle: { color: '#3f3f46' } },
        splitLine: { show: false },
      },
      yAxis: {
        type: 'value',
        min: config.yMin,
        max: config.yMax,
        axisLabel: {
          color: '#71717a',
          fontSize: 10,
          formatter: config.yAxisLabel === '%' ? '{value}%' : undefined,
        },
        splitLine: { lineStyle: { color: '#27272a' } },
      },
      series: echartsSeries,
    }
  }, [points, config, range])

  return (
    <ReactECharts
      option={option}
      style={{ width: '100%', height: '200px'}}
      notMerge={true}
      lazyUpdate={true}
      theme="dark"
    />
  )
}
