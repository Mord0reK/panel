import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

import { ContainersTable } from '@/components/containers/ContainersTable'
import type { Container } from '@/types'

// Mock next/navigation to avoid "invariant expected app router to be mounted"
jest.mock('next/navigation', () => ({
  useRouter: () => ({ push: jest.fn(), refresh: jest.fn(), replace: jest.fn() }),
  usePathname: () => '/',
  useSearchParams: () => new URLSearchParams(),
}))

// Mock useServerMetrics — SSE/EventSource is not available in JSDOM
jest.mock('@/hooks/useServerMetrics', () => ({
  useServerMetrics: () => ({
    hostPoints: [],
    containerPoints: new Map(),
    connected: false,
    error: null,
  }),
}))

// Mock ContainerActions — tested separately
jest.mock('@/components/containers/ContainerActions', () => ({
  ContainerActions: ({ containerId }: { containerId: string }) => (
    <div data-testid={`actions-${containerId}`}>Actions</div>
  ),
}))

// Mock api to avoid real HTTP calls in unit tests
const mockDeleteContainers = jest.fn()
jest.mock('@/lib/api', () => ({
  api: {
    serverCommand: jest.fn().mockResolvedValue({}),
    deleteContainers: (...args: unknown[]) => mockDeleteContainers(...args),
  },
}))

function makeContainer(overrides: Partial<Container> = {}): Container {
  return {
    id: 1,
    agent_uuid: 'uuid-1',
    container_id: 'abc123',
    name: 'nginx',
    image: 'nginx:latest',
    project: 'web',
    service: 'frontend',
    state: 'running',
    first_seen: '2025-01-01T00:00:00Z',
    last_seen: new Date().toISOString(),
    ...overrides,
  }
}

describe('ContainersTable', () => {
  beforeEach(() => {
    mockDeleteContainers.mockReset()
    mockDeleteContainers.mockResolvedValue({ deleted: [], failed: [] })
  })

  it('shows empty state when no containers', () => {
    render(<ContainersTable uuid="uuid-1" containers={[]} />)

    expect(screen.getByText('Brak kontenerów')).toBeInTheDocument()
  })

  it('renders container name', () => {
    render(
      <ContainersTable
        uuid="uuid-1"
        containers={[makeContainer({ name: 'my-app' })]}
      />,
    )

    expect(screen.getByText('my-app')).toBeInTheDocument()
  })

  it('renders container image as badge', () => {
    render(
      <ContainersTable
        uuid="uuid-1"
        containers={[makeContainer({ image: 'redis:7' })]}
      />,
    )

    expect(screen.getAllByText('redis:7').length).toBeGreaterThan(0)
  })

  it('renders service name', () => {
    render(
      <ContainersTable
        uuid="uuid-1"
        containers={[makeContainer({ service: 'api' })]}
      />,
    )

    expect(screen.getAllByText('api').length).toBeGreaterThan(0)
  })

  it('renders actions for each container', () => {
    render(
      <ContainersTable
        uuid="uuid-1"
        containers={[
          makeContainer({ container_id: 'c1' }),
          makeContainer({ container_id: 'c2', id: 2, name: 'redis' }),
        ]}
      />,
    )

    expect(screen.getByTestId('actions-c1')).toBeInTheDocument()
    expect(screen.getByTestId('actions-c2')).toBeInTheDocument()
  })

  it('renders table headers', () => {
    render(
      <ContainersTable
        uuid="uuid-1"
        containers={[makeContainer()]}
      />,
    )

    expect(screen.getByText('Nazwa')).toBeInTheDocument()
    expect(screen.getByText('Stan')).toBeInTheDocument()
    expect(screen.getByText('Obraz')).toBeInTheDocument()
    expect(screen.getByText('Serwis')).toBeInTheDocument()
    expect(screen.getByText('Akcje')).toBeInTheDocument()
  })

  it('hides less important columns on smaller screens', () => {
    render(
      <ContainersTable
        uuid="uuid-1"
        containers={[makeContainer()]}
      />,
    )

    expect(screen.getByText('Health').closest('th')).toHaveClass('hidden')
    expect(screen.getByText('Health').closest('th')).toHaveClass('lg:table-cell')
    expect(screen.getByText('Obraz').closest('th')).toHaveClass('hidden')
    expect(screen.getByText('Obraz').closest('th')).toHaveClass('xl:table-cell')
    expect(screen.getByText('Serwis').closest('th')).toHaveClass('hidden')
    expect(screen.getByText('Serwis').closest('th')).toHaveClass('lg:table-cell')
  })

  it('renders project name as group header', () => {
    render(
      <ContainersTable
        uuid="uuid-1"
        containers={[makeContainer({ project: 'myproject' })]}
      />,
    )

    // project name appears in group header (uppercase via CSS, but text content is the same)
    expect(screen.getByText('myproject')).toBeInTheDocument()
  })

  it('renders standalone header for containers without project', () => {
    render(
      <ContainersTable
        uuid="uuid-1"
        containers={[makeContainer({ project: '' })]}
      />,
    )

    expect(screen.getByText('Standalone')).toBeInTheDocument()
  })

  it('groups containers by project', () => {
    render(
      <ContainersTable
        uuid="uuid-1"
        containers={[
          makeContainer({ container_id: 'c1', name: 'app-1', project: 'proj-a' }),
          makeContainer({ container_id: 'c2', id: 2, name: 'app-2', project: 'proj-b' }),
          makeContainer({ container_id: 'c3', id: 3, name: 'app-3', project: 'proj-a' }),
        ]}
      />,
    )

    expect(screen.getByText('proj-a')).toBeInTheDocument()
    expect(screen.getByText('proj-b')).toBeInTheDocument()
  })

  it('renders multiple containers as separate rows', () => {
    render(
      <ContainersTable
        uuid="uuid-1"
        containers={[
          makeContainer({ container_id: 'c1', name: 'app-1' }),
          makeContainer({ container_id: 'c2', id: 2, name: 'app-2' }),
          makeContainer({ container_id: 'c3', id: 3, name: 'app-3' }),
        ]}
      />,
    )

    expect(screen.getByText('app-1')).toBeInTheDocument()
    expect(screen.getByText('app-2')).toBeInTheDocument()
    expect(screen.getByText('app-3')).toBeInTheDocument()
  })
})

describe('BulkActionBar', () => {
  const twoContainers = [
    makeContainer({ container_id: 'c1', name: 'nginx' }),
    makeContainer({ container_id: 'c2', id: 2, name: 'redis', project: 'app' }),
  ]

  beforeEach(() => {
    mockDeleteContainers.mockReset()
    mockDeleteContainers.mockResolvedValue({ deleted: [], failed: [] })
  })

  it('shows Usuń button when bulkMode is on', () => {
    render(
      <ContainersTable
        uuid="uuid-1"
        containers={twoContainers}
        bulkMode={true}
        onToggleBulk={() => {}}
      />,
    )
    expect(screen.getByText('Usuń')).toBeInTheDocument()
  })

  it('does not show Usuń button when bulkMode is off', () => {
    render(
      <ContainersTable
        uuid="uuid-1"
        containers={twoContainers}
        bulkMode={false}
        onToggleBulk={() => {}}
      />,
    )
    expect(screen.queryByText('Usuń')).not.toBeInTheDocument()
  })

  it('opens delete dialog when Usuń is clicked', async () => {
    const user = userEvent.setup()
    render(
      <ContainersTable
        uuid="uuid-1"
        containers={twoContainers}
        bulkMode={true}
        onToggleBulk={() => {}}
      />,
    )

    // Select all containers via the header checkbox so the Usuń button is enabled
    const selectAllCheckbox = screen.getByLabelText('Zaznacz wszystkie')
    await user.click(selectAllCheckbox)

    await user.click(screen.getByText('Usuń'))
    expect(screen.getByTestId('bulk-delete-password')).toBeInTheDocument()
  })
})
