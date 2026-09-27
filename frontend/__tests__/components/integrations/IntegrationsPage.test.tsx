import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import '@testing-library/jest-dom'
import IntegrationsPage from '@/app/(dashboard)/settings/integrations/page'
import IntegrationSelector from '@/components/integrations/IntegrationSelector'
import { api } from '@/lib/api'
import { kinds, makeIntegration } from './fixtures'

vi.mock('@/lib/api', () => ({
  api: {
    getIntegrations: vi.fn(),
    getIntegrationKinds: vi.fn(),
    testSavedIntegration: vi.fn(),
    deleteIntegration: vi.fn(),
  },
}))

describe('IntegrationsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(api.getIntegrationKinds).mockResolvedValue({ data: kinds })
  })

  it('lists integrations with their status and tests one', async () => {
    vi.mocked(api.getIntegrations).mockResolvedValue({
      data: [
        makeIntegration(),
        makeIntegration({
          id: 'i2',
          kind: 'radarr',
          name: 'Movies',
          last_test_ok: false,
          last_error: 'timeout',
        }),
      ],
    })
    vi.mocked(api.testSavedIntegration).mockResolvedValue({ data: { ok: true, latency_ms: 4 } })
    render(<IntegrationsPage />)

    expect(await screen.findByText('Movies')).toBeInTheDocument()
    expect(screen.getByText('Radarr')).toBeInTheDocument()
    expect(screen.getByText('Failed: timeout')).toBeInTheDocument()
    expect(screen.getByText('Not tested')).toBeInTheDocument()

    fireEvent.click(screen.getAllByText('Test')[0])
    expect(await screen.findByText('Connected')).toBeInTheDocument()
    expect(api.testSavedIntegration).toHaveBeenCalledWith('i1')
  })

  it('deletes after confirming', async () => {
    vi.mocked(api.getIntegrations).mockResolvedValue({ data: [makeIntegration()] })
    vi.mocked(api.deleteIntegration).mockResolvedValue({ data: undefined })
    render(<IntegrationsPage />)

    fireEvent.click(await screen.findByText('Delete'))
    fireEvent.click(screen.getByText('Delete', { selector: '[role=alertdialog] button' }))
    await waitFor(() => expect(screen.getByText('No integrations yet')).toBeInTheDocument())
    expect(api.deleteIntegration).toHaveBeenCalledWith('i1')
    expect(screen.getByText(/Supported apps: Radarr, Sonarr, Multi/)).toBeInTheDocument()
  })
})

describe('IntegrationSelector', () => {
  it('lists integrations with their app name', () => {
    const onChange = vi.fn()
    render(
      <IntegrationSelector
        value=""
        onChange={onChange}
        integrations={[makeIntegration(), makeIntegration({ id: 'x', kind: 'gone', name: 'Old' })]}
        kinds={kinds}
      />
    )
    expect(screen.getByRole('option', { name: 'None' })).toBeInTheDocument()
    expect(screen.getByRole('option', { name: 'Sonarr (Sonarr)' })).toBeInTheDocument()
    expect(screen.getByRole('option', { name: 'Old (gone)' })).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText('Live numbers'), { target: { value: 'i1' } })
    expect(onChange).toHaveBeenCalledWith('i1')
  })
})
