'use client'

import { useCallback, useEffect, useState } from 'react'

import { ServerNavItem } from '@/components/sidebar/ServerNavItem'
import {
  SidebarMenu,
  SidebarMenuSkeleton,
} from '@/components/ui/sidebar'
import { api } from '@/lib/api'
import type { Server } from '@/types'

const POLL_INTERVAL_MS = 30_000

export function ServerNavList() {
  const [servers, setServers] = useState<Server[] | null>(null)

  const fetchServers = useCallback(async () => {
    try {
      const data = await api.getServers()
      setServers(data)
    } catch {
      // Przy błędzie zachowujemy ostatnią znaną listę
    }
  }, [])

  useEffect(() => {
    fetchServers()

    const id = setInterval(fetchServers, POLL_INTERVAL_MS)
    return () => clearInterval(id)
  }, [fetchServers])

  if (servers === null) {
    return (
      <SidebarMenu>
        {Array.from({ length: 3 }).map((_, i) => (
          <SidebarMenuSkeleton key={i} showIcon />
        ))}
      </SidebarMenu>
    )
  }

  if (servers.length === 0) {
    return (
      <SidebarMenu>
        <p className="px-3 py-2 text-xs text-muted-foreground">
          Brak serwerów
        </p>
      </SidebarMenu>
    )
  }

  return (
    <SidebarMenu>
      {servers.map((server) => (
        <ServerNavItem key={server.uuid} server={server} />
      ))}
    </SidebarMenu>
  )
}
