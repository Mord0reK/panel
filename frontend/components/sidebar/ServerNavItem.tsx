'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'
import {
  BarChart3Icon,
  ChevronRightIcon,
  ContainerIcon,
  FileTextIcon,
  ServerIcon,
} from 'lucide-react'

import { Badge } from '@/components/ui/badge'
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
} from '@/components/ui/sidebar'
import type { Server } from '@/types'

const SUB_ITEMS = [
  {
    label: 'Metryki',
    icon: BarChart3Icon,
    segment: 'metrics',
    disabled: false,
  },
  {
    label: 'Logi',
    icon: FileTextIcon,
    segment: 'logs',
    disabled: true,
  },
  {
    label: 'Kontenery',
    icon: ContainerIcon,
    segment: 'containers',
    disabled: false,
  },
] as const

interface ServerNavItemProps {
  server: Server
}

export function ServerNavItem({ server }: ServerNavItemProps) {
  const pathname = usePathname()
  const basePath = `/servers/${server.uuid}`
  const isServerActive = pathname.startsWith(basePath)

  return (
    <Collapsible asChild defaultOpen={isServerActive} className="group/collapsible">
      <SidebarMenuItem>
        <CollapsibleTrigger asChild>
          <SidebarMenuButton
            isActive={isServerActive}
            tooltip={server.hostname}
          >
            <ServerIcon className="size-4" />
            <span className="truncate">{server.hostname}</span>
            {!server.approved && (
              <Badge variant="outline" className="ml-auto text-[10px] px-1.5 py-0">
                Oczekuje
              </Badge>
            )}
            <ChevronRightIcon className="ml-auto size-4 shrink-0 transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90" />
          </SidebarMenuButton>
        </CollapsibleTrigger>

        <CollapsibleContent>
          <SidebarMenuSub>
            {SUB_ITEMS.map((item) => {
              const href = `${basePath}/${item.segment}`
              const isActive = pathname === href

              return (
                <SidebarMenuSubItem key={item.segment}>
                  <SidebarMenuSubButton
                    asChild
                    isActive={isActive}
                    aria-disabled={item.disabled}
                    className={item.disabled ? 'pointer-events-none opacity-50' : ''}
                  >
                    <Link href={item.disabled ? '#' : href}>
                      <item.icon className="size-4" />
                      <span>{item.label}</span>
                    </Link>
                  </SidebarMenuSubButton>
                </SidebarMenuSubItem>
              )
            })}
          </SidebarMenuSub>
        </CollapsibleContent>
      </SidebarMenuItem>
    </Collapsible>
  )
}
