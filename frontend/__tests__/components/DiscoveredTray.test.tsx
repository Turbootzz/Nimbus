import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import DiscoveredTray from '@/components/DiscoveredTray'
import { api } from '@/lib/api'
import { INTEGRATION_PREFILL_KEY } from '@/types'
import type { DiscoveredContainer, DiscoveryResponse, Service, User } from '@/types'

vi.mock('@/lib/api', () => ({
  api: {
    getCurrentUser: vi.fn(),
    getDockerDiscovery: vi.fn(),
    addDiscovered: vi.fn(),
    ignoreDiscovered: vi.fn(),
    unignoreDiscovered: vi.fn(),
  },
}))

const container = (overrides: Partial<DiscoveredContainer>): DiscoveredContainer => ({
  key: 'i-1/sonarr',
  integration_id: 'i-1',
  integration_name: 'Homelab',
  container: 'sonarr',
  image: 'lscr.io/linuxserver/sonarr:latest',
  state: 'running',
  enabled: true,
  name: 'Sonarr',
  url: 'https://sonarr.lan',
  url_guessed: false,
  icon_url: '',
  group: 'Media',
  description: '',
  ignored: false,
  ...overrides,
})

const labelled = container({
  integration: { kind: 'sonarr', name: 'Sonarr', base_url: 'https://sonarr.lan' },
})
const guessed = container({
  key: 'i-1/web',
  container: 'web',
  image: 'nginx',
  enabled: false,
  name: 'web',
  url: 'http://nimbus.lan:8080',
  url_guessed: true,
  group: '',
})
const ignoredOne = container({ key: 'i-1/db', container: 'db', name: 'db', ignored: true })

const discovery = (containers: DiscoveredContainer[]): { data: DiscoveryResponse } => ({
  data: { integrations: 1, containers, errors: [] },
})

const asRole = (role: User['role']) =>
  vi.mocked(api.getCurrentUser).mockResolvedValue({ data: { role } as User })

describe('DiscoveredTray', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    asRole('admin')
    vi.mocked(api.getDockerDiscovery).mockResolvedValue(discovery([labelled, guessed, ignoredOne]))
  })

  it('never asks for containers when the user is not an admin', async () => {
    asRole('user')
    const { container: root } = render(<DiscoveredTray />)

    await waitFor(() => expect(api.getCurrentUser).toHaveBeenCalled())
    expect(api.getDockerDiscovery).not.toHaveBeenCalled()
    expect(root).toBeEmptyDOMElement()
  })

  it('renders nothing without a Docker integration', async () => {
    vi.mocked(api.getDockerDiscovery).mockResolvedValue({
      data: { integrations: 0, containers: [], errors: [] },
    })
    const { container: root } = render(<DiscoveredTray />)

    await waitFor(() => expect(api.getDockerDiscovery).toHaveBeenCalled())
    expect(root).toBeEmptyDOMElement()
  })

  it('adds a labelled container in one click and suggests its integration', async () => {
    const service = { id: 's-1', name: 'Sonarr' } as Service
    vi.mocked(api.addDiscovered).mockResolvedValue({ data: service })
    const onAdded = vi.fn()
    render(<DiscoveredTray onAdded={onAdded} />)

    await userEvent.click(await screen.findByRole('button', { name: 'Add' }))

    expect(api.addDiscovered).toHaveBeenCalledWith({
      key: 'i-1/sonarr',
      name: 'Sonarr',
      url: 'https://sonarr.lan',
      group: 'Media',
      icon_url: '',
      description: '',
    })
    await waitFor(() => expect(onAdded).toHaveBeenCalledWith(service))
    expect(screen.queryByText('https://sonarr.lan')).not.toBeInTheDocument()
    // Handed over in sessionStorage, never in the address
    const setUp = screen.getByRole('link', { name: 'Set it up' })
    expect(setUp).toHaveAttribute('href', '/settings/integrations')
    await userEvent.click(setUp)
    expect(JSON.parse(sessionStorage.getItem(INTEGRATION_PREFILL_KEY)!)).toEqual(
      labelled.integration
    )
  })

  it('makes the admin review a guessed address before adding', async () => {
    vi.mocked(api.addDiscovered).mockResolvedValue({ data: { id: 's-2' } as Service })
    render(<DiscoveredTray />)

    expect(await screen.findByText('guess')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Review' }))
    expect(api.addDiscovered).not.toHaveBeenCalled()

    const address = screen.getByLabelText('Address')
    expect(address).toHaveValue('http://nimbus.lan:8080')
    await userEvent.clear(address)
    await userEvent.type(address, 'https://www.lan')
    await userEvent.type(screen.getByLabelText('Group'), 'Web')
    await userEvent.click(screen.getByRole('button', { name: 'Add service' }))

    expect(api.addDiscovered).toHaveBeenCalledWith(
      expect.objectContaining({ key: 'i-1/web', name: 'web', url: 'https://www.lan', group: 'Web' })
    )
  })

  it('closes the review form with Escape', async () => {
    render(<DiscoveredTray />)

    await userEvent.click(await screen.findByRole('button', { name: 'Review' }))
    await userEvent.type(screen.getByLabelText('Address'), '{Escape}')

    expect(screen.queryByLabelText('Address')).not.toBeInTheDocument()
  })

  it('ignores and un-ignores containers', async () => {
    vi.mocked(api.ignoreDiscovered).mockResolvedValue({ data: undefined })
    vi.mocked(api.unignoreDiscovered).mockResolvedValue({ data: undefined })
    render(<DiscoveredTray />)

    const ignoreButtons = await screen.findAllByRole('button', { name: 'Ignore' })
    await userEvent.click(ignoreButtons[0])
    expect(api.ignoreDiscovered).toHaveBeenCalledWith('i-1/sonarr')

    await userEvent.click(await screen.findByRole('button', { name: 'Ignored (2)' }))
    await userEvent.click(screen.getByRole('button', { name: 'Stop ignoring db' }))
    expect(api.unignoreDiscovered).toHaveBeenCalledWith('i-1/db')
    expect(await screen.findByRole('button', { name: 'Ignored (1)' })).toBeInTheDocument()
  })

  it('shows label text as text and a failing integration', async () => {
    vi.mocked(api.getDockerDiscovery).mockResolvedValue({
      data: {
        integrations: 2,
        containers: [
          container({ name: '<img src=x onerror=alert(1)>', description: '<b>bold</b>' }),
        ],
        errors: [
          {
            integration_id: 'i-2',
            integration_name: 'Portainer',
            error: 'credentials were rejected',
          },
        ],
      },
    })
    const { container: root } = render(<DiscoveredTray />)

    expect(await screen.findByText('<img src=x onerror=alert(1)>')).toBeInTheDocument()
    expect(screen.getByText('<b>bold</b>')).toBeInTheDocument()
    expect(root.querySelector('img, b')).toBeNull()
    expect(screen.getByText('Portainer: credentials were rejected')).toBeInTheDocument()
  })

  it('refreshes on demand', async () => {
    render(<DiscoveredTray />)

    await userEvent.click(await screen.findByRole('button', { name: /refresh/i }))
    await waitFor(() => expect(api.getDockerDiscovery).toHaveBeenCalledTimes(2))
  })

  it('is a link to the tray on the dashboard, counting only new containers', async () => {
    render(<DiscoveredTray variant="link" />)

    const link = await screen.findByRole('link', { name: '2 discovered containers' })
    expect(link).toHaveAttribute('href', '/services#discovered')
    expect(within(link).queryByRole('button')).toBeNull()
  })
})
