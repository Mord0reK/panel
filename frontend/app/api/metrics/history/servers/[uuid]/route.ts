import { NextRequest, NextResponse } from 'next/server'
import { backendFetch } from '@/lib/backend'

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ uuid: string }> },
) {
  const { uuid } = await params
  const range = request.nextUrl.searchParams.get('range') ?? '1h'

  const res = await backendFetch(
    `/api/metrics/history/servers/${uuid}?range=${range}`,
  )
  const data = await res.json()
  return NextResponse.json(data, { status: res.status })
}
