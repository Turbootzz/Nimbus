import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import '@testing-library/jest-dom'
import BoardManager, { openBoardManager } from '@/components/boards/BoardManager'
import BoardSwitcher from '@/components/boards/BoardSwitcher'
import DashboardView from '@/components/DashboardView'
import ServiceForm from '@/components/ServiceForm'
import WidgetModal from '@/components/widgets/WidgetModal'
import BoardPage from '@/app/(dashboard)/board/[slug]/page'
import NewServicePage from '@/app/(dashboard)/services/new/page'
import { getBoardsSnapshot, refreshBoards, resetBoards } from '@/hooks/useBoards'
import { setBoardOnScreen, useBoardReadOnly } from '@/lib/dashboard-store'
import { api } from '@/lib/api'
import type { Board, Group, Service, Widget } from '@/types'

const replace = vi.fn()
const push = vi.fn()
let params: Record<string, string> = {}
let searchParams = new URLSearchParams()
vi.mock('next/navigation', () => ({
  useRouter: () => ({ replace, push }),
  useParams: () => params,
  useSearchParams: () => searchParams,
}))
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
    getCurrentUser: vi.fn(),
    createBoard: vi.fn(),
    updateBoard: vi.fn(),
    reorderBoards: vi.fn(),
    deleteBoard: vi.fn(),
  },
}))
// Polls right away instead of waiting for the full fetch to age
vi.mock('@/lib/polling', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/lib/polling')>()),
  shouldSkipPoll: () => false,
}))
// The live data has its own tests
let stream: { boardId?: string; onResync: () => Promise<void> | void }
vi.mock('@/hooks/useDashboardStream', () => ({
  useDashboardStream: (options: typeof stream) => {
    stream = options
    return { snapshots: {} }
  },
  snapshotKey: (kind: string, id: string) => `${kind}:${id}`,
}))
vi.mock('@/components/DiscoveredTray', () => ({ default: () => <div data-testid="tray" /> }))
vi.mock('@/components/ServicesGrid', () => ({
  default: ({
    tiles,
    onEditService,
  }: {
    tiles: { id: string; kind: string; service?: unknown }[]
    onEditService?: (service: unknown) => void
  }) => (
    <ul>
      {tiles.map((tile) => (
        <li key={tile.id}>
          {tile.id}
          {onEditService && tile.service !== undefined && (
            <button onClick={() => onEditService(tile.service)}>Edit {tile.id}</button>
          )}
        </li>
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
  params = {}
  searchParams = new URLSearchParams()
  vi.mocked(api.getBoards).mockResolvedValue({ data: [home] })
  vi.mocked(api.getServices).mockResolvedValue({ data: [service] })
  vi.mocked(api.getGroups).mockResolvedValue({ data: [] })
  vi.mocked(api.getWidgets).mockResolvedValue({ data: [] })
  vi.mocked(api.getWidgetTypes).mockResolvedValue({ data: [] })
  vi.mocked(api.getBoardIntegrations).mockResolvedValue({ data: [] })
  vi.mocked(api.getIntegrations).mockResolvedValue({ data: [] })
  vi.mocked(api.getIntegrationKinds).mockResolvedValue({ data: [] })
})

afterEach(() => {
  vi.useRealTimers()
  act(() => setBoardOnScreen(null))
})

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
    expect(screen.getByRole('link', { name: 'Family · shared' })).toBeInTheDocument()

    // Choosing the default board makes /dashboard stay there
    fireEvent.click(screen.getByRole('link', { name: 'Home' }))
    expect(store['nimbus-last-board']).toBe('')
  })

  it("marks its owner's shared board too", async () => {
    await loadBoards([home, { ...lab, is_shared: true }])
    render(<BoardSwitcher variant="list" collapsed />)
    expect(screen.getByRole('link', { name: 'Lab · shared' })).toBeInTheDocument()
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

  it('edits a service with the groups of its own board', async () => {
    await loadBoards([home, lab])
    theme.enableServiceGrouping = true
    try {
      render(
        <ServiceForm
          service={{ ...service, board_id: 'b-lab' }}
          variant="modal"
          onSaved={vi.fn()}
          onCancel={vi.fn()}
        />
      )
      await waitFor(() => expect(api.getGroups).toHaveBeenCalledWith('b-lab'))
    } finally {
      theme.enableServiceGrouping = false
    }
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

  it('drops a service moved to another board from the default board', async () => {
    await loadBoards([home, lab])
    vi.mocked(api.updateService).mockResolvedValue({ data: { ...service, board_id: 'b-lab' } })
    store['nimbus-last-board'] = ''
    render(<DashboardView />)
    fireEvent.click(await screen.findByRole('button', { name: /^Edit$/ }))
    fireEvent.click(screen.getByRole('button', { name: 'Edit s1' }))
    fireEvent.change(await screen.findByLabelText('Board'), { target: { value: 'b-lab' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save Changes' }))
    await waitFor(() => expect(screen.queryByText('s1')).not.toBeInTheDocument())
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

  it('refetches a kiosk every five minutes without the spinner', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true })
    window.history.replaceState(null, '', '/board/lab?kiosk=1')
    render(<DashboardView board={lab} />)
    expect(await screen.findByText('s1')).toBeInTheDocument()
    expect(api.getServices).toHaveBeenCalledTimes(1)
    let answer!: (response: { data: Service[] }) => void
    vi.mocked(api.getServices).mockReturnValueOnce(new Promise((resolve) => (answer = resolve)))
    await act(() => vi.advanceTimersByTimeAsync(5 * 60 * 1000))
    expect(api.getServices).toHaveBeenCalledTimes(2)
    // While the refetch is out, the board stays on screen
    expect(screen.queryByText('Loading dashboard...')).not.toBeInTheDocument()
    expect(screen.getByText('s1')).toBeInTheDocument()
    await act(async () => answer({ data: [{ ...service, id: 's2' }] }))
    expect(screen.getByText('s2')).toBeInTheDocument()
    expect(screen.queryByText('s1')).not.toBeInTheDocument()
  })

  it('leaves a board that is no longer shared', async () => {
    render(<DashboardView board={family} />)
    expect(await screen.findByText('s1')).toBeInTheDocument()
    expect(stream.boardId).toBe('b-family')
    vi.mocked(api.getServices).mockResolvedValue({ error: { message: 'Board not found' } })
    vi.mocked(api.getBoards).mockClear()
    await act(() => stream.onResync())
    expect(api.getBoards).toHaveBeenCalled()
    expect(replace).toHaveBeenCalledWith('/dashboard')
  })
})

describe('the boards list', () => {
  it('loads again when the boards changed during a load', async () => {
    let answer!: (response: { data: Board[] }) => void
    vi.mocked(api.getBoards).mockReturnValueOnce(new Promise((resolve) => (answer = resolve)))
    vi.mocked(api.getBoards).mockResolvedValueOnce({ data: [home, lab] })
    const first = refreshBoards()
    // A board was created while the first load was out
    const second = refreshBoards()
    expect(refreshBoards()).toBe(second)
    answer({ data: [home] })
    await first
    await second
    expect(api.getBoards).toHaveBeenCalledTimes(2)
    expect(getBoardsSnapshot()).toEqual([home, lab])
  })

  it('tells a failed load from a missing board, and loads again on the next visit', async () => {
    vi.mocked(api.getBoards).mockResolvedValue({ error: { message: 'Server error' } })
    params = { slug: 'lab' }
    const { unmount } = render(<BoardPage />)
    expect(await screen.findByText(/Boards could not be loaded/)).toBeInTheDocument()
    expect(screen.queryByText('Board not found')).not.toBeInTheDocument()
    unmount()

    vi.mocked(api.getBoards).mockResolvedValue({ data: [home, lab] })
    render(<BoardPage />)
    expect(await screen.findByText('s1')).toBeInTheDocument()
    expect(api.getServices).toHaveBeenCalledWith('b-lab')
  })
})

describe('adding a service to a board', () => {
  it('only offers the form for a board of the user', async () => {
    await loadBoards([home, lab, family])
    searchParams = new URLSearchParams('board=b-family&from=dashboard')
    const { unmount } = render(<NewServicePage />)
    expect(await screen.findByText('Board not found')).toBeInTheDocument()
    expect(screen.queryByLabelText(/Service Name/)).not.toBeInTheDocument()
    unmount()

    searchParams = new URLSearchParams('board=b-lab&from=dashboard')
    render(<NewServicePage />)
    expect(await screen.findByLabelText(/Service Name/)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Back to Dashboard/ })).toHaveAttribute(
      'href',
      '/board/lab'
    )
  })

  it('shows the groups of the board it was given last', async () => {
    theme.enableServiceGrouping = true
    const group = (id: string, name: string, boardId: string) =>
      ({ id, name, board_id: boardId, is_default: false }) as Group
    let answerLab!: (response: { data: Group[] }) => void
    vi.mocked(api.getGroups).mockImplementation((boardId) =>
      boardId === 'b-lab'
        ? new Promise((resolve) => (answerLab = resolve))
        : Promise.resolve({ data: [group('g-home', 'Home apps', 'b-home')] })
    )
    try {
      const { rerender } = render(<ServiceForm boardId="b-lab" onSaved={() => {}} />)
      rerender(<ServiceForm boardId="b-home" onSaved={() => {}} />)
      expect(await screen.findByRole('option', { name: /Home apps/ })).toBeInTheDocument()
      await act(async () => answerLab({ data: [group('g-lab', 'Lab apps', 'b-lab')] }))
      expect(screen.queryByRole('option', { name: /Lab apps/ })).not.toBeInTheDocument()
      expect(screen.getByRole('option', { name: /Home apps/ })).toBeInTheDocument()
    } finally {
      theme.enableServiceGrouping = false
    }
  })
})

describe('BoardManager', () => {
  beforeEach(async () => {
    vi.mocked(api.getCurrentUser).mockResolvedValue({ data: { role: 'admin' } as never })
    vi.mocked(api.createBoard).mockResolvedValue({ data: lab })
    vi.mocked(api.deleteBoard).mockResolvedValue({ data: undefined })
    await loadBoards([home, lab])
  })

  const open = async () => {
    render(<BoardManager />)
    act(() => openBoardManager())
    return screen.findByRole('dialog', { name: 'Boards' })
  }

  it('creates a shared board', async () => {
    await open()
    fireEvent.change(screen.getByLabelText('New board'), { target: { value: ' Family ' } })
    fireEvent.click(await screen.findByLabelText('Share with every user, read-only'))
    vi.mocked(api.getBoards).mockClear()
    fireEvent.click(screen.getByRole('button', { name: 'Create' }))
    await waitFor(() =>
      expect(api.createBoard).toHaveBeenCalledWith({ name: 'Family', is_shared: true })
    )
    await waitFor(() => expect(api.getBoards).toHaveBeenCalled())
    expect(screen.getByLabelText('New board')).toHaveValue('')
  })

  it('deletes a board and moves its tiles, or deletes them too', async () => {
    await open()
    expect(screen.queryByRole('button', { name: 'Delete Home' })).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Delete Lab' }))
    expect(screen.getByLabelText('Board to move them to')).toHaveValue('b-home')
    fireEvent.click(screen.getByRole('button', { name: 'Delete board' }))
    await waitFor(() => expect(api.deleteBoard).toHaveBeenCalledWith('b-lab', 'b-home'))
    expect(push, 'the board was not on screen').not.toHaveBeenCalled()

    fireEvent.click(screen.getByRole('button', { name: 'Delete Lab' }))
    fireEvent.click(screen.getByLabelText('Delete them with the board'))
    fireEvent.click(screen.getByRole('button', { name: 'Delete board' }))
    await waitFor(() => expect(api.deleteBoard).toHaveBeenLastCalledWith('b-lab', undefined))
  })

  it('leaves the deleted board when it is on screen', async () => {
    act(() => setBoardOnScreen('b-lab'))
    await open()
    fireEvent.click(screen.getByRole('button', { name: 'Delete Lab' }))
    fireEvent.click(screen.getByRole('button', { name: 'Delete board' }))
    await waitFor(() => expect(push).toHaveBeenCalledWith('/dashboard'))
    expect(store['nimbus-last-board']).toBe('')
  })
})
