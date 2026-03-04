'use client'

import { useCallback, useEffect, useRef, useState } from 'react'

import { useSSE } from '@/hooks/useSSE'
import type {
  ContainerHistory,
  LiveServerContainer,
  LiveServerEvent,
  LiveServerHost,
  RawContainerMetricPoint,
  RawHostMetricPoint,
} from '@/types'
import { normalizeLiveHost, normalizeLiveContainer } from '@/types'

const MAX_BUFFER_SIZE = 65
const CONTAINER_MISSING_GRACE_TICKS = 3

export interface ServerMetricsState {
  /** Ostatnie 60 punktów hosta do wyświetlenia (snake_case) */
  hostPoints: LiveServerHost[]
  /** Bufor ostatnich 60 punktów per kontener — mapa containerId → points */
  containerPoints: Map<string, LiveServerContainer[]>
  /** Czy połączenie SSE jest aktywne */
  connected: boolean
  /** Ostatni błąd SSE */
  error: string | null
}

/**
 * Hook subskrybujący SSE `/api/metrics/live/servers/[uuid]`.
 * Normalizuje PascalCase → snake_case. Utrzymuje sliding window 60 punktów.
 * Na starcie pobiera 1m historię z backendu, by wypełnić wykres od razu.
 */
export function useServerMetrics(
  uuid: string,
  enabled = true
): ServerMetricsState {
  const [hostPoints, setHostPoints] = useState<LiveServerHost[]>([])
  const [containerPoints, setContainerPoints] = useState<
    Map<string, LiveServerContainer[]>
  >(() => new Map())

  // Ref do śledzenia ostatniego timestampa prefilla — unikamy duplikatów z SSE
  const lastPrefillTimestampRef = useRef<number>(0)

  // Ref na bufor kontenerów — unikamy stale closure w onMessage
  const containerBufferRef = useRef<Map<string, LiveServerContainer[]>>(
    new Map()
  )
  const missingContainerTicksRef = useRef<Map<string, number>>(new Map())

  // Reset buffera gdy uuid lub enabled się zmienia
  useEffect(() => {
    if (!enabled) return
    setHostPoints([])
    setContainerPoints(new Map())
    containerBufferRef.current = new Map()
    missingContainerTicksRef.current = new Map()
    lastPrefillTimestampRef.current = 0
  }, [uuid, enabled])

  // Pre-fill buffera z historii ostatniej minuty
  useEffect(() => {
    if (!enabled) return
    let cancelled = false

    async function prefill() {
      try {
        const res = await fetch(
          `/api/metrics/history/servers/${uuid}?range=1m`,
        )
        if (!res.ok || cancelled) return

        const data = (await res.json()) as {
          host?: { points?: RawHostMetricPoint[] }
          containers?: ContainerHistory[]
        }
        const points = data.host?.points ?? []
        if (cancelled) return

        // Prefill kontenerów
        if ((data.containers ?? []).length > 0) {
          const buf = containerBufferRef.current
          const missingTicks = missingContainerTicksRef.current
          for (const c of data.containers!) {
            const mapped: LiveServerContainer[] = (c.points as RawContainerMetricPoint[]).map(
              (p) => ({
                container_id: c.container_id,
                timestamp: p.timestamp,
                cpu: p.cpu,
                mem_used: p.mem_used,
                mem_percent: 0,
                disk_used: p.disk_used,
                disk_percent: 0,
                net_rx: p.net_rx_bytes,
                net_tx: p.net_tx_bytes,
                state: '',
                health: '',
                status: '',
              }),
            )
            buf.set(
              c.container_id,
              mapped.length > MAX_BUFFER_SIZE ? mapped.slice(-MAX_BUFFER_SIZE) : mapped,
            )
            missingTicks.set(c.container_id, 0)
          }
          if (!cancelled) setContainerPoints(new Map(buf))
        }

        if (points.length === 0) return

        const mapped: LiveServerHost[] = points.map((p) => ({
          timestamp: p.timestamp,
          cpu: p.cpu,
          mem_used: p.mem_used,
          mem_percent: p.mem_percent,
          disk_used: p.disk_used ?? 0,
          disk_read_bytes_per_sec: p.disk_read_bytes_per_sec,
          disk_write_bytes_per_sec: p.disk_write_bytes_per_sec,
          net_rx_bytes_per_sec: p.net_rx_bytes_per_sec,
          net_tx_bytes_per_sec: p.net_tx_bytes_per_sec,
          disk_used_percent: p.disk_used_percent ?? 0,
        }))

        const last = mapped[mapped.length - 1]
        lastPrefillTimestampRef.current = last?.timestamp ?? 0

        setHostPoints(
          mapped.length > MAX_BUFFER_SIZE
            ? mapped.slice(-MAX_BUFFER_SIZE)
            : mapped,
        )
      } catch {
        // Nie blokuj SSE gdy prefill się nie powiódł
      }
    }

    prefill()
    return () => {
      cancelled = true
    }
  }, [uuid, enabled])

  const onMessage = useCallback((raw: unknown) => {
    const event = raw as LiveServerEvent
    if (!event?.host) return

    // Host — pomijamy punkty już uwzględnione w prefill
    const normalizedHost = normalizeLiveHost(event.host)
    if (normalizedHost.timestamp > lastPrefillTimestampRef.current) {
      setHostPoints((prev) => {
        const next = [...prev, normalizedHost]
        return next.length > MAX_BUFFER_SIZE
          ? next.slice(next.length - MAX_BUFFER_SIZE)
          : next
      })
    }

    // Containers
    if (event.containers && Array.isArray(event.containers)) {
      const buf = containerBufferRef.current
      const missingTicks = missingContainerTicksRef.current
      const seenContainerIDs = new Set<string>()

      for (const rawContainer of event.containers) {
        const key = rawContainer.ContainerID
        if (!key) continue
        seenContainerIDs.add(key)
        missingTicks.delete(key)

        const normalized = normalizeLiveContainer(rawContainer)
        const existing = buf.get(key) ?? []

        const lastPoint = existing[existing.length - 1]
        const next =
          lastPoint && lastPoint.timestamp === normalized.timestamp
            ? [...existing.slice(0, -1), normalized]
            : [...existing, normalized]

        buf.set(
          key,
          next.length > MAX_BUFFER_SIZE
            ? next.slice(next.length - MAX_BUFFER_SIZE)
            : next
        )
      }

      for (const key of Array.from(buf.keys())) {
        if (seenContainerIDs.has(key)) continue

        const missedTicks = (missingTicks.get(key) ?? 0) + 1
        if (missedTicks >= CONTAINER_MISSING_GRACE_TICKS) {
          buf.delete(key)
          missingTicks.delete(key)
          continue
        }

        missingTicks.set(key, missedTicks)
      }

      setContainerPoints(new Map(buf))
    }
  }, [])

  const { connected, error } = useSSE({
    url: `/api/metrics/live/servers/${uuid}`,
    onMessage,
    enabled,
  })

  return { hostPoints, containerPoints, connected, error }
}
