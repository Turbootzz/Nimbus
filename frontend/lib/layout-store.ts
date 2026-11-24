import { useSyncExternalStore } from 'react'
import type { LayoutMode } from '@/types'

// Layout mode backed by localStorage, read with useSyncExternalStore like the
// sidebar state. ThemeContext keeps it in sync with the preferences API.

const STORAGE_KEY = 'nimbus-layout'
const listeners = new Set<() => void>()
// The preferences could not be loaded, so there is nothing left to wait for
let settled = false

export function subscribeLayout(onStoreChange: () => void) {
  listeners.add(onStoreChange)
  // Follow a change made in another tab
  const onStorage = (e: StorageEvent) => {
    if (e.key === STORAGE_KEY) onStoreChange()
  }
  window.addEventListener('storage', onStorage)
  return () => {
    listeners.delete(onStoreChange)
    window.removeEventListener('storage', onStorage)
  }
}

export function getLayoutSnapshot(): LayoutMode {
  return localStorage.getItem(STORAGE_KEY) === 'canvas' ? 'canvas' : 'classic'
}

export function getLayoutServerSnapshot(): LayoutMode {
  return 'classic'
}

export function setLayoutMode(value: LayoutMode) {
  localStorage.setItem(STORAGE_KEY, value)
  listeners.forEach((l) => l())
}

// useLayoutMode is the layout the user picked; classic on the server
export function useLayoutMode(): LayoutMode {
  return useSyncExternalStore(subscribeLayout, getLayoutSnapshot, getLayoutServerSnapshot)
}

// Pending: this browser has not stored a layout yet (a new user, or a new
// browser) and the preferences are still loading. The dashboard stays hidden
// until then, so it never shows one layout first and then the other.
export function getLayoutPendingSnapshot(): boolean {
  return !settled && localStorage.getItem(STORAGE_KEY) === null
}

// Ends the wait when the preferences are in, or could not be loaded
export function settleLayout() {
  settled = true
  listeners.forEach((l) => l())
}

export function useLayoutPending(): boolean {
  return useSyncExternalStore(subscribeLayout, getLayoutPendingSnapshot, () => false)
}
