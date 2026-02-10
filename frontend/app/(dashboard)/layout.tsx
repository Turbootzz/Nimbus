'use client'

import { useState, useMemo, useEffect } from 'react'
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

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const [isSidebarOpen, setIsSidebarOpen] = useState(false)
  const [isDesktopCollapsed, setIsDesktopCollapsed] = useState(() => {
    if (typeof window !== 'undefined') {
      return localStorage.getItem('nimbus-sidebar-collapsed') === 'true'
    }
    return false
  })
  const pathname = usePathname()

  useEffect(() => {
    localStorage.setItem('nimbus-sidebar-collapsed', String(isDesktopCollapsed))
  }, [isDesktopCollapsed])

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
