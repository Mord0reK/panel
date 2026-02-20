'use client'

import dynamic from 'next/dynamic'
import { useCallback, useEffect, useState } from 'react'
import type { ComponentType } from 'react'

import { RangeDropdown } from '@/components/metrics/RangeDropdown'
import { Skeleton } from '@/components/ui/skeleton'
import { useServerMetrics } from '@/hooks/useServerMetrics'
import { api } from '@/lib/api'
import type {
  AggregatedHostMetricPoint,
  LiveServerHost,
  MetricRange,
} from '@/types'

type ChartType = 'cpu' | 'ram' | 'disk' | 'net'

const ChartSkeleton = () => <Skeleton className="h-50 w-full" />

// Ładuj wykresy bez SSR — ECharts wymaga DOM i powoduje hydration mismatch
const LiveChart = dynamic(
  () => import('@/components/charts/LiveChart').then((m) => m.LiveChart),
  { ssr: false, loading: ChartSkeleton },
) as ComponentType<{ points: LiveServerHost[]; type: ChartType }>

const HistoryChart = dynamic(
  () => import('@/components/charts/HistoryChart').then((m) => m.HistoryChart),
  { ssr: false, loading: ChartSkeleton },
) as ComponentType<{
  points: AggregatedHostMetricPoint[]
  type: ChartType
  range: MetricRange
}>

const CHART_TYPES = ['cpu', 'ram', 'disk', 'net'] as const

interface MetricsGridProps {
  uuid: string
}

export function MetricsGrid({ uuid }: MetricsGridProps) {
  const [range, setRange] = useState<MetricRange>('1m')
  const isLive = range === '1m'

  // Live data — SSE
  const { hostPoints, connected, error } = useServerMetrics(uuid, isLive)

  // History data — jednorazowy fetch
  const [historyPoints, setHistoryPoints] = useState<
    AggregatedHostMetricPoint[]
  >([])
  const [historyLoading, setHistoryLoading] = useState(false)
  const [historyError, setHistoryError] = useState<string | null>(null)

  const fetchHistory = useCallback(
    async (r: MetricRange) => {
      setHistoryLoading(true)
      setHistoryError(null)
      try {
        const data = await api.getMetricsHistory(uuid, r)
        setHistoryPoints(
          (data.host?.points ?? []) as AggregatedHostMetricPoint[],
        )
      } catch (err) {
        setHistoryError(
          err instanceof Error ? err.message : 'Błąd ładowania danych',
        )
        setHistoryPoints([])
      } finally {
        setHistoryLoading(false)
      }
    },
    [uuid],
  )

  // Przy zmianie zakresu na historyczny — ładuj dane
  useEffect(() => {
    if (!isLive) {
      fetchHistory(range)
    }
  }, [range, isLive, fetchHistory])

  const handleRangeChange = (newRange: MetricRange) => {
    setRange(newRange)
  }

  return (
    <div className="space-y-4">
      {/* Header — dropdown + status */}
      <div className="flex items-center gap-5">
        <RangeDropdown value={range} onChange={handleRangeChange} />

        {isLive && (
          <div className="flex items-center gap-2 text-xs text-zinc-500">
            {connected ? (
              <>
                <span className="inline-block size-2 rounded-full bg-red-500 animate-pulse" />
                Na żywo
              </>
            ) : error ? (
              <>
                <span className="inline-block size-2 rounded-full bg-red-400" />
                Błąd połączenia
              </>
            ) : (
              <>
                <span role="status" aria-label="Łączenie" className="inline-flex items-center">
                  <svg className="animate-spin h-3 w-3 text-yellow-400" viewBox="0 0 24 24" fill="none" aria-hidden>
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z"></path>
                  </svg>
                </span>
                Łączenie…
              </>
            )}
          </div>
        )}
      </div>

      {/* Błąd history */}
      {!isLive && historyError && (
        <div className="rounded-md border border-red-900 bg-red-950/50 p-3 text-sm text-red-400">
          {historyError}
        </div>
      )}

      {/* Grid wykresów */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        {CHART_TYPES.map((chartType) => (
          <div
            key={chartType}
            className="rounded-lg border border-zinc-800 bg-zinc-900/50 p-3"
          >
            {isLive ? (
              <LiveChart points={hostPoints} type={chartType} />
            ) : historyLoading ? (
              <Skeleton className="h-50 w-full" />
            ) : (
              <HistoryChart
                points={historyPoints}
                type={chartType}
                range={range}
              />
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
