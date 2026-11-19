import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import '@testing-library/jest-dom'
import IntegrationsPage from '@/app/(dashboard)/settings/integrations/page'
import IntegrationSelector from '@/components/integrations/IntegrationSelector'
import { api } from '@/lib/api'
import { kinds, makeIntegration } from './fixtures'

// What the config file locks; null: no config file
const configFile = vi.hoisted(() => ({ status: null as { read_only: boolean } | null }))
vi.mock('@/hooks/useConfigFile', () => ({ useConfigFile: () => configFile.status }))
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
    configFile.status = null
    vi.mocked(api.getIntegrationKinds).mockResolvedValue({ data: kinds })
  })

  it('only tests integrations on a dashboard a config file locks', async () => {
    configFile.status = { read_only: true }
    vi.mocked(api.getIntegrations).mockResolvedValue({ data: [makeIntegration()] })
    render(<IntegrationsPage />)

    expect(await screen.findByText('Test')).toBeInTheDocument()
    for (const control of ['Add Integration', 'Edit', 'Delete']) {
      expect(screen.queryByText(control)).not.toBeInTheDocument()
    }
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

  it('flags an imported integration that still needs its credentials', async () => {
    vi.mocked(api.getIntegrations).mockResolvedValue({
      data: [
        makeIntegration({ has_credentials: false }),
        makeIntegration({ id: 'i2', name: 'Open', auth_type: 'none', has_credentials: false }),
      ],
    })
    render(<IntegrationsPage />)

    expect(await screen.findByText(/needs credentials/i)).toBeInTheDocument()
    // No credentials is fine for an integration that takes none
    expect(screen.getByText('Not tested')).toBeInTheDocument()
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
