'use client'

import { useState, useEffect } from 'react'
import { useTranslations } from 'next-intl'
import { ThemedInput } from '@/components/ui/ThemedInput'
import { Toggle } from '@/components/ui/Toggle'
import { api } from '@/lib/api'
import type {
  SystemSetting,
  SMTPStatusResponse,
  SMTPTLSMode,
  UpdateSMTPSettingsRequest,
} from '@/types'

const SMTP_KEYS = [
  'smtp_host',
  'smtp_port',
  'smtp_username',
  'smtp_password',
  'smtp_from_email',
  'smtp_from_name',
  'smtp_enabled',
  'smtp_tls_mode',
  'smtp_tls_skip_verify',
] as const

type SMTPKey = (typeof SMTP_KEYS)[number]

const BOOLEAN_KEYS: SMTPKey[] = ['smtp_enabled', 'smtp_tls_skip_verify']

// '' lets the backend pick: implicit TLS on port 465, STARTTLS elsewhere
type TLSSelection = SMTPTLSMode | ''

const TLS_MODES: TLSSelection[] = ['', 'starttls', 'tls', 'none']

interface SMTPFormData {
  smtp_host: string
  smtp_port: string
  smtp_username: string
  smtp_password: string
  smtp_from_email: string
  smtp_from_name: string
  smtp_enabled: boolean
  smtp_tls_mode: TLSSelection
  smtp_tls_skip_verify: boolean
}

const defaultFormData: SMTPFormData = {
  smtp_host: '',
  smtp_port: '587',
  smtp_username: '',
  smtp_password: '',
  smtp_from_email: '',
  smtp_from_name: 'Nimbus',
  smtp_enabled: false,
  smtp_tls_mode: '',
  smtp_tls_skip_verify: false,
}

function toTLSSelection(value: string): TLSSelection {
  return TLS_MODES.includes(value as TLSSelection) ? (value as TLSSelection) : ''
}

export default function SMTPSettingsSection({ settings }: { settings: SystemSetting[] }) {
  const t = useTranslations('admin.smtp')
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
      const key = setting.key as SMTPKey
      if (!SMTP_KEYS.includes(key)) continue

      if (key === 'smtp_tls_mode') {
        data.smtp_tls_mode = toTLSSelection(setting.value)
      } else if (BOOLEAN_KEYS.includes(key)) {
        ;(data as Record<string, string | boolean>)[key] = setting.value === 'true'
      } else {
        ;(data as Record<string, string | boolean>)[key] = setting.value
      }
    }
    setFormData(data)
  }, [settings])

  const buildPayload = (): UpdateSMTPSettingsRequest => ({
    smtp_host: formData.smtp_host,
    smtp_port: formData.smtp_port,
    smtp_username: formData.smtp_username,
    smtp_password: formData.smtp_password,
    smtp_from_email: formData.smtp_from_email,
    smtp_from_name: formData.smtp_from_name,
    smtp_enabled: String(formData.smtp_enabled),
    smtp_tls_mode: formData.smtp_tls_mode,
    smtp_tls_skip_verify: String(formData.smtp_tls_skip_verify),
  })

  const handleSave = async () => {
    setIsSaving(true)
    setError('')
    setSuccess('')

    try {
      const response = await api.updateSMTPSettings(buildPayload())
      if (response.error) {
        setError(response.error.message)
      } else {
        setSuccess(t('saved'))
      }
    } catch {
      setError(t('saveFailed'))
    } finally {
      setIsSaving(false)
    }
  }

  const handleTest = async () => {
    setIsTesting(true)
    setError('')
    setSuccess('')

    try {
      const response = await api.testSMTPConnection(buildPayload())
      if (response.data?.success) {
        setSuccess(t('testSucceeded'))
      } else {
        setError(response.data?.message || response.error?.message || t('testFailed'))
      }
    } catch {
      setError(t('testError'))
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
            color: 'var(--color-on-primary, white)',
            opacity: 0.9,
          }}
        >
          {t('fromEnv')}
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
          {t('notConfigured')}
        </div>
      )}

      <Toggle
        enabled={formData.smtp_enabled}
        onChange={(enabled) => updateField('smtp_enabled', enabled)}
        label={t('enable')}
        description={t('enableDescription')}
      />

      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <div>
          <label
            htmlFor="smtp-host"
            className="mb-1 block text-xs font-medium"
            style={{ color: 'var(--color-text-secondary)' }}
          >
            {t('host')}
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
            {t('port')}
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
            htmlFor="smtp-tls-mode"
            className="mb-1 block text-xs font-medium"
            style={{ color: 'var(--color-text-secondary)' }}
          >
            {t('encryption')}
          </label>
          <select
            id="smtp-tls-mode"
            value={formData.smtp_tls_mode}
            onChange={(e) => updateField('smtp_tls_mode', e.target.value)}
            className="border-card-border focus:border-primary focus:ring-opacity-50 w-full rounded-md border px-4 py-2 transition focus:ring-2 focus:outline-none"
            style={{
              backgroundColor: 'var(--color-background)',
              color: 'var(--color-text-primary)',
            }}
          >
            {TLS_MODES.map((mode) => (
              <option key={mode} value={mode}>
                {t(`encryptionModes.${mode || 'auto'}`)}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label
            htmlFor="smtp-username"
            className="mb-1 block text-xs font-medium"
            style={{ color: 'var(--color-text-secondary)' }}
          >
            {t('username')}
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
            {t('password')}
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
            {t('fromEmail')}
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
            {t('fromName')}
          </label>
          <ThemedInput
            id="smtp-from-name"
            value={formData.smtp_from_name}
            onChange={(e) => updateField('smtp_from_name', e.target.value)}
            placeholder="Nimbus"
          />
        </div>
      </div>

      {formData.smtp_tls_mode === 'none' ? (
        <div
          className="rounded-lg p-3 text-sm"
          style={{ backgroundColor: 'var(--color-error)', color: 'white', opacity: 0.9 }}
        >
          {t('unencryptedWarning')}
        </div>
      ) : (
        <Toggle
          enabled={formData.smtp_tls_skip_verify}
          onChange={(enabled) => updateField('smtp_tls_skip_verify', enabled)}
          label={t('skipVerify')}
          description={t('skipVerifyDescription')}
        />
      )}

      <div className="flex space-x-3">
        <button
          onClick={handleSave}
          disabled={isSaving}
          className="rounded-lg px-4 py-2 text-sm font-medium text-white transition disabled:cursor-not-allowed disabled:opacity-50"
          style={{
            backgroundColor: 'var(--color-primary)',
            color: 'var(--color-on-primary, white)',
          }}
        >
          {isSaving ? t('saving') : t('save')}
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
          {isTesting ? t('testing') : t('test')}
        </button>
      </div>
    </div>
  )
}
