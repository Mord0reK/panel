'use client'

import { useMemo, useState } from 'react'
import { useRouter } from 'next/navigation'
import { PlayIcon, SquareIcon, RotateCwIcon } from 'lucide-react'

import { ContainerActions } from '@/components/containers/ContainerActions'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { useServerMetrics } from '@/hooks/useServerMetrics'
import { api } from '@/lib/api'
import { formatRelativeTime } from '@/lib/formatters'
import type { Container } from '@/types'

interface ContainersTableProps {
  uuid: string
  containers: Container[]
}

// ---------------------------------------------------------------------------
// Uptime helper
// ---------------------------------------------------------------------------

// Parses Docker's raw status string into a short Polish uptime string.
// "Up 2 hours" → "Od 2h"
// "Up 3 minutes" → "Od 3min"
// "Up 5 seconds" → "Od 5s"
// "Exited (0) 2 hours ago" → "" (not running, don't show)
function parseUptime(status: string): string {
  if (!status) return ''
  const m = status.match(/^Up (\d+)\s+(second|minute|hour|day|week|month)/)
  if (!m) return ''
  const n = parseInt(m[1], 10)
  const unit = m[2]
  if (unit.startsWith('second')) return `Od ${n}s`
  if (unit.startsWith('minute')) return `Od ${n}min`
  if (unit.startsWith('hour')) return `Od ${n}h`
  if (unit.startsWith('day')) return `Od ${n}d`
  if (unit.startsWith('week')) return `Od ${n}tyg`
  if (unit.startsWith('month')) return `Od ${n}mies`
  return ''
}

// ---------------------------------------------------------------------------
// State badge
// ---------------------------------------------------------------------------

const STATE_LABELS: Record<string, string> = {
  running: 'Uruchomiony',
  exited: 'Wyłączony',
  stopped: 'Zatrzymany',
  paused: 'Wstrzymany',
  restarting: 'Restartowanie',
  removing: 'Usuwanie',
  removed: 'Usunięty',
  created: 'Utworzony',
  dead: 'Martwy',
}

function StateBadge({ state }: { state: string }) {
  const label = STATE_LABELS[state] ?? state

  switch (state) {
    case 'running':
      return (
        <span className="inline-flex items-center gap-1 rounded-full bg-emerald-500/10 px-2 py-0.5 text-xs font-medium text-emerald-400 ring-1 ring-emerald-500/20">
          <span className="size-1.5 rounded-full bg-emerald-400" />
          {label}
        </span>
      )
    case 'exited':
    case 'stopped':
      return (
        <span className="inline-flex items-center gap-1 rounded-full bg-zinc-800 px-2 py-0.5 text-xs font-medium text-zinc-400 ring-1 ring-zinc-700">
          <span className="size-1.5 rounded-full bg-zinc-500" />
          {label}
        </span>
      )
    case 'paused':
      return (
        <span className="inline-flex items-center gap-1 rounded-full bg-amber-500/10 px-2 py-0.5 text-xs font-medium text-amber-400 ring-1 ring-amber-500/20">
          <span className="size-1.5 rounded-full bg-amber-400" />
          {label}
        </span>
      )
    case 'restarting':
      return (
        <span className="inline-flex items-center gap-1 rounded-full bg-blue-500/10 px-2 py-0.5 text-xs font-medium text-blue-400 ring-1 ring-blue-500/20">
          <span className="size-1.5 rounded-full bg-blue-400" />
          {label}
        </span>
      )
    case 'removing':
    case 'removed':
      return (
        <span className="inline-flex items-center gap-1 rounded-full bg-zinc-800/60 px-2 py-0.5 text-xs font-medium text-zinc-600 ring-1 ring-zinc-700/50">
          <span className="size-1.5 rounded-full bg-zinc-600" />
          {label}
        </span>
      )
    case 'created':
      return (
        <span className="inline-flex items-center gap-1 rounded-full bg-cyan-500/10 px-2 py-0.5 text-xs font-medium text-cyan-400 ring-1 ring-cyan-500/20">
          <span className="size-1.5 rounded-full bg-cyan-400" />
          {label}
        </span>
      )
    case 'dead':
      return (
        <span className="inline-flex items-center gap-1 rounded-full bg-red-500/10 px-2 py-0.5 text-xs font-medium text-red-400 ring-1 ring-red-500/20">
          <span className="size-1.5 rounded-full bg-red-500" />
          {label}
        </span>
      )
    default:
      return state ? (
        <span className="inline-flex items-center gap-1 rounded-full bg-zinc-800 px-2 py-0.5 text-xs font-medium text-zinc-500 ring-1 ring-zinc-700">
          {label}
        </span>
      ) : null
  }
}

// ---------------------------------------------------------------------------
// Health badge
// ---------------------------------------------------------------------------

function HealthBadge({ health }: { health: string }) {
  switch (health) {
    case 'healthy':
      return (
        <span className="inline-flex items-center gap-1 rounded-full bg-emerald-500/10 px-2 py-0.5 text-xs font-medium text-emerald-400 ring-1 ring-emerald-500/20">
          <span className="size-1.5 rounded-full bg-emerald-400" />
          Zdrowy
        </span>
      )
    case 'unhealthy':
      return (
        <span className="inline-flex items-center gap-1 rounded-full bg-red-500/10 px-2 py-0.5 text-xs font-medium text-red-400 ring-1 ring-red-500/20">
          <span className="size-1.5 rounded-full bg-red-400" />
          Niezdrowy
        </span>
      )
    case 'starting':
      return (
        <span className="inline-flex items-center gap-1 rounded-full bg-amber-500/10 px-2 py-0.5 text-xs font-medium text-amber-400 ring-1 ring-amber-500/20">
          <span className="size-1.5 rounded-full bg-amber-400" />
          Startuje
        </span>
      )
    default:
      return (
        <span className="inline-flex items-center gap-1 rounded-full bg-zinc-800 px-2 py-0.5 text-xs font-medium text-zinc-500 ring-1 ring-zinc-700">
          Nieznany
        </span>
      )
  }
}

// ---------------------------------------------------------------------------
// Project group header row
// ---------------------------------------------------------------------------

type ComposeAction = 'compose-stop' | 'compose-start' | 'compose-restart'

interface ProjectGroupHeaderProps {
  uuid: string
  projectName: string
  containerCount: number
  isStandalone?: boolean
  showActions?: boolean
}

function ProjectGroupHeader({ uuid, projectName, containerCount, isStandalone, showActions }: ProjectGroupHeaderProps) {
  const [pending, setPending] = useState<ComposeAction | null>(null)

  async function handleAction(action: ComposeAction) {
    setPending(action)
    try {
      await api.serverCommand(uuid, { action, target: projectName })
    } finally {
      setPending(null)
    }
  }

  return (
    <tr className="border-b border-zinc-700/60 bg-zinc-900/80">
      <td colSpan={5} className="px-4 py-2">
        <div className="flex items-center gap-2">
          <span className="text-xs font-semibold uppercase tracking-wider text-zinc-400">
            {isStandalone ? 'Standalone' : projectName}
          </span>
          <span className="rounded-full bg-zinc-700/60 px-1.5 py-0.5 text-xs text-zinc-500">
            {containerCount}
          </span>
        </div>
      </td>
      <td className="px-4 py-2" />
      <td className="px-4 py-2 text-right">
        {showActions && (
          <div className="flex items-center justify-end gap-1">
            <Button
              variant="outline"
              size="sm"
              disabled={pending !== null}
              onClick={() => handleAction('compose-start')}
              className="h-6 px-2 text-xs"
              title="Uruchom cały projekt"
            >
              <PlayIcon className={`size-3 ${pending === 'compose-start' ? 'animate-spin' : ''}`} />
            </Button>
            <Button
              variant="destructive"
              size="sm"
              disabled={pending !== null}
              onClick={() => handleAction('compose-stop')}
              className="h-6 px-2 text-xs"
              title="Zatrzymaj cały projekt"
            >
              <SquareIcon className={`size-3 ${pending === 'compose-stop' ? 'animate-spin' : ''}`} />
            </Button>
            <Button
              variant="outline"
              size="sm"
              disabled={pending !== null}
              onClick={() => handleAction('compose-restart')}
              className="h-6 px-2 text-xs"
              title="Restartuj cały projekt"
            >
              <RotateCwIcon className={`size-3 ${pending === 'compose-restart' ? 'animate-spin' : ''}`} />
            </Button>
          </div>
        )}
      </td>
    </tr>
  )
}

// ---------------------------------------------------------------------------
// Container row
// ---------------------------------------------------------------------------

function ContainerRow({
  uuid,
  c,
  onDeleted,
  onAction,
}: {
  uuid: string
  c: Container
  onDeleted: (id: string) => void
  onAction: () => void
}) {
  const isRemoved = c.state === 'removed'
  return (
    <tr
      className={[
        'border-b border-zinc-800/50 transition-colors',
        isRemoved ? 'opacity-40 hover:opacity-60' : 'hover:bg-zinc-900/30',
      ].join(' ')}
    >
      <td className="px-4 py-3">
        <div className="font-medium text-zinc-200">{c.name}</div>
      </td>
      <td className="px-4 py-3">
        <div className="flex flex-wrap items-center gap-1.5">
          <StateBadge state={c.state} />
          <HealthBadge health={c.health ?? ''} />
          {c.status && parseUptime(c.status) && (
            <span className="text-xs text-zinc-500 ml-1">{parseUptime(c.status)}</span>
          )}
        </div>
      </td>
      <td className="px-4 py-3">
        <Badge variant="secondary" className="font-mono text-xs">
          {c.image}
        </Badge>
      </td>
      <td className="px-4 py-3 text-zinc-400">{c.service || '—'}</td>
      <td className="px-4 py-3 text-zinc-400" suppressHydrationWarning>
        {formatRelativeTime(c.last_seen)}
      </td>
      <td className="px-4 py-3 text-right" colSpan={2}>
        <ContainerActions
          uuid={uuid}
          containerId={c.container_id}
          containerName={c.name}
          onDeleted={() => onDeleted(c.container_id)}
          onAction={onAction}
        />
      </td>
    </tr>
  )
}

// ---------------------------------------------------------------------------
// Main table
// ---------------------------------------------------------------------------

function ProjectGroup({
  uuid,
  projectKey,
  group,
  isStandalone,
  onDeleted,
  onAction,
}: {
  uuid: string
  projectKey: string
  group: Container[]
  isStandalone: boolean
  onDeleted: (id: string) => void
  onAction: () => void
}) {
  const showActions = !isStandalone && group.length > 1

  return (
    <>
      <ProjectGroupHeader
        uuid={uuid}
        projectName={projectKey}
        containerCount={group.length}
        isStandalone={isStandalone}
        showActions={showActions}
      />
      {group.map((c) => (
        <ContainerRow
          key={c.container_id}
          uuid={uuid}
          c={c}
          onDeleted={onDeleted}
          onAction={onAction}
        />
      ))}
    </>
  )
}

export function ContainersTable({ uuid, containers }: ContainersTableProps) {
  const router = useRouter()
  const [localContainers, setLocalContainers] = useState<Container[]>(containers)

  // Live container states from SSE — merged on top of DB data
  const { containerPoints } = useServerMetrics(uuid)

  // Merge: take DB list, override `state`/`health`/`status` with latest live value when available
  const mergedContainers = useMemo<Container[]>(() => {
    return localContainers.map((c) => {
      const points = containerPoints.get(c.container_id)
      const latest = points?.[points.length - 1]
      if (!latest) return c
      return {
        ...c,
        state: latest.state || c.state,
        health: latest.health || c.health,
        status: latest.status || c.status,
      }
    })
  }, [localContainers, containerPoints])

  function handleDeleted(containerId: string) {
    setLocalContainers((prev) => prev.filter((c) => c.container_id !== containerId))
    router.refresh()
  }

  function handleAction() {
    router.refresh()
  }

  if (mergedContainers.length === 0) {
    return (
      <div className="rounded-lg border border-zinc-800 bg-zinc-900/50 p-8 text-center text-sm text-zinc-500">
        Brak kontenerów
      </div>
    )
  }

  // Group by project; containers without a project go to "__standalone__"
  const grouped = new Map<string, Container[]>()
  for (const c of mergedContainers) {
    const key = c.project || '__standalone__'
    const group = grouped.get(key) ?? []
    group.push(c)
    grouped.set(key, group)
  }

  // Sort: named projects first (alphabetically), standalone last
  const sortedKeys = Array.from(grouped.keys()).sort((a, b) => {
    if (a === '__standalone__') return 1
    if (b === '__standalone__') return -1
    return a.localeCompare(b)
  })

  return (
    <div className="overflow-x-auto rounded-lg border border-zinc-800">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-zinc-800 bg-zinc-900/50 text-left text-xs text-zinc-500">
            <th className="px-4 py-3 font-medium">Nazwa</th>
            <th className="px-4 py-3 font-medium">Stan</th>
            <th className="px-4 py-3 font-medium">Obraz</th>
            <th className="px-4 py-3 font-medium">Serwis</th>
            <th className="px-4 py-3 font-medium">Ostatnio widziany</th>
            <th className="px-4 py-3 text-right font-medium" colSpan={2}>Akcje</th>
          </tr>
        </thead>
        <tbody>
          {sortedKeys.map((key) => {
            const group = grouped.get(key)!
            const isStandalone = key === '__standalone__'
            return (
              <ProjectGroup
                key={key}
                uuid={uuid}
                projectKey={key}
                group={group}
                isStandalone={isStandalone}
                onDeleted={handleDeleted}
                onAction={handleAction}
              />
            )
          })}
        </tbody>
      </table>
    </div>
  )
}
