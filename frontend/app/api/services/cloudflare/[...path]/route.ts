import { NextRequest, NextResponse } from 'next/server'
import { backendFetch } from '@/lib/backend'

// Catch-all for /api/services/cloudflare/* paths not handled by static routes.
// Ensures /test, /config, /stats still proxy correctly after the cloudflare/
// static directory was created (static dirs shadow [service] dynamic routes).
async function handler(
  request: NextRequest,
  { params }: { params: Promise<{ path: string[] }> }
) {
  const { path } = await params
  const endpoint = `/api/services/cloudflare/${path.join('/')}`

  const body =
    request.method !== 'GET' && request.method !== 'HEAD'
      ? await request.text().catch(() => null)
      : null

  const res = await backendFetch(endpoint, {
    method: request.method,
    ...(body != null ? { body } : {}),
  })

  const text = await res.text()
  let data: unknown
  try {
    data = JSON.parse(text)
  } catch {
    data = { error: text.trim() }
  }

  return NextResponse.json(data, { status: res.status })
}

export { handler as GET, handler as POST, handler as PUT, handler as DELETE, handler as PATCH }
