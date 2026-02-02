'use client'

import { useEffect, useState } from 'react'
import { api } from '@/lib/api'
import type { SystemSetting } from '@/types'

export default function AdminSettingsPage() {
  const [settings, setSettings] = useState<SystemSetting[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')
  const [actionLoading, setActionLoading] = useState<string | null>(null)

  useEffect(() => {
    fetchSettings()
  }, [])

  const fetchSettings = async () => {
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
  }

  const handleToggle = async (key: string, currentValue: string) => {
    setActionLoading(key)
    const newValue = currentValue === 'true' ? 'false' : 'true'

    try {
      const response = await api.updateSystemSetting(key, { value: newValue })
      if (response.data) {
        setSettings((prev) => prev.map((s) => (s.key === key ? response.data! : s)))
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

  const getSettingLabel = (key: string): string => {
    const labels: Record<string, string> = {
      public_registration_enabled: 'Public Registration',
    }
    return labels[key] || key
  }

  const getSettingDescription = (key: string): string => {
    const descriptions: Record<string, string> = {
      public_registration_enabled:
        'Allow new users to register accounts. When disabled, only admins can create new users.',
    }
    return descriptions[key] || ''
  }

  // Check if public_registration_enabled exists, if not show it as disabled
  const publicRegSetting = settings.find((s) => s.key === 'public_registration_enabled')

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <div
          className="h-8 w-8 animate-spin rounded-full border-4 border-solid border-current border-r-transparent"
          style={{ color: 'var(--color-primary)' }}
        />
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-2xl px-4 py-6 sm:px-6 lg:px-8">
      <div className="mb-6">
        <h1 className="text-2xl font-bold" style={{ color: 'var(--color-text-primary)' }}>
          System Settings
        </h1>
        <p className="mt-1" style={{ color: 'var(--color-text-secondary)' }}>
          Configure system-wide settings for Nimbus
        </p>
      </div>

      {error && (
        <div
          className="mb-4 rounded-lg border p-3 text-sm"
          style={{
            backgroundColor: 'var(--color-error)',
            borderColor: 'var(--color-error)',
            color: 'white',
            opacity: 0.9,
          }}
        >
          {error}
        </div>
      )}

      <div
        className="rounded-lg border"
        style={{
          backgroundColor: 'var(--color-card)',
          borderColor: 'var(--color-card-border)',
        }}
      >
        <div className="divide-y" style={{ borderColor: 'var(--color-card-border)' }}>
          {/* Registration Setting */}
          <div className="flex items-center justify-between p-4">
            <div className="flex-1 pr-4">
              <h3 className="font-medium" style={{ color: 'var(--color-text-primary)' }}>
                {getSettingLabel('public_registration_enabled')}
              </h3>
              <p className="mt-1 text-sm" style={{ color: 'var(--color-text-secondary)' }}>
                {getSettingDescription('public_registration_enabled')}
              </p>
            </div>
            <button
              onClick={() =>
                handleToggle('public_registration_enabled', publicRegSetting?.value || 'true')
              }
              disabled={actionLoading === 'public_registration_enabled'}
              className={`relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:ring-2 focus:ring-offset-2 focus:outline-none disabled:cursor-not-allowed disabled:opacity-50`}
              style={{
                backgroundColor:
                  (publicRegSetting?.value || 'true') === 'true'
                    ? 'var(--color-success)'
                    : 'var(--color-card-elevated)',
              }}
              role="switch"
              aria-checked={(publicRegSetting?.value || 'true') === 'true'}
            >
              <span
                className={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${
                  (publicRegSetting?.value || 'true') === 'true' ? 'translate-x-5' : 'translate-x-0'
                }`}
              />
            </button>
          </div>
        </div>
      </div>

      <p className="mt-4 text-sm" style={{ color: 'var(--color-text-muted)' }}>
        Changes take effect immediately.
      </p>
    </div>
  )
}
