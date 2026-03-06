import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

import { ServerNavItem } from '@/components/sidebar/ServerNavItem'
import type { Server } from '@/types'

// ---- Mocks ----

let mockPathname = '/dashboard'

jest.mock('next/navigation', () => ({
  usePathname: () => mockPathname,
}))

jest.mock('next/link', () => {
  return function MockLink({
    children,
    href,
    ...rest
  }: {
    children: React.ReactNode
    href: string
    [key: string]: unknown
  }) {
    return (
      <a href={href} {...rest}>
        {children}
      </a>
    )
  }
})

// ---- Helpers ----

function makeServer(overrides: Partial<Server> = {}): Server {
  return {
    uuid: 'test-uuid-123',
    hostname: 'web-server-01',
    approved: true,
    cpu_model: 'Intel Xeon',
    cpu_cores: 8,
    memory_total: 17179869184,
    platform: 'linux',
    kernel: '6.1.0',
    architecture: 'amd64',
    last_seen: new Date().toISOString(),
    created_at: new Date().toISOString(),
    ...overrides,
  }
}

// shadcn/ui sidebar requires a provider context; we provide a minimal mock
jest.mock('@/components/ui/sidebar', () => {
  const React = require('react')
  const setOpenMobile = jest.fn()

  return {
    useSidebar: () => ({ setOpenMobile }),
    SidebarMenuItem: ({ children, ...props }: React.ComponentProps<'li'>) => (
      <li data-sidebar="menu-item" {...props}>{children}</li>
    ),
    SidebarMenuButton: ({
      children,
      isActive,
      tooltip: _tooltip,
      ...props
    }: React.ComponentProps<'button'> & { isActive?: boolean; tooltip?: string }) => (
      <button data-active={isActive} data-sidebar="menu-button" {...props}>
        {children}
      </button>
    ),
    SidebarMenuSub: ({ children, ...props }: React.ComponentProps<'ul'>) => (
      <ul data-sidebar="menu-sub" {...props}>{children}</ul>
    ),
    SidebarMenuSubItem: ({ children, ...props }: React.ComponentProps<'li'>) => (
      <li data-sidebar="menu-sub-item" {...props}>{children}</li>
    ),
    SidebarMenuSubButton: ({
      children,
      isActive,
      asChild: _asChild,
      ...props
    }: React.ComponentProps<'a'> & { isActive?: boolean; asChild?: boolean }) => (
      <div data-active={isActive} data-sidebar="menu-sub-button" {...props}>
        {children}
      </div>
    ),
  }
})

// ---- Tests ----

describe('ServerNavItem', () => {
  beforeEach(() => {
    mockPathname = '/dashboard'
  })

  it('renders the server hostname', () => {
    render(<ServerNavItem server={makeServer()} />)

    expect(screen.getByText('web-server-01')).toBeInTheDocument()
  })

  it('shows "Offline" badge for offline servers', () => {
    render(<ServerNavItem server={makeServer({ online: false })} />)

    expect(screen.getByText('Offline')).toBeInTheDocument()
  })

  it('does not show Offline badge for online servers', () => {
    render(<ServerNavItem server={makeServer({ online: true })} />)

    expect(screen.queryByText('Offline')).not.toBeInTheDocument()
  })

  it('renders sub-items: Metryki, Logi, Kontenery', () => {
    render(<ServerNavItem server={makeServer()} />)

    // Collapsible is open by default only when the server is active.
    // Click trigger to open it:
    const trigger = screen.getByText('web-server-01').closest('button')!
    // The collapsible trigger should open on click
    expect(trigger).toBeInTheDocument()
  })

  it('highlights the active route for Metryki', () => {
    mockPathname = '/servers/test-uuid-123/metrics'

    render(<ServerNavItem server={makeServer()} />)

    // The server button should be marked as active
    const serverButton = screen.getByText('web-server-01').closest('[data-sidebar="menu-button"]')
    expect(serverButton).toHaveAttribute('data-active', 'true')
  })

  it('highlights the active route for Kontenery', () => {
    mockPathname = '/servers/test-uuid-123/containers'

    render(<ServerNavItem server={makeServer()} />)

    const serverButton = screen.getByText('web-server-01').closest('[data-sidebar="menu-button"]')
    expect(serverButton).toHaveAttribute('data-active', 'true')
  })

  it('marks server as inactive when a different server is active', () => {
    mockPathname = '/servers/different-uuid/metrics'

    render(<ServerNavItem server={makeServer()} />)

    const serverButton = screen.getByText('web-server-01').closest('[data-sidebar="menu-button"]')
    expect(serverButton).toHaveAttribute('data-active', 'false')
  })

  it('renders Logi link as disabled', () => {
    mockPathname = '/servers/test-uuid-123/metrics'

    render(<ServerNavItem server={makeServer()} />)

    const logiLink = screen.getByText('Logi').closest('a')
    expect(logiLink).toHaveAttribute('href', '#')
  })

  it('renders correct href for Metryki', () => {
    mockPathname = '/servers/test-uuid-123/metrics'

    render(<ServerNavItem server={makeServer()} />)

    const link = screen.getByText('Metryki').closest('a')
    expect(link).toHaveAttribute('href', '/servers/test-uuid-123/metrics')
  })

  it('renders correct href for Kontenery', () => {
    mockPathname = '/servers/test-uuid-123/metrics'

    render(<ServerNavItem server={makeServer()} />)

    const link = screen.getByText('Kontenery').closest('a')
    expect(link).toHaveAttribute('href', '/servers/test-uuid-123/containers')
  })

  it('opens collapsible by default when server route is active', () => {
    mockPathname = '/servers/test-uuid-123/metrics'

    render(<ServerNavItem server={makeServer()} />)

    // When the server is active, collapsible is defaultOpen=true,
    // so sub-items should be visible
    expect(screen.getByText('Metryki')).toBeVisible()
    expect(screen.getByText('Kontenery')).toBeVisible()
  })
})
