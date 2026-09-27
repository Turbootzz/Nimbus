import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, it, expect, vi } from 'vitest'
import '@testing-library/jest-dom'
import WidgetModal from '@/components/widgets/WidgetModal'
import { api } from '@/lib/api'
import { backendStaticTypes, makeWidget } from './fixtures'

vi.mock('@/lib/api', () => ({
  api: {
    createWidget: vi.fn(),
    updateWidget: vi.fn(),
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

    render(
      <WidgetModal types={backendStaticTypes} groupId="g1" onClose={() => {}} onSaved={onSaved} />
    )
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

    render(<WidgetModal types={backendStaticTypes} onClose={() => {}} onSaved={onSaved} />)
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
      <WidgetModal
        types={backendStaticTypes}
        widget={widget}
        onClose={() => {}}
        onSaved={onSaved}
      />
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
        types={backendStaticTypes}
        widget={makeWidget({ type: 'future' })}
        onClose={() => {}}
        onSaved={() => {}}
      />
    )
    expect(screen.getByText(/can't be edited/)).toBeInTheDocument()
    expect(screen.queryByText('Clock')).not.toBeInTheDocument()
  })
})
