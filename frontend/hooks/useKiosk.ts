import { useSyncExternalStore } from 'react'
import { isKioskSearch } from '@/lib/kiosk'

const subscribe = () => () => {}

// Kiosk mode, read from the URL on every render: a kiosk page has no links
// that change the query, and following a route re-renders the layout. False
// on the server; ThemeScript and globals.css hide the chrome until React
// knows.
export function useKiosk(): boolean {
  return useSyncExternalStore(
    subscribe,
    () => isKioskSearch(window.location.search),
    () => false
  )
}
