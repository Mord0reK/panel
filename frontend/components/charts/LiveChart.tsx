'use client'

import { useRef, useEffect, useMemo } from 'react'
import ReactECharts from 'echarts-for-react'
import type { EChartsOption } from 'echarts'

import { formatTimestamp } from '@/lib/formatters'
import type { LiveServerHost } from '@/types'

// ---------------------------------------------------------------------------
// Typy wykresu
// ---------------------------------------------------------------------------

export type LiveChartType = 'cpu' | 'ram' | 'disk' | 'net'

interface LiveChartProps {
  /** Bufor ostatnich 60 punktów hosta */
  points: LiveServerHost[]
  /** Typ wykresu determinujący pola i formatowanie */
  type: LiveChartType
}

// ---------------------------------------------------------------------------
// Konfiguracja per typ wykresu
// ---------------------------------------------------------------------------

interface ChartConfig {
  title: string
  fields: (keyof LiveServerHost)[]
  seriesNames: string[]
  colors: string[]
  tooltipFormatter: (value: number) => string
  yAxisLabel: string
  yMin?: number
  yMax?: number
}

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

const CHART_CONFIGS: Record<LiveChartType, ChartConfig> = {
  cpu: {
    title: 'CPU',
    fields: ['cpu'],
    seriesNames: ['CPU'],
    colors: ['#60a5fa'],
    tooltipFormatter: (v) => `${v.toFixed(2)}%`,
    yAxisLabel: '%',
    yMin: 0,
    yMax: 100,
  },
  ram: {
    title: 'RAM',
    fields: ['mem_used'],
    seriesNames: ['Użyta pamięć'],
    colors: ['#a78bfa'],
    tooltipFormatter: fmtBytes,
    yAxisLabel: '',
    yMin: 0,
  },
  disk: {
    title: 'Dysk I/O',
    fields: ['disk_read_bytes_per_sec', 'disk_write_bytes_per_sec'],
    seriesNames: ['Odczyt', 'Zapis'],
    colors: ['#34d399', '#f87171'],
    tooltipFormatter: fmtBytesPerSec,
    yAxisLabel: '',
    yMin: 0,
  },
  net: {
    title: 'Sieć',
    fields: ['net_rx_bytes_per_sec', 'net_tx_bytes_per_sec'],
    seriesNames: ['RX', 'TX'],
    colors: ['#38bdf8', '#fb923c'],
    tooltipFormatter: fmtBytesPerSec,
    yAxisLabel: '',
    yMin: 0,
  },
}

// ---------------------------------------------------------------------------
// Komponent
// ---------------------------------------------------------------------------

export function LiveChart({ points, type }: LiveChartProps) {
  const chartRef = useRef<ReactECharts>(null)
  const config = CHART_CONFIGS[type]

  // Stabilna ref na tooltip formatter
  const tooltipFmtRef = useRef(config.tooltipFormatter)
  tooltipFmtRef.current = config.tooltipFormatter

  // Inicjalne opcje — ustawiane raz przy mount, NIGDY nie zmieniane przez props
  const initialOption = useMemo<EChartsOption>(() => {
    const showLegend = config.fields.length > 1
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
          let html = `<div style="margin-bottom:4px">${list[0].axisValue}</div>`
          for (const item of list) {
            html += `<div>${item.seriesName}: <b>${tooltipFmtRef.current(item.value)}</b></div>`
          }
          return html
        },
      },
      legend: showLegend
        ? {
            show: true,
            textStyle: { color: '#a1a1aa', fontSize: 11 },
            top: 4,
            right: 16,
          }
        : { show: false },
      xAxis: {
        type: 'category',
        data: [] as string[],
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
      series: config.fields.map((_, i) => ({
        name: config.seriesNames[i],
        type: 'line' as const,
        data: [] as number[],
        smooth: true,
        showSymbol: false,
        lineStyle: { width: 2 },
        areaStyle: { opacity: 0.1 },
        color: config.colors[i],
      })),
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [type])

  // Aktualizuj dane co 1s przez setOption (merge mode — zachowuje yAxis, grid, itp.)
  useEffect(() => {
    const instance = chartRef.current?.getEchartsInstance()
    if (!instance) return

    const timestamps = points.map((p) => formatTimestamp(p.timestamp, '1m'))

    const series = config.fields.map((field, i) => ({
      name: config.seriesNames[i],
      type: 'line' as const,
      data: points.map((p) => p[field] as number),
      smooth: true,
      showSymbol: false,
      lineStyle: { width: 2 },
      areaStyle: { opacity: 0.1 },
      color: config.colors[i],
    }))

    instance.setOption(
      { xAxis: { data: timestamps }, series },
      { notMerge: false, lazyUpdate: false },
    )
  }, [points, config])

  return (
    <ReactECharts
      ref={chartRef}
      option={initialOption}
      style={{ width: '100%', height: '200px' }}
      notMerge={false}
      lazyUpdate={true}
      theme="dark"
    />
  )
}
