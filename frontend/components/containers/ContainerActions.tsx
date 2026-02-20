'use client'

import { useState } from 'react'
import { PlayIcon, SquareIcon, RotateCwIcon } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { api } from '@/lib/api'
import type { ContainerAction } from '@/types'

interface ContainerActionsProps {
  uuid: string
  containerId: string
}

const ACTIONS: {
  action: ContainerAction
  label: string
  icon: typeof PlayIcon
  variant: 'default' | 'destructive' | 'outline' | 'secondary' | 'ghost' | 'link'
}[] = [
  { action: 'start', label: 'Start', icon: PlayIcon, variant: 'outline' },
  { action: 'stop', label: 'Stop', icon: SquareIcon, variant: 'destructive' },
  { action: 'restart', label: 'Restart', icon: RotateCwIcon, variant: 'outline' },
]

export function ContainerActions({ uuid, containerId }: ContainerActionsProps) {
  const [pending, setPending] = useState<ContainerAction | null>(null)
  const [error, setError] = useState<string | null>(null)

  async function handleAction(action: ContainerAction) {
    setPending(action)
    setError(null)

    try {
      await api.containerCommand(uuid, containerId, action)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Błąd wykonania akcji')
    } finally {
      setPending(null)
    }
  }

  return (
    <div className="flex items-center gap-1.5">
      {ACTIONS.map(({ action, label, icon: Icon, variant }) => (
        <Button
          key={action}
          variant={variant}
          size="sm"
          disabled={pending !== null}
          onClick={() => handleAction(action)}
          className="h-7 px-2 text-xs"
          title={label}
        >
          <Icon
            className={`size-3.5 ${pending === action ? 'animate-spin' : ''}`}
          />
          <span className="sr-only">{label}</span>
        </Button>
      ))}

      {error && (
        <span className="ml-1 text-xs text-red-400" title={error}>
          Błąd
        </span>
      )}
    </div>
  )
}
