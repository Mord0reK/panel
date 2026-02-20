import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

import { ContainerActions } from '@/components/containers/ContainerActions'

// Mock API
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const mockContainerCommand: jest.Mock<any, any> = jest.fn(() => Promise.resolve({}))
jest.mock('@/lib/api', () => ({
  api: {
    containerCommand: (...args: string[]) => mockContainerCommand(...args),
  },
}))

describe('ContainerActions', () => {
  beforeEach(() => {
    mockContainerCommand.mockClear()
    mockContainerCommand.mockResolvedValue({})
  })

  it('renders 3 action buttons', () => {
    render(<ContainerActions uuid="uuid-1" containerId="c-1" />)

    expect(screen.getByTitle('Start')).toBeInTheDocument()
    expect(screen.getByTitle('Stop')).toBeInTheDocument()
    expect(screen.getByTitle('Restart')).toBeInTheDocument()
  })

  it('calls containerCommand with start action on click', async () => {
    const user = userEvent.setup()
    render(<ContainerActions uuid="uuid-1" containerId="c-1" />)

    await user.click(screen.getByTitle('Start'))

    expect(mockContainerCommand).toHaveBeenCalledWith('uuid-1', 'c-1', 'start')
  })

  it('calls containerCommand with stop action on click', async () => {
    const user = userEvent.setup()
    render(<ContainerActions uuid="uuid-1" containerId="c-1" />)

    await user.click(screen.getByTitle('Stop'))

    expect(mockContainerCommand).toHaveBeenCalledWith('uuid-1', 'c-1', 'stop')
  })

  it('calls containerCommand with restart action on click', async () => {
    const user = userEvent.setup()
    render(<ContainerActions uuid="uuid-1" containerId="c-1" />)

    await user.click(screen.getByTitle('Restart'))

    expect(mockContainerCommand).toHaveBeenCalledWith(
      'uuid-1',
      'c-1',
      'restart',
    )
  })

  it('disables all buttons while action is pending', async () => {
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

    await user.click(screen.getByTitle('Start'))

    // All buttons should be disabled
    expect(screen.getByTitle('Start')).toBeDisabled()
    expect(screen.getByTitle('Stop')).toBeDisabled()
    expect(screen.getByTitle('Restart')).toBeDisabled()

    // Resolve to clean up
    resolvePromise!()
    await waitFor(() => {
      expect(screen.getByTitle('Start')).not.toBeDisabled()
    })
  })

  it('re-enables buttons after action completes', async () => {
    const user = userEvent.setup()
    render(<ContainerActions uuid="uuid-1" containerId="c-1" />)

    await user.click(screen.getByTitle('Stop'))

    await waitFor(() => {
      expect(screen.getByTitle('Start')).not.toBeDisabled()
      expect(screen.getByTitle('Stop')).not.toBeDisabled()
      expect(screen.getByTitle('Restart')).not.toBeDisabled()
    })
  })

  it('shows error indicator on failure', async () => {
    mockContainerCommand.mockRejectedValue(new Error('timeout'))

    const user = userEvent.setup()
    render(<ContainerActions uuid="uuid-1" containerId="c-1" />)

    await user.click(screen.getByTitle('Start'))

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

    await user.click(screen.getByTitle('Start'))
    await waitFor(() => {
      expect(screen.getByText('Błąd')).toBeInTheDocument()
    })

    await user.click(screen.getByTitle('Stop'))
    await waitFor(() => {
      expect(screen.queryByText('Błąd')).not.toBeInTheDocument()
    })
  })
})
