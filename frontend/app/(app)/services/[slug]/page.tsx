'use client'

import { useParams } from 'next/navigation'
import { useServices } from '@/hooks/useServices'
import { Skeleton } from '@/components/ui/skeleton'

export default function ServicePage() {
  const { slug } = useParams()
  const { services, loading } = useServices()

  const service = services.find(s => s.slug === slug)

  if (loading) {
    return (
      <div className="space-y-6 p-6">
        <Skeleton className="h-10 w-[200px]" />
        <Skeleton className="h-[400px] w-full" />
      </div>
    )
  }

  if (!service || !service.is_enabled) {
    return (
      <div className="p-6 text-center">
        <h2 className="text-xl font-semibold">Usługa niedostępna</h2>
        <p className="text-muted-foreground">Ta usługa nie została skonfigurowana lub nie istnieje.</p>
      </div>
    )
  }

  return (
    <div className="space-y-6 p-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">{service.name}</h1>
        <p className="text-muted-foreground">{service.description}</p>
      </div>

      <div className="rounded-lg border border-dashed p-20 text-center">
        <p className="text-muted-foreground">
          Tutaj pojawi się dedykowany interfejs dla usługi <strong>{service.name}</strong>.
          Dane będą pobierane przez backendowy proxy: <code>/api/services/{service.slug}/proxy/...</code>
        </p>
      </div>
    </div>
  )
}
