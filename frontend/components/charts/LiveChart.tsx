'use client'

import { useMemo, useRef, useEffect } from 'react'
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
// Helpery formatowania
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

// ---------------------------------------------------------------------------
// Konfiguracja per typ wykresu
// ---------------------------------------------------------------------------

interface ChartConfig {
  title: string
  fields: (keyof LiveServerHost)[]
  seriesNames: string[]
  colors: string[]
  tooltipFormatter: (value: number) => string
  yAxisFormatter: (value: number) => string
  yMin?: number
  yMax?: number
}

const CHART_CONFIGS: Record<LiveChartType, ChartConfig> = {
  cpu: {
    title: 'Wykorzystanie CPU',
    fields: ['cpu'],
    seriesNames: ['CPU'],
    colors: ['#60a5fa'],
    tooltipFormatter: (v) => `${v.toFixed(2)}%`,
    yAxisFormatter: (v) => `${v}%`,
    yMin: 0,
  },
  ram: {
    title: 'Wykorzystanie pamięci RAM',
    fields: ['mem_used'],
    seriesNames: ['Użyta pamięć'],
    colors: ['#31db42'],
    tooltipFormatter: fmtBytes,
    yAxisFormatter: fmtBytes,
    yMin: 0,
  },
  disk: {
    title: 'Przepustowość dysku (I/O)',
    fields: ['disk_read_bytes_per_sec', 'disk_write_bytes_per_sec'],
    seriesNames: ['Odczyt', 'Zapis'],
    colors: ['#34d399', '#f87171'],
    tooltipFormatter: fmtBytesPerSec,
    yAxisFormatter: fmtBytesPerSec,
    yMin: 0,
  },
  net: {
    title: 'Sieć',
    fields: ['net_rx_bytes_per_sec', 'net_tx_bytes_per_sec'],
    seriesNames: ['Odebrane', 'Wysłane'],
    colors: ['#38bdf8', '#fb923c'],
    tooltipFormatter: fmtBytesPerSec,
    yAxisFormatter: fmtBytesPerSec,
    yMin: 0,
  },
}

// ---------------------------------------------------------------------------
// Komponent
// ---------------------------------------------------------------------------

export function LiveChart({ points, type }: LiveChartProps) {
  const config = CHART_CONFIGS[type]
  const showLegend = config.fields.length > 1

  const { tooltipFormatter, yAxisFormatter } = config

  // Śledzi czy animacja wejścia już się odbyła.
  // Zapobiega restartu 500ms animacji gdy nowy punkt pojawi się
  // zanim poprzednia animacja dobiegnie końca.
  const hasAnimatedRef = useRef(false)
  const hasPoints = points.length > 0

  useEffect(() => {
    if (!hasPoints) {
      // Dane wyczyszczone (zmiana serwera) — pozwól na nową animację
      hasAnimatedRef.current = false
    } else {
      // Po pierwszym renderze z danymi oznacz animację jako wykonaną
      hasAnimatedRef.current = true
    }
  }, [hasPoints])

  // useMemo oblicza pełną opcję przy każdej zmianie points/type
  const option = useMemo<EChartsOption>(() => {
    // Pokazuj ostatnie 60 punktów — bufor trzyma 65, 5 extra daje płynniejszą animację scrollowania
    const displayPts = points.slice(-60)
    const timestamps = displayPts.map((p) => formatTimestamp(p.timestamp, '1m'))

    // Animuj 500ms tylko przy pierwszym pojawieniu się danych.
    // hasAnimatedRef.current jest aktualizowany przez useEffect po renderze,
    // więc pierwszy render z danymi ma shouldAnimate=true, każdy kolejny false.
    const shouldAnimate = hasPoints && !hasAnimatedRef.current

    return {
      backgroundColor: 'transparent',
      animation: true,
      animationDuration: shouldAnimate ? 500 : 0,
      animationEasing: 'cubicOut',
      animationDurationUpdate: 0,
      grid: {
        top: showLegend ? 44 : 36,
        right: 16,
        bottom: 32,
        left: 72,
      },
      title: {
        text: config.title,
        textStyle: { color: '#e4e4e7', fontSize: 13, fontWeight: 500 },
        left: 8,
        top: 6,
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
          let html = `<div style="margin-bottom:4px;color:#a1a1aa">${list[0].axisValue}</div>`
          for (const item of list) {
            html += `<div>${item.seriesName}: <b>${tooltipFormatter(item.value)}</b></div>`
          }
          return html
        },
      },
      legend: showLegend
        ? {
            show: true,
            textStyle: { color: '#a1a1aa', fontSize: 11 },
            top: 6,
            right: 16,
          }
        : { show: false },
      xAxis: {
        type: 'category',
        data: timestamps,
        axisLabel: {
          color: '#71717a',
          fontSize: 10,
          interval: 'auto',
          hideOverlap: true,
        },
        axisLine: { lineStyle: { color: '#3f3f46' } },
        splitLine: { show: false },
        boundaryGap: false,
      },
      yAxis: {
        type: 'value',
        min: config.yMin,
        max: config.yMax,
        axisLabel: {
          color: '#71717a',
          fontSize: 10,
          formatter: (value: number) => yAxisFormatter(value),
          width: 64,
          overflow: 'truncate',
        },
        splitLine: { lineStyle: { color: '#27272a' } },
      },
      series: config.fields.map((field, i) => ({
        name: config.seriesNames[i],
        type: 'line' as const,
        data: displayPts.map((p) => p[field] as number),
        smooth: true,
        showSymbol: false,
        lineStyle: { width: 2 },
        areaStyle: { opacity: 0.12 },
        color: config.colors[i],
        emphasis: { disabled: true },
      })),
    }
  }, [points, type, config, showLegend, tooltipFormatter, yAxisFormatter, hasPoints])

  return (
    <ReactECharts
      option={option}
      style={{ width: '100%', height: '220px' }}
      notMerge={false}
      lazyUpdate={false}
    />
  )
}