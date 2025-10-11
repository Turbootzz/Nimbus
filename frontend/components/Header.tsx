'use client'

import { useEffect, useState } from 'react'
import { Bars3Icon, SunIcon, MoonIcon, UserCircleIcon } from '@heroicons/react/24/outline'
import { api } from '@/lib/api'
import type { User } from '@/types'

interface HeaderProps {
  onMenuClick: () => void
}

export default function Header({ onMenuClick }: HeaderProps) {
  const [user, setUser] = useState<User | null>(null)
  const [isDarkMode, setIsDarkMode] = useState(false)

  useEffect(() => {
    // Fetch current user
    const fetchUser = async () => {
      const response = await api.getCurrentUser()
      if (response.data) {
        setUser(response.data)
      }
    }
    fetchUser()

    // Check current theme
    const isDark =
      localStorage.getItem('theme') === 'dark' ||
      (!localStorage.getItem('theme') && window.matchMedia('(prefers-color-scheme: dark)').matches)
    setIsDarkMode(isDark)
    if (isDark) {
      document.documentElement.classList.add('dark')
      document.documentElement.classList.remove('light')
    } else {
      document.documentElement.classList.add('light')
      document.documentElement.classList.remove('dark')
    }
  }, [])

  const toggleTheme = () => {
    const newTheme = !isDarkMode
    setIsDarkMode(newTheme)

    if (newTheme) {
      document.documentElement.classList.add('dark')
      document.documentElement.classList.remove('light')
      localStorage.setItem('theme', 'dark')
    } else {
      document.documentElement.classList.remove('dark')
      document.documentElement.classList.add('light')
      localStorage.setItem('theme', 'light')
    }
  }

  return (
    <header className="bg-card border-card-border sticky top-0 z-30 border-b">
      <div className="flex h-16 items-center justify-between px-4 sm:px-6 lg:px-8">
        {/* Mobile menu button */}
        <button
          onClick={onMenuClick}
          className="text-text-secondary hover:text-text-primary hover:bg-card-border rounded-md p-2 focus:outline-none lg:hidden"
        >
          <Bars3Icon className="h-6 w-6" />
        </button>

        {/* Page title (desktop) */}
        <div className="hidden lg:block">
          <h1 className="text-text-primary text-xl font-semibold">Dashboard</h1>
        </div>

        {/* Right section */}
        <div className="flex items-center space-x-4">
          {/* Theme toggle */}
          <button
            onClick={toggleTheme}
            className="text-text-secondary hover:text-text-primary hover:bg-card-border rounded-md p-2 transition-colors focus:outline-none"
            aria-label="Toggle theme"
          >
            {isDarkMode ? <SunIcon className="h-5 w-5" /> : <MoonIcon className="h-5 w-5" />}
          </button>

          {/* User info */}
          <div className="flex items-center space-x-3">
            <UserCircleIcon className="text-text-secondary h-8 w-8" />
            <div className="hidden sm:block">
              <div className="text-text-primary text-sm font-medium">
                {user?.name || 'Loading...'}
              </div>
              <div className="text-text-muted text-xs">{user?.email || ''}</div>
            </div>
          </div>
        </div>
      </div>
    </header>
  )
}
