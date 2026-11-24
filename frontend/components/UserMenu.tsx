'use client'

import { useState, useEffect, useRef, useCallback, useId } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import {
  UserCircleIcon,
  ArrowRightOnRectangleIcon,
  UserIcon,
  ChevronDownIcon,
  CheckIcon,
  PencilSquareIcon,
} from '@heroicons/react/24/outline'
import { api } from '@/lib/api'
import UserAvatar from '@/components/UserAvatar'
import { mainNavigation } from '@/lib/navigation'
import { toggleEditMode, useEditMode } from '@/lib/dashboard-store'
import InstallAppMenuItem from '@/components/InstallAppMenuItem'
import { useHoverStyle, hoverStyles } from '@/hooks/useHoverStyle'
import type { User } from '@/types'

interface UserMenuProps {
  // The canvas layout has no sidebar, so its menu links to the main pages
  // and toggles the dashboard's edit mode
  withNavigation?: boolean
}

export default function UserMenu({ withNavigation = false }: UserMenuProps) {
  const [user, setUser] = useState<User | null>(null)
  const [isOpen, setIsOpen] = useState(false)
  const [isLoading, setIsLoading] = useState(true)
  const menuRef = useRef<HTMLDivElement>(null)
  const triggerRef = useRef<HTMLButtonElement>(null)
  const menuId = useId()
  const isEditMode = useEditMode()
  const router = useRouter()
  const menuItemHover = useHoverStyle(hoverStyles.menuItem)

  useEffect(() => {
    // Fetch current user
    const fetchUser = async () => {
      const response = await api.getCurrentUser()
      if (response.data) {
        setUser(response.data)
      }
      setIsLoading(false)
    }
    fetchUser()

    // Close dropdown when clicking outside
    const handleClickOutside = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setIsOpen(false)
      }
    }

    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const handleSignOut = useCallback(() => {
    api.logout()
    router.push('/login')
  }, [router])

  // Escape closes the menu and hands the focus back to its button
  const handleKeyDown = (event: React.KeyboardEvent) => {
    if (event.key === 'Escape' && isOpen) {
      event.stopPropagation()
      setIsOpen(false)
      triggerRef.current?.focus()
    }
  }

  const handleEditLayout = () => {
    setIsOpen(false)
    triggerRef.current?.focus()
    toggleEditMode()
  }

  const handleProfile = useCallback(() => {
    setIsOpen(false)
    router.push('/settings/profile')
  }, [router])

  if (isLoading) {
    return (
      <div className="flex items-center space-x-3">
        <UserCircleIcon className="h-8 w-8" style={{ color: 'var(--color-text-secondary)' }} />
        <div className="hidden sm:block">
          <div className="text-sm font-medium" style={{ color: 'var(--color-text-secondary)' }}>
            Loading...
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="relative" ref={menuRef} onKeyDown={handleKeyDown}>
      {/* User button */}
      <button
        ref={triggerRef}
        type="button"
        aria-haspopup="true"
        aria-expanded={isOpen}
        aria-controls={isOpen ? menuId : undefined}
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center space-x-3 rounded-md p-2 transition-colors"
        style={{
          backgroundColor: isOpen ? 'var(--color-card-border)' : 'transparent',
        }}
        onMouseEnter={(e) => {
          if (!isOpen) {
            e.currentTarget.style.backgroundColor = 'var(--color-card-border)'
          }
        }}
        onMouseLeave={(e) => {
          if (!isOpen) {
            e.currentTarget.style.backgroundColor = 'transparent'
          }
        }}
      >
        <UserAvatar
          avatarUrl={user?.avatar_url}
          name={user?.name || 'User'}
          size={32}
          className="h-8 w-8"
        />
        <div className="hidden text-left sm:block">
          <div className="text-sm font-medium" style={{ color: 'var(--color-text-primary)' }}>
            {user?.name || 'User'}
          </div>
        </div>
        <ChevronDownIcon
          className="hidden h-4 w-4 sm:block"
          style={{
            color: 'var(--color-text-muted)',
            transform: isOpen ? 'rotate(180deg)' : 'rotate(0deg)',
            transition: 'transform 0.2s',
          }}
        />
      </button>

      {/* Dropdown menu */}
      {isOpen && (
        <div
          id={menuId}
          className="absolute right-0 mt-2 w-56 rounded-md border shadow-lg"
          style={{
            backgroundColor: 'var(--color-card)',
            borderColor: 'var(--color-card-border)',
          }}
        >
          <div className="py-1">
            {withNavigation && (
              <>
                {mainNavigation(user?.role === 'admin').map((item) => (
                  <Link
                    key={item.href}
                    href={item.href}
                    onClick={() => setIsOpen(false)}
                    className="flex w-full items-center px-4 py-2 text-sm transition-colors"
                    style={{ color: 'var(--color-text-primary)' }}
                    {...menuItemHover}
                  >
                    <item.icon
                      className="mr-3 h-5 w-5"
                      style={{ color: 'var(--color-text-muted)' }}
                    />
                    {item.name}
                  </Link>
                ))}
                <button
                  type="button"
                  onClick={handleEditLayout}
                  className="flex w-full items-center px-4 py-2 text-sm transition-colors"
                  style={{ color: 'var(--color-text-primary)' }}
                  {...menuItemHover}
                >
                  {isEditMode ? (
                    <CheckIcon className="mr-3 h-5 w-5" style={{ color: 'var(--color-success)' }} />
                  ) : (
                    <PencilSquareIcon
                      className="mr-3 h-5 w-5"
                      style={{ color: 'var(--color-text-muted)' }}
                    />
                  )}
                  {isEditMode ? 'Done editing' : 'Edit layout'}
                </button>
                {process.env.NEXT_PUBLIC_NIMBUS_CLOUD === 'true' && (
                  <a
                    href="https://nimbusapp.dev"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-text-muted block px-4 py-1 text-xs hover:opacity-80"
                  >
                    Hosted by Nimbus Cloud
                  </a>
                )}
                <div
                  className="my-1 h-px"
                  style={{ backgroundColor: 'var(--color-card-border)' }}
                />
              </>
            )}

            {/* Profile option */}
            <button
              onClick={handleProfile}
              className="flex w-full items-center px-4 py-2 text-sm transition-colors"
              style={{ color: 'var(--color-text-primary)' }}
              {...menuItemHover}
            >
              <UserIcon className="mr-3 h-5 w-5" style={{ color: 'var(--color-text-muted)' }} />
              Profile Settings
            </button>

            <InstallAppMenuItem onSelect={() => setIsOpen(false)} />

            {/* Divider */}
            <div className="my-1 h-px" style={{ backgroundColor: 'var(--color-card-border)' }} />

            {/* Sign out option */}
            <button
              onClick={handleSignOut}
              className="flex w-full items-center px-4 py-2 text-sm transition-colors"
              style={{ color: 'var(--color-error)' }}
              {...menuItemHover}
            >
              <ArrowRightOnRectangleIcon
                className="mr-3 h-5 w-5"
                style={{ color: 'var(--color-error)' }}
              />
              Sign Out
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
