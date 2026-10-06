import { useSyncExternalStore } from 'react'
import type { LayoutMode } from '@/types'

// Layout mode backed by localStorage, read with useSyncExternalStore like the
// sidebar state. ThemeContext keeps it in sync with the preferences API.

const STORAGE_KEY = 'nimbus-layout'
const listeners = new Set<() => void>()

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
