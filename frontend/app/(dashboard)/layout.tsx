'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import Sidebar from '@/components/Sidebar'
import Header from '@/components/Header'

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter()
  const [isLoading, setIsLoading] = useState(true)
  const [isSidebarOpen, setIsSidebarOpen] = useState(false)

  useEffect(() => {
    // Check if user is authenticated by calling the backend
    // The httpOnly cookie will be sent automatically
    const checkAuth = async () => {
      try {
        const response = await fetch('http://localhost:8080/api/v1/auth/me', {
          method: 'GET',
          credentials: 'include', // Required to send httpOnly cookie
        })

        if (response.ok) {
          setIsLoading(false)
        } else {
          router.push('/login')
        }
      } catch (error) {
        console.error('Auth check error:', error)
        router.push('/login')
      }
    }

    checkAuth()
  }, [router])

  if (isLoading) {
    return (
      <div className="bg-background flex min-h-screen items-center justify-center">
        <div className="text-center">
          <div className="border-primary mx-auto mb-4 h-12 w-12 animate-spin rounded-full border-t-2 border-b-2"></div>
          <p className="text-text-secondary">Loading...</p>
        </div>
      </div>
    )
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
        <Header onMenuClick={() => setIsSidebarOpen(true)} />

        {/* Page content */}
        <main className="p-4 sm:p-6 lg:p-8">{children}</main>
      </div>
    </div>
  )
}
