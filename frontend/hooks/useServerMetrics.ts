'use client'

import { useCallback, useRef, useState } from 'react'

import { useSSE } from '@/hooks/useSSE'
import type {
  LiveServerEvent,
  LiveServerHost,
  LiveServerContainer,
} from '@/types'
import { normalizeLiveHost, normalizeLiveContainer } from '@/types'

const MAX_BUFFER_SIZE = 60

export interface ServerMetricsState {
  /** Bufor ostatnich 60 punktów hosta (snake_case) */
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
 */
export function useServerMetrics(
  uuid: string,
  enabled = true
): ServerMetricsState {
  const [hostPoints, setHostPoints] = useState<LiveServerHost[]>([])
  const [containerPoints, setContainerPoints] = useState<
    Map<string, LiveServerContainer[]>
  >(() => new Map())

  // Ref na bufor kontenerów — unikamy stale closure w onMessage
  const containerBufferRef = useRef<Map<string, LiveServerContainer[]>>(
    new Map()
  )

  const onMessage = useCallback((raw: unknown) => {
    const event = raw as LiveServerEvent
    if (!event?.host) return

    // Host
    const normalizedHost = normalizeLiveHost(event.host)
    setHostPoints((prev) => {
      const next = [...prev, normalizedHost]
      return next.length > MAX_BUFFER_SIZE
        ? next.slice(next.length - MAX_BUFFER_SIZE)
        : next
    })

    // Containers
    if (event.containers && Array.isArray(event.containers)) {
      const buf = containerBufferRef.current

      for (const rawContainer of event.containers) {
        const normalized = normalizeLiveContainer(rawContainer)
        // Używamy indeksu jako klucza (backend nie wysyła container_id w live)
        const key = String(event.containers.indexOf(rawContainer))
        const existing = buf.get(key) ?? []
        const next = [...existing, normalized]
        buf.set(
          key,
          next.length > MAX_BUFFER_SIZE
            ? next.slice(next.length - MAX_BUFFER_SIZE)
            : next
        )
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
