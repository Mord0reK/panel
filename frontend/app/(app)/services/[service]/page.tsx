'use client'

import Image from 'next/image'
import Link from 'next/link'
import { ArrowLeftIcon, RefreshCwIcon, Settings2Icon } from 'lucide-react'
import { useParams } from 'next/navigation'
import { useCallback, useEffect, useState } from 'react'

import { AdGuardHomeDashboard } from '@/components/services/AdGuardHomeDashboard'
import { Button } from '@/components/ui/button'
import { api } from '@/lib/api'
import { cn } from '@/lib/utils'
import type {
  AdGuardHomeDashboardResponse,
  ServiceDefinition,
} from '@/types'

const FALLBACK_SERVICE_ICON = '/icons/info.svg'

function ServicePlaceholder({
  serviceName,
  serviceKey,
}: {
  serviceName: string
  serviceKey: string
}) {
  return (
    <div className="flex min-h-full items-center justify-center rounded-xl border border-zinc-800 bg-zinc-950 px-4 py-8 sm:px-6">
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

        <div className="flex flex-col justify-center gap-3 sm:flex-row">
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

function LoadingState() {
  return (
    <div
      data-testid="service-dashboard-loading"
      className="overflow-hidden rounded-[2rem] border border-zinc-800 bg-zinc-950/80 p-5 sm:p-6"
    >
      <div className="animate-pulse space-y-4">
        <div className="h-5 w-28 rounded-full bg-zinc-800" />
        <div className="h-10 w-72 rounded-full bg-zinc-900" />
        <div className="h-24 rounded-[1.5rem] bg-zinc-900" />
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <div className="h-32 rounded-[1.5rem] bg-zinc-900" />
          <div className="h-32 rounded-[1.5rem] bg-zinc-900" />
          <div className="h-32 rounded-[1.5rem] bg-zinc-900" />
          <div className="h-32 rounded-[1.5rem] bg-zinc-900" />
        </div>
      </div>
    </div>
  )
}

function ErrorState({ message }: { message: string }) {
  return (
    <div
      data-testid="service-dashboard-error"
      className="rounded-[2rem] border border-red-900/70 bg-red-950/40 p-6"
    >
      <h2 className="text-lg font-semibold text-red-200">
        Nie udało się pobrać danych AdGuard Home
      </h2>
      <p className="mt-3 max-w-2xl text-sm leading-6 text-red-100/80">
        {message}
      </p>
      <div className="mt-5 flex flex-col gap-3 sm:flex-row">
        <Button asChild variant="outline" size="sm">
          <Link href="/settings/services">Sprawdź konfigurację integracji</Link>
        </Button>
        <Button asChild size="sm">
          <Link href="/dashboard">Powrót do dashboardu</Link>
        </Button>
      </div>
    </div>
  )
}

export default function ServicePage() {
  const params = useParams()
  const serviceKey = params?.service as string
  const [service, setService] = useState<ServiceDefinition | null>(null)
  const [dashboard, setDashboard] =
    useState<AdGuardHomeDashboardResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const loadPageData = useCallback(
    async (options?: { refresh?: boolean }) => {
      if (!serviceKey) {
        return
      }

      if (options?.refresh) {
        setRefreshing(true)
      } else {
        setLoading(true)
      }

      setError(null)

      try {
        const services = await api.getServices()
        const matchedService =
          services.find((item) => item.key === serviceKey) ?? null
        setService(matchedService)

        if (serviceKey === 'adguardhome') {
          const nextDashboard = await api.getServiceStats(serviceKey)
          setDashboard(nextDashboard)
        } else {
          setDashboard(null)
        }
      } catch (err) {
        setDashboard(null)
        setError(
          err instanceof Error
            ? err.message
            : 'Nie udało się pobrać danych usługi.'
        )
      } finally {
        setLoading(false)
        setRefreshing(false)
      }
    },
    [serviceKey]
  )

  useEffect(() => {
    loadPageData()
  }, [loadPageData])

  const serviceName = service?.display_name ?? serviceKey
  const serviceIcon = service?.icon || FALLBACK_SERVICE_ICON
  const isAdGuardHome = serviceKey === 'adguardhome'

  if (!isAdGuardHome) {
    return (
      <ServicePlaceholder serviceName={serviceName} serviceKey={serviceKey} />
    )
  }

  return (
    <div className="space-y-6">

      {loading ? <LoadingState /> : null}
      {!loading && error ? <ErrorState message={error} /> : null}
      {!loading && !error && dashboard ? (
        <AdGuardHomeDashboard serviceName={serviceName} dashboard={dashboard} />
      ) : null}
      {!loading && !error && !dashboard ? (
        <ErrorState message="Brak danych dashboardu dla tej usługi." />
      ) : null}
    </div>
  )
}
