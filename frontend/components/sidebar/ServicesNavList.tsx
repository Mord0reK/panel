'use client'

import Link from 'next/link'
import * as Icons from 'lucide-react'
import { useServices } from '@/hooks/useServices'
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from '@/components/ui/sidebar'

export function ServicesNavList() {
  const { services, loading } = useServices()
  const enabledServices = services.filter(s => s.is_enabled)

  if (loading || enabledServices.length === 0) return null

  return (
    <SidebarMenu>
      {enabledServices.map((service) => {
        // Use icon from Lucide or fallback to Box
        const IconComponent = (Icons as any)[service.icon] || Icons.Box

        return (
          <SidebarMenuItem key={service.slug}>
            <SidebarMenuButton asChild tooltip={service.name}>
              <Link href={`/services/${service.slug}`}>
                <IconComponent className="size-4" />
                <span>{service.name}</span>
              </Link>
            </SidebarMenuButton>
          </SidebarMenuItem>
        )
      })}
    </SidebarMenu>
  )
}
