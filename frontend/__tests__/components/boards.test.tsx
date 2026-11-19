import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import '@testing-library/jest-dom'
import BoardSwitcher from '@/components/boards/BoardSwitcher'
import DashboardView from '@/components/DashboardView'
import ServiceForm from '@/components/ServiceForm'
import WidgetModal from '@/components/widgets/WidgetModal'
import { refreshBoards, resetBoards } from '@/hooks/useBoards'
import { setBoardOnScreen, useBoardReadOnly } from '@/lib/dashboard-store'
import { api } from '@/lib/api'
import type { Board, Service, Widget } from '@/types'

const replace = vi.fn()
vi.mock('next/navigation', () => ({ useRouter: () => ({ replace, push: vi.fn() }) }))
vi.mock('@/hooks/useConfigFile', () => ({ useConfigFile: () => null }))
vi.mock('@/lib/api', () => ({
  api: {
    getBoards: vi.fn(),
    getServices: vi.fn(),
    getGroups: vi.fn(),
    getWidgets: vi.fn(),
    getWidgetTypes: vi.fn(),
    getBoardIntegrations: vi.fn(),
    getIntegrations: vi.fn(),
    getIntegrationKinds: vi.fn(),
    updateService: vi.fn(),
    updateWidget: vi.fn(),
  },
}))
// The live data has its own tests
vi.mock('@/hooks/useDashboardStream', () => ({
  useDashboardStream: () => ({ snapshots: {} }),
  snapshotKey: (kind: string, id: string) => `${kind}:${id}`,
}))
vi.mock('@/components/DiscoveredTray', () => ({ default: () => <div data-testid="tray" /> }))
vi.mock('@/components/ServicesGrid', () => ({
  default: ({ tiles }: { tiles: { id: string }[] }) => (
    <ul>
      {tiles.map((tile) => (
        <li key={tile.id}>{tile.id}</li>
      ))}
    </ul>
  ),
}))
vi.mock('@/components/IconSelector', () => ({ default: () => null }))
const theme = {
  openInNewTab: false,
  enableCardResizing: true,
  enableServiceGrouping: false,
  cardScale: 'medium',
  viewMode: 'grid',
  statusStrip: { enabled: false, chips: [] },
  groupDisplay: 'tabs',
  loading: false,
}
vi.mock('@/contexts/ThemeContext', () => ({ useTheme: () => theme }))

// happy-dom's localStorage has no clear()
const store: Record<string, string> = {}
Object.defineProperty(window, 'localStorage', {
  value: {
    getItem: (key: string) => store[key] ?? null,
    setItem: (key: string, value: string) => {
      store[key] = String(value)
    },
    removeItem: (key: string) => {
      delete store[key]
    },
  },
})

const board = (id: string, name: string, extra: Partial<Board> = {}): Board => ({
  id,
  name,
  slug: name.toLowerCase(),
  position: 0,
  is_default: false,
  is_shared: false,
  visibility: 'all',
  owned: true,
  can_edit: true,
  ...extra,
})
const home = board('b-home', 'Home', { is_default: true })
const lab = board('b-lab', 'Lab')
const family = board('b-family', 'Family', { owned: false, can_edit: false, is_shared: true })

const service: Service = {
  id: 's1',
  board_id: 'b-home',
  name: 'Plex',
  url: 'https://plex.test',
  icon: '🎬',
  icon_type: 'emoji',
  status: 'online',
  position: 0,
  card_size: '2x1',
  group_id: 'g1',
  monitoring_enabled: true,
  created_at: '2026-01-01T00:00:00Z',
}

async function loadBoards(boards: Board[]) {
  vi.mocked(api.getBoards).mockResolvedValue({ data: boards })
  await act(() => refreshBoards())
}

beforeEach(() => {
  vi.clearAllMocks()
  resetBoards()
  Object.keys(store).forEach((key) => delete store[key])
  window.history.replaceState(null, '', '/')
  vi.mocked(api.getBoards).mockResolvedValue({ data: [home] })
  vi.mocked(api.getServices).mockResolvedValue({ data: [service] })
  vi.mocked(api.getGroups).mockResolvedValue({ data: [] })
  vi.mocked(api.getWidgets).mockResolvedValue({ data: [] })
  vi.mocked(api.getWidgetTypes).mockResolvedValue({ data: [] })
  vi.mocked(api.getBoardIntegrations).mockResolvedValue({ data: [] })
  vi.mocked(api.getIntegrations).mockResolvedValue({ data: [] })
  vi.mocked(api.getIntegrationKinds).mockResolvedValue({ data: [] })
})

afterEach(() => act(() => setBoardOnScreen(null)))

describe('BoardSwitcher', () => {
  it('is not there with one board', async () => {
    await loadBoards([home])
    const { container } = render(<BoardSwitcher variant="menu" />)
    expect(container).toBeEmptyDOMElement()
    render(<BoardSwitcher variant="list" />)
    expect(screen.queryByRole('group', { name: 'Boards' })).not.toBeInTheDocument()
  })

  it('links every board and marks the one on screen', async () => {
    await loadBoards([home, lab, family])
    act(() => setBoardOnScreen('b-lab'))
    render(<BoardSwitcher variant="list" />)
    const links = screen.getAllByRole('link')
    expect(links.map((l) => l.getAttribute('href'))).toEqual([
      '/dashboard',
      '/board/lab',
      '/board/family',
    ])
    expect(screen.getByRole('link', { name: 'Lab' })).toHaveAttribute('aria-current', 'page')

    // Choosing the default board makes /dashboard stay there
    fireEvent.click(screen.getByRole('link', { name: 'Home' }))
    expect(store['nimbus-last-board']).toBe('')
  })

  it('opens a menu in the canvas top bar', async () => {
    await loadBoards([home, family])
    act(() => setBoardOnScreen(''))
    render(<BoardSwitcher variant="menu" />)
    fireEvent.click(screen.getByRole('button', { name: /Home/ }))
    expect(screen.getByRole('link', { name: /Family/ })).toHaveAttribute('href', '/board/family')
    expect(screen.getByRole('button', { name: 'Manage boards' })).toBeInTheDocument()
  })
})

describe('moving a tile to another board', () => {
  it('offers nothing with one board of your own, even with a shared one', async () => {
    await loadBoards([home, family])
    render(<ServiceForm service={service} variant="modal" onSaved={vi.fn()} onCancel={vi.fn()} />)
    await waitFor(() => expect(api.getIntegrations).toHaveBeenCalled())
    expect(screen.queryByLabelText('Board')).not.toBeInTheDocument()
  })

  it('moves a service out of its group', async () => {
    await loadBoards([home, lab, family])
    vi.mocked(api.updateService).mockResolvedValue({ data: { ...service, board_id: 'b-lab' } })
    const onSaved = vi.fn()
    render(<ServiceForm service={service} variant="modal" onSaved={onSaved} onCancel={vi.fn()} />)
    const select = await screen.findByLabelText('Board')
    expect(Array.from(select.querySelectorAll('option')).map((o) => o.textContent)).toEqual([
      'Home (this board)',
      'Lab',
    ])
    fireEvent.change(select, { target: { value: 'b-lab' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save Changes' }))
    await waitFor(() => expect(onSaved).toHaveBeenCalled())
    expect(vi.mocked(api.updateService).mock.calls[0][1]).toMatchObject({
      board_id: 'b-lab',
      group_id: '',
    })
  })

  it('leaves the board alone when it is not changed', async () => {
    await loadBoards([home, lab])
    vi.mocked(api.updateService).mockResolvedValue({ data: service })
    const onSaved = vi.fn()
    render(<ServiceForm service={service} variant="modal" onSaved={onSaved} onCancel={vi.fn()} />)
    await screen.findByLabelText('Board')
    fireEvent.click(screen.getByRole('button', { name: 'Save Changes' }))
    await waitFor(() => expect(onSaved).toHaveBeenCalled())
    expect(vi.mocked(api.updateService).mock.calls[0][1]).not.toHaveProperty('board_id')
  })

  it('moves a widget', async () => {
    await loadBoards([home, lab])
    const widget = {
      id: 'w1',
      board_id: 'b-lab',
      type: 'clock',
      title: 'Clock',
      group_id: null,
      config: {},
      enabled: true,
    } as unknown as Widget
    vi.mocked(api.updateWidget).mockResolvedValue({ data: { ...widget, board_id: 'b-home' } })
    const onSaved = vi.fn()
    render(<WidgetModal types={[]} widget={widget} onClose={vi.fn()} onSaved={onSaved} />)
    fireEvent.change(await screen.findByLabelText('Board'), { target: { value: 'b-home' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save Changes' }))
    await waitFor(() => expect(onSaved).toHaveBeenCalled())
    expect(vi.mocked(api.updateWidget).mock.calls[0][1]).toMatchObject({ board_id: 'b-home' })
  })
})

function ReadOnlyProbe() {
  return <p>{useBoardReadOnly() ? 'locked' : 'editable'}</p>
}

describe('DashboardView', () => {
  it('reads the default board as before boards existed', async () => {
    render(<DashboardView />)
    expect(await screen.findByText('s1')).toBeInTheDocument()
    for (const call of [api.getServices, api.getGroups, api.getWidgets]) {
      expect(vi.mocked(call).mock.calls).toEqual([[undefined]])
    }
    expect(screen.getByRole('button', { name: /Edit/ })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Add Service/ })).toHaveAttribute(
      'href',
      '/services/new?from=dashboard'
    )
    expect(screen.getByTestId('tray')).toBeInTheDocument()
    expect(api.getBoardIntegrations).not.toHaveBeenCalled()
  })

  it('shows a shared board read-only', async () => {
    render(
      <>
        <DashboardView board={family} />
        <ReadOnlyProbe />
      </>
    )
    expect(await screen.findByText('s1')).toBeInTheDocument()
    expect(api.getServices).toHaveBeenCalledWith('b-family')
    expect(api.getBoardIntegrations).toHaveBeenCalledWith('b-family')
    expect(screen.getByText('Read-only: shared board')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Edit/ })).not.toBeInTheDocument()
    expect(screen.queryByRole('link', { name: /Add Service/ })).not.toBeInTheDocument()
    expect(screen.queryByTestId('tray')).not.toBeInTheDocument()
    expect(screen.getByText('locked'), 'the user menu offers no "Edit layout"').toBeInTheDocument()
  })

  it('adds to the board it shows', async () => {
    render(<DashboardView board={lab} />)
    expect(await screen.findByText('s1')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Add Service/ })).toHaveAttribute(
      'href',
      '/services/new?from=dashboard&board=b-lab'
    )
    expect(screen.queryByTestId('tray'), 'discovery adds to the default board').toBeNull()
  })

  it('shows a kiosk without buttons or notes', async () => {
    window.history.replaceState(null, '', '/board/lab?kiosk=1')
    render(<DashboardView board={lab} />)
    expect(await screen.findByText('s1')).toBeInTheDocument()
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
    expect(screen.queryByRole('link')).not.toBeInTheDocument()
    expect(screen.queryByText(/Read-only/)).not.toBeInTheDocument()
  })

  it('sends a new user with an empty board to the shared board', async () => {
    vi.mocked(api.getBoards).mockResolvedValue({ data: [home, family] })
    vi.mocked(api.getServices).mockResolvedValue({ data: [] })
    render(<DashboardView />)
    await waitFor(() => expect(replace).toHaveBeenCalledWith('/board/family'))
  })

  it('stays when the user chose the default board, or in a kiosk', async () => {
    vi.mocked(api.getBoards).mockResolvedValue({ data: [home, family] })
    vi.mocked(api.getServices).mockResolvedValue({ data: [] })
    store['nimbus-last-board'] = ''
    const { unmount } = render(<DashboardView />)
    await screen.findByText(/Edit/)
    await act(() => refreshBoards())
    unmount()

    delete store['nimbus-last-board']
    window.history.replaceState(null, '', '/dashboard?kiosk=1')
    render(<DashboardView />)
    await waitFor(() => expect(api.getBoards).toHaveBeenCalled())
    expect(replace).not.toHaveBeenCalled()
  })
})
