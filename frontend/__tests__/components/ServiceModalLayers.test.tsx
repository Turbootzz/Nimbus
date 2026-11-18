import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { beforeEach, describe, it, expect, vi } from 'vitest'
import '@testing-library/jest-dom'
import ServiceModal from '@/components/ServiceModal'
import { api } from '@/lib/api'
import type { Service } from '@/types'

// The real IconSelector and EmojiPicker: this is about the picker on top of the modal
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

describe('ServiceModal with the emoji picker open', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(api.getIntegrations).mockResolvedValue({ data: [] })
    vi.mocked(api.getIntegrationKinds).mockResolvedValue({ data: [] })
  })

  const openPicker = async (onClose = vi.fn()) => {
    render(<ServiceModal service={service} onClose={onClose} onSaved={() => {}} />)
    await waitFor(() => expect(api.getIntegrations).toHaveBeenCalled())
    // A re-render of the form must not put the modal back on top
    fireEvent.change(screen.getByLabelText(/Service Name/), { target: { value: 'Jellyfin' } })
    fireEvent.click(screen.getByText('Pick Emoji'))
    return { onClose, picker: screen.getByRole('dialog', { name: 'Choose an Emoji' }) }
  }

  it('closes only the picker on the first Esc and the modal on the second', async () => {
    const { onClose, picker } = await openPicker()
    expect(picker).toHaveFocus()

    fireEvent.keyDown(document, { key: 'Escape' })
    expect(screen.queryByRole('dialog', { name: 'Choose an Emoji' })).not.toBeInTheDocument()
    expect(onClose).not.toHaveBeenCalled()
    expect(screen.getByLabelText(/Service Name/)).toHaveValue('Jellyfin')

    fireEvent.keyDown(document, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('does not submit the form when the picker is closed with its button', async () => {
    const { onClose, picker } = await openPicker()

    fireEvent.click(within(picker).getByLabelText('Close'))
    expect(screen.queryByRole('dialog', { name: 'Choose an Emoji' })).not.toBeInTheDocument()
    expect(api.updateService).not.toHaveBeenCalled()
    expect(onClose).not.toHaveBeenCalled()
  })
})
