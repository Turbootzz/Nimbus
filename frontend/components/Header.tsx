'use client'

import { Bars3Icon } from '@heroicons/react/24/outline'
import UserMenu from './UserMenu'
import ThemeToggle from './ThemeToggle'

interface HeaderProps {
  onMenuClick: () => void
}

export default function Header({ onMenuClick }: HeaderProps) {
  return (
    <header
      className="sticky top-0 z-30 border-b"
      style={{
        backgroundColor: 'var(--color-card)',
        borderColor: 'var(--color-card-border)',
      }}
    >
      <div className="flex h-16 items-center justify-between px-4 sm:px-6 lg:px-8">
        {/* Mobile menu button */}
        <button
          onClick={onMenuClick}
          className="rounded-md p-2 focus:outline-none lg:hidden"
          style={{ color: 'var(--color-text-secondary)' }}
          onMouseEnter={(e) => {
            e.currentTarget.style.backgroundColor = 'var(--color-card-border)'
            e.currentTarget.style.color = 'var(--color-text-primary)'
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.backgroundColor = 'transparent'
            e.currentTarget.style.color = 'var(--color-text-secondary)'
          }}
        >
          <Bars3Icon className="h-6 w-6" />
        </button>

        {/* Page title (desktop) */}
        <div className="hidden lg:block">
          <h1 className="text-xl font-semibold" style={{ color: 'var(--color-text-primary)' }}>
            Dashboard
          </h1>
        </div>

        {/* Right section */}
        <div className="flex items-center space-x-4">
          {/* Theme toggle */}
          <ThemeToggle />

          {/* User menu */}
          <UserMenu />
        </div>
      </div>
    </header>
  )
}
