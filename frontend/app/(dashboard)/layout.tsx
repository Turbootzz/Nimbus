'use client'

import { useState, useMemo, useEffect, useSyncExternalStore } from 'react'
import { usePathname } from 'next/navigation'
import { useTranslations, type Messages } from 'next-intl'
import Sidebar from '@/components/Sidebar'
import Header from '@/components/Header'
import CanvasTopBar from '@/components/layout/CanvasTopBar'
import CommandPalette from '@/components/CommandPalette'
import CustomCss from '@/components/CustomCss'
import ConfigFileBanner from '@/components/ConfigFileBanner'
import BoardManager from '@/components/boards/BoardManager'
import {
  subscribeSidebar,
  getSidebarSnapshot,
  getSidebarServerSnapshot,
  setSidebarCollapsed,
} from '@/lib/sidebar-store'
import {
  getLayoutPendingSnapshot,
  getLayoutSnapshot,
  useLayoutMode,
  useLayoutPending,
} from '@/lib/layout-store'
import { isDashboardPath } from '@/lib/boards'
import { useKiosk } from '@/hooks/useKiosk'
import { isKioskSearch } from '@/lib/kiosk'

// Route title configuration (ordered by specificity - more specific routes first)
// title: its key in layout.routes
const routeTitles: {
  path: string
  title: keyof Messages['layout']['routes']
  exact?: boolean
}[] = [
  { path: '/dashboard', title: 'dashboard', exact: true },
  { path: '/services', title: 'services' },
  { path: '/metrics', title: 'metrics' },
  { path: '/admin/users', title: 'userManagement' },
  { path: '/admin', title: 'admin' },
  { path: '/settings/profile', title: 'profileSettings' },
  { path: '/settings/theme', title: 'themeSettings' },
  { path: '/settings', title: 'settings' },
]

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const t = useTranslations('layout.routes')
  const [isSidebarOpen, setIsSidebarOpen] = useState(false)
  const isDesktopCollapsed = useSyncExternalStore(
    subscribeSidebar,
    getSidebarSnapshot,
    getSidebarServerSnapshot
  )
  const pathname = usePathname()
  // Canvas is only for a board; every other page keeps the sidebar
  const dashboard = isDashboardPath(pathname)
  const canvas = useLayoutMode() === 'canvas' && dashboard
  const layoutPending = useLayoutPending()
  // Kiosk: a board and nothing around it, for a wall screen
  const kiosk = useKiosk() && dashboard

  // Clean up the pre-hydration sidebar attribute once React takes over. The
  // layout one follows the stored choice, not this render: during hydration
  // this render is still classic, and the canvas follows right after. While
  // the choice is unknown the dashboard stays hidden (globals.css).
  useEffect(() => {
    document.documentElement.removeAttribute('data-sidebar-collapsed')
  }, [])
  useEffect(() => {
    const root = document.documentElement
    // From the URL, not this render: while hydrating it is not kiosk yet,
    // and the attribute ThemeScript set must stay
    root.toggleAttribute('data-kiosk', dashboard && isKioskSearch(window.location.search))
    if (!dashboard) {
      root.removeAttribute('data-layout')
    } else if (getLayoutSnapshot() === 'canvas') {
      root.setAttribute('data-layout', 'canvas')
    } else if (getLayoutPendingSnapshot()) {
      root.setAttribute('data-layout', 'pending')
    } else {
      root.removeAttribute('data-layout')
    }
  }, [canvas, layoutPending, dashboard, kiosk])

  const pageTitle = useMemo(() => {
    const route = routeTitles.find((r) =>
      r.exact ? pathname === r.path : pathname === r.path || pathname.startsWith(r.path + '/')
    )
    return t(route?.title ?? 'dashboard')
  }, [pathname, t])

  // One tree for both layouts: only the chrome around <main> changes, so the
  // page itself never remounts when the layout switches
  return (
    <div className="min-h-screen">
      {/* Mobile sidebar backdrop */}
      {!canvas && !kiosk && isSidebarOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/50 lg:hidden"
          onClick={() => setIsSidebarOpen(false)}
        />
      )}

      {/* Sidebar */}
      {!canvas && !kiosk && (
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
          canvas || kiosk
            ? ''
            : `transition-all duration-300 ${isDesktopCollapsed ? 'lg:pl-16' : 'lg:pl-52'}`
        }
      >
        {kiosk ? null : canvas ? (
          <CanvasTopBar />
        ) : (
          <Header onMenuClick={() => setIsSidebarOpen(true)} title={pageTitle} />
        )}

        {/* Page content */}
        <main
          className={canvas ? 'mx-auto max-w-screen-2xl p-4 sm:p-6 lg:p-8' : 'p-4 sm:p-6 lg:p-8'}
        >
          {!kiosk && <ConfigFileBanner />}
          {children}
        </main>
      </div>

      {!kiosk && <CommandPalette />}
      {!kiosk && <BoardManager />}
      <CustomCss />
    </div>
  )
}
