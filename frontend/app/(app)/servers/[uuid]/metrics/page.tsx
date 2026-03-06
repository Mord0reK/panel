import { redirect } from 'next/navigation'

import { MetricsGrid } from '@/components/metrics/MetricsGrid'
import { ServerInfo } from '@/components/servers/ServerInfo'
import { backendFetch } from '@/lib/backend'
import type { ServerDetailResponse } from '@/types'

export default async function MetricsPage({
  params,
}: {
  params: Promise<{ uuid: string }>
}) {
  const { uuid } = await params

  const res = await backendFetch(`/api/servers/${uuid}`)
  if (res.status === 401) redirect('/login')
  if (!res.ok) throw new Error(`Failed to fetch server: ${res.status}`)

  const detail = (await res.json()) as ServerDetailResponse

  return (
    <section className="space-y-4 sm:space-y-6">
      {/* Sekcja statyczna — Server Component */}
      <ServerInfo server={detail.server} />

      {/* Sekcja wykresów — Client Component */}
      <MetricsGrid uuid={uuid} containers={detail.containers ?? []} />
    </section>
  )
}
