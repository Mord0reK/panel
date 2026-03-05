'use client'

import Link from 'next/link'
import { ArrowLeftIcon } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import { Button } from '@/components/ui/button'
import { api } from '@/lib/api'
import type { ServiceDefinition } from '@/types'

export default function ServicePlaceholder() {
  const params = useParams()
  const serviceKey = params?.service as string
  const [service, setService] = useState<ServiceDefinition | null>(null)

  useEffect(() => {
    if (serviceKey) {
      api
        .getServices()
        .then((services) => {
          const found = services.find((s) => s.key === serviceKey)
          setService(found ?? null)
        })
        .catch(() => setService(null))
    }
  }, [serviceKey])

  const serviceName = service?.display_name ?? serviceKey

  return (
    <div className="flex min-h-screen items-center justify-center bg-zinc-950 px-4">
      <div className="w-full max-w-md space-y-6 text-center">
        <div className="space-y-2">
          <h1 className="text-2xl font-semibold text-zinc-100">
            {serviceName}
          </h1>
          <p className="text-sm text-zinc-400">
            Strona dla usługi{' '}
            <span className="font-mono text-zinc-300">{serviceKey}</span> jest w
            budowie.
          </p>
        </div>

        <div className="flex justify-center gap-3">
          <Button asChild variant="outline" size="sm">
            <Link href="/dashboard">
              <ArrowLeftIcon className="mr-2 size-4" />
              Powrót do dashboardu
            </Link>
          </Button>
          <Button asChild size="sm">
            <Link href="/settings/services">Ustawienia integracji</Link>
          </Button>
        </div>

        <div className="mt-8 rounded-lg border border-zinc-800 bg-zinc-900/50 p-4 text-left">
          <h2 className="mb-2 text-sm font-medium text-zinc-200">
            Planowana funkcjonalność:
          </h2>
          <ul className="space-y-1 text-xs text-zinc-400">
            <li>• Statystyki z {serviceName}</li>
            <li>• Wykresy i monitorowanie w czasie rzeczywistym</li>
            <li>• Zarządzanie konfiguracją usługi</li>
          </ul>
        </div>
      </div>
    </div>
  )
}
