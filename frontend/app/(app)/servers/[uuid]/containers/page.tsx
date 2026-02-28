import { redirect } from 'next/navigation'

import { ContainersPageClient } from './ContainersPageClient'
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

  return <ContainersPageClient uuid={uuid} detail={detail} />
}
