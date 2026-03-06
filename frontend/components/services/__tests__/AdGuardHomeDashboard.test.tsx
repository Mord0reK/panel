import { render, screen } from '@testing-library/react'

import { AdGuardHomeDashboard } from '@/components/services/AdGuardHomeDashboard'
import type { AdGuardHomeDashboardResponse } from '@/types'

function makeDashboard(
  overrides: Partial<AdGuardHomeDashboardResponse> = {}
): AdGuardHomeDashboardResponse {
  return {
    service_key: 'adguardhome',
    status: {
      protection_enabled: true,
      running: true,
    },
    version: {
      current_version: '0.107.55',
      latest_version: '0.107.57',
      announcement: 'Nowe filtry i poprawki stabilności.',
      announcement_url: 'https://example.test/release',
      can_auto_update: true,
      update_available: true,
      check_disabled: false,
      check_failed: false,
    },
    stats: {
      time_units: 'hours',
      num_dns_queries: 12_500,
      num_blocked_filtering: 5_000,
      num_replaced_safebrowsing: 45,
      num_replaced_safesearch: 25,
      num_replaced_parental: 10,
      avg_processing_time: 1.75,
      dns_queries: [10, 20, 30, 40],
      blocked_filtering: [5, 10, 12, 18],
      replaced_safebrowsing: [1, 2, 3, 4],
      replaced_parental: [0, 1, 0, 2],
      top_clients: [
        { name: 'MacBook-Pro', count: 3200 },
        { name: 'iPhone', count: 1800 },
      ],
      top_queried_domains: [{ name: 'github.com', count: 800 }],
      top_blocked_domains: [{ name: 'ads.example.com', count: 350 }],
    },
    ...overrides,
  }
}

describe('AdGuardHomeDashboard', () => {
  it('renders key status and rankings', () => {
    render(
      <AdGuardHomeDashboard
        serviceName="AdGuard Home"
        dashboard={makeDashboard()}
      />
    )

    expect(screen.getByText('AdGuard Home')).toBeInTheDocument()
    expect(screen.getByText('Aktywna ochrona')).toBeInTheDocument()
    expect(screen.getByText('Dostępna aktualizacja')).toBeInTheDocument()
    expect(screen.getByText('40.00%')).toBeInTheDocument()
    expect(screen.getByText('MacBook-Pro')).toBeInTheDocument()
    expect(screen.getByText('ads.example.com')).toBeInTheDocument()
    expect(screen.getByText('1.75 ms')).toBeInTheDocument()
  })

  it('shows current version state when update is not available', () => {
    render(
      <AdGuardHomeDashboard
        serviceName="AdGuard Home"
        dashboard={makeDashboard({
          version: {
            current_version: '0.107.57',
            latest_version: '0.107.57',
            announcement: '',
            announcement_url: '',
            can_auto_update: true,
            update_available: false,
            check_disabled: false,
            check_failed: false,
          },
        })}
      />
    )

    expect(screen.getByText('Aktualna wersja')).toBeInTheDocument()
    expect(
      screen.getByText(/Instancja działa na wersji 0.107.57/)
    ).toBeInTheDocument()
  })
})