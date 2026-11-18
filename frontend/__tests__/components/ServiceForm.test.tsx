import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, it, expect, vi } from 'vitest'
import '@testing-library/jest-dom'
import ServiceForm from '@/components/ServiceForm'
import { api } from '@/lib/api'
import type { Group, Service } from '@/types'

vi.mock('@/hooks/useConfigFile', () => ({ useConfigFile: () => null }))
vi.mock('@/lib/api', () => ({
  api: {
    createService: vi.fn(),
    updateService: vi.fn(),
    uploadServiceIcon: vi.fn(),
    getGroups: vi.fn(),
    getIntegrations: vi.fn(),
    getIntegrationKinds: vi.fn(),
  },
}))

const theme = { enableServiceGrouping: true }
vi.mock('@/contexts/ThemeContext', () => ({ useTheme: () => theme }))

// The icon picker has its own tests
vi.mock('@/components/IconSelector', () => ({ default: () => <div data-testid="icon-selector" /> }))

const group = (overrides: Partial<Group>): Group =>
  ({
    id: 'g1',
    name: 'Home',
    color: '#000000',
    position: 0,
    is_default: false,
    monitoring_enabled: true,
    ...overrides,
  }) as Group

const service: Service = {
  id: 's1',
  name: 'Plex',
  url: 'https://plex.test',
  icon: '🎬',
  icon_type: 'emoji',
  description: 'Movies',
  status: 'online',
  position: 0,
  card_size: '2x1',
  group_id: 'g2',
  integration_id: 'i1',
  monitoring_enabled: true,
  created_at: '2026-01-01T00:00:00Z',
}

const fill = (label: RegExp, value: string) =>
  fireEvent.change(screen.getByLabelText(label), { target: { value } })

describe('ServiceForm', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    theme.enableServiceGrouping = true
    vi.mocked(api.getGroups).mockResolvedValue({
      data: [group({ id: 'g1', is_default: true }), group({ id: 'g2', name: 'Media' })],
    })
    vi.mocked(api.getIntegrations).mockResolvedValue({ data: [] })
    vi.mocked(api.getIntegrationKinds).mockResolvedValue({ data: [] })
  })

  it('creates a service in the preselected group', async () => {
    const created = { ...service, id: 'new' }
    vi.mocked(api.createService).mockResolvedValue({ data: created })
    const onSaved = vi.fn()

    render(<ServiceForm initialGroupId="g2" cancelHref="/services" onSaved={onSaved} />)
    await waitFor(() => expect(screen.getByLabelText('Group')).toHaveValue('g2'))
    fill(/Service Name/, ' Plex ')
    fill(/Service URL/, 'https://plex.test')
    fireEvent.click(screen.getByText('Create Service'))

    await waitFor(() => expect(onSaved).toHaveBeenCalledWith(created))
    expect(api.createService).toHaveBeenCalledWith({
      name: 'Plex',
      url: 'https://plex.test',
      icon: '🔗',
      icon_type: 'emoji',
      icon_image_path: '',
      description: '',
      monitoring_enabled: true,
      group_id: 'g2',
      integration_id: undefined,
    })
    expect(screen.getByText('Cancel').closest('a')).toHaveAttribute('href', '/services')
  })

  it('falls back to the default group and sends none without grouping', async () => {
    vi.mocked(api.createService).mockResolvedValue({ data: service })

    const { unmount } = render(<ServiceForm initialGroupId="gone" onSaved={() => {}} />)
    await waitFor(() => expect(screen.getByLabelText('Group')).toHaveValue('g1'))
    unmount()

    theme.enableServiceGrouping = false
    render(<ServiceForm onSaved={() => {}} />)
    expect(screen.queryByLabelText('Group')).not.toBeInTheDocument()
    fill(/Service Name/, 'Plex')
    fill(/Service URL/, 'https://plex.test')
    fireEvent.click(screen.getByText('Create Service'))

    await waitFor(() => expect(api.createService).toHaveBeenCalled())
    expect(vi.mocked(api.createService).mock.calls[0][0].group_id).toBeUndefined()
  })

  it('edits a service, keeping its group and sending an empty integration to unlink', async () => {
    const updated = { ...service, name: 'Plex 2' }
    vi.mocked(api.updateService).mockResolvedValue({ data: updated })
    const onSaved = vi.fn()

    render(<ServiceForm service={{ ...service, integration_id: undefined }} onSaved={onSaved} />)
    expect(screen.getByLabelText(/Service Name/)).toHaveValue('Plex')
    expect(screen.getByLabelText(/Service URL/)).toHaveValue('https://plex.test')
    expect(screen.getByLabelText('Description')).toHaveValue('Movies')
    // The stored group wins over the default group
    await waitFor(() => expect(screen.getByLabelText('Group')).toHaveValue('g2'))

    fill(/Service Name/, 'Plex 2')
    fireEvent.click(screen.getByText('Save Changes'))

    await waitFor(() => expect(onSaved).toHaveBeenCalledWith(updated))
    expect(api.updateService).toHaveBeenCalledWith('s1', {
      name: 'Plex 2',
      url: 'https://plex.test',
      icon: '🎬',
      icon_type: 'emoji',
      icon_image_path: '',
      description: 'Movies',
      monitoring_enabled: true,
      group_id: 'g2',
      integration_id: '',
    })
    expect(api.createService).not.toHaveBeenCalled()
  })

  it('rejects an invalid URL before calling the API', async () => {
    const onSaved = vi.fn()
    render(<ServiceForm onSaved={onSaved} />)
    fill(/Service Name/, 'Plex')
    fill(/Service URL/, 'not a url')
    fireEvent.submit(screen.getByText('Create Service').closest('form')!)

    expect(await screen.findByRole('alert')).toHaveTextContent('Please enter a valid URL')
    expect(api.createService).not.toHaveBeenCalled()
    expect(onSaved).not.toHaveBeenCalled()
  })

  it('shows the backend error and lets the user try again', async () => {
    vi.mocked(api.updateService).mockResolvedValue({ error: { message: 'Access denied' } })
    const onSaved = vi.fn()
    const onSavingChange = vi.fn()

    render(
      <ServiceForm
        service={service}
        variant="modal"
        onSaved={onSaved}
        onCancel={() => {}}
        onSavingChange={onSavingChange}
      />
    )
    fireEvent.click(screen.getByText('Save Changes'))

    expect(await screen.findByRole('alert')).toHaveTextContent('Access denied')
    expect(onSaved).not.toHaveBeenCalled()
    expect(screen.getByText('Save Changes')).toBeEnabled()
    expect(onSavingChange.mock.calls).toEqual([[true], [false]])
  })

  it('stops saving when the request throws', async () => {
    vi.mocked(api.updateService).mockRejectedValue(new Error('Network down'))
    vi.spyOn(console, 'error').mockImplementation(() => {})

    render(<ServiceForm service={service} onSaved={() => {}} />)
    fireEvent.click(screen.getByText('Save Changes'))

    expect(await screen.findByRole('alert')).toHaveTextContent('Network down')
    expect(screen.getByText('Save Changes')).toBeEnabled()
  })
})
