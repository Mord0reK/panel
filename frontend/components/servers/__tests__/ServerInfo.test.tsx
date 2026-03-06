import { render, screen } from '@testing-library/react'

import { ServerInfo } from '@/components/servers/ServerInfo'
import type { Server } from '@/types'

function makeServer(overrides: Partial<Server> = {}): Server {
  return {
    uuid: 'test-uuid',
    hostname: 'web-01',
    approved: true,
    cpu_model: 'Intel Core i7-12700K',
    cpu_cores: 12,
    cpu_threads: 20,
    memory_total: 34_359_738_368, // 32 GB
    platform: 'Ubuntu 24.04',
    kernel: '6.8.0-45-generic',
    architecture: 'amd64',
    last_seen: '2025-01-01T00:00:00Z',
    created_at: '2025-01-01T00:00:00Z',
    ...overrides,
  }
}

describe('ServerInfo', () => {
  it('renders hostname', () => {
    render(<ServerInfo server={makeServer()} />)
    expect(screen.getByRole('heading', { name: 'web-01' })).toBeInTheDocument()
  })

  it('renders CPU model', () => {
    render(<ServerInfo server={makeServer()} />)
    expect(screen.getByText(/Intel Core i7-12700K/)).toBeInTheDocument()
  })

  it('renders CPU cores count', () => {
    render(<ServerInfo server={makeServer()} />)
    expect(screen.getByText(/12 rdzeni/)).toBeInTheDocument()
  })

  it('renders formatted RAM total', () => {
    render(<ServerInfo server={makeServer()} />)
    // 34_359_738_368 bytes = 32 GB
    expect(screen.getByText('32 GB')).toBeInTheDocument()
  })

  it('renders platform', () => {
    render(<ServerInfo server={makeServer()} />)
    expect(screen.getByText('Ubuntu 24.04')).toBeInTheDocument()
  })

  it('renders kernel version', () => {
    render(<ServerInfo server={makeServer()} />)
    expect(screen.getByText('6.8.0-45-generic')).toBeInTheDocument()
  })

  it('renders architecture', () => {
    render(<ServerInfo server={makeServer()} />)
    expect(screen.getByText('amd64')).toBeInTheDocument()
  })

  it('omits CPU stats when cpu_model is missing', () => {
    render(<ServerInfo server={makeServer({ cpu_model: '' })} />)
    expect(screen.queryByText(/rdzeni/)).not.toBeInTheDocument()
  })

  it('omits CPU stats when cpu_cores is zero', () => {
    render(<ServerInfo server={makeServer({ cpu_cores: 0 })} />)
    expect(screen.queryByText(/rdzeni/)).not.toBeInTheDocument()
  })

  it('omits memory stats when memory_total is zero', () => {
    render(<ServerInfo server={makeServer({ memory_total: 0 })} />)
    expect(screen.queryByText('32 GB')).not.toBeInTheDocument()
  })

  it('stacks the layout on mobile before switching to a horizontal row', () => {
    const { container } = render(<ServerInfo server={makeServer()} />)

    expect(container.firstChild).toHaveClass('flex-col')
    expect(container.firstChild).toHaveClass('sm:flex-row')
  })
})
