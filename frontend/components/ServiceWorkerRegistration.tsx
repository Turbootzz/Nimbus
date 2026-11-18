'use client'

import { useEffect } from 'react'
// Imported here so the install prompt is captured on every page
import '@/lib/pwa'

/**
 * Registers the service worker in production builds. In `next dev` it removes
 * a worker left behind by an earlier production run on the same origin, which
 * would otherwise keep serving cached build assets.
 */
export default function ServiceWorkerRegistration() {
  useEffect(() => {
    if (!('serviceWorker' in navigator)) return
    if (process.env.NODE_ENV !== 'production') {
      navigator.serviceWorker
        .getRegistrations()
        .then((registrations) => registrations.forEach((r) => r.unregister()))
        .catch(() => {})
      return
    }
    // The build id in the URL makes every deploy install a fresh worker
    navigator.serviceWorker
      .register(`/sw.js?v=${process.env.NEXT_PUBLIC_BUILD_ID}`, { scope: '/' })
      .catch((error) => console.warn('Service worker registration failed:', error))
  }, [])

  return null
}
