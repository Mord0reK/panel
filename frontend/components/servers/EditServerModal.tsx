'use client'

import { useEffect, useState } from 'react'
import Image from 'next/image'
import {
  ServerIcon,
  DatabaseIcon,
  CloudIcon,
  BoxIcon,
  MonitorIcon,
  HardDriveIcon,
  NetworkIcon,
  ShieldIcon,
  GlobeIcon,
  CpuIcon,
  CheckIcon,
} from 'lucide-react'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { api } from '@/lib/api'
import { cn } from '@/lib/utils'
import type { CustomIcon, Server } from '@/types'

// ---------------------------------------------------------------------------
// Lucide icon catalogue available for servers
// ---------------------------------------------------------------------------
const LUCIDE_ICONS: Array<{
  id: string
  label: string
  Icon: React.ComponentType<{ className?: string }>
}> = [
  { id: 'lucide:server', label: 'Serwer', Icon: ServerIcon },
  { id: 'lucide:database', label: 'Baza danych', Icon: DatabaseIcon },
  { id: 'lucide:cloud', label: 'Chmura', Icon: CloudIcon },
  { id: 'lucide:box', label: 'Kontener', Icon: BoxIcon },
  { id: 'lucide:monitor', label: 'Monitor', Icon: MonitorIcon },
  { id: 'lucide:hard-drive', label: 'Dysk', Icon: HardDriveIcon },
  { id: 'lucide:network', label: 'Sieć', Icon: NetworkIcon },
  { id: 'lucide:shield', label: 'Bezpieczeństwo', Icon: ShieldIcon },
  { id: 'lucide:globe', label: 'Web', Icon: GlobeIcon },
  { id: 'lucide:cpu', label: 'CPU', Icon: CpuIcon },
]

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------
interface EditServerModalProps {
  server: Server | null
  open: boolean
  onClose: () => void
  onSaved: (updated: Server) => void
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------
export function EditServerModal({
  server,
  open,
  onClose,
  onSaved,
}: EditServerModalProps) {
  const [displayName, setDisplayName] = useState('')
  const [selectedIcon, setSelectedIcon] = useState('')
  const [customIcons, setCustomIcons] = useState<CustomIcon[]>([])
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Sync state when server changes.
  useEffect(() => {
    if (server) {
      setDisplayName(server.display_name ?? '')
      setSelectedIcon(server.icon ?? '')
      setError(null)
    }
  }, [server])

  // Fetch custom icons once when modal opens.
  useEffect(() => {
    if (!open) return
    api.getIcons().then(setCustomIcons).catch(() => setCustomIcons([]))
  }, [open])

  async function handleSave() {
    if (!server) return
    setSaving(true)
    setError(null)
    try {
      const updated = await api.patchServer(server.uuid, {
        display_name: displayName,
        icon: selectedIcon,
      })
      onSaved(updated)
      onClose()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Błąd zapisu')
    } finally {
      setSaving(false)
    }
  }

  if (!server) return null

  return (
    <Dialog open={open} onOpenChange={(v) => !v && onClose()}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Edytuj serwer</DialogTitle>
        </DialogHeader>

        <div className="space-y-5">
          {/* ── Read-only info ── */}
          <div className="rounded-lg border bg-muted/30 px-4 py-3 text-sm space-y-1.5">
            <InfoRow label="Hostname" value={server.hostname} />
            <InfoRow label="System" value={server.platform || '—'} />
            <InfoRow label="Kernel" value={server.kernel || '—'} />
            <InfoRow label="Architektura" value={server.architecture || '—'} />
            <InfoRow label="CPU" value={server.cpu_model || '—'} />
            <InfoRow
              label="Rdzenie"
              value={server.cpu_cores ? String(server.cpu_cores) : '—'}
            />
          </div>

          {/* ── Display name ── */}
          <div className="space-y-1.5">
            <Label htmlFor="display-name">Nazwa</Label>
            <Input
              id="display-name"
              placeholder={server.hostname}
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
            />
          </div>

          {/* ── Icon picker ── */}
          <div className="space-y-1.5">
            <Label>Ikona</Label>
            <Tabs defaultValue="builtin">
              <TabsList className="mb-3">
                <TabsTrigger value="builtin">Wbudowane</TabsTrigger>
                <TabsTrigger value="custom">Własne</TabsTrigger>
              </TabsList>

              {/* Built-in Lucide icons */}
              <TabsContent value="builtin">
                <div className="grid grid-cols-5 gap-2">
                  {LUCIDE_ICONS.map(({ id, label, Icon }) => (
                    <IconTile
                      key={id}
                      selected={selectedIcon === id}
                      label={label}
                      onClick={() =>
                        setSelectedIcon(selectedIcon === id ? '' : id)
                      }
                    >
                      <Icon className="size-6" />
                    </IconTile>
                  ))}
                </div>
              </TabsContent>

              {/* Custom icons from /public/icons/ */}
              <TabsContent value="custom">
                {customIcons.length === 0 ? (
                  <p className="text-sm text-muted-foreground">
                    Brak własnych ikon. Wrzuć pliki PNG/SVG do folderu{' '}
                    <code className="text-xs">public/icons/</code>.
                  </p>
                ) : (
                  <div className="grid grid-cols-5 gap-2">
                    {customIcons.map(({ name, url }) => {
                      const id = `custom:${name}`
                      return (
                        <IconTile
                          key={id}
                          selected={selectedIcon === id}
                          label={name}
                          onClick={() =>
                            setSelectedIcon(selectedIcon === id ? '' : id)
                          }
                        >
                          <Image
                            src={url}
                            alt={name}
                            width={24}
                            height={24}
                            className="size-6 object-contain"
                            unoptimized
                          />
                        </IconTile>
                      )
                    })}
                  </div>
                )}
              </TabsContent>
            </Tabs>
          </div>

          {error && <p className="text-sm text-destructive">{error}</p>}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={saving}>
            Anuluj
          </Button>
          <Button onClick={handleSave} disabled={saving}>
            {saving ? 'Zapisuję…' : 'Zapisz'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ---------------------------------------------------------------------------
// Small helpers
// ---------------------------------------------------------------------------
function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center gap-2">
      <span className="w-28 shrink-0 text-muted-foreground">{label}</span>
      <span className="truncate font-mono text-xs">{value}</span>
    </div>
  )
}

function IconTile({
  selected,
  label,
  onClick,
  children,
}: {
  selected: boolean
  label: string
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <button
      type="button"
      title={label}
      onClick={onClick}
      className={cn(
        'relative flex flex-col items-center justify-center gap-1 rounded-lg border p-2 text-xs transition-colors',
        selected
          ? 'border-primary bg-primary/10 text-primary'
          : 'border-border hover:border-primary/50 hover:bg-muted/50',
      )}
    >
      {selected && (
        <CheckIcon className="absolute right-1 top-1 size-3 text-primary" />
      )}
      {children}
    </button>
  )
}
