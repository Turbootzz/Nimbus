import { fireEvent, render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import '@testing-library/jest-dom'
import WidgetCard from '@/components/widgets/WidgetCard'
import { backendStaticTypes, makeSnapshot, makeWidget } from './fixtures'

const clockMeta = backendStaticTypes.find((t) => t.type === 'clock')!
const note = makeWidget({ title: 'Todo', config: { content: 'Buy milk' } })

describe('WidgetCard', () => {
  it('shows the title and the rendered widget', () => {
    render(<WidgetCard widget={note} openInNewTab={false} />)
    expect(screen.getByText('Todo')).toBeInTheDocument()
    expect(screen.getByText('Buy milk')).toBeInTheDocument()
    expect(screen.queryByLabelText('Edit widget')).not.toBeInTheDocument()
  })

  it('offers edit and delete in edit mode', () => {
    const onEdit = vi.fn()
    const onDelete = vi.fn()
    const onSizeChange = vi.fn()
    render(
      <WidgetCard
        widget={note}
        openInNewTab={false}
        isEditMode
        onEdit={onEdit}
        onDelete={onDelete}
        onSizeChange={onSizeChange}
      />
    )
    fireEvent.click(screen.getByLabelText('Edit widget'))
    fireEvent.click(screen.getByLabelText('Delete widget'))
    expect(onEdit).toHaveBeenCalledWith(note)
    expect(onDelete).toHaveBeenCalledWith(note)
    expect(onSizeChange).not.toHaveBeenCalled()
  })

  it('cycles through the sizes the type allows on click', () => {
    const onSizeChange = vi.fn()
    const clock = makeWidget({ type: 'clock', card_size: '2x2' })
    const { container } = render(
      <WidgetCard
        widget={clock}
        meta={{ ...clockMeta, allowed_sizes: ['2x1', '2x2'] }}
        openInNewTab={false}
        isEditMode
        onSizeChange={onSizeChange}
      />
    )
    fireEvent.click(container.firstChild as HTMLElement)
    expect(onSizeChange).toHaveBeenCalledWith(clock, '2x1')
  })

  it('does not resize when resizing is off or only one size fits', () => {
    const onSizeChange = vi.fn()
    const clock = makeWidget({ type: 'clock', card_size: '2x1' })
    const { container, rerender } = render(
      <WidgetCard
        widget={clock}
        meta={clockMeta}
        openInNewTab={false}
        isEditMode
        enableCardResizing={false}
        onSizeChange={onSizeChange}
      />
    )
    fireEvent.click(container.firstChild as HTMLElement)
    rerender(
      <WidgetCard
        widget={clock}
        meta={{ ...clockMeta, allowed_sizes: ['2x1'] }}
        openInNewTab={false}
        isEditMode
        onSizeChange={onSizeChange}
      />
    )
    fireEvent.click(container.firstChild as HTMLElement)
    expect(onSizeChange).not.toHaveBeenCalled()
    expect(screen.queryByText('2x1')).not.toBeInTheDocument()
  })

  it('hides edit for a type this version does not know', () => {
    render(
      <WidgetCard
        widget={makeWidget({ type: 'future' })}
        openInNewTab={false}
        isEditMode
        onEdit={vi.fn()}
        onDelete={vi.fn()}
      />
    )
    expect(screen.queryByLabelText('Edit widget')).not.toBeInTheDocument()
    expect(screen.getByLabelText('Delete widget')).toBeInTheDocument()
  })

  describe('polled widgets', () => {
    const weather = makeWidget({ type: 'weather', title: 'Weather', config: { location: 'Home' } })

    it('shows a loading block until the first data arrives', () => {
      render(<WidgetCard widget={weather} openInNewTab={false} />)
      expect(screen.getByRole('status')).toHaveTextContent('Loading')
    })

    it('shows the error when there is no data yet', () => {
      render(
        <WidgetCard
          widget={weather}
          snapshot={makeSnapshot({
            payload: null,
            error: 'weather service: unexpected status 429',
          })}
          openInNewTab={false}
        />
      )
      expect(screen.getByText(/Could not load: weather service/)).toBeInTheDocument()
    })

    it('keeps old data with a warning when an update failed', () => {
      render(
        <WidgetCard
          widget={weather}
          snapshot={makeSnapshot({ error: 'timeout', stale: true })}
          openInNewTab={false}
        />
      )
      expect(screen.getByText('17°')).toBeInTheDocument()
      expect(screen.getByLabelText('Last update failed: timeout')).toBeInTheDocument()
    })

    it('marks data loaded at startup as possibly out of date', () => {
      render(
        <WidgetCard
          widget={weather}
          snapshot={makeSnapshot({ stale: true })}
          openInNewTab={false}
        />
      )
      expect(screen.getByLabelText('This data may be out of date')).toBeInTheDocument()
    })

    it('offers refresh in edit mode', () => {
      const onRefresh = vi.fn()
      render(
        <WidgetCard
          widget={weather}
          snapshot={makeSnapshot()}
          openInNewTab={false}
          isEditMode
          onRefresh={onRefresh}
        />
      )
      fireEvent.click(screen.getByLabelText('Refresh widget'))
      expect(onRefresh).toHaveBeenCalledWith(weather)
    })

    it('has no refresh for static widgets', () => {
      render(<WidgetCard widget={note} openInNewTab={false} isEditMode onRefresh={vi.fn()} />)
      expect(screen.queryByLabelText('Refresh widget')).not.toBeInTheDocument()
    })
  })
})
