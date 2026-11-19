import { useSyncExternalStore } from 'react'

// Kiosk mode (?kiosk=1 on a dashboard page) hides everything around the
// tiles, for a wall screen. It is read from the URL on every render: a page
// in kiosk mode has no links that change the query, and following the route
// re-renders the layout.

export function isKioskSearch(search: string): boolean {
  return new URLSearchParams(search).get('kiosk') === '1'
}

const subscribe = () => () => {}

// False on the server; ThemeScript and globals.css hide the chrome until
// React knows
export function useKiosk(): boolean {
  return useSyncExternalStore(
    subscribe,
    () => isKioskSearch(window.location.search),
    () => false
  )
}
