import { NextRequest, NextResponse } from 'next/server'
import { backendFetch } from '@/lib/backend'

export async function PUT(
  _request: NextRequest,
  { params }: { params: Promise<{ uuid: string }> },
) {
  const { uuid } = await params
  const res = await backendFetch(`/api/servers/${uuid}/approve`, {
    method: 'PUT',
  })
  const data = await res.json()
  return NextResponse.json(data, { status: res.status })
}
