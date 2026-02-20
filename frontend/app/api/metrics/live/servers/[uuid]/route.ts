import { NextRequest } from 'next/server'
import { backendStream } from '@/lib/backend'

export const dynamic = 'force-dynamic'

export async function GET(
  _request: NextRequest,
  { params }: { params: Promise<{ uuid: string }> },
) {
  const { uuid } = await params
  return backendStream(`/api/metrics/live/servers/${uuid}`)
}
