import { render, screen } from '@testing-library/react'

import { ServerCard } from '@/components/servers/ServerCard'
import type { LiveServerSnapshot } from '@/types'

// Mock next/link
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

function makeSnapshot(
  overrides: Partial<LiveServerSnapshot> = {}
): LiveServerSnapshot {
  return {
    uuid: 'uuid-1',
    hostname: 'web-01',
    cpu: 42.5,
    memory: 8_589_934_592, // 8 GB
    mem_percent: 50,
    memory_total: 16_000_000_000,
    disk_used_percent: 50,
    disk_used: 10_737_418_240, // 10 GB approx
    disk_read_bytes_per_sec: 1_048_576, // 1 MB/s
    disk_write_bytes_per_sec: 524_288, // 512 KB/s
    net_rx_bytes_per_sec: 2_097_152, // 2 MB/s
    net_tx_bytes_per_sec: 1_048_576, // 1 MB/s
    ...overrides,
  }
}

describe('ServerCard', () => {
  it('renders hostname', () => {
    render(
      <ServerCard hostname="web-01" uuid="uuid-1" snapshot={makeSnapshot()} />
    )

    expect(screen.getByText('web-01')).toBeInTheDocument()
  })

  it('shows Online when snapshot is provided', () => {
    render(
      <ServerCard hostname="web-01" uuid="uuid-1" snapshot={makeSnapshot()} />
    )

    expect(screen.getByText('Online')).toBeInTheDocument()
  })

  it('shows Offline when snapshot is null', () => {
    render(
      <ServerCard hostname="web-01" uuid="uuid-1" snapshot={null} />
    )

    expect(screen.getByText('Offline')).toBeInTheDocument()
  })

  it('displays CPU percentage', () => {
    render(
      <ServerCard
        hostname="web-01"
        uuid="uuid-1"
        snapshot={makeSnapshot({ cpu: 75.3 })}
      />
    )

    // percentage number and symbol are separate nodes, just assert number
    expect(screen.getByText('75.30')).toBeInTheDocument()
  })



  it('shows dashes for metrics when offline', () => {
    render(
      <ServerCard hostname="web-01" uuid="uuid-1" snapshot={null} />
    )

    // Multiple dashes appear for each stat; just ensure at least three
    const dashes = screen.getAllByText('—')
    expect(dashes.length).toBeGreaterThanOrEqual(3)
  })

  it('links to correct server metrics page', () => {
    render(
      <ServerCard hostname="web-01" uuid="uuid-1" snapshot={makeSnapshot()} />
    )

    const link = screen.getByRole('link')
    expect(link).toHaveAttribute('href', '/servers/uuid-1/metrics')
  })

  it('displays disk percentage with absolute GB when available', () => {
    render(
      <ServerCard
        hostname="web-01"
        uuid="uuid-1"
        snapshot={makeSnapshot({ disk_used_percent: 74.4, disk_used: 11_000_000_000 })}
      />
    )

    // number portion of percent is separate
    expect(screen.getByText('74.40')).toBeInTheDocument()
    // formatter rounding can differ by a hundredth depending on unit conversion
    expect(
      screen.getByText((content) => /10\.2[45]/.test(content))
    ).toBeInTheDocument()
  })

  it('displays CPU progress bar at correct width', () => {
    const { container } = render(
      <ServerCard
        hostname="web-01"
        uuid="uuid-1"
        snapshot={makeSnapshot({ cpu: 60 })}
      />
    )

    const progressBar = container.querySelector('[style*=\"width\"]')
    expect(progressBar).toHaveStyle({ width: '60%' })
  })

  it('caps CPU progress bar at 100%', () => {
    const { container } = render(
      <ServerCard
        hostname="web-01"
        uuid="uuid-1"
        snapshot={makeSnapshot({ cpu: 120 })}
      />
    )

    const progressBar = container.querySelector('[style*="width"]')
    expect(progressBar).toHaveStyle({ width: '100%' })
  })

  it('uses a single-column metrics grid on mobile before switching to two columns', () => {
    const { container } = render(
      <ServerCard hostname="web-01" uuid="uuid-1" snapshot={makeSnapshot()} />
    )

    const metricsGrid = container.querySelector('.grid')
    expect(metricsGrid).toHaveClass('grid-cols-1')
    expect(metricsGrid).toHaveClass('sm:grid-cols-2')
  })

  it('reduces outer padding on mobile', () => {
    render(
      <ServerCard hostname="web-01" uuid="uuid-1" snapshot={makeSnapshot()} />
    )

    const link = screen.getByRole('link')
    expect(link).toHaveClass('p-3')
    expect(link).toHaveClass('sm:p-4')
  })
})
