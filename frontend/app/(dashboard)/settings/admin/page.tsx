'use client'

import { useEffect, useState, useCallback } from 'react'
import { useRouter } from 'next/navigation'
import { api } from '@/lib/api'
import { Toggle } from '@/components/ui/Toggle'
import type { SystemSetting, User } from '@/types'

export default function AdminSettingsPage() {
  const router = useRouter()
  const [currentUser, setCurrentUser] = useState<User | null>(null)
  const [isCheckingAuth, setIsCheckingAuth] = useState(true)
  const [settings, setSettings] = useState<SystemSetting[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')
  const [actionLoading, setActionLoading] = useState<string | null>(null)

  // Check admin role before loading settings
  useEffect(() => {
    const checkAdminAccess = async () => {
      try {
        const response = await api.getCurrentUser()
        if (!response.data || response.data.role !== 'admin') {
          router.replace('/settings')
          return
        }
        setCurrentUser(response.data)
        setIsCheckingAuth(false)
      } catch {
        router.replace('/settings')
      }
    }
    checkAdminAccess()
  }, [router])

  const fetchSettings = useCallback(async () => {
    setIsLoading(true)
    setError('')
    try {
      const response = await api.getSystemSettings()
      if (response.data) {
        setSettings(response.data.settings || [])
      } else if (response.error) {
        setError(response.error.message || 'Failed to load settings')
      }
    } catch (err) {
      setError('Failed to load settings')
      console.error('Failed to fetch settings:', err)
    } finally {
      setIsLoading(false)
    }
  }, [])

  // Only fetch settings after confirming admin access
  useEffect(() => {
    if (currentUser && currentUser.role === 'admin') {
      fetchSettings()
    }
  }, [currentUser, fetchSettings])

  const handleToggle = async (key: string, currentValue: string) => {
    setActionLoading(key)
    const newValue = currentValue === 'true' ? 'false' : 'true'

    try {
      const response = await api.updateSystemSetting(key, { value: newValue })
      if (response.data) {
        setSettings((prev) => {
          const exists = prev.some((s) => s.key === key)
          if (exists) {
            return prev.map((s) => (s.key === key ? response.data! : s))
          }
          return [...prev, response.data!]
        })
      } else if (response.error) {
        setError(response.error.message || 'Failed to update setting')
      }
    } catch (err) {
      setError('Failed to update setting')
      console.error('Failed to update setting:', err)
    } finally {
      setActionLoading(null)
    }
  }

  const publicRegSetting = settings.find((s) => s.key === 'public_registration_enabled')

  // Show loading while checking auth or loading settings
  if (isCheckingAuth || isLoading) {
    return (
      <div className="flex min-h-100 items-center justify-center">
        <div className="border-primary h-8 w-8 animate-spin rounded-full border-4 border-t-transparent" />
      </div>
    )
  }

  return (
    <div className="max-w-4xl p-4 sm:p-6">
      <h1 className="text-text-primary mb-2 text-2xl font-bold sm:text-3xl">Admin Settings</h1>
      <p className="text-text-secondary mb-2 text-sm sm:text-base">
        Configure system-wide settings for Nimbus
      </p>
      <p className="text-text-muted mb-8 text-xs sm:text-sm">Changes take effect immediately</p>

      {error && (
        <div className="mb-6 rounded-lg border border-red-500/20 bg-red-500/10 p-4 text-red-500">
          {error}
        </div>
      )}

      <div className="space-y-6">
        {/* User Registration */}
        <div className="bg-card border-card-border rounded-lg border p-6">
          <h2 className="text-text-primary mb-2 text-xl font-semibold">User Registration</h2>
          <p className="text-text-secondary mb-4 text-sm">
            Control how new users can join your Nimbus instance
          </p>

          <Toggle
            enabled={(publicRegSetting?.value || 'true') === 'true'}
            onChange={() =>
              handleToggle('public_registration_enabled', publicRegSetting?.value || 'true')
            }
            label="Public Registration"
            description="Allow new users to register accounts. When disabled, only admins can create new users."
            disabled={actionLoading === 'public_registration_enabled'}
          />
        </div>
      </div>
    </div>
  )
}
