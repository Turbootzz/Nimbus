import { useSyncExternalStore } from 'react'

// Shared with the service worker; public/sw-routing.js holds the same values
// (a test keeps them equal)
export const API_CACHE = 'nimbus-api-v1'
export const CACHED_AT_HEADER = 'X-Nimbus-Cached-At'
export const CLEAR_API_CACHE = 'CLEAR_API_CACHE'

/**
 * Drops the API responses kept for offline use. Called when the session
 * changes (login, logout, 401) so one user's dashboard never shows up for the
 * next on a shared device. Never throws.
 */
export async function clearApiCache(): Promise<void> {
  if (typeof window === 'undefined') return
  try {
    // The worker also stops in-flight responses from being stored afterwards
    navigator.serviceWorker?.controller?.postMessage({ type: CLEAR_API_CACHE })
    if ('caches' in window) await caches.delete(API_CACHE)
  } catch (error) {
    console.warn('Could not clear the offline cache:', error)
  }
}

// localStorage key: the user the offline copy was made for
const CACHE_OWNER_KEY = 'offlineCacheOwner'

/**
 * Ties the offline copy to a user: when someone else turns out to be signed in
 * (a session change no purge saw, such as a swapped cookie), it is dropped.
 * Never throws.
 */
export async function bindApiCacheTo(userId: string): Promise<void> {
  if (typeof window === 'undefined') return
  try {
    if (localStorage.getItem(CACHE_OWNER_KEY) === userId) return
    await clearApiCache()
    localStorage.setItem(CACHE_OWNER_KEY, userId)
  } catch (error) {
    console.warn('Could not bind the offline cache:', error)
  }
}

// ============================================
// Install prompt
// ============================================

// Chromium only; Safari and Firefox never fire it
interface BeforeInstallPromptEvent extends Event {
  prompt: () => Promise<void>
  userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>
}

let installEvent: BeforeInstallPromptEvent | null = null
const listeners = new Set<() => void>()

function setInstallEvent(event: BeforeInstallPromptEvent | null) {
  installEvent = event
  listeners.forEach((l) => l())
}

// The browser fires this once, usually long before the user menu opens, so it
// is captured as soon as this module loads (the root layout imports it)
if (typeof window !== 'undefined') {
  window.addEventListener('beforeinstallprompt', (event) => {
    // Keep the browser's own banner away; the user menu offers the install
    event.preventDefault()
    setInstallEvent(event as BeforeInstallPromptEvent)
  })
  window.addEventListener('appinstalled', () => setInstallEvent(null))
}

function subscribeInstall(onStoreChange: () => void) {
  listeners.add(onStoreChange)
  return () => {
    listeners.delete(onStoreChange)
  }
}

function isStandalone(): boolean {
  return window.matchMedia?.('(display-mode: standalone)').matches ?? false
}

// True when the browser offers an install and the app is not running installed
export function useCanInstall(): boolean {
  return useSyncExternalStore(
    subscribeInstall,
    () => installEvent !== null && !isStandalone(),
    () => false
  )
}

// Shows the browser's install dialog; an event can only be used once
export async function promptInstall(): Promise<void> {
  const event = installEvent
  if (!event) return
  setInstallEvent(null)
  await event.prompt()
}
