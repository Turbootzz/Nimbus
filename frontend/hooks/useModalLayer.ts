'use client'

import { useEffect, useRef, useState } from 'react'

// Open layers, bottom to top. Esc and Tab go to the top one only, so a picker
// opened from a modal closes without taking the modal with it.
const layers: object[] = []

// ponytail: by selector only; filter on visibility if a dialog ever hides a control with CSS
const FOCUSABLE =
  'a[href], button:not([disabled]), input:not([disabled]):not([type="hidden"]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'

// For a modal or a picker on top of one: Esc closes it (unless canClose is
// false, e.g. while saving), focus moves into it when it opens, Tab stays
// inside it, and focus goes back to what opened it when it closes. Put the
// returned ref on the dialog element, with tabIndex={-1}.
export function useModalLayer<T extends HTMLElement = HTMLDivElement>(
  onClose: () => void,
  canClose = true
) {
  const ref = useRef<T>(null)
  // Read while rendering: an autofocused field inside takes the focus before
  // any effect runs
  const [opener] = useState(() => (typeof document === 'undefined' ? null : document.activeElement))
  // Callers pass inline callbacks; the layer must keep its place in the
  // stack when they change, so it registers once and reads the latest here
  const latest = useRef({ onClose, canClose })
  useEffect(() => {
    latest.current = { onClose, canClose }
  })

  useEffect(() => {
    const layer = {}
    const el = ref.current
    layers.push(layer)
    // An autofocused field inside keeps the focus
    if (el && !el.contains(document.activeElement)) el.focus()

    const handleKeyDown = (e: KeyboardEvent) => {
      if (layers[layers.length - 1] !== layer) return
      if (e.key === 'Escape' && latest.current.canClose) latest.current.onClose()
      if (e.key !== 'Tab' || !el) return

      // Wrap at the ends. Focus in another overlay (one that does not use
      // this hook) is left alone.
      const active = document.activeElement
      if (active !== document.body && !el.contains(active)) return
      const focusable = el.querySelectorAll<HTMLElement>(FOCUSABLE)
      const first = focusable[0]
      const last = focusable[focusable.length - 1]
      // On the dialog itself, or on a control that got disabled while focused
      const outside = !Array.from(focusable).includes(active as HTMLElement)
      if (!first) e.preventDefault()
      else if (e.shiftKey && (active === first || outside)) {
        e.preventDefault()
        last.focus()
      } else if (!e.shiftKey && (active === last || outside)) {
        e.preventDefault()
        first.focus()
      }
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => {
      document.removeEventListener('keydown', handleKeyDown)
      layers.splice(layers.indexOf(layer), 1)
      // Not on a dev StrictMode re-run, where the layer is still mounted
      if (!el?.isConnected && opener instanceof HTMLElement && opener.isConnected) opener.focus()
    }
  }, [opener])

  return ref
}
