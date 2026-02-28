import { redirect } from 'next/navigation'

import { ContainersTable } from '@/components/containers/ContainersTable'
import { backendFetch } from '@/lib/backend'
import type { ServerDetailResponse } from '@/types'

export default async function ContainersPage({
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
    <main className="space-y-6 p-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold text-zinc-100">
          Kontenery — {detail.server.hostname}
        </h1>
      </div>

      <ContainersTable uuid={uuid} containers={detail.containers ?? []} />
    </main>
  )
}
