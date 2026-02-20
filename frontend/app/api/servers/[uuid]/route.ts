import { NextRequest, NextResponse } from 'next/server'
import { backendFetch } from '@/lib/backend'

export async function GET(
  _request: NextRequest,
  { params }: { params: Promise<{ uuid: string }> },
) {
  const { uuid } = await params
  const res = await backendFetch(`/api/servers/${uuid}`)
  const data = await res.json()
  return NextResponse.json(data, { status: res.status })
}

export async function DELETE(
  _request: NextRequest,
  { params }: { params: Promise<{ uuid: string }> },
) {
  const { uuid } = await params
  const res = await backendFetch(`/api/servers/${uuid}`, { method: 'DELETE' })
  const data = await res.json()
  return NextResponse.json(data, { status: res.status })
}
