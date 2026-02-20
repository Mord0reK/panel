import { renderHook, act } from '@testing-library/react'

import { useLiveAll } from '@/hooks/useLiveAll'
import type { LiveAllEvent } from '@/types'

// --- Mock useSSE ---

let capturedOnMessage: ((data: unknown) => void) | null = null
let mockEnabled = true
const mockSSEReturn = { connected: true, error: null }

jest.mock('@/hooks/useSSE', () => ({
  useSSE: ({ onMessage, enabled }: { url: string; onMessage: (data: unknown) => void; enabled?: boolean }) => {
    capturedOnMessage = onMessage
    mockEnabled = enabled ?? true
    return mockSSEReturn
  },
}))

describe('useLiveAll', () => {
  beforeEach(() => {
    capturedOnMessage = null
  })

  it('starts with empty snapshots map', () => {
    const { result } = renderHook(() => useLiveAll())

    expect(result.current.snapshots.size).toBe(0)
  })

  it('exposes connected state from useSSE', () => {
    const { result } = renderHook(() => useLiveAll())

    expect(result.current.connected).toBe(true)
  })

  it('updates snapshots map on SSE message', () => {
    const { result } = renderHook(() => useLiveAll())

    const event: LiveAllEvent = {
      servers: [
        {
          uuid: 'uuid-1',
          hostname: 'web-01',
          cpu: 50,
          memory: 1_000_000,
          disk_read_bytes_per_sec: 0,
          disk_write_bytes_per_sec: 0,
          net_rx_bytes_per_sec: 100,
          net_tx_bytes_per_sec: 200,
        },
        {
          uuid: 'uuid-2',
          hostname: 'db-01',
          cpu: 80,
          memory: 2_000_000,
          disk_read_bytes_per_sec: 0,
          disk_write_bytes_per_sec: 0,
          net_rx_bytes_per_sec: 300,
          net_tx_bytes_per_sec: 400,
        },
      ],
    }

    act(() => {
      capturedOnMessage?.(event)
    })

    expect(result.current.snapshots.size).toBe(2)
    expect(result.current.snapshots.get('uuid-1')?.hostname).toBe('web-01')
    expect(result.current.snapshots.get('uuid-2')?.cpu).toBe(80)
  })

  it('replaces entire map on each event (server disappears = offline)', () => {
    const { result } = renderHook(() => useLiveAll())

    // First event: 2 servers
    act(() => {
      capturedOnMessage?.({
        servers: [
          {
            uuid: 'uuid-1',
            hostname: 'web-01',
            cpu: 10,
            memory: 100,
            disk_read_bytes_per_sec: 0,
            disk_write_bytes_per_sec: 0,
            net_rx_bytes_per_sec: 0,
            net_tx_bytes_per_sec: 0,
          },
          {
            uuid: 'uuid-2',
            hostname: 'db-01',
            cpu: 20,
            memory: 200,
            disk_read_bytes_per_sec: 0,
            disk_write_bytes_per_sec: 0,
            net_rx_bytes_per_sec: 0,
            net_tx_bytes_per_sec: 0,
          },
        ],
      })
    })

    expect(result.current.snapshots.size).toBe(2)

    // Second event: only 1 server (uuid-2 went offline)
    act(() => {
      capturedOnMessage?.({
        servers: [
          {
            uuid: 'uuid-1',
            hostname: 'web-01',
            cpu: 15,
            memory: 150,
            disk_read_bytes_per_sec: 0,
            disk_write_bytes_per_sec: 0,
            net_rx_bytes_per_sec: 0,
            net_tx_bytes_per_sec: 0,
          },
        ],
      })
    })

    expect(result.current.snapshots.size).toBe(1)
    expect(result.current.snapshots.has('uuid-2')).toBe(false)
  })

  it('ignores events without servers array', () => {
    const { result } = renderHook(() => useLiveAll())

    act(() => {
      capturedOnMessage?.({})
    })

    expect(result.current.snapshots.size).toBe(0)

    act(() => {
      capturedOnMessage?.(null)
    })

    expect(result.current.snapshots.size).toBe(0)
  })

  it('handles empty servers array', () => {
    const { result } = renderHook(() => useLiveAll())

    act(() => {
      capturedOnMessage?.({ servers: [] })
    })

    expect(result.current.snapshots.size).toBe(0)
  })
})
