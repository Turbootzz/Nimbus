'use client'

import { useEffect, useState, useCallback } from 'react'
import { useRouter } from 'next/navigation'
import { useTranslations } from 'next-intl'
import { api } from '@/lib/api'
import { Toggle } from '@/components/ui/Toggle'
import SMTPSettingsSection from '@/components/SMTPSettingsSection'
import StatusPageSettings from '@/components/StatusPageSettings'
import type { SystemSetting, User } from '@/types'

export default function AdminSettingsPage() {
  const t = useTranslations('admin.page')
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
        setError(response.error.message || t('loadFailed'))
      }
    } catch (err) {
      setError(t('loadFailed'))
      console.error('Failed to fetch settings:', err)
    } finally {
      setIsLoading(false)
    }
  }, [t])

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
        setError(response.error.message || t('updateFailed'))
      }
    } catch (err) {
      setError(t('updateFailed'))
      console.error('Failed to update setting:', err)
    } finally {
      setActionLoading(null)
    }
  }

  const publicRegSetting = settings.find((s) => s.key === 'public_registration_enabled')
  const autoAddValue = settings.find((s) => s.key === 'discovery_auto_add')?.value || 'false'

  // Show loading while checking auth or loading settings
  if (isCheckingAuth || isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="border-primary h-8 w-8 animate-spin rounded-full border-4 border-t-transparent" />
      </div>
    )
  }

  return (
    <div className="max-w-4xl p-4 sm:p-6">
      <div className="wallpaper-panel mb-8">
        <h1 className="text-text-primary mb-2 text-2xl font-bold sm:text-3xl">{t('title')}</h1>
        <p className="text-text-secondary mb-2 text-sm sm:text-base">{t('subtitle')}</p>
        <p className="text-text-muted text-xs sm:text-sm">{t('immediate')}</p>
      </div>

      {error && (
        <div className="mb-6 rounded-lg border border-red-500/20 bg-red-500/10 p-4 text-red-500">
          {error}
        </div>
      )}

      <div className="space-y-6">
        {/* User Registration */}
        <div className="glass-card border-card-border rounded-lg border p-6">
          <h2 className="text-text-primary mb-2 text-xl font-semibold">
            {t('registration.title')}
          </h2>
          <p className="text-text-secondary mb-4 text-sm">{t('registration.description')}</p>

          <Toggle
            id="public-registration-toggle"
            enabled={(publicRegSetting?.value || 'true') === 'true'}
            onChange={() =>
              handleToggle('public_registration_enabled', publicRegSetting?.value || 'true')
            }
            label={t('registration.toggle')}
            description={t('registration.toggleDescription')}
            disabled={actionLoading === 'public_registration_enabled'}
          />
        </div>

        {/* Docker discovery */}
        <div className="glass-card border-card-border rounded-lg border p-6">
          <h2 className="text-text-primary mb-2 text-xl font-semibold">{t('discovery.title')}</h2>
          <p className="text-text-secondary mb-4 text-sm">{t('discovery.description')}</p>

          <Toggle
            id="discovery-auto-add-toggle"
            enabled={autoAddValue === 'true'}
            onChange={() => handleToggle('discovery_auto_add', autoAddValue)}
            label={t('discovery.toggle')}
            description={t('discovery.toggleDescription')}
            disabled={actionLoading === 'discovery_auto_add'}
          />
        </div>

        {/* SMTP / Email Configuration */}
        <div className="glass-card border-card-border rounded-lg border p-6">
          <h2 className="text-text-primary mb-2 text-xl font-semibold">{t('smtp.title')}</h2>
          <p className="text-text-secondary mb-4 text-sm">{t('smtp.description')}</p>
          <SMTPSettingsSection settings={settings} />
        </div>

        {/* Public status page */}
        <div className="glass-card border-card-border rounded-lg border p-6">
          <h2 className="text-text-primary mb-2 text-xl font-semibold">{t('statusPage.title')}</h2>
          <p className="text-text-secondary mb-4 text-sm">{t('statusPage.description')}</p>
          <StatusPageSettings />
        </div>
      </div>
    </div>
  )
}
