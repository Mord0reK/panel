import { NextRequest, NextResponse } from 'next/server'
import { backendFetch } from '@/lib/backend'

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ service: string }> }
) {
  const { service } = await params

  const res = await backendFetch(`/api/services/${service}/config`, {
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

export async function PUT(
  request: NextRequest,
  { params }: { params: Promise<{ service: string }> }
) {
  const { service } = await params
  const body = await request.json().catch(() => ({}))

  const res = await backendFetch(`/api/services/${service}/config`, {
    method: 'PUT',
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
