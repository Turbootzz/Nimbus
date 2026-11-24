import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import '@testing-library/jest-dom'
import StatusPageSettings from '@/components/StatusPageSettings'
import { api } from '@/lib/api'
import type { ApiResponse, Group, StatusPageSettings as Settings } from '@/types'

vi.mock('@/lib/api', () => ({
  api: {
    getStatusPageSettings: vi.fn(),
    getGroups: vi.fn(),
    updateStatusPageSettings: vi.fn(),
    uploadServiceIcon: vi.fn(),
  },
}))

const stored: Settings = {
  enabled: true,
  group_id: 'group-1',
  title: 'Home lab',
  logo: '',
  show_links: false,
}
const group: Group = {
  id: 'group-1',
  name: 'Public',
  color: '#0ea5e9',
  position: 0,
  is_default: false,
  monitoring_enabled: true,
  created_at: '2026-11-01T00:00:00Z',
}

describe('StatusPageSettings', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(api.getStatusPageSettings).mockResolvedValue({ data: stored })
    vi.mocked(api.getGroups).mockResolvedValue({ data: [group] })
  })

  it('offers no Save when the settings fail to load', async () => {
    vi.mocked(api.getStatusPageSettings).mockResolvedValue({ error: { message: 'Server down' } })

    render(<StatusPageSettings />)

    expect(await screen.findByRole('alert')).toHaveTextContent('Server down')
    expect(screen.queryByRole('button', { name: 'Save status page' })).not.toBeInTheDocument()
  })

  it('offers no Save when the groups fail to load, so the stored group is kept', async () => {
    vi.mocked(api.getGroups).mockResolvedValue({ error: { message: 'Groups unavailable' } })

    render(<StatusPageSettings />)

    expect(await screen.findByRole('alert')).toHaveTextContent('Groups unavailable')
    expect(screen.queryByRole('button', { name: 'Save status page' })).not.toBeInTheDocument()
    expect(screen.queryByText(/no longer exists/)).not.toBeInTheDocument()
  })

  it('locks the form while saving', async () => {
    let finish: (response: ApiResponse<Settings>) => void = () => {}
    vi.mocked(api.updateStatusPageSettings).mockReturnValue(
      new Promise((resolve) => {
        finish = resolve
      })
    )
    const user = userEvent.setup()

    render(<StatusPageSettings />)
    const title = await screen.findByLabelText('Title')
    expect(title).toHaveValue('Home lab')

    await user.click(screen.getByRole('button', { name: 'Save status page' }))
    const controls = [
      title,
      screen.getByLabelText('Group to show'),
      screen.getByRole('switch', { name: 'Enable the status page' }),
      screen.getByRole('switch', { name: 'Show links' }),
      screen.getByRole('button', { name: 'Upload' }),
    ]
    controls.forEach((control) => expect(control).toBeDisabled())

    finish({ data: stored })
    await waitFor(() => expect(title).toBeEnabled())
    controls.forEach((control) => expect(control).toBeEnabled())
    expect(api.updateStatusPageSettings).toHaveBeenCalledWith(stored)
  })
})
