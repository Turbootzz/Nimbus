import { useState } from 'react'
import { fireEvent, render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import '@testing-library/jest-dom'
import { useModalLayer } from '@/hooks/useModalLayer'

function Layer({ name, onClose }: { name: string; onClose: () => void }) {
  const [childOpen, setChildOpen] = useState(false)
  const ref = useModalLayer(onClose)
  return (
    <div ref={ref} tabIndex={-1} role="dialog" aria-label={name}>
      <button>{name} first</button>
      <button onClick={() => setChildOpen(true)}>{name} open child</button>
      <button>{name} last</button>
      {childOpen && name === 'modal' && <Layer name="picker" onClose={() => setChildOpen(false)} />}
    </div>
  )
}

function Page({ autoFocus = false }: { autoFocus?: boolean }) {
  const [open, setOpen] = useState(false)
  return (
    <>
      <button onClick={() => setOpen(true)}>Edit</button>
      {open && autoFocus && <AutoFocusLayer onClose={() => setOpen(false)} />}
      {open && !autoFocus && <Layer name="modal" onClose={() => setOpen(false)} />}
    </>
  )
}

function AutoFocusLayer({ onClose }: { onClose: () => void }) {
  const ref = useModalLayer(onClose)
  return (
    <div ref={ref} tabIndex={-1} role="dialog">
      <input aria-label="Name" autoFocus />
    </div>
  )
}

// jsdom does not move focus on click or Tab: focus by hand, then fire the key
const open = (label: string) => {
  screen.getByText(label).focus()
  fireEvent.click(screen.getByText(label))
}
const tab = (shiftKey = false) =>
  fireEvent.keyDown(document.activeElement!, { key: 'Tab', shiftKey })

describe('useModalLayer focus', () => {
  it('wraps Tab and Shift+Tab inside the layer', () => {
    render(<Page />)
    open('Edit')
    expect(screen.getByRole('dialog', { name: 'modal' })).toHaveFocus()

    tab()
    expect(screen.getByText('modal first')).toHaveFocus()
    tab(true)
    expect(screen.getByText('modal last')).toHaveFocus()
    tab()
    expect(screen.getByText('modal first')).toHaveFocus()
  })

  it('leaves Tab alone between the ends', () => {
    render(<Page />)
    open('Edit')
    screen.getByText('modal first').focus()

    expect(tab()).toBe(true) // not prevented: the browser moves the focus
  })

  it('traps in the top layer only and hands the trap back when it closes', () => {
    render(<Page />)
    open('Edit')
    open('modal open child')
    expect(screen.getByRole('dialog', { name: 'picker' })).toHaveFocus()

    // 'picker last' is also the last control of the modal around it
    screen.getByText('picker last').focus()
    tab()
    expect(screen.getByText('picker first')).toHaveFocus()
    tab(true)
    expect(screen.getByText('picker last')).toHaveFocus()

    fireEvent.keyDown(document, { key: 'Escape' })
    expect(screen.queryByRole('dialog', { name: 'picker' })).not.toBeInTheDocument()
    expect(screen.getByText('modal open child')).toHaveFocus()

    screen.getByText('modal last').focus()
    tab()
    expect(screen.getByText('modal first')).toHaveFocus()
  })

  it('returns focus to the opener on close, also past an autofocused field', () => {
    render(<Page autoFocus />)
    open('Edit')
    expect(screen.getByLabelText('Name')).toHaveFocus()

    fireEvent.keyDown(document, { key: 'Escape' })
    expect(screen.getByText('Edit')).toHaveFocus()
  })
})
