'use client'

import { useState, useMemo } from 'react'
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
  const pathname = usePathname()

  const pageTitle = useMemo(() => {
    const route = routeTitles.find((r) =>
      r.exact
        ? pathname === r.path
        : pathname === r.path || pathname.startsWith(r.path + '/')
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
      <Sidebar isOpen={isSidebarOpen} setIsOpen={setIsSidebarOpen} />

      {/* Main content */}
      <div className="lg:pl-64">
        {/* Header */}
        <Header onMenuClick={() => setIsSidebarOpen(true)} title={pageTitle} />

        {/* Page content */}
        <main className="p-4 sm:p-6 lg:p-8">{children}</main>
      </div>
    </div>
  )
}
