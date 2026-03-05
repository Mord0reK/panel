import { NextRequest, NextResponse } from 'next/server'
import { backendFetch } from '@/lib/backend'

export async function POST(
  request: NextRequest,
  { params }: { params: Promise<{ service: string }> }
) {
  const { service } = await params
  const body = await request.json().catch(() => ({}))

  const res = await backendFetch(`/api/services/${service}/test`, {
    method: 'POST',
    body: JSON.stringify(body),
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
