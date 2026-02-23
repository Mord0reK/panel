'use client'

import { useState, useEffect } from 'react'
import { useServices, Service } from '@/hooks/useServices'
import { ServiceConfigForm } from '@/components/services/ServiceConfigForm'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Skeleton } from '@/components/ui/skeleton'

export default function ServicesSettingsPage() {
  const { services, loading, refresh } = useServices()
  const [configs, setConfigs] = useState<Record<string, Record<string, string>>>({})
  const [loadingConfigs, setLoadingConfigs] = useState(true)

  useEffect(() => {
    if (services.length > 0) {
      fetchAllConfigs()
    } else if (!loading) {
      setLoadingConfigs(false)
    }
  }, [services, loading])

  const fetchAllConfigs = async () => {
    const newConfigs: Record<string, Record<string, string>> = {}
    await Promise.all(
      services.map(async (s) => {
        try {
          const res = await fetch(`/api/services/${s.slug}/config`)
          if (res.ok) {
            newConfigs[s.slug] = await res.json()
          }
        } catch (e) {
          console.error(`Failed to fetch config for ${s.slug}`, e)
        }
      })
    )
    setConfigs(newConfigs)
    setLoadingConfigs(false)
  }

  if (loading || loadingConfigs) {
    return (
      <div className="space-y-6 p-6">
        <Skeleton className="h-10 w-[200px]" />
        <Skeleton className="h-[400px] w-full" />
      </div>
    )
  }

  if (services.length === 0) {
    return (
      <div className="p-6 text-center">
        <h2 className="text-xl font-semibold">Brak dostępnych usług</h2>
        <p className="text-muted-foreground">Nie zarejestrowano żadnych usług na backendzie.</p>
      </div>
    )
  }

  return (
    <div className="space-y-6 p-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Usługi</h1>
        <p className="text-muted-foreground">
          Konfiguruj zewnętrzne usługi zintegrowane z panelem.
        </p>
      </div>

      <Tabs defaultValue={services[0]?.slug} className="space-y-4">
        <TabsList>
          {services.map((s) => (
            <TabsTrigger key={s.slug} value={s.slug}>
              {s.name}
            </TabsTrigger>
          ))}
        </TabsList>
        {services.map((s) => (
          <TabsContent key={s.slug} value={s.slug}>
            <div className="rounded-xl border bg-card text-card-foreground shadow">
              <div className="flex flex-col space-y-1.5 p-6">
                <h3 className="font-semibold leading-none tracking-tight">{s.name}</h3>
                <p className="text-sm text-muted-foreground">{s.description}</p>
              </div>
              <div className="p-6 pt-0">
                <ServiceConfigForm
                  service={s}
                  initialConfig={configs[s.slug] || {}}
                  onSave={refresh}
                />
              </div>
            </div>
          </TabsContent>
        ))}
      </Tabs>
    </div>
  )
}
