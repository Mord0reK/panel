'use client'

import { useCallback, useState } from 'react'

import { useSSE } from '@/hooks/useSSE'
import type { LiveAllEvent, LiveServerSnapshot } from '@/types'

interface UseLiveAllReturn {
  /** Mapa uuid → ostatni snapshot. Serwer obecny w mapie = online */
  snapshots: Map<string, LiveServerSnapshot>
  /** Czy połączenie SSE jest aktywne */
  connected: boolean
  /** Ostatni błąd SSE (null = brak) */
  error: string | null
}

/**
 * Hook subskrybujący SSE `/api/metrics/live/all`.
 * Utrzymuje mapę `uuid → ostatni LiveServerSnapshot`.
 * Serwery nieobecne w bieżącym evencie traktowane jako offline.
 */
export function useLiveAll(): UseLiveAllReturn {
  const [snapshots, setSnapshots] = useState<Map<string, LiveServerSnapshot>>(
    () => new Map()
  )

  const onMessage = useCallback((raw: unknown) => {
    const event = raw as LiveAllEvent
    if (!event?.servers) return

    const next = new Map<string, LiveServerSnapshot>()
    for (const s of event.servers) {
      next.set(s.uuid, s)
    }
    setSnapshots(next)
  }, [])

  const { connected, error } = useSSE({
    url: '/api/metrics/live/all',
    onMessage,
  })

  return { snapshots, connected, error }
}
