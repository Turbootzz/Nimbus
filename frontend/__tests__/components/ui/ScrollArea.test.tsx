import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import '@testing-library/jest-dom'
import ScrollArea from '@/components/ui/ScrollArea'

// A strip of the given widths; jsdom does no layout
function strip(scrollWidth: number) {
  render(
    <ScrollArea orientation="horizontal" data-testid="strip">
      tabs
    </ScrollArea>
  )
  const el = screen.getByTestId('strip')
  Object.defineProperty(el, 'scrollWidth', { value: scrollWidth })
  Object.defineProperty(el, 'clientWidth', { value: 300 })
  return el
}

describe('ScrollArea', () => {
  it('scrolls a horizontal strip with the wheel, without the page', () => {
    const listen = vi.spyOn(HTMLElement.prototype, 'addEventListener')
    const el = strip(900)
    // A browser ignores preventDefault in a passive listener (jsdom does not)
    expect(listen).toHaveBeenCalledWith('wheel', expect.any(Function), { passive: false })
    listen.mockRestore()
    // fireEvent returns false when the default (scrolling the page) is prevented
    expect(fireEvent.wheel(el, { deltaY: 40 })).toBe(false)
    expect(el.scrollLeft).toBe(40)
  })

  it('leaves the wheel to the page when there is nothing to scroll', () => {
    const el = strip(300)
    expect(fireEvent.wheel(el, { deltaY: 40 })).toBe(true)
    expect(el.scrollLeft).toBe(0)
  })
})
