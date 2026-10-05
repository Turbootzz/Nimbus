import { fireEvent, render, screen, within } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import '@testing-library/jest-dom'
import CalendarWidget from '@/components/widgets/renderers/CalendarWidget'
import CalendarForm from '@/components/widgets/forms/CalendarForm'
import { monthWeeks } from '@/lib/calendar'
import type { CalendarPayload, CalendarWidgetConfig } from '@/types'
import { makeSnapshot, makeWidget } from './fixtures'

vi.mock('@/hooks/useIntegrations', () => ({
  useIntegrations: () => ({
    isLoading: false,
    error: null,
    integrations: [
      { id: 's1', kind: 'sonarr', name: 'Sonarr' },
      { id: 'p1', kind: 'pihole', name: 'Pi-hole' },
    ],
  }),
}))

const config: CalendarWidgetConfig = {
  integrations: ['s1'],
  ical_urls: [],
  days: 14,
  verify_tls: true,
}
const payload: CalendarPayload = {
  from: '2026-10-01',
  sources: [
    { name: 'Sonarr', kind: 'sonarr' },
    { name: 'cal.test', kind: 'ical' },
  ],
  events: [
    { title: 'Old episode', start: '2026-10-02', all_day: true, source: 0 },
    { title: 'Andor S01E03', start: '2026-10-07', all_day: true, source: 0 },
    { title: 'Bin day', start: '2026-10-07', all_day: true, source: 1 },
    { title: 'Far away', start: '2026-10-30', all_day: true, source: 1 },
  ],
}

describe('calendar helpers', () => {
  it('lays out a month in weeks starting on Monday', () => {
    const weeks = monthWeeks('2026-10-01')
    expect(weeks[0]).toEqual([
      null,
      null,
      null,
      '2026-10-01',
      '2026-10-02',
      '2026-10-03',
      '2026-10-04',
    ])
    expect(weeks.at(-1)).toEqual([
      '2026-10-26',
      '2026-10-27',
      '2026-10-28',
      '2026-10-29',
      '2026-10-30',
      '2026-10-31',
      null,
    ])
  })
})

describe('CalendarWidget', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 9, 5, 12, 0))
  })
  afterEach(() => vi.useRealTimers())

  const base = { widget: makeWidget({ type: 'calendar' }), config, openInNewTab: false }

  it('shows the month and the coming days, and filters by a clicked day', () => {
    render(<CalendarWidget {...base} cardSize="2x2" snapshot={makeSnapshot({ payload })} />)
    const agenda = screen.getByRole('list')
    expect(
      within(agenda)
        .getAllByRole('listitem')
        .map((li) => li.textContent)
    ).toEqual([expect.stringContaining('Andor S01E03'), expect.stringContaining('Bin day')])
    expect(screen.queryByText('Old episode')).not.toBeInTheDocument()
    expect(screen.queryByText('Far away')).not.toBeInTheDocument()

    // The label is in the browser's language, so find the day by its number
    const day30 = screen.getAllByRole('button').find((b) => b.textContent === '30')!
    expect(day30).toHaveAccessibleName(expect.stringContaining('has events'))
    fireEvent.click(day30)
    expect(screen.getByText('Far away')).toBeInTheDocument()
    expect(screen.queryByText('Bin day')).not.toBeInTheDocument()
  })

  it('shows only the agenda on a smaller tile, and failed sources', () => {
    render(
      <CalendarWidget
        {...base}
        cardSize="2x1"
        snapshot={makeSnapshot({ payload: { ...payload, failed: ['Radarr: timeout'] } })}
      />
    )
    expect(screen.queryByRole('table')).not.toBeInTheDocument()
    expect(screen.getByText('1 source could not be loaded')).toHaveAttribute(
      'title',
      'Radarr: timeout'
    )
  })
})

describe('CalendarForm', () => {
  it('offers only Sonarr and Radarr, and adds feeds', () => {
    const onChange = vi.fn()
    render(<CalendarForm config={{ ...config, integrations: [] }} onChange={onChange} />)
    expect(screen.queryByText('Pi-hole')).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole('checkbox', { name: 'Sonarr' }))
    expect(onChange).toHaveBeenLastCalledWith(expect.objectContaining({ integrations: ['s1'] }))

    fireEvent.click(screen.getByText('Add calendar'))
    expect(onChange).toHaveBeenLastCalledWith(expect.objectContaining({ ical_urls: [''] }))
  })
})
