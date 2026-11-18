import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import '@testing-library/jest-dom'
import UptimeBar from '@/components/status/UptimeBar'

describe('UptimeBar', () => {
  it('names the whole row with its percentage for screen readers', () => {
    render(<UptimeBar label="24 hours" bars={[100, 100]} hoursPerBar={1} uptime={99.95} />)
    expect(screen.getByRole('img', { name: '24 hours: 99.95% uptime' })).toBeInTheDocument()
    expect(screen.getByText('99.95%')).toBeInTheDocument()
  })

  it('renders one bar per period with a tooltip and a tone', () => {
    render(<UptimeBar label="24 hours" bars={[null, 50, 95, 100]} hoursPerBar={1} uptime={81.67} />)
    const bars = Array.from(screen.getByRole('img').children)
    expect(bars.map((bar) => bar.getAttribute('data-tone'))).toEqual([
      'none',
      'down',
      'partial',
      'up',
    ])
    expect(bars.map((bar) => bar.getAttribute('title'))).toEqual([
      '3 hours ago: no data',
      '2 hours ago: 50% uptime',
      '1 hour ago: 95% uptime',
      'This hour: 100% uptime',
    ])
  })

  it('says so when there is no data at all', () => {
    render(<UptimeBar label="7 days" bars={[null, null]} hoursPerBar={6} uptime={null} />)
    expect(screen.getByRole('img', { name: '7 days: no data' })).toBeInTheDocument()
    expect(screen.getByText('No data')).toBeInTheDocument()
  })
})
