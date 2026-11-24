'use client'

import {
  forwardRef,
  useRef,
  useImperativeHandle,
  useEffect,
  type ReactNode,
  type HTMLAttributes,
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

    // Convert vertical scroll to horizontal for horizontal orientation, while
    // the strip can move that way; at its ends the page scrolls on. React
    // listens to wheel passively, where preventDefault can't keep the page
    // from scrolling along.
    useEffect(() => {
      const el = innerRef.current
      if (orientation !== 'horizontal' || !el) return
      const handleWheel = (e: globalThis.WheelEvent) => {
        // Ctrl+wheel and a trackpad pinch zoom the browser
        if (e.ctrlKey || e.deltaY === 0 || e.deltaX !== 0) return
        const end = el.scrollWidth - el.clientWidth - 1 // 1px for rounding
        if (e.deltaY < 0 ? el.scrollLeft <= 0 : el.scrollLeft >= end) return
        e.preventDefault()
        el.scrollLeft += e.deltaY
      }
      el.addEventListener('wheel', handleWheel, { passive: false })
      return () => el.removeEventListener('wheel', handleWheel)
    }, [orientation])

    return (
      <div ref={innerRef} className={`${overflowClass} ${scrollbarClass} ${className}`} {...props}>
        {children}
      </div>
    )
  }
)

ScrollArea.displayName = 'ScrollArea'

export default ScrollArea
