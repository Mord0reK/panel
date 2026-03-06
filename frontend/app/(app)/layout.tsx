import { AppSidebar } from '@/components/sidebar/Sidebar'
import { SidebarInset, SidebarProvider, SidebarTrigger } from '@/components/ui/sidebar'
import { Separator } from '@/components/ui/separator'

export default function AppLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset>
        <header className="flex h-14 shrink-0 items-center gap-2 border-b px-3 sm:h-12 sm:px-4">
          <SidebarTrigger className="-ml-1" />
          <Separator orientation="vertical" className="mr-2 hidden h-4 sm:block" />
        </header>
        <main className="flex-1 px-3 py-3 sm:px-4 sm:py-4 lg:px-6">
          {children}
        </main>
      </SidebarInset>
    </SidebarProvider>
  )
}
