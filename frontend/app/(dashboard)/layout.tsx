'use client'

import { useState } from 'react'
import { usePathname } from 'next/navigation'
import Sidebar from '@/components/Sidebar'
import Header from '@/components/Header'

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const [isSidebarOpen, setIsSidebarOpen] = useState(false)
  const pathname = usePathname()

  // Determine page title based on current route
  const getPageTitle = () => {
    if (pathname === '/dashboard') return 'Dashboard'
    if (pathname.startsWith('/services')) return 'Services'
    if (pathname.startsWith('/settings/profile')) return 'Profile Settings'
    if (pathname.startsWith('/settings/theme')) return 'Theme Settings'
    if (pathname.startsWith('/settings')) return 'Settings'
    return 'Dashboard' // Fallback
  }

  return (
    <div className="bg-background min-h-screen">
      {/* Mobile sidebar backdrop */}
      {isSidebarOpen && (
        <div
          className="bg-opacity-50 fixed inset-0 z-40 bg-black lg:hidden"
          onClick={() => setIsSidebarOpen(false)}
        />
      )}

      {/* Sidebar */}
      <Sidebar isOpen={isSidebarOpen} setIsOpen={setIsSidebarOpen} />

      {/* Main content */}
      <div className="lg:pl-64">
        {/* Header */}
        <Header onMenuClick={() => setIsSidebarOpen(true)} title={getPageTitle()} />

        {/* Page content */}
        <main className="p-4 sm:p-6 lg:p-8">{children}</main>
      </div>
    </div>
  )
}
