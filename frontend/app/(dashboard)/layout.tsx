'use client'

import { useState, useMemo, useEffect, useSyncExternalStore } from 'react'
import { usePathname } from 'next/navigation'
import Sidebar from '@/components/Sidebar'
import Header from '@/components/Header'
import CanvasTopBar from '@/components/layout/CanvasTopBar'
import CommandPalette from '@/components/CommandPalette'
import {
  subscribeSidebar,
  getSidebarSnapshot,
  getSidebarServerSnapshot,
  setSidebarCollapsed,
} from '@/lib/sidebar-store'
import { getLayoutSnapshot, useLayoutMode } from '@/lib/layout-store'

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

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const [isSidebarOpen, setIsSidebarOpen] = useState(false)
  const isDesktopCollapsed = useSyncExternalStore(
    subscribeSidebar,
    getSidebarSnapshot,
    getSidebarServerSnapshot
  )
  const pathname = usePathname()
  // Canvas is only for the dashboard; every other page keeps the sidebar
  const canvas = useLayoutMode() === 'canvas' && pathname === '/dashboard'

  // Clean up the pre-hydration sidebar attribute once React takes over. The
  // canvas one follows the stored choice, not this render: during hydration
  // this render is still classic, and the canvas follows right after.
  useEffect(() => {
    document.documentElement.removeAttribute('data-sidebar-collapsed')
  }, [])
  useEffect(() => {
    const root = document.documentElement
    if (getLayoutSnapshot() === 'canvas' && pathname === '/dashboard') {
      root.setAttribute('data-layout', 'canvas')
    } else {
      root.removeAttribute('data-layout')
    }
  }, [canvas, pathname])

  const pageTitle = useMemo(() => {
    const route = routeTitles.find((r) =>
      r.exact ? pathname === r.path : pathname === r.path || pathname.startsWith(r.path + '/')
    )
    return route?.title ?? 'Dashboard'
  }, [pathname])

  // One tree for both layouts: only the chrome around <main> changes, so the
  // page itself never remounts when the layout switches
  return (
    <div className="min-h-screen">
      {/* Mobile sidebar backdrop */}
      {!canvas && isSidebarOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/50 lg:hidden"
          onClick={() => setIsSidebarOpen(false)}
        />
      )}

      {/* Sidebar */}
      {!canvas && (
        <Sidebar
          isOpen={isSidebarOpen}
          setIsOpen={setIsSidebarOpen}
          isDesktopCollapsed={isDesktopCollapsed}
          setIsDesktopCollapsed={setSidebarCollapsed}
        />
      )}

      {/* Main content; its padding must match the sidebar widths (w-16, w-52) */}
      <div
        data-main-content
        className={
          canvas
            ? ''
            : `transition-all duration-300 ${isDesktopCollapsed ? 'lg:pl-16' : 'lg:pl-52'}`
        }
      >
        {canvas ? (
          <CanvasTopBar />
        ) : (
          <Header onMenuClick={() => setIsSidebarOpen(true)} title={pageTitle} />
        )}

        {/* Page content */}
        <main
          className={canvas ? 'mx-auto max-w-screen-2xl p-4 sm:p-6 lg:p-8' : 'p-4 sm:p-6 lg:p-8'}
        >
          {children}
        </main>
      </div>

      <CommandPalette />
    </div>
  )
}
