import { NextRequest, NextResponse } from 'next/server'
import { backendFetch } from '@/lib/backend'

export async function POST(
  request: NextRequest,
  { params }: { params: Promise<{ uuid: string }> },
) {
  const { uuid } = await params
  const body = await request.json()

  const res = await backendFetch(`/api/servers/${uuid}/command`, {
    method: 'POST',
    body: JSON.stringify(body),
  })

  const data = await res.json().catch(() => null)
  return NextResponse.json(data, { status: res.status })
}
