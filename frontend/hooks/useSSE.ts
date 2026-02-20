'use client'

import { useCallback, useEffect, useRef, useState } from 'react'

interface UseSSEOptions {
  /** URL endpointu SSE (np. /api/metrics/live/all) */
  url: string
  /** Callback wywoływany przy każdym evencie SSE (dane JSON) */
  onMessage: (data: unknown) => void
  /** Czy hook jest aktywny (domyślnie true) */
  enabled?: boolean
}

interface UseSSEReturn {
  /** Czy połączenie jest aktywne */
  connected: boolean
  /** Ostatni błąd (null = brak) */
  error: string | null
}

const INITIAL_RETRY_MS = 1_000
const MAX_RETRY_MS = 30_000

/**
 * Hook opakowujący EventSource z auto-reconnect (exponential backoff).
 *
 * Ponieważ SSE przechodzi przez Next.js Route Handler (same-origin),
 * httpOnly cookie `token` jest dołączane automatycznie.
 */
export function useSSE({ url, onMessage, enabled = true }: UseSSEOptions): UseSSEReturn {
  const [connected, setConnected] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const retryMs = useRef(INITIAL_RETRY_MS)
  const retryTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const eventSourceRef = useRef<EventSource | null>(null)
  // Stabilna referencja do callbacka, unikamy re-tworzenia EventSource przy
  // każdej zmianie onMessage
  const onMessageRef = useRef(onMessage)
  onMessageRef.current = onMessage

  const connect = useCallback(() => {
    // Sprzątamy poprzednie
    if (eventSourceRef.current) {
      eventSourceRef.current.close()
      eventSourceRef.current = null
    }

    const es = new EventSource(url)
    eventSourceRef.current = es

    es.onopen = () => {
      setConnected(true)
      setError(null)
      retryMs.current = INITIAL_RETRY_MS
    }

    es.onmessage = (event) => {
      try {
        const data: unknown = JSON.parse(event.data as string)
        onMessageRef.current(data)
      } catch {
        // Ignorujemy nieprawidłowe JSON-y
      }
    }

    es.onerror = () => {
      es.close()
      eventSourceRef.current = null
      setConnected(false)
      setError('SSE connection lost')

      // Exponential backoff
      retryTimer.current = setTimeout(() => {
        retryMs.current = Math.min(retryMs.current * 2, MAX_RETRY_MS)
        connect()
      }, retryMs.current)
    }
  }, [url])

  useEffect(() => {
    if (!enabled) {
      eventSourceRef.current?.close()
      eventSourceRef.current = null
      setConnected(false)
      return
    }

    connect()

    return () => {
      eventSourceRef.current?.close()
      eventSourceRef.current = null
      if (retryTimer.current) clearTimeout(retryTimer.current)
      setConnected(false)
    }
  }, [connect, enabled])

  return { connected, error }
}
