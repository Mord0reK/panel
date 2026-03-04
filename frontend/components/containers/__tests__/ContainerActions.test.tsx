import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

import { ContainerActions } from '@/components/containers/ContainerActions'

// Mock API
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const mockContainerCommand: jest.Mock<any, any> = jest.fn(() => Promise.resolve({}))
const mockCheckContainerUpdate: jest.Mock<any, any> = jest.fn(() => Promise.resolve({}))
const mockUpdateContainer: jest.Mock<any, any> = jest.fn(() => Promise.resolve({}))

jest.mock('@/lib/api', () => ({
  api: {
    containerCommand: (...args: string[]) => mockContainerCommand(...args),
    checkContainerUpdate: (...args: string[]) => mockCheckContainerUpdate(...args),
    updateContainer: (...args: string[]) => mockUpdateContainer(...args),
  },
}))

describe('ContainerActions', () => {
  beforeEach(() => {
    mockContainerCommand.mockClear()
    mockContainerCommand.mockResolvedValue({})
    mockCheckContainerUpdate.mockClear()
    mockCheckContainerUpdate.mockResolvedValue({ updates: [{ status: 'up_to_date', update_available: false }] })
    mockUpdateContainer.mockClear()
    mockUpdateContainer.mockResolvedValue({ results: [{ success: true }] })
  })

  it('renders actions dropdown trigger', () => {
    render(<ContainerActions uuid="uuid-1" containerId="c-1" />)

    expect(screen.getByTitle('Akcje')).toBeInTheDocument()
  })

  it('calls containerCommand with start action on click', async () => {
    const user = userEvent.setup()
    render(<ContainerActions uuid="uuid-1" containerId="c-1" />)

    await user.click(screen.getByTitle('Akcje'))
    await user.click(screen.getByText('Uruchom'))

    expect(mockContainerCommand).toHaveBeenCalledWith('uuid-1', 'c-1', 'start')
  })

  it('calls containerCommand with stop action on click', async () => {
    const user = userEvent.setup()
    render(<ContainerActions uuid="uuid-1" containerId="c-1" />)

    await user.click(screen.getByTitle('Akcje'))
    await user.click(screen.getByText('Zatrzymaj'))

    expect(mockContainerCommand).toHaveBeenCalledWith('uuid-1', 'c-1', 'stop')
  })

  it('calls containerCommand with restart action on click', async () => {
    const user = userEvent.setup()
    render(<ContainerActions uuid="uuid-1" containerId="c-1" />)

    await user.click(screen.getByTitle('Akcje'))
    await user.click(screen.getByText('Restartuj'))

    expect(mockContainerCommand).toHaveBeenCalledWith(
      'uuid-1',
      'c-1',
      'restart',
    )
  })

  it('disables dropdown trigger while action is pending', async () => {
    // Make the promise never resolve during the test
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    let resolvePromise: (value?: any) => void
    mockContainerCommand.mockReturnValue(
      new Promise((resolve) => {
        resolvePromise = resolve
      }),
    )

    const user = userEvent.setup()
    render(<ContainerActions uuid="uuid-1" containerId="c-1" />)

    await user.click(screen.getByTitle('Akcje'))
    await user.click(screen.getByText('Uruchom'))

    // Trigger should be disabled
    expect(screen.getByTitle('Akcje')).toBeDisabled()

    // Resolve to clean up
    resolvePromise!()
    await waitFor(() => {
      expect(screen.getByTitle('Akcje')).not.toBeDisabled()
    })
  })

  it('re-enables dropdown trigger after action completes', async () => {
    const user = userEvent.setup()
    render(<ContainerActions uuid="uuid-1" containerId="c-1" />)

    await user.click(screen.getByTitle('Akcje'))
    await user.click(screen.getByText('Zatrzymaj'))

    await waitFor(() => {
      expect(screen.getByTitle('Akcje')).not.toBeDisabled()
    })
  })

  it('shows error indicator on failure', async () => {
    mockContainerCommand.mockRejectedValue(new Error('timeout'))

    const user = userEvent.setup()
    render(<ContainerActions uuid="uuid-1" containerId="c-1" />)

    await user.click(screen.getByTitle('Akcje'))
    await user.click(screen.getByText('Uruchom'))

    await waitFor(() => {
      expect(screen.getByText('Błąd')).toBeInTheDocument()
    })
  })

  it('clears error on next action attempt', async () => {
    // First call fails
    mockContainerCommand.mockRejectedValueOnce(new Error('fail'))
    // Second call succeeds
    mockContainerCommand.mockResolvedValueOnce({})

    const user = userEvent.setup()
    render(<ContainerActions uuid="uuid-1" containerId="c-1" />)

    await user.click(screen.getByTitle('Akcje'))
    await user.click(screen.getByText('Uruchom'))
    await waitFor(() => {
      expect(screen.getByText('Błąd')).toBeInTheDocument()
    })

    await user.click(screen.getByTitle('Akcje'))
    await user.click(screen.getByText('Zatrzymaj'))
    await waitFor(() => {
      expect(screen.queryByText('Błąd')).not.toBeInTheDocument()
    })
  })
})
