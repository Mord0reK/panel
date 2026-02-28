'use client'

import { useState } from 'react'
import {
  PlayIcon,
  SquareIcon,
  RotateCwIcon,
  MoreHorizontalIcon,
  Trash2Icon,
  AlertTriangleIcon,
} from 'lucide-react'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { api } from '@/lib/api'
import type { ContainerAction } from '@/types'

interface ContainerActionsProps {
  uuid: string
  containerId: string
  containerName?: string
  onDeleted?: () => void
  onAction?: () => void
}

export function ContainerActions({ uuid, containerId, containerName, onDeleted, onAction }: ContainerActionsProps) {
  const [pending, setPending] = useState<ContainerAction | 'delete' | null>(null)
  const [error, setError] = useState<string | null>(null)

  // Delete dialog state
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [password, setPassword] = useState('')
  const [deleteError, setDeleteError] = useState<string | null>(null)

  async function handleAction(action: ContainerAction) {
    setPending(action)
    setError(null)
    try {
      await api.containerCommand(uuid, containerId, action)
      onAction?.()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Błąd wykonania akcji')
    } finally {
      setPending(null)
    }
  }

  async function handleDelete() {
    if (!password) {
      setDeleteError('Wprowadź hasło')
      return
    }
    setPending('delete')
    setDeleteError(null)
    try {
      await api.deleteContainer(uuid, containerId, password)
      setDeleteOpen(false)
      setPassword('')
      onDeleted?.()
    } catch (err) {
      const msg = err instanceof Error ? err.message : ''
      // Backend returns plaintext errors wrapped in JSON by the proxy: { error: "..." }
      let parsed = msg
      try { parsed = (JSON.parse(msg) as { error?: string }).error ?? msg } catch { /* not JSON */ }
      if (parsed.includes('invalid password')) {
        setDeleteError('Nieprawidłowe hasło')
      } else {
        setDeleteError(parsed || 'Błąd usuwania kontenera')
      }
    } finally {
      setPending(null)
    }
  }

  function openDeleteDialog() {
    setPassword('')
    setDeleteError(null)
    setDeleteOpen(true)
  }

  const isBusy = pending !== null

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="ghost"
            size="sm"
            disabled={isBusy}
            className="h-7 w-7 p-0"
            title="Akcje"
          >
            <MoreHorizontalIcon className={`size-4 ${isBusy ? 'animate-spin' : ''}`} />
            <span className="sr-only">Akcje</span>
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-44">
          <DropdownMenuItem
            onClick={() => handleAction('start')}
            disabled={isBusy}
          >
            <PlayIcon className="size-4" />
            Uruchom
          </DropdownMenuItem>
          <DropdownMenuItem
            onClick={() => handleAction('stop')}
            disabled={isBusy}
          >
            <SquareIcon className="size-4" />
            Zatrzymaj
          </DropdownMenuItem>
          <DropdownMenuItem
            onClick={() => handleAction('restart')}
            disabled={isBusy}
          >
            <RotateCwIcon className="size-4" />
            Restartuj
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            variant="destructive"
            onClick={openDeleteDialog}
            disabled={isBusy}
          >
            <Trash2Icon className="size-4" />
            Usuń z panelu
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      {error && (
        <span className="ml-1 text-xs text-red-400" title={error}>
          Błąd
        </span>
      )}

      <Dialog open={deleteOpen} onOpenChange={(open) => {
        if (!open && pending !== 'delete') {
          setDeleteOpen(false)
          setPassword('')
          setDeleteError(null)
        }
      }}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <AlertTriangleIcon className="size-5 text-red-400" />
              Usuń kontener z panelu
            </DialogTitle>
            <DialogDescription>
              Kontener{' '}
              <span className="font-mono text-xs text-zinc-200">
                {containerName || containerId}
              </span>{' '}
              zostanie trwale usunięty z bazy danych panelu. Nie wpłynie to na działający kontener Docker.
              Ta operacja jest nieodwracalna.
            </DialogDescription>
          </DialogHeader>

          <div className="flex flex-col gap-3 py-2">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="delete-password">Potwierdź hasłem do panelu</Label>
              <Input
                id="delete-password"
                type="password"
                placeholder="Hasło"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') handleDelete()
                }}
                autoFocus
                data-testid="container-delete-password"
              />
            </div>
            {deleteError && (
              <p className="text-sm text-red-400">{deleteError}</p>
            )}
          </div>

          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setDeleteOpen(false)
                setPassword('')
                setDeleteError(null)
              }}
              disabled={pending === 'delete'}
            >
              Anuluj
            </Button>
            <Button
              variant="destructive"
              onClick={handleDelete}
              disabled={pending === 'delete' || !password}
              data-testid="container-delete-confirm"
            >
              {pending === 'delete' ? 'Usuwanie…' : 'Usuń kontener'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
