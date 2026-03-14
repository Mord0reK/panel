'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'
import Image from 'next/image'
import {
  ChevronRightIcon,
  SettingsIcon,
  BarChart3Icon,
} from 'lucide-react'

import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import {
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
  useSidebar,
} from '@/components/ui/sidebar'
import type { ServiceDefinition } from '@/types'

const FALLBACK_SERVICE_ICON = '/icons/info.svg'

function normalizeServiceIcon(icon: string | undefined): string {
  if (!icon) return FALLBACK_SERVICE_ICON
  if (icon.startsWith('/icons/')) return icon
  return icon
}

const SUB_ITEMS: Record<string, Array<{
  label: string
  icon: any
  path: string
  disabled: boolean
}>> = {
  cloudflare: [
    {
      label: 'Statystyki',
      icon: BarChart3Icon,
      path: '/services/cloudflare',
      disabled: true,
    },
    {
      label: 'Rekordy DNS',
      icon: SettingsIcon,
      path: '/services/cloudflare/dns',
      disabled: false,
    }
  ],
}

interface ServiceNavItemProps {
  service: ServiceDefinition
}

export function ServiceNavItem({ service }: ServiceNavItemProps) {
  const pathname = usePathname()
  const { setOpenMobile } = useSidebar()
  const displayName = service.display_name
  const items = SUB_ITEMS[service.key] || []

  function handleLinkClick() {
    setOpenMobile(false)
  }

  if (items.length === 0) {
    return null
  }

  return (
    <Collapsible asChild className="group/collapsible">
      <SidebarMenuItem>
        <CollapsibleTrigger asChild>
          <SidebarMenuButton tooltip={displayName}>
            <div className="flex size-6 shrink-0 items-center justify-center overflow-hidden">
              <Image
                src={normalizeServiceIcon(service.icon)}
                alt={`${displayName} icon`}
                width={20}
                height={20}
                className="size-4 object-contain"
                unoptimized
              />
            </div>
            <span className="truncate">{displayName}</span>
            <ChevronRightIcon className="ml-auto size-4 shrink-0 transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90" />
          </SidebarMenuButton>
        </CollapsibleTrigger>

        <CollapsibleContent>
          <SidebarMenuSub>
            {items.map((item) => (
              <SidebarMenuSubItem key={item.path}>
                <SidebarMenuSubButton
                  asChild
                  aria-disabled={item.disabled}
                  className={item.disabled ? 'pointer-events-none opacity-50' : ''}
                >
                  <Link href={item.disabled ? '#' : item.path} onClick={handleLinkClick}>
                    <item.icon className="size-4" />
                    <span>{item.label}</span>
                  </Link>
                </SidebarMenuSubButton>
              </SidebarMenuSubItem>
            ))}
          </SidebarMenuSub>
        </CollapsibleContent>
      </SidebarMenuItem>
    </Collapsible>
  )
}
