'use client'

import { useState, useEffect } from 'react'
import { ThemedInput } from '@/components/ui/ThemedInput'
import { Toggle } from '@/components/ui/Toggle'
import { api } from '@/lib/api'
import type { SystemSetting, SMTPStatusResponse } from '@/types'

const SMTP_KEYS = [
  'smtp_host',
  'smtp_port',
  'smtp_username',
  'smtp_password',
  'smtp_from_email',
  'smtp_from_name',
  'smtp_enabled',
] as const

type SMTPKey = (typeof SMTP_KEYS)[number]

interface SMTPFormData {
  smtp_host: string
  smtp_port: string
  smtp_username: string
  smtp_password: string
  smtp_from_email: string
  smtp_from_name: string
  smtp_enabled: boolean
}

const defaultFormData: SMTPFormData = {
  smtp_host: '',
  smtp_port: '587',
  smtp_username: '',
  smtp_password: '',
  smtp_from_email: '',
  smtp_from_name: 'Nimbus',
  smtp_enabled: false,
}

export default function SMTPSettingsSection({ settings }: { settings: SystemSetting[] }) {
  const [formData, setFormData] = useState<SMTPFormData>(defaultFormData)
  const [isSaving, setIsSaving] = useState(false)
  const [isTesting, setIsTesting] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [smtpStatus, setSMTPStatus] = useState<SMTPStatusResponse | null>(null)

  // Fetch SMTP configuration source
  useEffect(() => {
    api.getSMTPStatus().then((res) => {
      if (res.data) setSMTPStatus(res.data)
    })
  }, [])

  // Initialize form from settings
  useEffect(() => {
    const data = { ...defaultFormData }
    for (const setting of settings) {
      if (SMTP_KEYS.includes(setting.key as SMTPKey)) {
        if (setting.key === 'smtp_enabled') {
          data.smtp_enabled = setting.value === 'true'
        } else {
          ;(data as Record<string, string | boolean>)[setting.key] = setting.value
        }
      }
    }
    setFormData(data)
  }, [settings])

  const handleSave = async () => {
    setIsSaving(true)
    setError('')
    setSuccess('')

    try {
      const response = await api.updateSMTPSettings({
        smtp_host: formData.smtp_host,
        smtp_port: formData.smtp_port,
        smtp_username: formData.smtp_username,
        smtp_password: formData.smtp_password,
        smtp_from_email: formData.smtp_from_email,
        smtp_from_name: formData.smtp_from_name,
        smtp_enabled: String(formData.smtp_enabled),
      })
      if (response.error) {
        setError(response.error.message)
      } else {
        setSuccess('SMTP settings saved successfully')
      }
    } catch {
      setError('Failed to save settings')
    } finally {
      setIsSaving(false)
    }
  }

  const handleTest = async () => {
    setIsTesting(true)
    setError('')
    setSuccess('')

    try {
      const response = await api.testSMTPConnection({
        smtp_host: formData.smtp_host,
        smtp_port: formData.smtp_port,
        smtp_username: formData.smtp_username,
        smtp_password: formData.smtp_password,
        smtp_from_email: formData.smtp_from_email,
        smtp_from_name: formData.smtp_from_name,
        smtp_enabled: String(formData.smtp_enabled),
      })
      if (response.data?.success) {
        setSuccess('SMTP connection successful')
      } else {
        setError(response.data?.message || response.error?.message || 'Connection test failed')
      }
    } catch {
      setError('Failed to test SMTP connection')
    } finally {
      setIsTesting(false)
    }
  }

  const updateField = (key: SMTPKey, value: string | boolean) => {
    setFormData((prev) => ({ ...prev, [key]: value }))
  }

  return (
    <div className="space-y-4">
      {error && (
        <div
          className="rounded-lg p-3 text-sm"
          style={{
            backgroundColor: 'var(--color-error)',
            color: 'white',
            opacity: 0.9,
          }}
        >
          {error}
        </div>
      )}

      {success && (
        <div
          className="rounded-lg p-3 text-sm"
          style={{
            backgroundColor: 'var(--color-success, #22c55e)',
            color: 'white',
            opacity: 0.9,
          }}
        >
          {success}
        </div>
      )}

      {smtpStatus?.source === 'env' && (
        <div
          className="rounded-lg p-3 text-sm"
          style={{
            backgroundColor: 'var(--color-primary)',
            color: 'white',
            opacity: 0.9,
          }}
        >
          SMTP is pre-configured. You can override these settings below.
        </div>
      )}

      {smtpStatus?.source === 'none' && (
        <div
          className="rounded-lg p-3 text-sm"
          style={{
            backgroundColor: 'var(--color-card-border)',
            color: 'var(--color-text-secondary)',
          }}
        >
          SMTP is not configured. Fill in the fields below and enable SMTP to allow password reset
          emails.
        </div>
      )}

      <Toggle
        enabled={formData.smtp_enabled}
        onChange={(enabled) => updateField('smtp_enabled', enabled)}
        label="Enable SMTP"
        description="Required for password reset emails"
      />

      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <div>
          <label
            htmlFor="smtp-host"
            className="mb-1 block text-xs font-medium"
            style={{ color: 'var(--color-text-secondary)' }}
          >
            SMTP Host
          </label>
          <ThemedInput
            id="smtp-host"
            value={formData.smtp_host}
            onChange={(e) => updateField('smtp_host', e.target.value)}
            placeholder="smtp.gmail.com"
          />
        </div>
        <div>
          <label
            htmlFor="smtp-port"
            className="mb-1 block text-xs font-medium"
            style={{ color: 'var(--color-text-secondary)' }}
          >
            SMTP Port
          </label>
          <ThemedInput
            id="smtp-port"
            value={formData.smtp_port}
            onChange={(e) => updateField('smtp_port', e.target.value)}
            placeholder="587"
          />
        </div>
        <div>
          <label
            htmlFor="smtp-username"
            className="mb-1 block text-xs font-medium"
            style={{ color: 'var(--color-text-secondary)' }}
          >
            Username
          </label>
          <ThemedInput
            id="smtp-username"
            value={formData.smtp_username}
            onChange={(e) => updateField('smtp_username', e.target.value)}
            placeholder="user@gmail.com"
          />
        </div>
        <div>
          <label
            htmlFor="smtp-password"
            className="mb-1 block text-xs font-medium"
            style={{ color: 'var(--color-text-secondary)' }}
          >
            Password
          </label>
          <ThemedInput
            id="smtp-password"
            type="password"
            value={formData.smtp_password}
            onChange={(e) => updateField('smtp_password', e.target.value)}
            placeholder="••••••••"
          />
        </div>
        <div>
          <label
            htmlFor="smtp-from-email"
            className="mb-1 block text-xs font-medium"
            style={{ color: 'var(--color-text-secondary)' }}
          >
            From Email
          </label>
          <ThemedInput
            id="smtp-from-email"
            type="email"
            value={formData.smtp_from_email}
            onChange={(e) => updateField('smtp_from_email', e.target.value)}
            placeholder="noreply@yourdomain.com"
          />
        </div>
        <div>
          <label
            htmlFor="smtp-from-name"
            className="mb-1 block text-xs font-medium"
            style={{ color: 'var(--color-text-secondary)' }}
          >
            From Name
          </label>
          <ThemedInput
            id="smtp-from-name"
            value={formData.smtp_from_name}
            onChange={(e) => updateField('smtp_from_name', e.target.value)}
            placeholder="Nimbus"
          />
        </div>
      </div>

      <div className="flex space-x-3">
        <button
          onClick={handleSave}
          disabled={isSaving}
          className="rounded-lg px-4 py-2 text-sm font-medium text-white transition disabled:cursor-not-allowed disabled:opacity-50"
          style={{ backgroundColor: 'var(--color-primary)' }}
        >
          {isSaving ? 'Saving...' : 'Save settings'}
        </button>
        <button
          onClick={handleTest}
          disabled={isTesting || isSaving}
          className="rounded-lg border px-4 py-2 text-sm font-medium transition disabled:cursor-not-allowed disabled:opacity-50"
          style={{
            borderColor: 'var(--color-card-border)',
            color: 'var(--color-text-primary)',
          }}
        >
          {isTesting ? 'Testing...' : 'Test connection'}
        </button>
      </div>
    </div>
  )
}
