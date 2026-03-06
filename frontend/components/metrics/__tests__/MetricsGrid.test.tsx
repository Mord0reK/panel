import { render, screen } from '@testing-library/react'

// Mock next/dynamic — synchroniczna rozwiązka dynamicznych importów w testach
jest.mock('next/dynamic', () => {
  const moduleMap: Record<string, string> = {
    LiveChart: '@/components/charts/LiveChart',
    HistoryChart: '@/components/charts/HistoryChart',
  }

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return function mockDynamic(loader: any) {
    const loaderStr = loader.toString()

    for (const [name, path] of Object.entries(moduleMap)) {
      if (loaderStr.includes(name)) {
        const mod = jest.requireMock(path) as Record<string, unknown>
        return mod[name] ?? mod.default ?? Object.values(mod)[0]
      }
    }

    return () => null
  }
})

import { MetricsGrid } from '@/components/metrics/MetricsGrid'

// Mock hooks i komponenty wykresów
jest.mock('@/hooks/useServerMetrics', () => ({
  useServerMetrics: jest.fn(() => ({
    hostPoints: [],
    containerPoints: new Map(),
    connected: true,
    error: null,
  })),
}))

jest.mock('@/components/charts/LiveChart', () => ({
  LiveChart: ({ type }: { type: string }) => (
    <div data-testid={`live-chart-${type}`}>LiveChart-{type}</div>
  ),
}))

jest.mock('@/components/charts/HistoryChart', () => ({
  HistoryChart: ({ type }: { type: string }) => (
    <div data-testid={`history-chart-${type}`}>HistoryChart-{type}</div>
  ),
}))

jest.mock('@/lib/api', () => ({
  api: {
    getMetricsHistory: jest.fn(() =>
      Promise.resolve({
        host: { points: [] },
        containers: [],
      }),
    ),
  },
}))

describe('MetricsGrid', () => {
  beforeEach(() => {
    const { useServerMetrics } = jest.requireMock('@/hooks/useServerMetrics')
    useServerMetrics.mockReturnValue({
      hostPoints: [],
      containerPoints: new Map(),
      connected: true,
      error: null,
    })
  })

  it('renders 4 LiveChart components in live mode (default)', () => {
    render(<MetricsGrid uuid="test-uuid" />)

    expect(screen.getByTestId('live-chart-cpu')).toBeInTheDocument()
    expect(screen.getByTestId('live-chart-ram')).toBeInTheDocument()
    expect(screen.getByTestId('live-chart-disk')).toBeInTheDocument()
    expect(screen.getByTestId('live-chart-net')).toBeInTheDocument()
  })

  it('shows Live status indicator when connected', () => {
    render(<MetricsGrid uuid="test-uuid" />)

    expect(screen.getByText('Na żywo')).toBeInTheDocument()
  })

  it('shows connection error indicator', () => {
    const { useServerMetrics } = jest.requireMock('@/hooks/useServerMetrics')
    useServerMetrics.mockReturnValue({
      hostPoints: [],
      containerPoints: new Map(),
      connected: false,
      error: 'Connection failed',
    })

    render(<MetricsGrid uuid="test-uuid" />)

    expect(screen.getByText('Błąd połączenia')).toBeInTheDocument()
  })

  it('shows connecting indicator when not connected and no error', () => {
    const { useServerMetrics } = jest.requireMock('@/hooks/useServerMetrics')
    useServerMetrics.mockReturnValue({
      hostPoints: [],
      containerPoints: new Map(),
      connected: false,
      error: null,
    })

    render(<MetricsGrid uuid="test-uuid" />)

    expect(screen.getByText('Łączenie…')).toBeInTheDocument()
  })

  it('renders RangeDropdown with live range label', () => {
    render(<MetricsGrid uuid="test-uuid" />)

    expect(screen.getByText('1 minuta (Na żywo)')).toBeInTheDocument()
  })

  it('passes uuid to useServerMetrics', () => {
    const { useServerMetrics } = jest.requireMock('@/hooks/useServerMetrics')

    render(<MetricsGrid uuid="my-uuid" />)

    expect(useServerMetrics).toHaveBeenCalledWith('my-uuid', true)
  })

  it('stacks toolbar controls on mobile before switching to a row layout', () => {
    const { container } = render(<MetricsGrid uuid="test-uuid" />)

    const toolbar = container.querySelector('.space-y-4 > .flex')
    expect(toolbar).toHaveClass('flex-col')
    expect(toolbar).toHaveClass('sm:flex-row')
  })
})
