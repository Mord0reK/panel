import {
  CpuIcon,
  HardDriveIcon,
  MemoryStickIcon,
  MonitorIcon,
  ServerIcon,
  TagIcon,
} from 'lucide-react'

import { formatBytes } from '@/lib/formatters'
import type { Server } from '@/types'

interface ServerInfoProps {
  server: Server
}

const items = (server: Server) => [
  { icon: MonitorIcon, label: 'Hostname', value: server.hostname },
  { icon: CpuIcon, label: 'CPU', value: server.cpu_model || '—' },
  {
    icon: TagIcon,
    label: 'Rdzenie',
    value: server.cpu_cores > 0 ? String(server.cpu_cores) : '—',
  },
  {
    icon: MemoryStickIcon,
    label: 'RAM',
    value: server.memory_total > 0 ? formatBytes(server.memory_total) : '—',
  },
  {
    icon: ServerIcon,
    label: 'Platforma',
    value: server.platform || '—',
  },
  {
    icon: HardDriveIcon,
    label: 'Kernel',
    value: server.kernel || '—',
  },
  {
    icon: TagIcon,
    label: 'Architektura',
    value: server.architecture || '—',
  },
]

/**
 * Sekcja informacji statycznych serwera.
 * Server Component — dane z `api.getServer()`.
 */
export function ServerInfo({ server }: ServerInfoProps) {
  return (
    <div className="grid grid-cols-2 gap-x-8 gap-y-3 rounded-lg border border-zinc-800 bg-zinc-900/50 p-5 sm:grid-cols-3 lg:grid-cols-4">
      {items(server).map((item) => (
        <div key={item.label} className="flex items-center gap-2.5">
          <item.icon className="size-4 shrink-0 text-zinc-500" />
          <div className="min-w-0">
            <p className="text-[11px] text-zinc-500">{item.label}</p>
            <p className="truncate text-sm text-zinc-200">{item.value}</p>
          </div>
        </div>
      ))}
    </div>
  )
}
