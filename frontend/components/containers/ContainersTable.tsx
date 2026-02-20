'use client'

import { ContainerActions } from '@/components/containers/ContainerActions'
import { Badge } from '@/components/ui/badge'
import { formatRelativeTime } from '@/lib/formatters'
import type { Container } from '@/types'

interface ContainersTableProps {
  uuid: string
  containers: Container[]
}

export function ContainersTable({ uuid, containers }: ContainersTableProps) {
  if (containers.length === 0) {
    return (
      <div className="rounded-lg border border-zinc-800 bg-zinc-900/50 p-8 text-center text-sm text-zinc-500">
        Brak kontenerów
      </div>
    )
  }

  return (
    <div className="overflow-x-auto rounded-lg border border-zinc-800">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-zinc-800 bg-zinc-900/50 text-left text-xs text-zinc-500">
            <th className="px-4 py-3 font-medium">Nazwa</th>
            <th className="px-4 py-3 font-medium">Obraz</th>
            <th className="px-4 py-3 font-medium">Projekt</th>
            <th className="px-4 py-3 font-medium">Serwis</th>
            <th className="px-4 py-3 font-medium">Ostatnio widziany</th>
            <th className="px-4 py-3 text-right font-medium">Akcje</th>
          </tr>
        </thead>
        <tbody>
          {containers.map((c) => (
            <tr
              key={c.container_id}
              className="border-b border-zinc-800/50 transition-colors hover:bg-zinc-900/30"
            >
              <td className="px-4 py-3 font-medium text-zinc-200">{c.name}</td>
              <td className="px-4 py-3">
                <Badge variant="secondary" className="font-mono text-xs">
                  {c.image}
                </Badge>
              </td>
              <td className="px-4 py-3 text-zinc-400">
                {c.project || '—'}
              </td>
              <td className="px-4 py-3 text-zinc-400">
                {c.service || '—'}
              </td>
              <td
                className="px-4 py-3 text-zinc-400"
                suppressHydrationWarning
              >
                {formatRelativeTime(c.last_seen)}
              </td>
              <td className="px-4 py-3 text-right">
                <ContainerActions uuid={uuid} containerId={c.container_id} />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
