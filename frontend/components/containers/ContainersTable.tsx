'use client'

import { useEffect, useMemo, useState } from 'react'
import { useRouter } from 'next/navigation'
import {
  PlayIcon,
  SquareIcon,
  RotateCwIcon,
  CheckSquareIcon,
  XIcon,
  Trash2Icon,
  AlertTriangleIcon,
  ArrowRightIcon,
  RefreshCwIcon,
} from 'lucide-react'

import { ContainerActions } from '@/components/containers/ContainerActions'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useServerMetrics } from '@/hooks/useServerMetrics'
import { api } from '@/lib/api'
import { formatRelativeTime } from '@/lib/formatters'
import type { Container, ContainerAction, ContainerUpdateInfo } from '@/types'

interface ContainersTableProps {
  uuid: string
  containers: Container[]
  bulkMode: boolean
  onToggleBulk: () => void
}

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

const STATE_LABELS: Record<string, string> = {
  running: 'Uruchomiony',
  exited: 'Wyłączony',
  stopped: 'Zatrzymany',
  paused: 'Wstrzymany',
  restarting: 'Restartowanie',
  removing: 'Usuwanie',
  created: 'Utworzony',
  dead: 'Martwy',
}

function StateBadge({ state }: { state: string }) {
  const label = STATE_LABELS[state] ?? state

  switch (state) {
    case 'running':
      return (
        <span className="inline-flex items-center gap-1 rounded-full bg-emerald-500/10 px-2.5 py-1 text-xs font-medium text-emerald-400 ring-1 ring-emerald-500/20">
          <span className="size-2 rounded-full bg-emerald-400" />
          {label}
        </span>
      )
    case 'exited':
    case 'stopped':
      return (
        <span className="inline-flex items-center gap-1 rounded-full bg-zinc-800 px-2.5 py-1 text-xs font-medium text-zinc-400 ring-1 ring-zinc-700">
          <span className="size-2 rounded-full bg-zinc-500" />
          {label}
        </span>
      )
    case 'paused':
      return (
        <span className="inline-flex items-center gap-1 rounded-full bg-amber-500/10 px-2.5 py-1 text-xs font-medium text-amber-400 ring-1 ring-amber-500/20">
          <span className="size-2 rounded-full bg-amber-400" />
          {label}
        </span>
      )
    case 'restarting':
      return (
        <span className="inline-flex items-center gap-1 rounded-full bg-blue-500/10 px-2.5 py-1 text-xs font-medium text-blue-400 ring-1 ring-blue-500/20">
          <span className="size-2 rounded-full bg-blue-400" />
          {label}
        </span>
      )
    case 'removing':
      return (
        <span className="inline-flex items-center gap-1 rounded-full bg-zinc-800/60 px-2.5 py-1 text-xs font-medium text-zinc-600 ring-1 ring-zinc-700/50">
          <span className="size-2 rounded-full bg-zinc-600" />
          {label}
        </span>
      )
    case 'created':
      return (
        <span className="inline-flex items-center gap-1 rounded-full bg-cyan-500/10 px-2.5 py-1 text-xs font-medium text-cyan-400 ring-1 ring-cyan-500/20">
          <span className="size-2 rounded-full bg-cyan-400" />
          {label}
        </span>
      )
    case 'dead':
      return (
        <span className="inline-flex items-center gap-1 rounded-full bg-red-500/10 px-2.5 py-1 text-xs font-medium text-red-400 ring-1 ring-red-500/20">
          <span className="size-2 rounded-full bg-red-500" />
          {label}
        </span>
      )
    default:
      return state ? (
        <span className="inline-flex items-center gap-1 rounded-full bg-zinc-800 px-2.5 py-1 text-xs font-medium text-zinc-500 ring-1 ring-zinc-700">
          {label}
        </span>
      ) : null
  }
}

function HealthBadge({ health, uptime }: { health: string; uptime?: string }) {
  const uptimePart = uptime ? <span className="text-xs text-zinc-500 ml-1">{uptime}</span> : null

  switch (health) {
    case 'healthy':
      return (
        <div className="flex items-center gap-1.5">
          <span className="inline-flex items-center gap-1 rounded-full bg-emerald-500/10 px-2 py-0.5 text-xs font-medium text-emerald-400 ring-1 ring-emerald-500/20">
            <span className="size-1.5 rounded-full bg-emerald-400" />
            Zdrowy
          </span>
          {uptimePart}
        </div>
      )
    case 'unhealthy':
      return (
        <div className="flex items-center gap-1.5">
          <span className="inline-flex items-center gap-1 rounded-full bg-red-500/10 px-2 py-0.5 text-xs font-medium text-red-400 ring-1 ring-red-500/20">
            <span className="size-1.5 rounded-full bg-red-400" />
            Niezdrowy
          </span>
          {uptimePart}
        </div>
      )
    case 'starting':
      return (
        <div className="flex items-center gap-1.5">
          <span className="inline-flex items-center gap-1 rounded-full bg-amber-500/10 px-2 py-0.5 text-xs font-medium text-amber-400 ring-1 ring-amber-500/20">
            <span className="size-1.5 rounded-full bg-amber-400" />
            Startuje
          </span>
          {uptimePart}
        </div>
      )
    default:
      return (
        <div className="flex items-center gap-1.5">
          <span className="text-xs text-zinc-600">Nieznany</span>
          {uptimePart}
        </div>
      )
  }
}

function UpdateBadge({ updateInfo }: { updateInfo?: ContainerUpdateInfo }) {
  if (!updateInfo) return null

  switch (updateInfo.status) {
    case 'up_to_date':
      return (
        <span className="inline-flex items-center gap-1 rounded-full bg-emerald-500/10 px-2 py-0.5 text-xs font-medium text-emerald-400">
          Aktualny
        </span>
      )
    case 'update_available':
      return (
        <span className="inline-flex items-center gap-1 rounded-full bg-amber-500/10 px-2 py-0.5 text-xs font-medium text-amber-400 ring-1 ring-amber-500/20">
          <RefreshCwIcon className="size-3" />
          Aktualizacja
        </span>
      )
    case 'local':
      return (
        <span className="inline-flex items-center gap-1 rounded-full bg-zinc-700/50 px-2 py-0.5 text-xs font-medium text-zinc-400">
          Lokalny
        </span>
      )
    case 'rate_limited':
      return (
        <span className="inline-flex items-center gap-1 rounded-full bg-orange-500/10 px-2 py-0.5 text-xs font-medium text-orange-400">
          Rate limit
        </span>
      )
    default:
      return null
  }
}

function BulkCheckboxCell({
  bulkMode,
  checked,
  onCheckedChange,
  ariaLabel,
}: {
  bulkMode: boolean
  checked: boolean | 'indeterminate'
  onCheckedChange: (val: boolean) => void
  ariaLabel: string
}) {
  return (
    <td
      className={[
        'transition-all duration-200 overflow-hidden',
        bulkMode ? 'w-10 px-4 opacity-100' : 'w-0 px-0 opacity-0 pointer-events-none',
      ].join(' ')}
    >
      <Checkbox
        checked={checked}
        onCheckedChange={(val) => onCheckedChange(!!val)}
        aria-label={ariaLabel}
        tabIndex={bulkMode ? 0 : -1}
      />
    </td>
  )
}

function BulkCheckboxTh({
  bulkMode,
  checked,
  onCheckedChange,
  ariaLabel,
}: {
  bulkMode: boolean
  checked: boolean | 'indeterminate'
  onCheckedChange: (val: boolean) => void
  ariaLabel: string
}) {
  return (
    <th
      className={[
        'transition-all duration-200 overflow-hidden',
        bulkMode ? 'w-10 px-4 opacity-100' : 'w-0 px-0 opacity-0 pointer-events-none',
      ].join(' ')}
    >
      <Checkbox
        checked={checked}
        onCheckedChange={(val) => onCheckedChange(!!val)}
        aria-label={ariaLabel}
        tabIndex={bulkMode ? 0 : -1}
      />
    </th>
  )
}

type ComposeAction = 'compose-stop' | 'compose-start' | 'compose-restart'

interface ProjectGroupHeaderProps {
  uuid: string
  projectName: string
  containerCount: number
  isStandalone?: boolean
  showActions?: boolean
  bulkMode: boolean
  allSelected: boolean
  someSelected: boolean
  onSelectAll: (checked: boolean) => void
}

function ProjectGroupHeader({
  uuid,
  projectName,
  containerCount,
  isStandalone,
  showActions,
  bulkMode,
  allSelected,
  someSelected,
  onSelectAll,
}: ProjectGroupHeaderProps) {
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
      <BulkCheckboxCell
        bulkMode={bulkMode}
        checked={someSelected ? (allSelected ? true : 'indeterminate') : false}
        onCheckedChange={onSelectAll}
        ariaLabel="Zaznacz wszystkie w grupie"
      />
      <td colSpan={5} className="px-3 py-2 sm:px-4">
        <div className="flex items-center gap-2">
          <span className="text-xs font-semibold uppercase tracking-wider text-zinc-400">
            {isStandalone ? 'Standalone' : projectName}
          </span>
          <span className="rounded-full bg-zinc-700/60 px-1.5 py-0.5 text-xs text-zinc-500">
            {containerCount}
          </span>
        </div>
      </td>
      <td className="px-3 py-2 text-right sm:px-4">
        {showActions && !bulkMode && (
          <div className="flex flex-wrap items-center justify-end gap-1">
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

interface ContainerCardProps {
  uuid: string
  c: Container
  updateInfo?: ContainerUpdateInfo
  onDeleted: (id: string) => void
  onAction: () => void
}

function ContainerCard({ uuid, c, updateInfo, onDeleted, onAction }: ContainerCardProps) {
  const uptime = c.status ? parseUptime(c.status) : ''

  return (
    <div className="rounded-lg border border-zinc-800 bg-zinc-900/50 p-4 space-y-3">
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0 flex-1">
          <h3 className="font-medium text-zinc-200 truncate">{c.name}</h3>
          {c.project && (
            <p className="text-xs text-zinc-500 mt-0.5">
              {c.project}/{c.service}
            </p>
          )}
        </div>
        <ContainerActions
          uuid={uuid}
          containerId={c.container_id}
          containerName={c.name}
          onDeleted={() => onDeleted(c.container_id)}
          onAction={onAction}
        />
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <StateBadge state={c.state} />
        <HealthBadge health={c.health ?? ''} uptime={uptime} />
      </div>

      {uptime && (
        <p className="text-xs text-zinc-500">{c.status}</p>
      )}

      <div className="space-y-1.5 pt-1">
        <Badge variant="secondary" className="font-mono text-xs block w-full text-left truncate">
          {c.image}
        </Badge>
        {updateInfo && (
          <div className="flex items-center gap-2 flex-wrap">
            {updateInfo.current_version && (
              <span className="text-xs text-zinc-400">{updateInfo.current_version}</span>
            )}
            {updateInfo.update_available && updateInfo.latest_version && (
              <>
                <ArrowRightIcon className="size-3 text-amber-400" />
                <span className="text-xs text-amber-400 font-medium">{updateInfo.latest_version}</span>
              </>
            )}
            <UpdateBadge updateInfo={updateInfo} />
          </div>
        )}
      </div>
    </div>
  )
}

interface ContainerRowProps {
  uuid: string
  c: Container
  updateInfo?: ContainerUpdateInfo
  onDeleted: (id: string) => void
  onAction: () => void
  bulkMode: boolean
  selected: boolean
  onSelect: (id: string, checked: boolean) => void
}

function ContainerRow({
  uuid,
  c,
  updateInfo,
  onDeleted,
  onAction,
  bulkMode,
  selected,
  onSelect,
}: ContainerRowProps) {
  const uptime = c.status ? parseUptime(c.status) : ''

  return (
    <tr
      className={[
        'border-b border-zinc-800/50 transition-colors hover:bg-zinc-900/30',
        bulkMode && selected ? 'bg-zinc-800/40' : '',
      ]
        .filter(Boolean)
        .join(' ')}
    >
      <BulkCheckboxCell
        bulkMode={bulkMode}
        checked={selected}
        onCheckedChange={(val) => onSelect(c.container_id, val)}
        ariaLabel={`Zaznacz ${c.name}`}
      />
      <td className="px-3 py-3 sm:px-4">
        <div className="space-y-1">
          <div className="font-medium text-zinc-200">{c.name}</div>
          <div className="text-xs text-zinc-500 lg:hidden">{c.service || '—'}</div>
        </div>
      </td>
      <td className="px-3 py-3 sm:px-4">
        <StateBadge state={c.state} />
      </td>
      <td className="hidden px-4 py-3 lg:table-cell">
        <HealthBadge health={c.health ?? ''} uptime={uptime} />
      </td>
      <td className="hidden px-4 py-3 xl:table-cell">
        <div className="space-y-1">
          <Badge variant="secondary" className="font-mono text-xs block max-w-[200px] truncate">
            {c.image}
          </Badge>
          {updateInfo && (
            <div className="flex items-center gap-1">
              {updateInfo.current_version && (
                <span className="text-xs text-zinc-500">{updateInfo.current_version}</span>
              )}
              {updateInfo.update_available && updateInfo.latest_version && (
                <>
                  <ArrowRightIcon className="size-3 text-amber-400" />
                  <span className="text-xs text-amber-400">{updateInfo.latest_version}</span>
                </>
              )}
              <UpdateBadge updateInfo={updateInfo} />
            </div>
          )}
        </div>
      </td>
      <td className="hidden px-4 py-3 text-zinc-400 lg:table-cell">{c.service || '—'}</td>
      <td className="px-3 py-3 text-right sm:px-4">
        {!bulkMode && (
          <ContainerActions
            uuid={uuid}
            containerId={c.container_id}
            containerName={c.name}
            onDeleted={() => onDeleted(c.container_id)}
            onAction={onAction}
          />
        )}
      </td>
    </tr>
  )
}

interface ProjectGroupProps {
  uuid: string
  projectKey: string
  group: Container[]
  updateInfos: Map<string, ContainerUpdateInfo>
  isStandalone: boolean
  onDeleted: (id: string) => void
  onAction: () => void
  bulkMode: boolean
  selectedIds: Set<string>
  onSelect: (id: string, checked: boolean) => void
  onSelectGroup: (ids: string[], checked: boolean) => void
}

function ProjectGroup({
  uuid,
  projectKey,
  group,
  updateInfos,
  isStandalone,
  onDeleted,
  onAction,
  bulkMode,
  selectedIds,
  onSelect,
  onSelectGroup,
}: ProjectGroupProps) {
  const showActions = !isStandalone && group.length > 1
  const groupIds = group.map((c) => c.container_id)
  const allSelected = groupIds.every((id) => selectedIds.has(id))
  const someSelected = groupIds.some((id) => selectedIds.has(id))

  return (
    <>
      <ProjectGroupHeader
        uuid={uuid}
        projectName={projectKey}
        containerCount={group.length}
        isStandalone={isStandalone}
        showActions={showActions}
        bulkMode={bulkMode}
        allSelected={allSelected}
        someSelected={someSelected}
        onSelectAll={(checked) => onSelectGroup(groupIds, checked)}
      />
      {group.map((c) => (
        <ContainerRow
          key={c.container_id}
          uuid={uuid}
          c={c}
          updateInfo={updateInfos.get(c.container_id)}
          onDeleted={onDeleted}
          onAction={onAction}
          bulkMode={bulkMode}
          selected={selectedIds.has(c.container_id)}
          onSelect={onSelect}
        />
      ))}
    </>
  )
}

interface BulkActionBarProps {
  uuid: string
  selectedIds: Set<string>
  onDone: () => void
  onBulkDeleted: (deletedIds: string[]) => void
}

function BulkActionBar({ uuid, selectedIds, onDone, onBulkDeleted }: BulkActionBarProps) {
  const [pending, setPending] = useState<ContainerAction | null>(null)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [deletePassword, setDeletePassword] = useState('')
  const [deleteError, setDeleteError] = useState<string | null>(null)
  const [deleting, setDeleting] = useState(false)
  const count = selectedIds.size

  async function runBulkAction(action: ContainerAction) {
    if (count === 0) return
    setPending(action)
    try {
      await Promise.allSettled(
        Array.from(selectedIds).map((id) => api.containerCommand(uuid, id, action)),
      )
      onDone()
    } finally {
      setPending(null)
    }
  }

  async function handleBulkDelete() {
    if (count === 0 || deleting) return
    setDeleting(true)
    setDeleteError(null)
    try {
      const result = await api.deleteContainers(uuid, Array.from(selectedIds), deletePassword)
      onBulkDeleted(result.deleted)
      setDeleteOpen(false)
      setDeletePassword('')
      onDone()
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Błąd usuwania'
      if (msg.includes('invalid password')) {
        setDeleteError('Nieprawidłowe hasło')
      } else {
        setDeleteError(msg)
      }
    } finally {
      setDeleting(false)
    }
  }

  return (
    <div className="flex flex-col gap-3 rounded-lg border border-zinc-700 bg-zinc-900 px-3 py-3 sm:flex-row sm:items-center sm:px-4 sm:py-2.5">
      <span className="text-sm font-medium text-zinc-300">
        Zaznaczono: <span className="text-zinc-100">{count}</span>
      </span>
      <div className="flex w-full flex-wrap items-center gap-1.5 sm:ml-auto sm:w-auto sm:justify-end">
        <Button
          variant="outline"
          size="sm"
          disabled={count === 0 || pending !== null}
          onClick={() => runBulkAction('start')}
          className="h-7 flex-1 gap-1.5 px-3 text-xs sm:flex-none"
        >
          <PlayIcon className={`size-3 ${pending === 'start' ? 'animate-spin' : ''}`} />
          Start
        </Button>
        <Button
          variant="outline"
          size="sm"
          disabled={count === 0 || pending !== null}
          onClick={() => runBulkAction('stop')}
          className="h-7 flex-1 gap-1.5 px-3 text-xs border-red-800 text-red-400 hover:bg-red-900/30 hover:text-red-300 sm:flex-none"
        >
          <SquareIcon className={`size-3 ${pending === 'stop' ? 'animate-spin' : ''}`} />
          Stop
        </Button>
        <Button
          variant="outline"
          size="sm"
          disabled={count === 0 || pending !== null}
          onClick={() => runBulkAction('restart')}
          className="h-7 flex-1 gap-1.5 px-3 text-xs sm:flex-none"
        >
          <RotateCwIcon className={`size-3 ${pending === 'restart' ? 'animate-spin' : ''}`} />
          Restart
        </Button>
        <Button
          variant="outline"
          size="sm"
          disabled={count === 0 || pending !== null || deleting}
          onClick={() => {
            setDeletePassword('')
            setDeleteError(null)
            setDeleteOpen(true)
          }}
          className="h-7 w-full gap-1.5 px-3 text-xs border-red-800/60 text-red-400 hover:bg-red-950/50 hover:text-red-300 hover:border-red-700 sm:w-auto"
        >
          <Trash2Icon className="size-3" />
          Usuń
        </Button>
      </div>

      <Dialog
        open={deleteOpen}
        onOpenChange={(open) => {
          if (!open && !deleting) {
            setDeleteOpen(false)
            setDeletePassword('')
            setDeleteError(null)
          }
        }}
      >
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <AlertTriangleIcon className="size-5 text-red-400" />
              Usuń kontenery z panelu
            </DialogTitle>
            <DialogDescription>
              Zostanie trwale usuniętych{' '}
              <span className="font-semibold text-zinc-200">{count}</span>{' '}
              {count === 1 ? 'kontener' : 'kontenerów'} z bazy danych panelu.
              Nie wpłynie to na działające kontenery Docker.
              Ta operacja jest nieodwracalna.
            </DialogDescription>
          </DialogHeader>

          <div className="flex flex-col gap-3 py-2">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="bulk-delete-password">Potwierdź hasłem do panelu</Label>
              <Input
                id="bulk-delete-password"
                type="password"
                placeholder="Hasło"
                value={deletePassword}
                onChange={(e) => setDeletePassword(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') handleBulkDelete()
                }}
                autoFocus
                data-testid="bulk-delete-password"
                disabled={deleting}
              />
            </div>
            {deleteError && <p className="text-sm text-red-400">{deleteError}</p>}
          </div>

          <DialogFooter>
            <Button
              variant="outline"
              disabled={deleting}
              onClick={() => {
                setDeleteOpen(false)
                setDeletePassword('')
                setDeleteError(null)
              }}
            >
              Anuluj
            </Button>
            <Button
              variant="destructive"
              disabled={!deletePassword || deleting}
              onClick={handleBulkDelete}
              data-testid="bulk-delete-confirm"
            >
              {deleting ? 'Usuwanie…' : `Usuń ${count}`}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

export function ContainersTable({ uuid, containers, bulkMode, onToggleBulk }: ContainersTableProps) {
  const router = useRouter()
  const [localContainers, setLocalContainers] = useState<Container[]>(containers)
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())
  const [updateInfos, setUpdateInfos] = useState<Map<string, ContainerUpdateInfo>>(new Map())
  const [checkingUpdates, setCheckingUpdates] = useState(false)

  const { containerPoints } = useServerMetrics(uuid)

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

  useEffect(() => {
    async function checkUpdates() {
      if (mergedContainers.length === 0) return
      setCheckingUpdates(true)
      try {
        const containerIds = mergedContainers.map((c) => c.container_id)
        const updates = await api.checkAllContainersUpdates(uuid, containerIds)
        const infoMap = new Map<string, ContainerUpdateInfo>()
        for (const u of updates) {
          infoMap.set(u.container_id, u)
        }
        setUpdateInfos(infoMap)
      } catch {
        // ignore errors
      } finally {
        setCheckingUpdates(false)
      }
    }
    checkUpdates()
  }, [uuid, mergedContainers.length])

  function handleDeleted(containerId: string) {
    setLocalContainers((prev) => prev.filter((c) => c.container_id !== containerId))
    setSelectedIds((prev) => {
      const next = new Set(prev)
      next.delete(containerId)
      return next
    })
    router.refresh()
  }

  function handleAction() {
    router.refresh()
  }

  function handleBulkDeleted(ids: string[]) {
    ids.forEach((id) => handleDeleted(id))
  }

  useEffect(() => {
    if (!bulkMode) setSelectedIds(new Set())
  }, [bulkMode])

  function handleSelect(id: string, checked: boolean) {
    setSelectedIds((prev) => {
      const next = new Set(prev)
      if (checked) next.add(id)
      else next.delete(id)
      return next
    })
  }

  function handleSelectGroup(ids: string[], checked: boolean) {
    setSelectedIds((prev) => {
      const next = new Set(prev)
      if (checked) ids.forEach((id) => next.add(id))
      else ids.forEach((id) => next.delete(id))
      return next
    })
  }

  if (mergedContainers.length === 0) {
    return (
      <div className="rounded-lg border border-zinc-800 bg-zinc-900/50 px-4 py-6 text-center text-sm text-zinc-500 sm:p-8">
        Brak kontenerów
      </div>
    )
  }

  const grouped = new Map<string, Container[]>()
  for (const c of mergedContainers) {
    const key = c.project || '__standalone__'
    const group = grouped.get(key) ?? []
    group.push(c)
    grouped.set(key, group)
  }

  const sortedKeys = Array.from(grouped.keys()).sort((a, b) => {
    if (a === '__standalone__') return 1
    if (b === '__standalone__') return -1
    return a.localeCompare(b)
  })

  const allSelected = mergedContainers.length > 0 && mergedContainers.every((c) => selectedIds.has(c.container_id))
  const someSelected = mergedContainers.some((c) => selectedIds.has(c.container_id))

  return (
    <div className="space-y-3">
      {bulkMode && (
        <BulkActionBar
          uuid={uuid}
          selectedIds={selectedIds}
          onDone={handleAction}
          onBulkDeleted={handleBulkDeleted}
        />
      )}

      {checkingUpdates && (
        <div className="flex items-center gap-2 text-xs text-zinc-500">
          <RefreshCwIcon className="size-3 animate-spin" />
          Sprawdzam aktualizacje...
        </div>
      )}

      <div className="md:hidden space-y-3">
        {mergedContainers.map((c) => (
          <ContainerCard
            key={c.container_id}
            uuid={uuid}
            c={c}
            updateInfo={updateInfos.get(c.container_id)}
            onDeleted={handleDeleted}
            onAction={handleAction}
          />
        ))}
      </div>

      <div className="hidden md:block overflow-x-auto rounded-lg border border-zinc-800">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-zinc-800 bg-zinc-900/50 text-left text-xs text-zinc-500">
              <BulkCheckboxTh
                bulkMode={bulkMode}
                checked={someSelected ? (allSelected ? true : 'indeterminate') : false}
                onCheckedChange={(val) => {
                  if (val) setSelectedIds(new Set(mergedContainers.map((c) => c.container_id)))
                  else setSelectedIds(new Set())
                }}
                ariaLabel="Zaznacz wszystkie"
              />
              <th className="px-3 py-3 font-medium sm:px-4">Nazwa</th>
              <th className="px-3 py-3 font-medium sm:px-4">Stan</th>
              <th className="hidden px-4 py-3 font-medium lg:table-cell">Health</th>
              <th className="hidden px-4 py-3 font-medium xl:table-cell">Obraz</th>
              <th className="hidden px-4 py-3 font-medium lg:table-cell">Serwis</th>
              <th className="px-3 py-3 text-right font-medium sm:px-4">Akcje</th>
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
                  updateInfos={updateInfos}
                  isStandalone={isStandalone}
                  onDeleted={handleDeleted}
                  onAction={handleAction}
                  bulkMode={bulkMode}
                  selectedIds={selectedIds}
                  onSelect={handleSelect}
                  onSelectGroup={handleSelectGroup}
                />
              )
            })}
          </tbody>
        </table>
      </div>
    </div>
  )
}
