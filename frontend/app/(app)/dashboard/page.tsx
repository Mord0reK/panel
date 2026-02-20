'use client'

import { useEffect, useState } from 'react'

import { ServerCard } from '@/components/servers/ServerCard'
import { Skeleton } from '@/components/ui/skeleton'
import { useLiveAll } from '@/hooks/useLiveAll'
import { api } from '@/lib/api'
import type { Server } from '@/types'

export default function DashboardPage() {
  const [servers, setServers] = useState<Server[] | null>(null)
  const { snapshots, connected } = useLiveAll()

  useEffect(() => {
    api.getServers().then(setServers).catch(() => {})
  }, [])

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-foreground">Dashboard</h1>
        <span
          className={`flex items-center gap-1.5 text-xs ${
            connected ? 'text-emerald-400' : 'text-muted-foreground'
          }`}
        >
          <span
            className={`inline-block size-2 rounded-full ${
              connected ? 'bg-emerald-400' : 'bg-muted-foreground'
            }`}
          />
          {connected ? 'Live' : 'Łączenie…'}
        </span>
      </div>

      {servers === null ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-48 rounded-lg" />
          ))}
        </div>
      ) : servers.length === 0 ? (
        <p className="text-muted-foreground">
          Brak serwerów. Podłącz agenta, aby rozpocząć monitorowanie.
        </p>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {servers.map((server) => (
            <ServerCard
              key={server.uuid}
              hostname={server.hostname}
              uuid={server.uuid}
              snapshot={snapshots.get(server.uuid) ?? null}
            />
          ))}
        </div>
      )}
    </div>
  )
}
