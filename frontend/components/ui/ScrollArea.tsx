'use client'

import {
  forwardRef,
  useRef,
  useImperativeHandle,
  useCallback,
  type ReactNode,
  type HTMLAttributes,
  type WheelEvent,
} from 'react'

interface ScrollAreaProps extends HTMLAttributes<HTMLDivElement> {
  children: ReactNode
  orientation?: 'horizontal' | 'vertical' | 'both'
  // Scrolls by wheel, touch and focus only, without a visible bar
  hideScrollbar?: boolean
}

const ScrollArea = forwardRef<HTMLDivElement, ScrollAreaProps>(
  (
    { children, orientation = 'vertical', hideScrollbar = false, className = '', ...props },
    ref
  ) => {
    const innerRef = useRef<HTMLDivElement>(null)

    // Expose the inner ref to parent components
    useImperativeHandle(ref, () => innerRef.current as HTMLDivElement)

    const overflowClass =
      orientation === 'horizontal'
        ? 'overflow-x-auto overflow-y-hidden'
        : orientation === 'vertical'
          ? 'overflow-y-auto overflow-x-hidden'
          : 'overflow-auto'

    const scrollbarClass = hideScrollbar
      ? '[scrollbar-width:none] [&::-webkit-scrollbar]:hidden'
      : '[&::-webkit-scrollbar-thumb]:bg-text-muted [&::-webkit-scrollbar-thumb:hover]:bg-text-secondary [scrollbar-width:thin] [scrollbar-color:var(--color-text-muted)_transparent] [&::-webkit-scrollbar]:h-1.5 [&::-webkit-scrollbar]:w-1.5 [&::-webkit-scrollbar-thumb]:rounded-full [&::-webkit-scrollbar-track]:bg-transparent'

    // Convert vertical scroll to horizontal for horizontal orientation
    const handleWheel = useCallback(
      (e: WheelEvent<HTMLDivElement>) => {
        if (orientation !== 'horizontal' || !innerRef.current) return

        // Only convert if there's vertical scroll delta and no horizontal
        if (e.deltaY !== 0 && e.deltaX === 0) {
          e.preventDefault()
          innerRef.current.scrollLeft += e.deltaY
        }
      },
      [orientation]
    )

    return (
      <div
        ref={innerRef}
        onWheel={handleWheel}
        className={`${overflowClass} ${scrollbarClass} ${className}`}
        {...props}
      >
        {children}
      </div>
    )
  }
)

ScrollArea.displayName = 'ScrollArea'

export default ScrollArea
