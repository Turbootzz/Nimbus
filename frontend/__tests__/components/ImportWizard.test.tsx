import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import '@testing-library/jest-dom'
import ImportWizard from '@/components/backup/ImportWizard'
import { api } from '@/lib/api'
import type { ConvertResult, Group, ImportSummary } from '@/types'

vi.mock('@/hooks/useConfigFile', () => ({ useConfigFile: () => null }))
vi.mock('@/lib/api', () => ({
  api: {
    convertConfig: vi.fn(),
    importDashboard: vi.fn(),
    getGroups: vi.fn(),
  },
}))

const groups: Group[] = [
  {
    id: 'g-home',
    name: 'Home',
    color: '#0ea5e9',
    position: 0,
    is_default: true,
    monitoring_enabled: true,
    created_at: '2026-01-01T00:00:00Z',
  },
  {
    id: 'g-media',
    name: 'Media',
    color: '#0ea5e9',
    position: 1,
    is_default: false,
    monitoring_enabled: true,
    created_at: '2026-01-01T00:00:00Z',
  },
]

const converted: ConvertResult = {
  format: 'homepage',
  document: {
    version: 1,
    groups: [
      { ref: 'group-media', name: 'Media' },
      { ref: 'group-network', name: 'Network' },
    ],
    integrations: [
      { ref: 'integration-sonarr', kind: 'sonarr', name: 'Sonarr', base_url: 'http://sonarr.lan' },
    ],
    services: [
      {
        name: 'Sonarr',
        url: 'http://sonarr.lan',
        group_ref: 'group-media',
        integration_ref: 'integration-sonarr',
        position: 0,
      },
      { name: 'Pi-hole', url: 'http://pi.hole/admin', group_ref: 'group-network', position: 1 },
    ],
    widgets: [],
  },
  warnings: ['Emby: Nimbus has no "emby" integration, it is imported as a link only'],
}

const summary = (dryRun: boolean): ImportSummary => ({
  mode: 'merge',
  dry_run: dryRun,
  groups: { created: 1, removed: 0 },
  services: { created: 2, removed: 0 },
  widgets: { created: 0, removed: 0 },
  integrations: { created: 1, removed: 0 },
  preferences: false,
  needs_credentials: 1,
  needs_secrets: 0,
})

const sentDocument = (call: number) =>
  JSON.parse(vi.mocked(api.importDashboard).mock.calls[call][0] as string)

describe('ImportWizard', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(api.getGroups).mockResolvedValue({ data: groups })
    vi.mocked(api.convertConfig).mockResolvedValue({ data: converted })
    vi.mocked(api.importDashboard).mockImplementation(async (_doc, _mode, dryRun) => ({
      data: summary(!!dryRun),
    }))
  })

  const convert = async (user: ReturnType<typeof userEvent.setup>) => {
    render(<ImportWizard />)
    expect(screen.getByRole('button', { name: 'Convert' })).toBeDisabled()
    await user.type(screen.getByLabelText('Config'), '- Media:')
    await user.click(screen.getByRole('button', { name: 'Convert' }))
    await screen.findByRole('button', { name: 'Import' })
  }

  it('converts a pasted config and shows what it holds before importing', async () => {
    const user = userEvent.setup()
    await convert(user)

    expect(api.convertConfig).toHaveBeenCalledWith('- Media:', 'auto', window.location.hostname)
    expect(screen.getByText(/2 services, 2 groups, 1 integrations/)).toBeInTheDocument()
    expect(screen.getByText(/no "emby" integration/)).toBeInTheDocument()
    expect(
      screen.getByRole('row', { name: /Sonarr.*http:\/\/sonarr\.lan.*Media.*sonarr/ })
    ).toBeInTheDocument()
    expect(screen.getByText(/comes without credentials/)).toBeInTheDocument()
    expect(
      screen.getByRole('row', { name: /Sonarr sonarr http:\/\/sonarr\.lan/ })
    ).toBeInTheDocument()

    // Checked without changing anything: a merge that matches groups
    expect(api.importDashboard).toHaveBeenCalledTimes(1)
    expect(vi.mocked(api.importDashboard).mock.calls[0].slice(1)).toEqual(['merge', true, true])
  })

  it('maps groups onto existing ones and imports the mapped document', async () => {
    const user = userEvent.setup()
    await convert(user)

    // Media exists already, so it is the suggestion and can't be made twice
    const media = screen.getByLabelText('Group for Media') as HTMLSelectElement
    expect(media.value).toBe('g-media')
    expect(within(media).queryByRole('option', { name: /New group/ })).not.toBeInTheDocument()

    await user.selectOptions(screen.getByLabelText('Group for Network'), 'g-home')
    await waitFor(() => expect(api.importDashboard).toHaveBeenCalledTimes(2))
    await user.click(await screen.findByRole('button', { name: 'Import' }))

    await screen.findByText('Import finished.')
    expect(vi.mocked(api.importDashboard).mock.calls[2].slice(1)).toEqual(['merge', false, true])
    expect(sentDocument(2).groups).toEqual([
      { ref: 'group-media', name: 'Media' },
      { ref: 'group-network', name: 'Home', default: true },
    ])
    expect(screen.getByRole('link', { name: 'Open the dashboard' })).toHaveAttribute(
      'href',
      '/dashboard'
    )
  })

  it('sends the chosen format and host', async () => {
    const user = userEvent.setup()
    render(<ImportWizard />)
    await user.selectOptions(screen.getByLabelText('Format'), 'compose')
    await user.type(screen.getByLabelText('Docker host'), 'nas.lan')
    await user.type(screen.getByLabelText('Config'), 'services:')
    await user.click(screen.getByRole('button', { name: 'Convert' }))
    await waitFor(() =>
      expect(api.convertConfig).toHaveBeenCalledWith('services:', 'compose', 'nas.lan')
    )
  })

  it('reads an uploaded file into the config', async () => {
    const user = userEvent.setup()
    render(<ImportWizard />)
    await user.upload(
      screen.getByLabelText('Config file'),
      new File(['sections: []'], 'conf.yml', { type: 'text/yaml' })
    )
    await waitFor(() => expect(screen.getByLabelText('Config')).toHaveValue('sections: []'))
  })

  it('shows why a config could not be converted', async () => {
    const user = userEvent.setup()
    vi.mocked(api.convertConfig).mockResolvedValue({
      error: { message: 'Could not convert the config: the format was not recognised' },
    })
    render(<ImportWizard />)
    await user.type(screen.getByLabelText('Config'), 'hello')
    await user.click(screen.getByRole('button', { name: 'Convert' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('the format was not recognised')
    expect(api.importDashboard).not.toHaveBeenCalled()
    expect(screen.queryByRole('button', { name: 'Import' })).not.toBeInTheDocument()
  })

  it('shows the problems the import finds, by entity', async () => {
    const user = userEvent.setup()
    vi.mocked(api.importDashboard).mockResolvedValue({
      error: {
        message: 'The dashboard document is not valid',
        details: [{ path: 'services[1]', message: 'Invalid URL format' }],
      },
    })
    render(<ImportWizard />)
    await user.type(screen.getByLabelText('Config'), '- Media:')
    await user.click(screen.getByRole('button', { name: 'Convert' }))

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('services[1]')
    expect(alert).toHaveTextContent('Invalid URL format')
    expect(screen.queryByRole('button', { name: 'Import' })).not.toBeInTheDocument()
  })
})
