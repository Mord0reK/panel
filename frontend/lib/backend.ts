import { cookies } from 'next/headers'

const BACKEND_URL = process.env.BACKEND_URL ?? 'http://localhost:8080'

/**
 * Server-side helper — wysyła żądanie do backendu z tokenem z httpOnly cookie.
 * Używany wyłącznie w Route Handlerach (server-side).
 */
export async function backendFetch(
  path: string,
  init?: RequestInit
): Promise<Response> {
  const cookieStore = await cookies()
  const token = cookieStore.get('token')?.value

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(init?.headers as Record<string, string>),
  }

  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  return fetch(`${BACKEND_URL}${path}`, {
    ...init,
    headers,
  })
}

/**
 * Server-side helper — strumieniuje SSE z backendu bezpośrednio do klienta.
 * Używany w Route Handlerach dla endpointów /metrics/live/*.
 */
export async function backendStream(path: string): Promise<Response> {
  const cookieStore = await cookies()
  const token = cookieStore.get('token')?.value

  const headers: Record<string, string> = {
    Accept: 'text/event-stream',
    'Cache-Control': 'no-cache',
  }

  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const upstream = await fetch(`${BACKEND_URL}${path}`, { headers })

  if (!upstream.ok || !upstream.body) {
    return new Response('Backend SSE error', { status: upstream.status })
  }

  return new Response(upstream.body, {
    headers: {
      'Content-Type': 'text/event-stream',
      'Cache-Control': 'no-cache, no-transform',
      Connection: 'keep-alive',
      'X-Accel-Buffering': 'no',
    },
  })
}
