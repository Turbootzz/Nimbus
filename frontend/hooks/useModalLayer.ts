'use client'

import { useEffect, useRef } from 'react'

// Open layers, bottom to top. Esc goes to the top one only, so a picker
// opened from a modal closes without taking the modal with it.
const layers: object[] = []

// For a modal or a picker on top of one: Esc closes it (unless canClose is
// false, e.g. while saving) and focus moves into it when it opens. Put the
// returned ref on the dialog element, with tabIndex={-1}.
export function useModalLayer<T extends HTMLElement = HTMLDivElement>(
  onClose: () => void,
  canClose = true
) {
  const ref = useRef<T>(null)
  // Callers pass inline callbacks; the layer must keep its place in the
  // stack when they change, so it registers once and reads the latest here
  const latest = useRef({ onClose, canClose })
  useEffect(() => {
    latest.current = { onClose, canClose }
  })

  useEffect(() => {
    const layer = {}
    layers.push(layer)
    // An autofocused field inside keeps the focus
    if (ref.current && !ref.current.contains(document.activeElement)) ref.current.focus()

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key !== 'Escape' || layers[layers.length - 1] !== layer) return
      if (latest.current.canClose) latest.current.onClose()
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => {
      document.removeEventListener('keydown', handleKeyDown)
      layers.splice(layers.indexOf(layer), 1)
    }
  }, [])

  return ref
}
