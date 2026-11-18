import { act, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import '@testing-library/jest-dom'
import ServiceCard from '@/components/ServiceCard'
import ServiceChip from '@/components/ServiceChip'
import ServiceListItem from '@/components/ServiceListItem'
import ServicesGrid from '@/components/ServicesGrid'
import { api } from '@/lib/api'
import { clearServiceDetailCache } from '@/lib/service-detail'
import { mergeTiles } from '@/lib/tiles'
import type { MetricsResponse, Service } from '@/types'

vi.mock('@/lib/api', () => ({ api: { getServiceMetrics: vi.fn() } }))
vi.mock('@/components/graphs/UptimeSparkline', () => ({
  default: ({ dataPoints }: { dataPoints: unknown[] }) => (
    <div data-testid="sparkline">{dataPoints.length} points</div>
  ),
}))

const NOW = new Date('2026-11-18T12:00:00Z')

const service: Service = {
  id: 's1',
  name: 'Plex',
  url: 'https://plex.test',
  icon: '🎬',
  icon_type: 'emoji',
  status: 'online',
  response_time: 42,
  position: 0,
  card_size: '2x1',
  monitoring_enabled: true,
  created_at: '2026-01-01T00:00:00Z',
}

const metrics = (overrides: Partial<MetricsResponse> = {}): MetricsResponse => ({
  service_id: 's1',
  time_range: { start: '2026-11-17T12:00:00Z', end: '2026-11-18T12:00:00Z' },
  uptime_percentage: 99.456,
  total_checks: 288,
  online_count: 286,
  offline_count: 2,
  avg_response_time: 40,
  min_response_time: 20,
  max_response_time: 90,
  data_points: [
    {
      timestamp: '2026-11-18T11:00:00Z',
      check_count: 12,
      online_count: 12,
      uptime_percentage: 100,
      avg_response_time: 40,
    },
  ],
  last_checked_at: '2026-11-18T11:58:00Z',
  last_error: 'connection refused',
  last_error_at: '2026-11-18T09:00:00Z',
  ...overrides,
})

const card = <ServiceCard service={service} openInNewTab={false} />

// Lets the request and the lazy chart settle
const settle = () => act(() => vi.advanceTimersByTimeAsync(0))

async function hover(element: HTMLElement) {
  fireEvent.mouseEnter(element)
  await act(() => vi.advanceTimersByTimeAsync(400))
}

describe('service hover detail', () => {
  beforeEach(() => {
    vi.useFakeTimers({ now: NOW })
    vi.clearAllMocks()
    clearServiceDetailCache()
    vi.mocked(api.getServiceMetrics).mockResolvedValue({ data: metrics() })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('requests nothing until a tile is hovered', async () => {
    const tiles = mergeTiles([service, { ...service, id: 's2', name: 'Sonarr', position: 1 }], [])
    render(
      <ServicesGrid
        tiles={tiles}
        viewMode="grid"
        openInNewTab
        enableCardResizing
        cardScale="medium"
      />
    )
    await settle()
    expect(api.getServiceMetrics).not.toHaveBeenCalled()
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
  })

  it('shows uptime, sparkline, last check and last error after a short hover', async () => {
    render(card)
    const link = screen.getByRole('link', { name: /Plex/ })

    // Passing over the tile is not enough
    fireEvent.mouseEnter(link)
    await act(() => vi.advanceTimersByTimeAsync(100))
    fireEvent.mouseLeave(link)
    await act(() => vi.advanceTimersByTimeAsync(1000))
    expect(api.getServiceMetrics).not.toHaveBeenCalled()

    await hover(link)
    await settle()
    expect(api.getServiceMetrics).toHaveBeenCalledWith('s1', '24h', 60)
    const tooltip = screen.getByRole('tooltip')
    expect(link).toHaveAttribute('aria-describedby', tooltip.id)
    expect(tooltip).toHaveTextContent('https://plex.test')
    expect(tooltip).toHaveTextContent('99.5%')
    expect(tooltip).toHaveTextContent('42ms')
    expect(tooltip).toHaveTextContent('Last check2m ago')
    expect(tooltip).toHaveTextContent('Last error · 3h ago')
    expect(tooltip).toHaveTextContent('connection refused')
    expect(screen.getByTestId('sparkline')).toHaveTextContent('1 points')

    fireEvent.mouseLeave(link)
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
    expect(link).not.toHaveAttribute('aria-describedby')
  })

  it('caches the details for 60 seconds', async () => {
    render(card)
    const link = screen.getByRole('link', { name: /Plex/ })
    await hover(link)
    fireEvent.mouseLeave(link)

    await act(() => vi.advanceTimersByTimeAsync(50_000))
    await hover(link)
    expect(api.getServiceMetrics).toHaveBeenCalledTimes(1)
    fireEvent.mouseLeave(link)

    await act(() => vi.advanceTimersByTimeAsync(10_000))
    await hover(link)
    expect(api.getServiceMetrics).toHaveBeenCalledTimes(2)
  })

  it('does not keep a failed request, and says so', async () => {
    vi.mocked(api.getServiceMetrics).mockResolvedValueOnce({ error: { message: 'boom' } })
    render(card)
    const link = screen.getByRole('link', { name: /Plex/ })
    await hover(link)
    expect(screen.getByRole('tooltip')).toHaveTextContent('Details could not be loaded')
    fireEvent.mouseLeave(link)

    await hover(link)
    await settle()
    expect(api.getServiceMetrics).toHaveBeenCalledTimes(2)
    expect(screen.getByRole('tooltip')).toHaveTextContent('99.5%')
  })

  it('opens on keyboard focus and closes on Escape or blur', async () => {
    render(card)
    const link = screen.getByRole('link', { name: /Plex/ })

    act(() => link.focus())
    await settle()
    expect(screen.getByRole('tooltip')).toBeInTheDocument()
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()

    act(() => link.blur())
    act(() => link.focus())
    await settle()
    expect(screen.getByRole('tooltip')).toBeInTheDocument()
    act(() => link.blur())
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
  })

  it('opens on a long press without following the link', async () => {
    render(card)
    const link = screen.getByRole('link', { name: /Plex/ })

    // A tap is not a long press, and its mouse events are not a hover
    fireEvent.touchStart(link)
    await act(() => vi.advanceTimersByTimeAsync(100))
    fireEvent.touchEnd(link)
    fireEvent.mouseEnter(link)
    await act(() => vi.advanceTimersByTimeAsync(600))
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
    expect(fireEvent.click(link)).toBe(true)

    fireEvent.touchStart(link)
    await act(() => vi.advanceTimersByTimeAsync(500))
    expect(screen.getByRole('tooltip')).toBeInTheDocument()
    fireEvent.touchEnd(link)
    // fireEvent returns false when the default (navigation) was prevented
    expect(fireEvent.click(link)).toBe(false)
    expect(screen.getByRole('tooltip')).toBeInTheDocument()

    // A touch elsewhere closes it, and the next tap opens the service again
    fireEvent.pointerDown(document.body)
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
    expect(fireEvent.click(link)).toBe(true)
  })

  it('keeps the browser menu away during a touch, not on a right click', async () => {
    render(card)
    const link = screen.getByRole('link', { name: /Plex/ })
    expect(fireEvent.contextMenu(link)).toBe(true)

    // Android asks for its menu before the long press is due
    fireEvent.touchStart(link)
    await act(() => vi.advanceTimersByTimeAsync(400))
    expect(fireEvent.contextMenu(link)).toBe(false)
    await act(() => vi.advanceTimersByTimeAsync(100))
    expect(screen.getByRole('tooltip')).toHaveTextContent('online')
    fireEvent.touchEnd(link)
  })

  it('does not open after a touch the browser cancelled', async () => {
    render(card)
    const link = screen.getByRole('link', { name: /Plex/ })
    fireEvent.touchStart(link)
    fireEvent.touchCancel(link)
    await act(() => vi.advanceTimersByTimeAsync(600))
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
    expect(fireEvent.contextMenu(link)).toBe(true)
  })

  it('closes when the page scrolls', async () => {
    render(card)
    await hover(screen.getByRole('link', { name: /Plex/ }))
    expect(screen.getByRole('tooltip')).toBeInTheDocument()
    fireEvent.scroll(window)
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
  })

  it('shows only the link of a service that is not monitored', async () => {
    render(<ServiceCard service={service} openInNewTab={false} isMonitored={false} />)
    await hover(screen.getByRole('link', { name: /Plex/ }))
    expect(screen.getByRole('tooltip')).toHaveTextContent('https://plex.test')
    expect(screen.getByRole('tooltip')).toHaveTextContent('Not monitored')
    expect(api.getServiceMetrics).not.toHaveBeenCalled()
  })

  it('says so when there were no checks', async () => {
    vi.mocked(api.getServiceMetrics).mockResolvedValue({
      data: metrics({
        total_checks: 0,
        data_points: [],
        last_checked_at: undefined,
        last_error: undefined,
        last_error_at: undefined,
      }),
    })
    render(card)
    await hover(screen.getByRole('link', { name: /Plex/ }))
    const tooltip = screen.getByRole('tooltip')
    expect(tooltip).toHaveTextContent('No checks')
    expect(tooltip).toHaveTextContent('Last checkNever')
    expect(tooltip).not.toHaveTextContent('Last error')
    expect(screen.queryByTestId('sparkline')).not.toBeInTheDocument()
  })

  it('is off in edit mode', async () => {
    const { container } = render(
      <ServiceCard service={service} openInNewTab={false} isEditMode onSizeChange={vi.fn()} />
    )
    fireEvent.mouseEnter(container.firstElementChild!)
    await act(() => vi.advanceTimersByTimeAsync(600))
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
  })

  it.each([
    ['list', <ServiceListItem key="list" service={service} openInNewTab={false} />],
    ['chip', <ServiceChip key="chip" service={service} openInNewTab={false} />],
  ])('works on a %s tile', async (_, tile) => {
    render(tile)
    await hover(screen.getByRole('link', { name: /Plex/ }))
    await settle()
    expect(screen.getByRole('tooltip')).toHaveTextContent('99.5%')
    expect(screen.getByRole('tooltip')).toHaveTextContent('https://plex.test')
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
  })
})
