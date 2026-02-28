'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { CheckSquareIcon, XIcon } from 'lucide-react'

import { ContainersTable } from '@/components/containers/ContainersTable'
import { Button } from '@/components/ui/button'
import type { ServerDetailResponse } from '@/types'

interface ContainersPageClientProps {
  uuid: string
  detail: ServerDetailResponse
}

export function ContainersPageClient({ uuid, detail }: ContainersPageClientProps) {
  const [bulkMode, setBulkMode] = useState(false)

  function toggleBulk() {
    setBulkMode((v) => !v)
  }

  return (
    <main className="space-y-6 p-4">
      <div className="flex items-start justify-between">
        <div className="space-y-1">
          <h1 className="text-xl font-semibold text-zinc-100">
            Kontenery — {detail.server.hostname}
          </h1>
          <Button
            variant={bulkMode ? 'secondary' : 'ghost'}
            size="sm"
            onClick={toggleBulk}
            className="h-7 gap-1.5 px-2 text-xs text-zinc-400 hover:text-zinc-200"
          >
            {bulkMode ? (
              <>
                <XIcon className="size-3.5" />
                Zakończ zaznaczanie
              </>
            ) : (
              <>
                <CheckSquareIcon className="size-3.5" />
                Zaznacz wiele
              </>
            )}
          </Button>
        </div>
      </div>

      <ContainersTable
        uuid={uuid}
        containers={detail.containers ?? []}
        bulkMode={bulkMode}
        onToggleBulk={toggleBulk}
      />
    </main>
  )
}
