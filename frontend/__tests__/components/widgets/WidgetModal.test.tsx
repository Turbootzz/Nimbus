import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, it, expect, vi } from 'vitest'
import '@testing-library/jest-dom'
import WidgetModal from '@/components/widgets/WidgetModal'
import { api } from '@/lib/api'
import { backendTypes, makeWidget } from './fixtures'

vi.mock('@/lib/api', () => ({
  api: {
    createWidget: vi.fn(),
    updateWidget: vi.fn(),
    getIntegrations: vi.fn(),
    getIntegrationKinds: vi.fn(),
  },
}))

describe('WidgetModal', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('picks a type, then creates the widget in the given group', async () => {
    const created = makeWidget({ id: 'new', title: 'Notes' })
    vi.mocked(api.createWidget).mockResolvedValue({ data: created })
    const onSaved = vi.fn()

    render(<WidgetModal types={backendTypes} groupId="g1" onClose={() => {}} onSaved={onSaved} />)
    fireEvent.click(screen.getByText('Note'))
    fireEvent.change(screen.getByLabelText(/Title/), { target: { value: ' Notes ' } })
    fireEvent.change(screen.getByLabelText('Note'), { target: { value: '# Hi' } })
    fireEvent.click(screen.getByText('Add Widget', { selector: 'button' }))

    await waitFor(() => expect(onSaved).toHaveBeenCalledWith(created))
    expect(api.createWidget).toHaveBeenCalledWith({
      type: 'markdown',
      title: 'Notes',
      config: { content: '# Hi' },
      group_id: 'g1',
    })
  })

  it('shows the backend error and stays open', async () => {
    vi.mocked(api.createWidget).mockResolvedValue({
      error: { message: 'Invalid config: invalid URL: invalid URL format' },
    })
    const onSaved = vi.fn()

    render(<WidgetModal types={backendTypes} onClose={() => {}} onSaved={onSaved} />)
    fireEvent.click(screen.getByText('Embed'))
    fireEvent.click(screen.getByText('Add Widget', { selector: 'button' }))

    expect(await screen.findByText(/invalid URL format/)).toBeInTheDocument()
    expect(onSaved).not.toHaveBeenCalled()
    expect(api.createWidget).toHaveBeenCalledWith(
      expect.objectContaining({ type: 'iframe', group_id: undefined })
    )
  })

  it('edits an existing widget with its stored config', async () => {
    const widget = makeWidget({ id: 'w9', title: 'Old', config: { content: 'before' } })
    vi.mocked(api.updateWidget).mockResolvedValue({ data: { ...widget, title: 'New' } })
    const onSaved = vi.fn()

    render(
      <WidgetModal types={backendTypes} widget={widget} onClose={() => {}} onSaved={onSaved} />
    )
    expect(screen.getByText('Edit Note')).toBeInTheDocument()
    expect(screen.queryByLabelText('Back to widget types')).not.toBeInTheDocument()
    expect(screen.getByLabelText('Note')).toHaveValue('before')

    fireEvent.change(screen.getByLabelText(/Title/), { target: { value: 'New' } })
    fireEvent.click(screen.getByText('Save Changes'))

    await waitFor(() => expect(onSaved).toHaveBeenCalled())
    expect(api.updateWidget).toHaveBeenCalledWith('w9', {
      title: 'New',
      config: { content: 'before' },
    })
  })

  it('does not offer other types when editing an unknown one', () => {
    render(
      <WidgetModal
        types={backendTypes}
        widget={makeWidget({ type: 'future' })}
        onClose={() => {}}
        onSaved={() => {}}
      />
    )
    expect(screen.getByText(/can't be edited/)).toBeInTheDocument()
    expect(screen.queryByText('Clock')).not.toBeInTheDocument()
  })

  it('asks for an integration for widgets that read one', async () => {
    const integration = (id: string, kind: string, name: string) => ({
      id,
      kind,
      name,
      base_url: 'unix:///var/run/docker.sock',
      auth_type: 'none' as const,
      has_credentials: false,
      verify_tls: true,
      refresh_seconds: 60,
      options: {},
      last_test_at: null,
      last_test_ok: null,
      last_error: null,
      created_at: '',
      updated_at: '',
    })
    vi.mocked(api.getIntegrations).mockResolvedValue({
      data: [integration('d1', 'docker', 'Host'), integration('s1', 'sonarr', 'Sonarr')],
    })
    vi.mocked(api.getIntegrationKinds).mockResolvedValue({
      data: [{ kind: 'docker', name: 'Docker', auth_types: ['none'] }],
    })
    vi.mocked(api.createWidget).mockResolvedValue({
      data: makeWidget({ type: 'docker_containers' }),
    })
    const onSaved = vi.fn()

    render(<WidgetModal types={backendTypes} onClose={() => {}} onSaved={onSaved} />)
    fireEvent.click(screen.getByText('Docker containers'))
    fireEvent.click(screen.getByText('Add Widget', { selector: 'button' }))
    expect(await screen.findByText('Choose an integration')).toBeInTheDocument()
    expect(api.createWidget).not.toHaveBeenCalled()

    const select = screen.getByLabelText('Integration')
    await screen.findByRole('option', { name: 'Host (Docker)' })
    expect(screen.queryByRole('option', { name: /Sonarr/ })).not.toBeInTheDocument()
    fireEvent.change(select, { target: { value: 'd1' } })
    fireEvent.click(screen.getByText('Add Widget', { selector: 'button' }))

    await waitFor(() => expect(onSaved).toHaveBeenCalled())
    expect(api.createWidget).toHaveBeenCalledWith(
      expect.objectContaining({ type: 'docker_containers', integration_id: 'd1' })
    )
  })
})
