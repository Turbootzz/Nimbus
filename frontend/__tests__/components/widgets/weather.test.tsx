import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import '@testing-library/jest-dom'
import WeatherWidget from '@/components/widgets/renderers/WeatherWidget'
import WeatherForm from '@/components/widgets/forms/WeatherForm'
import type { WeatherWidgetConfig } from '@/types'
import { makeSnapshot, makeWidget } from './fixtures'

const config: WeatherWidgetConfig = {
  latitude: 52.37,
  longitude: 4.89,
  location: 'Amsterdam',
  units: 'metric',
}
const base = { widget: makeWidget({ type: 'weather' }), config, openInNewTab: false }

describe('WeatherWidget', () => {
  it('shows the current weather and the next days on a wide tile', () => {
    render(<WeatherWidget {...base} cardSize="2x1" snapshot={makeSnapshot()} />)
    expect(screen.getByText('17°')).toBeInTheDocument()
    expect(screen.getByText('Amsterdam')).toBeInTheDocument()
    // One row high: a short detail line, the wind is in its tooltip
    expect(screen.getByText('Rain · feels like 15°')).toHaveAttribute(
      'title',
      'Rain · feels like 15° · wind 7 km/h'
    )
    expect(screen.getAllByRole('listitem')).toHaveLength(3) // today is already shown
    expect(screen.getByLabelText('Thunderstorm')).toBeInTheDocument()
  })

  it('shows all days on a large tile and only the basics on a small one', () => {
    const { unmount } = render(<WeatherWidget {...base} cardSize="2x2" snapshot={makeSnapshot()} />)
    expect(screen.getAllByRole('listitem')).toHaveLength(4)
    expect(screen.getByText(/Rain · feels like 15° · wind 7 km\/h/)).toBeInTheDocument()
    unmount()

    render(<WeatherWidget {...base} cardSize="1x1" snapshot={makeSnapshot()} />)
    expect(screen.queryByRole('listitem')).not.toBeInTheDocument()
    expect(screen.queryByText(/feels like/)).not.toBeInTheDocument()
    expect(screen.getByText('17°')).toBeInTheDocument()
  })

  it('renders nothing without data (the card shows loading)', () => {
    const { container } = render(<WeatherWidget {...base} cardSize="2x1" />)
    expect(container).toBeEmptyDOMElement()
  })
})

describe('WeatherForm', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('fills in the coordinates of a found place', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          results: [{ name: 'Utrecht', latitude: 52.09, longitude: 5.12, country: 'Netherlands' }],
        }),
      })
    )
    const onChange = vi.fn()
    render(<WeatherForm config={{ location: '', units: 'metric' }} onChange={onChange} />)

    fireEvent.change(screen.getByLabelText('Find a place'), { target: { value: 'Utrecht' } })
    fireEvent.keyDown(screen.getByLabelText('Find a place'), { key: 'Enter' })
    fireEvent.click(await screen.findByText('Utrecht, Netherlands'))

    expect(onChange).toHaveBeenCalledWith({
      location: 'Utrecht',
      units: 'metric',
      latitude: 52.09,
      longitude: 5.12,
    })
  })

  it('shows the chosen place', () => {
    const { rerender } = render(
      <WeatherForm config={{ location: '', units: 'metric' }} onChange={() => {}} />
    )
    expect(screen.queryByRole('status')).not.toBeInTheDocument()

    rerender(
      <WeatherForm
        config={{ location: 'Utrecht', units: 'metric', latitude: 52.0908, longitude: 5.1222 }}
        onChange={() => {}}
      />
    )
    expect(screen.getByRole('status')).toHaveTextContent('Utrecht52.09, 5.12')
  })

  it('shows search errors and accepts coordinates by hand', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 500 }))
    const onChange = vi.fn()
    render(<WeatherForm config={{ location: '', units: 'metric' }} onChange={onChange} />)

    fireEvent.change(screen.getByLabelText('Find a place'), { target: { value: 'x' } })
    fireEvent.click(screen.getByLabelText('Search places'))
    await waitFor(() => expect(screen.getByText(/Location search failed/)).toBeInTheDocument())

    fireEvent.change(screen.getByLabelText('Latitude'), { target: { value: '48.85' } })
    expect(onChange).toHaveBeenLastCalledWith({ location: '', units: 'metric', latitude: 48.85 })
  })
})
