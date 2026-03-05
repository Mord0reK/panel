'use client'

import { useCallback, useEffect, useState } from 'react'
import {
  PencilIcon,
  RefreshCwIcon,
  ServerIcon,
  Trash2Icon,
  XCircleIcon,
} from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { SettingsNav } from '@/components/settings/SettingsNav'
import { ServerIconDisplay } from '@/components/servers/ServerIconDisplay'
import { EditServerModal } from '@/components/servers/EditServerModal'
import { api } from '@/lib/api'
import type { Server } from '@/types'

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------
function formatLastSeen(iso: string | null | undefined): string {
  if (!iso) return '—'
  const d = new Date(iso)
  if (isNaN(d.getTime())) return '—'
  return d.toLocaleString('pl-PL')
}

// ---------------------------------------------------------------------------
// Server row component
// ---------------------------------------------------------------------------
interface ServerRowProps {
  server: Server
  onEdit: (server: Server) => void
  onReject: (server: Server) => void
  onRestore: (server: Server) => void
  onDelete: (server: Server) => void
}

function ServerRow({
  server,
  onEdit,
  onReject,
  onRestore,
  onDelete,
}: ServerRowProps) {
  const displayName = server.display_name || server.hostname

  return (
    <div className="flex items-center gap-3 rounded-lg border p-3 transition-colors hover:bg-muted/30">
      <div className="flex size-9 shrink-0 items-center justify-center rounded-md border bg-muted">
        <ServerIconDisplay icon={server.icon} className="size-5" />
      </div>

      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span className="truncate font-medium text-sm">{displayName}</span>
          {displayName !== server.hostname && (
            <span className="truncate text-xs text-muted-foreground">
              ({server.hostname})
            </span>
          )}
        </div>
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <span>{server.platform || '—'}</span>
          <span>·</span>
          <span>{server.architecture || '—'}</span>
          <span>·</span>
          <span>ostatnio: {formatLastSeen(server.last_seen)}</span>
        </div>
      </div>

      <div className="flex items-center gap-1.5 shrink-0">
        {server.status !== 'rejected' && (
          <>
            <Badge
              variant={server.online ? 'default' : 'outline'}
              className={
                server.online
                  ? 'border-emerald-600 bg-emerald-600/15 text-emerald-400'
                  : 'border-muted-foreground/40 text-muted-foreground'
              }
            >
              {server.online ? 'Online' : 'Offline'}
            </Badge>

            <Button
              size="icon"
              variant="ghost"
              className="size-7"
              title="Edytuj"
              onClick={() => onEdit(server)}
            >
              <PencilIcon className="size-3.5" />
            </Button>
            <Button
              size="icon"
              variant="ghost"
              className="size-7 text-destructive hover:text-destructive"
              title="Odrzuć"
              onClick={() => onReject(server)}
            >
              <XCircleIcon className="size-3.5" />
            </Button>
          </>
        )}

        {server.status === 'rejected' && (
          <>
            <Button
              size="icon"
              variant="ghost"
              className="size-7 text-emerald-500 hover:text-emerald-400"
              title="Przywróć"
              onClick={() => onRestore(server)}
            >
              <RefreshCwIcon className="size-3.5" />
            </Button>
            <Button
              size="icon"
              variant="ghost"
              className="size-7 text-destructive hover:text-destructive"
              title="Usuń permanentnie"
              onClick={() => onDelete(server)}
            >
              <Trash2Icon className="size-3.5" />
            </Button>
          </>
        )}
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------
export default function SettingsServersPage() {
  const [servers, setServers] = useState<Server[]>([])
  const [loading, setLoading] = useState(true)
  const [editTarget, setEditTarget] = useState<Server | null>(null)
  const [editOpen, setEditOpen] = useState(false)

  const fetchServers = useCallback(async () => {
    try {
      const data = await api.getServers()
      setServers(data)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchServers()
  }, [fetchServers])

  function handleEdit(server: Server) {
    setEditTarget(server)
    setEditOpen(true)
  }

  async function handleReject(server: Server) {
    await api.patchServer(server.uuid, { status: 'rejected' })
    setServers((prev) =>
      prev.map((s) =>
        s.uuid === server.uuid ? { ...s, status: 'rejected' } : s
      )
    )
  }

  async function handleRestore(server: Server) {
    await api.patchServer(server.uuid, { status: 'active' })
    setServers((prev) =>
      prev.map((s) => (s.uuid === server.uuid ? { ...s, status: 'active' } : s))
    )
  }

  async function handleDelete(server: Server) {
    if (
      !confirm(
        `Na pewno usunąć serwer "${server.display_name || server.hostname}"? Tej akcji nie można cofnąć.`
      )
    ) {
      return
    }
    await api.deleteServer(server.uuid)
    setServers((prev) => prev.filter((s) => s.uuid !== server.uuid))
  }

  function handleSaved(updated: Server) {
    setServers((prev) =>
      prev.map((s) => (s.uuid === updated.uuid ? updated : s))
    )
  }

  const activeServers = servers.filter(
    (s) => (s.status ?? 'active') !== 'rejected'
  )
  const rejectedServers = servers.filter((s) => s.status === 'rejected')

  return (
    <div className="mx-auto max-w-3xl space-y-6">
      <div>
        <h1 className="text-xl font-semibold">Serwery</h1>
        <p className="text-sm text-muted-foreground">
          Zarządzaj nazwami, ikonami i statusami podłączonych serwerów.
        </p>
      </div>

      <SettingsNav />

      {loading ? (
        <div className="text-sm text-muted-foreground">Ładowanie…</div>
      ) : (
        <Tabs defaultValue="active">
          <TabsList>
            <TabsTrigger value="active">
              Aktywne
              {activeServers.length > 0 && (
                <span className="ml-1.5 rounded-full bg-muted px-1.5 py-0.5 text-xs font-medium tabular-nums">
                  {activeServers.length}
                </span>
              )}
            </TabsTrigger>
            <TabsTrigger value="rejected">
              Odrzucone
              {rejectedServers.length > 0 && (
                <span className="ml-1.5 rounded-full bg-destructive/20 text-destructive px-1.5 py-0.5 text-xs font-medium tabular-nums">
                  {rejectedServers.length}
                </span>
              )}
            </TabsTrigger>
          </TabsList>

          <TabsContent value="active" className="mt-4 space-y-2">
            {activeServers.length === 0 ? (
              <EmptyState
                icon={<ServerIcon className="size-8 text-muted-foreground" />}
                text="Brak aktywnych serwerów."
              />
            ) : (
              activeServers.map((s) => (
                <ServerRow
                  key={s.uuid}
                  server={s}
                  onEdit={handleEdit}
                  onReject={handleReject}
                  onRestore={handleRestore}
                  onDelete={handleDelete}
                />
              ))
            )}
          </TabsContent>

          <TabsContent value="rejected" className="mt-4 space-y-2">
            {rejectedServers.length === 0 ? (
              <EmptyState
                icon={<XCircleIcon className="size-8 text-muted-foreground" />}
                text="Brak odrzuconych serwerów."
              />
            ) : (
              rejectedServers.map((s) => (
                <ServerRow
                  key={s.uuid}
                  server={s}
                  onEdit={handleEdit}
                  onReject={handleReject}
                  onRestore={handleRestore}
                  onDelete={handleDelete}
                />
              ))
            )}
          </TabsContent>
        </Tabs>
      )}

      <EditServerModal
        server={editTarget}
        open={editOpen}
        onClose={() => setEditOpen(false)}
        onSaved={handleSaved}
      />
    </div>
  )
}

function EmptyState({ icon, text }: { icon: React.ReactNode; text: string }) {
  return (
    <div className="flex flex-col items-center justify-center gap-2 rounded-lg border border-dashed py-12 text-center">
      {icon}
      <p className="text-sm text-muted-foreground">{text}</p>
    </div>
  )
}
