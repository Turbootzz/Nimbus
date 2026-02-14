// Sidebar collapsed state backed by localStorage.
// Used with useSyncExternalStore in the dashboard layout to avoid hydration mismatches.

const sidebarListeners = new Set<() => void>()

export function subscribeSidebar(onStoreChange: () => void) {
  sidebarListeners.add(onStoreChange)
  return () => {
    sidebarListeners.delete(onStoreChange)
  }
}

export function getSidebarSnapshot() {
  return localStorage.getItem('nimbus-sidebar-collapsed') === 'true'
}

export function getSidebarServerSnapshot() {
  return false
}

export function setSidebarCollapsed(value: boolean) {
  localStorage.setItem('nimbus-sidebar-collapsed', String(value))
  sidebarListeners.forEach((l) => l())
}

/** Clear all listeners (for tests only) */
export function clearSidebarListeners() {
  sidebarListeners.clear()
}
