'use client'

import { useEffect, useState } from 'react'
import { api } from '@/lib/api'
import type { User } from '@/types'

export default function ProfilePage() {
  const [user, setUser] = useState<User | null>(null)
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    const fetchUser = async () => {
      try {
        const response = await api.getCurrentUser()
        if (response.data) {
          setUser(response.data)
        }
      } catch (error) {
        console.error('Failed to fetch user:', error)
        // Consider adding error state to show user-friendly message
      } finally {
        setIsLoading(false)
      }
    }
    fetchUser()
  }, [])

  if (isLoading) {
    return (
      <div className="p-4 sm:p-6">
        <h1
          className="mb-4 text-2xl font-bold sm:text-3xl"
          style={{ color: 'var(--color-text-primary)' }}
        >
          Profile
        </h1>
        <p style={{ color: 'var(--color-text-secondary)' }}>Loading...</p>
      </div>
    )
  }

  return (
    <div className="p-4 sm:p-6">
      <h1
        className="mb-4 text-2xl font-bold sm:text-3xl"
        style={{ color: 'var(--color-text-primary)' }}
      >
        Profile
      </h1>

      <div
        className="max-w-2xl rounded-lg border p-6"
        style={{
          backgroundColor: 'var(--color-card)',
          borderColor: 'var(--color-card-border)',
        }}
      >
        <div className="space-y-4">
          <div>
            <label
              className="mb-1 block text-sm font-medium"
              style={{ color: 'var(--color-text-secondary)' }}
            >
              Name
            </label>
            <p className="text-lg" style={{ color: 'var(--color-text-primary)' }}>
              {user?.name}
            </p>
          </div>

          <div>
            <label
              className="mb-1 block text-sm font-medium"
              style={{ color: 'var(--color-text-secondary)' }}
            >
              Email
            </label>
            <p className="text-lg" style={{ color: 'var(--color-text-primary)' }}>
              {user?.email}
            </p>
          </div>

          <div>
            <label
              className="mb-1 block text-sm font-medium"
              style={{ color: 'var(--color-text-secondary)' }}
            >
              Role
            </label>
            <p className="text-lg capitalize" style={{ color: 'var(--color-text-primary)' }}>
              {user?.role}
            </p>
          </div>

          <div className="pt-4">
            <p className="text-sm" style={{ color: 'var(--color-text-muted)' }}>
              Profile editing coming soon...
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}
