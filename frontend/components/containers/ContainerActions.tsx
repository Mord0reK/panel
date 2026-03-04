'use client'

import { useState } from 'react'
import {
  PlayIcon,
  SquareIcon,
  RotateCwIcon,
  MoreHorizontalIcon,
  Trash2Icon,
  AlertTriangleIcon,
  RefreshCwIcon,
} from 'lucide-react'
import { toast } from 'sonner'

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
  const [pending, setPending] = useState<ContainerAction | 'delete' | 'update' | null>(null)
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

  async function handleUpdateLogic(updateInfo: any, loadingToast: string | number) {
    try {
      // Check status
      if (updateInfo.status === 'local') {
        toast.dismiss(loadingToast)
        toast.info('Obraz lokalny', {
          description: 'Ten kontener używa lokalnego obrazu (nie można sprawdzić aktualizacji)'
        })
        return
      }

      if (updateInfo.status === 'rate_limited') {
        toast.dismiss(loadingToast)
        toast.warning('Limit zapytań', {
          description: 'Przekroczono limit zapytań do rejestru Docker. Spróbuj ponownie później.'
        })
        return
      }

      if (updateInfo.status === 'up_to_date' || !updateInfo.update_available) {
        toast.dismiss(loadingToast)
        toast.success('Kontener jest aktualny', {
          description: 'Już w najnowszej wersji'
        })
        return
      }

      // Update available - proceed with update
      toast.loading('Aktualizowanie kontenera...', {
        id: loadingToast,
        description: `Dostępna nowa wersja (${updateInfo.latest_version}). Rozpoczynanie aktualizacji...`
      })

      const updateResponse = await api.updateContainer(uuid, containerId)
      const updateData = updateResponse as any

      if (updateData.error) {
        toast.dismiss(loadingToast)
        toast.error('Błąd aktualizacji', {
          description: updateData.error
        })
        return
      }

      const results = updateData.results || []
      const result = results[0]

      if (result && result.success) {
        toast.dismiss(loadingToast)
        toast.success('Kontener zaktualizowany', {
          description: result.message || 'Kontener został pomyślnie zaktualizowany'
        })
        onAction?.()
      } else {
        toast.dismiss(loadingToast)
        toast.error('Błąd aktualizacji', {
          description: result?.message || 'Nie udało się zaktualizować kontenera'
        })
      }
    } catch (err) {
      toast.dismiss(loadingToast)
      const errorMsg = err instanceof Error ? err.message : 'Błąd aktualizacji'
      toast.error('Błąd aktualizacji', {
        description: errorMsg
      })
    }
  }

  async function handleUpdate() {
    setPending('update')
    setError(null)

    // Show loading toast
    const loadingToast = toast.loading('Sprawdzanie aktualizacji...')

    try {
      // First check for updates
      const checkResponse = await api.checkContainerUpdate(uuid, containerId)
      const checkData = checkResponse as any

      // Check if there's an error in the response
      if (checkData.error) {
        toast.dismiss(loadingToast)
        toast.error('Błąd sprawdzania aktualizacji', {
          description: checkData.error
        })
        setPending(null)
        return
      }

      // Get update info from the response
      const updates = checkData.updates || []
      const updateInfo = updates[0]

      if (!updateInfo) {
        // If there's no updates array but the status is OK, or if it's already up to date
        if (checkData.result?.updates?.[0]) {
          // Some responses might have nested result object
          const nestedUpdate = checkData.result.updates[0]
          await handleUpdateLogic(nestedUpdate, loadingToast)
          return
        }
        
        toast.dismiss(loadingToast)
        toast.error('Nie można sprawdzić aktualizacji')
        setPending(null)
        return
      }

      await handleUpdateLogic(updateInfo, loadingToast)
    } catch (err) {
      toast.dismiss(loadingToast)
      const errorMsg = err instanceof Error ? err.message : 'Błąd aktualizacji kontenera'
      toast.error('Błąd aktualizacji', {
        description: errorMsg
      })
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
            onClick={handleUpdate}
            disabled={isBusy}
          >
            <RefreshCwIcon className="size-4" />
            Aktualizuj
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
