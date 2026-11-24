import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import DashboardExport from '@/components/backup/DashboardExport'
import DashboardImport from '@/components/backup/DashboardImport'
import { api } from '@/lib/api'
import type { ImportSummary } from '@/types'

vi.mock('@/hooks/useConfigFile', () => ({ useConfigFile: () => null }))
vi.mock('@/lib/api', () => ({
  api: {
    exportDashboard: vi.fn(),
    importDashboard: vi.fn(),
  },
}))

const summary = (overrides: Partial<ImportSummary> = {}): ImportSummary => ({
  mode: 'merge',
  dry_run: true,
  groups: { created: 1, removed: 0 },
  services: { created: 4, removed: 0 },
  widgets: { created: 2, removed: 0 },
  integrations: { created: 1, removed: 0 },
  preferences: false,
  needs_credentials: 1,
  needs_secrets: 0,
  ...overrides,
})

const replaceSummary = summary({
  mode: 'replace',
  services: { created: 4, removed: 7 },
  widgets: { created: 2, removed: 3 },
  preferences: true,
})

const documentText = '{"version":1,"services":[]}'
const documentFile = () =>
  new File([documentText], 'nimbus-dashboard.json', { type: 'application/json' })

describe('DashboardExport', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    URL.createObjectURL = vi.fn(() => 'blob:export')
    URL.revokeObjectURL = vi.fn()
  })

  it('downloads the export as a JSON file', async () => {
    const user = userEvent.setup()
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
    vi.mocked(api.exportDashboard).mockResolvedValue({ data: { version: 1, services: [] } })

    render(<DashboardExport />)
    await user.click(screen.getByRole('button', { name: /export dashboard/i }))

    await waitFor(() => expect(click).toHaveBeenCalled())
    const blob = vi.mocked(URL.createObjectURL).mock.calls[0][0] as Blob
    expect(JSON.parse(await blob.text())).toEqual({ version: 1, services: [] })
    const link = click.mock.contexts[0] as HTMLAnchorElement
    expect(link.download).toMatch(/^nimbus-dashboard-\d{4}-\d{2}-\d{2}\.json$/)
    expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:export')
    click.mockRestore()
  })

  it('tells when images were left out, and when the export failed', async () => {
    const user = userEvent.setup()
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
    vi.mocked(api.exportDashboard).mockResolvedValueOnce({
      data: { version: 1, omitted_images: 2 },
    })
    render(<DashboardExport />)

    await user.click(screen.getByRole('button', { name: /export dashboard/i }))
    expect(await screen.findByText(/2 uploaded images were missing or too large/)).toBeVisible()

    vi.mocked(api.exportDashboard).mockResolvedValueOnce({ error: { message: 'Session expired' } })
    await user.click(screen.getByRole('button', { name: /export dashboard/i }))
    expect(await screen.findByText('Session expired')).toBeVisible()
    expect(click).toHaveBeenCalledTimes(1)
    click.mockRestore()
  })
})

describe('DashboardImport', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('previews a chosen file before importing it', async () => {
    const user = userEvent.setup()
    vi.mocked(api.importDashboard).mockResolvedValue({ data: summary() })
    render(<DashboardImport />)
    expect(screen.getByText(/same file twice adds everything twice/)).toBeVisible()

    await user.upload(screen.getByLabelText('Dashboard file'), documentFile())

    // A dry run with the file's text, in the default mode
    await waitFor(() =>
      expect(api.importDashboard).toHaveBeenCalledWith(documentText, 'merge', true)
    )
    const servicesRow = (await screen.findByRole('rowheader', { name: 'services' })).closest('tr')!
    expect(within(servicesRow).getByText('4')).toBeVisible()
    expect(screen.getByText(/kept as they are/)).toBeVisible()
    expect(screen.getByText(/1 integration comes without credentials/)).toBeVisible()
    expect(screen.queryByText(/deletes your current dashboard/i)).not.toBeInTheDocument()

    vi.mocked(api.importDashboard).mockResolvedValue({ data: summary({ dry_run: false }) })
    await user.click(screen.getByRole('button', { name: 'Import' }))

    await waitFor(() => expect(api.importDashboard).toHaveBeenLastCalledWith(documentText, 'merge'))
    expect(await screen.findByText('Import finished.')).toBeVisible()
    expect(screen.getByRole('link', { name: /open the dashboard/i })).toHaveAttribute(
      'href',
      '/dashboard'
    )
  })

  it('warns about replace and asks before deleting anything', async () => {
    const user = userEvent.setup()
    vi.mocked(api.importDashboard).mockResolvedValue({ data: summary() })
    render(<DashboardImport />)
    await user.upload(screen.getByLabelText('Dashboard file'), documentFile())
    await screen.findByRole('button', { name: 'Import' })

    vi.mocked(api.importDashboard).mockResolvedValue({ data: replaceSummary })
    await user.click(screen.getByRole('radio', { name: /replace/i }))

    // Switching the mode asks for a new preview
    await waitFor(() =>
      expect(api.importDashboard).toHaveBeenLastCalledWith(documentText, 'replace', true)
    )
    expect(await screen.findByText(/replace deletes your current dashboard/i)).toBeVisible()
    const servicesRow = screen.getByRole('rowheader', { name: 'services' }).closest('tr')!
    expect(within(servicesRow).getByText('7')).toBeVisible()
    expect(screen.getByText(/taken from the file/)).toBeVisible()

    await user.click(screen.getByRole('button', { name: 'Replace my dashboard' }))
    const dialog = screen.getByRole('alertdialog')
    expect(within(dialog).getByText(/deletes 10 existing items/)).toBeVisible()
    // The confirmation itself names what is lost
    expect(within(dialog).getByText(/status history.*overwrites your preferences/)).toBeVisible()

    // Escape backs out without importing
    await user.keyboard('{Escape}')
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
    expect(api.importDashboard).toHaveBeenCalledTimes(2)

    await user.click(screen.getByRole('button', { name: 'Replace my dashboard' }))
    await user.click(
      within(screen.getByRole('alertdialog')).getByRole('button', { name: 'Replace' })
    )
    await waitFor(() =>
      expect(api.importDashboard).toHaveBeenLastCalledWith(documentText, 'replace')
    )
  })

  it('says what a replace does to the public status page', async () => {
    const user = userEvent.setup()
    vi.mocked(api.importDashboard).mockResolvedValue({ data: summary() })
    render(<DashboardImport />)
    await user.upload(screen.getByLabelText('Dashboard file'), documentFile())
    await screen.findByRole('button', { name: 'Import' })

    vi.mocked(api.importDashboard).mockResolvedValue({
      data: { ...replaceSummary, status_page: { group: 'Media', kept: false } },
    })
    await user.click(screen.getByRole('radio', { name: /replace/i }))
    expect(
      await screen.findByText(/status page goes offline: the file has no group “Media”/)
    ).toBeVisible()

    await user.click(screen.getByRole('button', { name: 'Replace my dashboard' }))
    expect(
      within(screen.getByRole('alertdialog')).getByText(/status page goes offline/)
    ).toBeVisible()
    await user.keyboard('{Escape}')

    vi.mocked(api.importDashboard).mockResolvedValue({
      data: { ...replaceSummary, status_page: { group: 'Media', kept: true } },
    })
    await user.click(screen.getByRole('radio', { name: /merge/i }))
    await user.click(screen.getByRole('radio', { name: /replace/i }))
    expect(await screen.findByText(/status page moves to the group “Media”/)).toBeVisible()
  })

  it('does not blame the file when the status page could not follow its group', async () => {
    const user = userEvent.setup()
    vi.mocked(api.importDashboard).mockResolvedValue({ data: summary() })
    render(<DashboardImport />)
    await user.upload(screen.getByLabelText('Dashboard file'), documentFile())
    await screen.findByRole('button', { name: 'Import' })

    // The file has the group, but moving the page to it failed after the replace
    vi.mocked(api.importDashboard).mockResolvedValue({
      data: { ...replaceSummary, status_page: { group: 'Media', kept: true } },
    })
    await user.click(screen.getByRole('radio', { name: /replace/i }))
    await screen.findByText(/status page moves to the group “Media”/)
    vi.mocked(api.importDashboard).mockResolvedValue({
      data: { ...replaceSummary, dry_run: false, status_page: { group: 'Media', kept: false } },
    })
    await user.click(screen.getByRole('button', { name: 'Replace my dashboard' }))
    await user.click(
      within(screen.getByRole('alertdialog')).getByRole('button', { name: 'Replace' })
    )

    const finished = (await screen.findByText('Import finished.')).parentElement!
    expect(within(finished).getByText(/status page is offline.*Choose a group again/)).toBeVisible()
    expect(within(finished).queryByText(/has no group/)).not.toBeInTheDocument()
  })

  it('shows the custom CSS a file brings before it is imported', async () => {
    const user = userEvent.setup()
    vi.mocked(api.importDashboard).mockResolvedValue({ data: summary() })
    render(<DashboardImport />)
    await user.upload(screen.getByLabelText('Dashboard file'), documentFile())
    await screen.findByRole('button', { name: 'Import' })
    expect(screen.queryByLabelText('Custom CSS in the file')).not.toBeInTheDocument()

    vi.mocked(api.importDashboard).mockResolvedValue({
      data: { ...replaceSummary, custom_css: 'body { display: none }' },
    })
    await user.click(screen.getByRole('radio', { name: /replace/i }))
    expect(await screen.findByLabelText('Custom CSS in the file')).toHaveTextContent(
      'body { display: none }'
    )
    expect(screen.getByText(/read it before you import/i)).toBeVisible()
  })

  it('previews in the mode that is selected once the file is read', async () => {
    const user = userEvent.setup()
    vi.mocked(api.importDashboard).mockImplementation(async (_text, mode) => ({
      data: mode === 'replace' ? replaceSummary : summary(),
    }))
    let finishReading: (text: string) => void = () => {}
    const file = documentFile()
    vi.spyOn(file, 'text').mockReturnValue(new Promise((resolve) => (finishReading = resolve)))
    render(<DashboardImport />)

    await user.upload(screen.getByLabelText('Dashboard file'), file)
    // Switching while the file is still being read
    await user.click(screen.getByRole('radio', { name: /replace/i }))
    finishReading(documentText)

    await waitFor(() => expect(api.importDashboard).toHaveBeenCalled())
    const selected = (screen.getByRole('radio', { checked: true }) as HTMLInputElement).value
    expect(api.importDashboard).toHaveBeenLastCalledWith(documentText, selected, true)
    const action = selected === 'replace' ? 'Replace my dashboard' : 'Import'
    expect(await screen.findByRole('button', { name: action })).toBeEnabled()
  })

  it('offers no import while the preview is of another mode', async () => {
    const user = userEvent.setup()
    // A merge preview, whatever was asked
    vi.mocked(api.importDashboard).mockResolvedValue({ data: summary() })
    render(<DashboardImport />)

    await user.click(screen.getByRole('radio', { name: /replace/i }))
    await user.upload(screen.getByLabelText('Dashboard file'), documentFile())

    await waitFor(() =>
      expect(api.importDashboard).toHaveBeenCalledWith(documentText, 'replace', true)
    )
    await waitFor(() => expect(screen.queryByText(/checking the file/i)).not.toBeInTheDocument())
    expect(screen.queryByRole('button', { name: 'Replace my dashboard' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Import' })).not.toBeInTheDocument()
  })

  it('gives the controls back when the file can not be read', async () => {
    const user = userEvent.setup()
    const file = documentFile()
    vi.spyOn(file, 'text').mockRejectedValue(new Error('gone'))
    render(<DashboardImport />)

    await user.upload(screen.getByLabelText('Dashboard file'), file)

    expect(await screen.findByText('The file could not be read')).toBeVisible()
    expect(screen.getByRole('button', { name: /choose file/i })).toBeEnabled()
    expect(screen.getByRole('radio', { name: /replace/i })).toBeEnabled()
    expect(api.importDashboard).not.toHaveBeenCalled()
  })

  it('shows which entity and field of the file is wrong', async () => {
    const user = userEvent.setup()
    vi.mocked(api.importDashboard).mockResolvedValue({
      error: {
        message: 'The dashboard document is not valid',
        details: [{ path: 'services[2].group_ref', message: 'no group has the ref "media"' }],
      },
    })
    render(<DashboardImport />)

    await user.upload(screen.getByLabelText('Dashboard file'), documentFile())

    expect(await screen.findByText('The dashboard document is not valid')).toBeVisible()
    expect(screen.getByText('services[2].group_ref')).toBeVisible()
    expect(screen.getByText(/no group has the ref "media"/)).toBeVisible()
    expect(screen.queryByRole('button', { name: 'Import' })).not.toBeInTheDocument()
  })

  it('refuses a file over the size limit without sending it', async () => {
    const user = userEvent.setup()
    render(<DashboardImport />)
    const big = new File([new Uint8Array(10 * 1024 * 1024 + 1)], 'big.json', {
      type: 'application/json',
    })

    await user.upload(screen.getByLabelText('Dashboard file'), big)

    expect(await screen.findByText(/the file is too large/i)).toBeVisible()
    expect(api.importDashboard).not.toHaveBeenCalled()
  })
})
