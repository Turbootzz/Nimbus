import { describe, it, expect, vi, beforeEach } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import '@testing-library/jest-dom'
import ConfigFileBanner from '@/components/ConfigFileBanner'
import DashboardHeader from '@/components/DashboardHeader'
import UserMenu from '@/components/UserMenu'
import { api } from '@/lib/api'
import type { User } from '@/types'

vi.mock('next/navigation', () => ({ useRouter: () => ({ push: vi.fn() }) }))
vi.mock('@/lib/api', () => ({
  api: {
    getCurrentUser: vi.fn(),
    logout: vi.fn(),
  },
}))

describe('config file mode', () => {
  beforeEach(() => vi.clearAllMocks())

  const me = (config_file?: User['config_file']) =>
    vi.mocked(api.getCurrentUser).mockResolvedValue({
      data: {
        id: 'u1',
        email: 'admin@example.com',
        name: 'Admin',
        role: 'admin',
        provider: 'local',
        email_verified: true,
        created_at: '2026-01-01T00:00:00Z',
        config_file,
      },
    })

  it('shows no banner when the dashboard is not managed by a file', async () => {
    me()
    const { container } = render(<ConfigFileBanner />)
    await waitFor(() => expect(api.getCurrentUser).toHaveBeenCalled())
    expect(container).toBeEmptyDOMElement()
  })

  it('tells the owner that the dashboard follows the file', async () => {
    me({ read_only: false, preferences: false })
    render(<ConfigFileBanner />)
    const banner = await screen.findByRole('status')
    expect(banner).toHaveTextContent('Managed by config file.')
    expect(banner).toHaveTextContent('can be overwritten')
  })

  it('explains a read-only dashboard', async () => {
    me({ read_only: true, preferences: true })
    render(<ConfigFileBanner />)
    const banner = await screen.findByRole('status')
    expect(banner).toHaveTextContent('This dashboard is read-only')
    expect(banner).toHaveTextContent('Theme, layout and wallpaper settings come from the file too')
  })

  it('offers no edit or add buttons on a read-only dashboard', () => {
    const props = {
      groups: [],
      selectedGroupId: null,
      onSelectGroup: vi.fn(),
      onCreateGroup: vi.fn(),
      onEditGroup: vi.fn(),
      onDeleteGroup: vi.fn(),
      isEditMode: false,
      onToggleEditMode: vi.fn(),
      enableServiceGrouping: false,
      activeId: null,
      isDraggingTab: false,
      addServiceHref: '/services/new',
      onAddWidget: vi.fn(),
    }
    const { rerender } = render(<DashboardHeader {...props} />)
    expect(screen.getByRole('button', { name: /edit/i })).toBeInTheDocument()

    rerender(<DashboardHeader {...props} readOnly />)
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
    expect(screen.queryByRole('link')).not.toBeInTheDocument()
    expect(screen.getByText(/Read-only: managed by config file/)).toBeInTheDocument()

    // While it is not known whether a config file locks it: no buttons, no note
    rerender(<DashboardHeader {...props} readOnly readOnlyNote={null} />)
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
    expect(screen.queryByText(/Read-only/)).not.toBeInTheDocument()
  })

  it('offers "Edit layout" in the canvas menu unless the dashboard is read-only', async () => {
    me({ read_only: false, preferences: false })
    const { unmount } = render(<UserMenu withNavigation />)
    fireEvent.click(await screen.findByRole('button', { name: /Admin/ }))
    expect(await screen.findByRole('button', { name: 'Edit layout' })).toBeInTheDocument()
    unmount()

    me({ read_only: true, preferences: false })
    render(<UserMenu withNavigation />)
    fireEvent.click(await screen.findByRole('button', { name: /Admin/ }))
    expect(screen.getByRole('link', { name: 'Services' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Edit layout' })).not.toBeInTheDocument()
  })

  it('offers no "Edit layout" when it could not be loaded whether a file locks the dashboard', async () => {
    vi.mocked(api.getCurrentUser).mockResolvedValue({ error: { message: 'Network error' } })
    render(<UserMenu withNavigation />)
    fireEvent.click(await screen.findByRole('button', { name: /User/ }))
    expect(screen.getByRole('link', { name: 'Services' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Edit layout' })).not.toBeInTheDocument()
  })
})
