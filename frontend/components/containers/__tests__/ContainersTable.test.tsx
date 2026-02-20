import { render, screen } from '@testing-library/react'

import { ContainersTable } from '@/components/containers/ContainersTable'
import type { Container } from '@/types'

// Mock ContainerActions — tested separately
jest.mock('@/components/containers/ContainerActions', () => ({
  ContainerActions: ({ containerId }: { containerId: string }) => (
    <div data-testid={`actions-${containerId}`}>Actions</div>
  ),
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
    first_seen: '2025-01-01T00:00:00Z',
    last_seen: new Date().toISOString(),
    ...overrides,
  }
}

describe('ContainersTable', () => {
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

    expect(screen.getByText('redis:7')).toBeInTheDocument()
  })

  it('renders project name', () => {
    render(
      <ContainersTable
        uuid="uuid-1"
        containers={[makeContainer({ project: 'myproject' })]}
      />,
    )

    expect(screen.getByText('myproject')).toBeInTheDocument()
  })

  it('renders service name', () => {
    render(
      <ContainersTable
        uuid="uuid-1"
        containers={[makeContainer({ service: 'api' })]}
      />,
    )

    expect(screen.getByText('api')).toBeInTheDocument()
  })

  it('shows dash for empty project', () => {
    render(
      <ContainersTable
        uuid="uuid-1"
        containers={[makeContainer({ project: '' })]}
      />,
    )

    expect(screen.getAllByText('—').length).toBeGreaterThanOrEqual(1)
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
    expect(screen.getByText('Obraz')).toBeInTheDocument()
    expect(screen.getByText('Projekt')).toBeInTheDocument()
    expect(screen.getByText('Serwis')).toBeInTheDocument()
    expect(screen.getByText('Akcje')).toBeInTheDocument()
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
