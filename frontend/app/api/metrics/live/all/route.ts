import { backendStream } from '@/lib/backend'

export const dynamic = 'force-dynamic'

export async function GET() {
  return backendStream('/api/metrics/live/all')
}
