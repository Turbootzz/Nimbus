'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import { usePathname, useRouter } from 'next/navigation'
import {
  HomeIcon,
  ServerIcon,
  CogIcon,
  ArrowRightStartOnRectangleIcon,
  XMarkIcon,
  PlusIcon,
  UserGroupIcon,
} from '@heroicons/react/24/outline'
import { api } from '@/lib/api'
import type { User } from '@/types'

interface SidebarProps {
  isOpen: boolean
  setIsOpen: (open: boolean) => void
}

export default function Sidebar({ isOpen, setIsOpen }: SidebarProps) {
  const pathname = usePathname()
  const router = useRouter()
  const [currentUser, setCurrentUser] = useState<User | null>(null)

  // Load current user to check role
  useEffect(() => {
    const loadUser = async () => {
      try {
        const response = await api.getCurrentUser()
        if (response.data) {
          setCurrentUser(response.data)
        }
      } catch (error) {
        console.error('Failed to load user:', error)
      }
    }
    loadUser()
  }, [])

  const navigation = [
    { name: 'Dashboard', href: '/dashboard', icon: HomeIcon },
    { name: 'Services', href: '/services', icon: ServerIcon },
    ...(currentUser?.role === 'admin'
      ? [{ name: 'Users', href: '/admin/users', icon: UserGroupIcon }]
      : []),
    { name: 'Settings', href: '/settings', icon: CogIcon },
  ]

  const handleLogout = async () => {
    const apiUrl = process.env.NEXT_PUBLIC_API_URL
    if (!apiUrl) {
      console.error('NEXT_PUBLIC_API_URL is not configured')
      router.push('/login')
      return
    }

    try {
      // Call backend logout endpoint to clear httpOnly cookie
      await fetch(`${apiUrl}/auth/logout`, {
        method: 'POST',
        credentials: 'include', // Required to send httpOnly cookie
      })
    } catch (error) {
      console.error('Logout error:', error)
    } finally {
      router.push('/login')
    }
  }

  const isActive = (href: string) => {
    // Dashboard should only match exactly to avoid matching /dashboard-something
    if (href === '/dashboard') {
      return pathname === href
    }
    // Other routes should match if pathname starts with href (e.g., /services matches /services/123)
    return pathname.startsWith(href)
  }

  // Shared navigation rendering function
  // onNavigate callback is called when a link is clicked (used to close mobile sidebar)
  const renderNavigation = (onNavigate?: () => void) => (
    <>
      {navigation.map((item) => (
        <Link
          key={item.name}
          href={item.href}
          onClick={onNavigate}
          className={`group flex items-center rounded-md px-3 py-2 text-sm font-medium transition-colors ${
            isActive(item.href) ? 'text-white' : 'hover:bg-card-border hover:text-text-primary'
          } `}
          style={{
            backgroundColor: isActive(item.href) ? 'var(--color-primary)' : undefined,
            color: isActive(item.href) ? 'white' : 'var(--color-text-secondary)',
          }}
        >
          <item.icon
            className="group-hover:text-text-secondary mr-3 h-5 w-5 flex-shrink-0"
            style={{
              color: isActive(item.href) ? 'white' : 'var(--color-text-muted)',
            }}
            aria-hidden="true"
          />
          {item.name}
        </Link>
      ))}

      {/* Add Service Button */}
      <Link
        href="/services/new"
        onClick={onNavigate}
        className="group text-text-secondary hover:text-text-primary hover:bg-card-border flex items-center rounded-md px-3 py-2 text-sm font-medium transition-colors"
      >
        <PlusIcon
          className="text-text-muted group-hover:text-text-secondary mr-3 h-5 w-5 flex-shrink-0"
          aria-hidden="true"
        />
        Add Service
      </Link>
    </>
  )

  // Shared logout button rendering function
  const renderLogout = () => (
    <div className="border-sidebar-border flex-shrink-0 border-t p-4">
      <button
        onClick={handleLogout}
        className="group text-text-secondary hover:text-text-primary hover:bg-card-border flex w-full items-center rounded-md px-3 py-2 text-sm font-medium transition-colors"
      >
        <ArrowRightStartOnRectangleIcon
          className="text-text-muted group-hover:text-text-secondary mr-3 h-5 w-5 flex-shrink-0"
          aria-hidden="true"
        />
        Sign out
      </button>
    </div>
  )

  return (
    <>
      {/* Desktop sidebar */}
      <div className="hidden lg:fixed lg:inset-y-0 lg:flex lg:w-64 lg:flex-col">
        <div className="border-sidebar-border bg-sidebar flex flex-grow flex-col overflow-y-auto border-r">
          {/* Logo */}
          <div className="border-sidebar-border flex h-16 flex-shrink-0 items-center border-b px-6">
            <span className="text-2xl">☁️</span>
            <span className="text-text-primary ml-2 text-xl font-semibold">Nimbus</span>
          </div>

          {/* Navigation */}
          <nav className="flex-1 space-y-1 px-4 py-4">{renderNavigation()}</nav>

          {/* Logout button */}
          {renderLogout()}
        </div>
      </div>

      {/* Mobile sidebar */}
      <div
        className={`bg-sidebar border-sidebar-border fixed inset-y-0 left-0 z-50 w-64 transform border-r transition-transform lg:hidden ${isOpen ? 'translate-x-0' : '-translate-x-full'} `}
      >
        <div className="flex h-full flex-col">
          {/* Logo and close button */}
          <div className="border-sidebar-border flex h-16 items-center justify-between border-b px-6">
            <div className="flex items-center">
              <span className="text-2xl">☁️</span>
              <span className="text-text-primary ml-2 text-xl font-semibold">Nimbus</span>
            </div>
            <button
              onClick={() => setIsOpen(false)}
              className="text-text-secondary hover:text-text-primary rounded-md focus:outline-none"
              aria-label="Close sidebar"
            >
              <XMarkIcon className="h-6 w-6" aria-hidden="true" />
            </button>
          </div>

          {/* Navigation */}
          <nav className="flex-1 space-y-1 overflow-y-auto px-4 py-4">
            {renderNavigation(() => setIsOpen(false))}
          </nav>

          {/* Logout button */}
          {renderLogout()}
        </div>
      </div>
    </>
  )
}
