import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, it, expect, vi } from 'vitest'
import '@testing-library/jest-dom'
import ServiceModal from '@/components/ServiceModal'
import { api } from '@/lib/api'
import type { ApiResponse, Service } from '@/types'

vi.mock('@/lib/api', () => ({
  api: {
    updateService: vi.fn(),
    getGroups: vi.fn(),
    getIntegrations: vi.fn(),
    getIntegrationKinds: vi.fn(),
  },
}))

vi.mock('@/contexts/ThemeContext', () => ({
  useTheme: () => ({ enableServiceGrouping: false }),
}))

vi.mock('@/components/IconSelector', () => ({ default: () => <div data-testid="icon-selector" /> }))

const service: Service = {
  id: 's1',
  name: 'Plex',
  url: 'https://plex.test',
  icon: '🎬',
  icon_type: 'emoji',
  status: 'online',
  position: 0,
  card_size: '2x1',
  monitoring_enabled: true,
  created_at: '2026-01-01T00:00:00Z',
}

describe('ServiceModal', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(api.getIntegrations).mockResolvedValue({ data: [] })
    vi.mocked(api.getIntegrationKinds).mockResolvedValue({ data: [] })
  })

  it('is a labelled dialog that focuses the name and closes on Esc', async () => {
    const onClose = vi.fn()
    render(<ServiceModal service={service} onClose={onClose} onSaved={() => {}} />)

    expect(screen.getByRole('dialog', { name: 'Edit Service' })).toHaveAttribute(
      'aria-modal',
      'true'
    )
    expect(screen.getByLabelText(/Service Name/)).toHaveFocus()

    fireEvent.keyDown(document, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledTimes(1)
    fireEvent.click(screen.getByText('Cancel'))
    fireEvent.click(screen.getByLabelText('Close'))
    expect(onClose).toHaveBeenCalledTimes(3)
    await waitFor(() => expect(api.getIntegrations).toHaveBeenCalled())
  })

  it('saves the service and cannot be closed while saving', async () => {
    let finish: (response: ApiResponse<Service>) => void = () => {}
    vi.mocked(api.updateService).mockReturnValue(new Promise((resolve) => (finish = resolve)))
    const onClose = vi.fn()
    const onSaved = vi.fn()
    const updated = { ...service, name: 'Jellyfin' }

    render(<ServiceModal service={service} onClose={onClose} onSaved={onSaved} />)
    fireEvent.change(screen.getByLabelText(/Service Name/), { target: { value: 'Jellyfin' } })
    fireEvent.click(screen.getByText('Save Changes'))

    expect(await screen.findByText('Saving...')).toBeDisabled()
    expect(screen.getByLabelText('Close')).toBeDisabled()
    expect(screen.getByText('Cancel')).toBeDisabled()
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(onClose).not.toHaveBeenCalled()

    await act(async () => finish({ data: updated }))
    expect(onSaved).toHaveBeenCalledWith(updated)
    expect(api.updateService).toHaveBeenCalledWith(
      's1',
      expect.objectContaining({ name: 'Jellyfin', url: 'https://plex.test' })
    )
  })
})
