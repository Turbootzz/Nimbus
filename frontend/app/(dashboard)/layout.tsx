'use client'

import { useState, useMemo, useEffect, useSyncExternalStore, useCallback } from 'react'
import { usePathname } from 'next/navigation'
import Sidebar from '@/components/Sidebar'
import Header from '@/components/Header'

// Route title configuration (ordered by specificity - more specific routes first)
const routeTitles: { path: string; title: string; exact?: boolean }[] = [
  { path: '/dashboard', title: 'Dashboard', exact: true },
  { path: '/services', title: 'Services' },
  { path: '/metrics', title: 'Metrics' },
  { path: '/admin/users', title: 'User Management' },
  { path: '/admin', title: 'Admin' },
  { path: '/settings/profile', title: 'Profile Settings' },
  { path: '/settings/theme', title: 'Theme Settings' },
  { path: '/settings', title: 'Settings' },
]

// Sidebar collapsed state synced with localStorage via useSyncExternalStore
const sidebarListeners = new Set<() => void>()

function subscribeSidebar(onStoreChange: () => void) {
  sidebarListeners.add(onStoreChange)
  return () => {
    sidebarListeners.delete(onStoreChange)
  }
}

function getSidebarSnapshot() {
  return localStorage.getItem('nimbus-sidebar-collapsed') === 'true'
}

function getSidebarServerSnapshot() {
  return false
}

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const [isSidebarOpen, setIsSidebarOpen] = useState(false)
  const isDesktopCollapsed = useSyncExternalStore(
    subscribeSidebar,
    getSidebarSnapshot,
    getSidebarServerSnapshot
  )
  const setIsDesktopCollapsed = useCallback((value: boolean) => {
    localStorage.setItem('nimbus-sidebar-collapsed', String(value))
    sidebarListeners.forEach((l) => l())
  }, [])
  const pathname = usePathname()

  // Clean up pre-hydration data attribute once React takes over
  useEffect(() => {
    document.documentElement.removeAttribute('data-sidebar-collapsed')
  }, [])

  const pageTitle = useMemo(() => {
    const route = routeTitles.find((r) =>
      r.exact ? pathname === r.path : pathname === r.path || pathname.startsWith(r.path + '/')
    )
    return route?.title ?? 'Dashboard'
  }, [pathname])

  return (
    <div className="min-h-screen">
      {/* Mobile sidebar backdrop */}
      {isSidebarOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/50 lg:hidden"
          onClick={() => setIsSidebarOpen(false)}
        />
      )}

      {/* Sidebar */}
      <Sidebar
        isOpen={isSidebarOpen}
        setIsOpen={setIsSidebarOpen}
        isDesktopCollapsed={isDesktopCollapsed}
        setIsDesktopCollapsed={setIsDesktopCollapsed}
      />

      {/* Main content */}
      <div
        data-main-content
        className={`transition-all duration-300 ${isDesktopCollapsed ? 'lg:pl-16' : 'lg:pl-56'}`}
      >
        {/* Header */}
        <Header onMenuClick={() => setIsSidebarOpen(true)} title={pageTitle} />

        {/* Page content */}
        <main className="p-4 sm:p-6 lg:p-8">{children}</main>
      </div>
    </div>
  )
}
