import { fireEvent, render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import '@testing-library/jest-dom'
import WidgetCard from '@/components/widgets/WidgetCard'
import { backendStaticTypes, makeWidget } from './fixtures'

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
})
