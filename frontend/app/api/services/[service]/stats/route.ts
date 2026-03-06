import { NextResponse } from 'next/server'
import { backendFetch } from '@/lib/backend'

export async function GET(
  _request: Request,
  { params }: { params: Promise<{ service: string }> }
) {
  const { service } = await params

  const res = await backendFetch(`/api/services/${service}/stats`, {
    method: 'GET',
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