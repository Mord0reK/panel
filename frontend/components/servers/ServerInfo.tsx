import type { LucideIcon } from 'lucide-react'
import {
  CpuIcon,
  HardDriveIcon,
  MemoryStickIcon,
  MonitorIcon,
  ServerIcon,
  TagIcon,
} from 'lucide-react'

import { cn } from '@/lib/utils'
import { formatBytes } from '@/lib/formatters'
import type { Server } from '@/types'
import { ServerIconDisplay } from './ServerIconDisplay'

interface ServerInfoProps {
  server: Server
}

interface StatItem {
  icon: LucideIcon
  value: string
}

function buildStats(server: Server): StatItem[] {
  const stats: StatItem[] = []

  if (server.hostname) {
    stats.push({ icon: ServerIcon, value: server.hostname })
  }
  if (server.platform) {
    stats.push({ icon: MonitorIcon, value: server.platform })
  }
  if (server.cpu_model && server.cpu_cores > 0 && server.cpu_threads > 0) {
    stats.push({ icon: CpuIcon, value: `${server.cpu_model} (${server.cpu_cores} rdzeni, ${server.cpu_threads} wątków)` })
  }
  if (server.memory_total > 0) {
    stats.push({ icon: MemoryStickIcon, value: formatBytes(server.memory_total) })
  }
  if (server.kernel) {
    stats.push({ icon: HardDriveIcon, value: server.kernel })
  }
  if (server.architecture) {
    stats.push({ icon: TagIcon, value: server.architecture })
  }

  return stats
}

/**
 * Sekcja informacji statycznych serwera — kompaktowy poziomy pasek.
 * Server Component — dane z `api.getServer()`.
 */
export function ServerInfo({ server }: ServerInfoProps) {
  const displayName = server.display_name || server.hostname
  const stats = buildStats(server)

  return (
    <div className="flex items-center gap-4 rounded-lg border border-zinc-800 bg-zinc-900/50 px-5 py-4">
      {/* Ikona serwera */}
      <div className="flex size-11 shrink-0 items-center justify-center rounded-lg bg-zinc-800">
        <ServerIconDisplay icon={server.icon} className="size-6 text-zinc-300" />
      </div>

      {/* Treść */}
      <div className="min-w-0 flex-1">
        <h2 className="truncate text-base font-semibold text-zinc-100">{displayName}</h2>

        <div className="mt-1 flex flex-wrap items-center gap-x-0 text-sm text-zinc-400">
          {/* Status online */}
          {server.online !== undefined && (
            <>
              <span
                className={cn(
                  'flex items-center gap-1.5',
                  server.online ? 'text-emerald-400' : 'text-zinc-500',
                )}
              >
                <span
                  className={cn(
                    'size-1.5 rounded-full',
                    server.online ? 'bg-emerald-400' : 'bg-zinc-500',
                  )}
                />
                {server.online ? 'Działa' : 'Offline'}
              </span>
              {stats.length > 0 && (
                <span className="mx-2.5 text-zinc-700">|</span>
              )}
            </>
          )}

          {/* Statystyki serwera */}
          {stats.map((item, i) => (
            <span key={i} className="flex items-center">
              <span className="flex items-center gap-1.5">
                <item.icon className="size-3.5 shrink-0 text-zinc-500" />
                <span className="text-zinc-400">{item.value}</span>
              </span>
              {i < stats.length - 1 && (
                <span className="mx-2.5 text-zinc-700">|</span>
              )}
            </span>
          ))}
        </div>
      </div>
    </div>
  )
}
