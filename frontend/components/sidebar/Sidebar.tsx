'use client'

import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { LayoutDashboardIcon, LogOutIcon, SettingsIcon } from 'lucide-react'

import { ServerNavList } from '@/components/sidebar/ServerNavList'
import { ServicesNavList } from '@/components/sidebar/ServicesNavList'
import {
  Sidebar as SidebarPrimitive,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarSeparator,
} from '@/components/ui/sidebar'

export function AppSidebar() {
  const router = useRouter()

  async function handleLogout() {
    await fetch('/api/auth/logout', { method: 'POST' })
    router.push('/login')
  }

  return (
    <SidebarPrimitive variant="sidebar" collapsible="icon">
      {/* ---------- Header ---------- */}
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton size="lg" asChild>
              <Link href="/dashboard">
                <div className="flex aspect-square size-8 items-center justify-center rounded-lg bg-sidebar-primary text-sidebar-primary-foreground">
                  <LayoutDashboardIcon className="size-4" />
                </div>
                <div className="grid flex-1 text-left text-sm leading-tight">
                  <span className="truncate font-semibold">Panel</span>
                  <span className="truncate text-xs text-muted-foreground">
                    Server Monitor
                  </span>
                </div>
              </Link>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>

      <SidebarSeparator />

      {/* ---------- Content ---------- */}
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>Serwery</SidebarGroupLabel>
          <SidebarGroupContent>
            <ServerNavList />
          </SidebarGroupContent>
        </SidebarGroup>

        <SidebarGroup>
          <SidebarGroupLabel>Usługi</SidebarGroupLabel>
          <SidebarGroupContent>
            <ServicesNavList />
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>

      {/* ---------- Footer ---------- */}
      <SidebarFooter>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton asChild tooltip="Ustawienia">
              <Link href="/settings/servers">
                <SettingsIcon className="size-4" />
                <span>Ustawienia</span>
              </Link>
            </SidebarMenuButton>
          </SidebarMenuItem>
          <SidebarMenuItem>
            <SidebarMenuButton onClick={handleLogout} tooltip="Wyloguj się">
              <LogOutIcon className="size-4" />
              <span>Wyloguj się</span>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>
    </SidebarPrimitive>
  )
}
