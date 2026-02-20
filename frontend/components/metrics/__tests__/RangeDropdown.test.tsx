import { render, screen } from '@testing-library/react'

import { RangeDropdown } from '@/components/metrics/RangeDropdown'

describe('RangeDropdown', () => {
  it('renders LIVE badge for 1m value', () => {
    const onChange = jest.fn()
    render(<RangeDropdown value="1m" onChange={onChange} />)

    expect(screen.getByText('LIVE')).toBeInTheDocument()
    expect(screen.getByText('1m')).toBeInTheDocument()
  })

  it('renders non-live value label in trigger', () => {
    const onChange = jest.fn()
    render(<RangeDropdown value="5m" onChange={onChange} />)

    expect(screen.getByText('5 minut')).toBeInTheDocument()
  })

  it('renders trigger as combobox role', () => {
    const onChange = jest.fn()
    render(<RangeDropdown value="1m" onChange={onChange} />)

    expect(screen.getByRole('combobox')).toBeInTheDocument()
  })

  it('renders 15m label correctly', () => {
    const onChange = jest.fn()
    render(<RangeDropdown value="15m" onChange={onChange} />)

    expect(screen.getByText('15 minut')).toBeInTheDocument()
  })

  it('renders 1h label correctly', () => {
    const onChange = jest.fn()
    render(<RangeDropdown value="1h" onChange={onChange} />)

    expect(screen.getByText('1 godzina')).toBeInTheDocument()
  })

  it('renders 7d label correctly', () => {
    const onChange = jest.fn()
    render(<RangeDropdown value="7d" onChange={onChange} />)

    expect(screen.getByText('7 dni')).toBeInTheDocument()
  })
})
