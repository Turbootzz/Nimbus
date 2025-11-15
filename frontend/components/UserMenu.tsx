'use client'

import { useState, useEffect, useRef } from 'react'
import { useRouter } from 'next/navigation'
import {
  UserCircleIcon,
  ArrowRightOnRectangleIcon,
  UserIcon,
  ChevronDownIcon,
} from '@heroicons/react/24/outline'
import { api } from '@/lib/api'
import { getApiUrl } from '@/lib/utils/api-url'
import type { User } from '@/types'

export default function UserMenu() {
  const [user, setUser] = useState<User | null>(null)
  const [isOpen, setIsOpen] = useState(false)
  const [isLoading, setIsLoading] = useState(true)
  const menuRef = useRef<HTMLDivElement>(null)
  const router = useRouter()

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

  const handleSignOut = () => {
    api.logout()
    router.push('/login')
  }

  const handleProfile = () => {
    setIsOpen(false)
    router.push('/settings/profile')
  }

  const getAvatarUrl = (avatarUrl: string | undefined) => {
    if (!avatarUrl) return undefined
    // If it's a full URL (OAuth provider), return as-is
    if (avatarUrl.startsWith('http')) return avatarUrl
    // If it's a relative path (local upload), prepend API URL
    return getApiUrl() + avatarUrl
  }

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
    <div className="relative" ref={menuRef}>
      {/* User button */}
      <button
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
        {user?.avatar_url ? (
          <img
            src={getAvatarUrl(user.avatar_url)}
            alt={user.name}
            className="h-8 w-8 rounded-full object-cover"
          />
        ) : (
          <UserCircleIcon className="h-8 w-8" style={{ color: 'var(--color-text-secondary)' }} />
        )}
        <div className="hidden text-left sm:block">
          <div className="text-sm font-medium" style={{ color: 'var(--color-text-primary)' }}>
            {user?.name || 'User'}
          </div>
          <div className="text-xs" style={{ color: 'var(--color-text-muted)' }}>
            {user?.email || ''}
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
          className="absolute right-0 mt-2 w-56 rounded-md border shadow-lg"
          style={{
            backgroundColor: 'var(--color-card)',
            borderColor: 'var(--color-card-border)',
          }}
        >
          <div className="py-1">
            {/* Profile option */}
            <button
              onClick={handleProfile}
              className="flex w-full items-center px-4 py-2 text-sm transition-colors"
              style={{ color: 'var(--color-text-primary)' }}
              onMouseEnter={(e) => {
                e.currentTarget.style.backgroundColor = 'var(--color-card-border)'
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.backgroundColor = 'transparent'
              }}
            >
              <UserIcon className="mr-3 h-5 w-5" style={{ color: 'var(--color-text-muted)' }} />
              Profile Settings
            </button>

            {/* Divider */}
            <div className="my-1 h-px" style={{ backgroundColor: 'var(--color-card-border)' }} />

            {/* Sign out option */}
            <button
              onClick={handleSignOut}
              className="flex w-full items-center px-4 py-2 text-sm transition-colors"
              style={{ color: 'var(--color-error)' }}
              onMouseEnter={(e) => {
                e.currentTarget.style.backgroundColor = 'var(--color-card-border)'
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.backgroundColor = 'transparent'
              }}
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
