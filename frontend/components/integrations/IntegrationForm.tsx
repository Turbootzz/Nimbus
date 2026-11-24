'use client'

import { useState } from 'react'
import { useTranslations } from 'next-intl'
import { XMarkIcon } from '@heroicons/react/24/outline'
import { api } from '@/lib/api'
import { Toggle } from '@/components/ui/Toggle'
import { inputClass, labelClass } from '@/components/widgets/forms/fieldStyles'
import type {
  Integration,
  IntegrationAuthType,
  IntegrationCredentials,
  IntegrationKindMeta,
  IntegrationRequest,
  IntegrationTestResult,
} from '@/types'

interface IntegrationFormProps {
  kinds: IntegrationKindMeta[]
  // Edits this integration when set, otherwise adds a new one
  integration?: Integration | null
  // Fills in a new integration, e.g. one a container label asked for
  prefill?: { kind: string; name: string; base_url: string } | null
  onClose: () => void
  onSaved: (integration: Integration) => void
}

// Credentials to send: only what the auth type uses, and only when typed.
// Leaving them empty on edit keeps the saved ones.
function credentialsFor(
  authType: IntegrationAuthType,
  creds: IntegrationCredentials
): IntegrationCredentials | undefined {
  switch (authType) {
    case 'api_key':
      return creds.api_key ? { api_key: creds.api_key } : undefined
    case 'token':
      return creds.token ? { token: creds.token } : undefined
    case 'basic':
      return creds.username || creds.password
        ? { username: creds.username, password: creds.password }
        : undefined
    default:
      return undefined
  }
}

export default function IntegrationForm({
  kinds,
  integration,
  prefill,
  onClose,
  onSaved,
}: IntegrationFormProps) {
  const t = useTranslations('integrations.form')
  const authLabels: Record<IntegrationAuthType, string> = {
    none: t('auth.none'),
    api_key: t('auth.apiKey'),
    token: t('auth.token'),
    basic: t('auth.basic'),
  }
  const isEdit = !!integration
  const [kind, setKind] = useState(integration?.kind ?? prefill?.kind ?? kinds[0]?.kind ?? '')
  const meta = kinds.find((k) => k.kind === kind)
  const [name, setName] = useState(integration?.name ?? prefill?.name ?? meta?.name ?? '')
  const [baseUrl, setBaseUrl] = useState(integration?.base_url ?? prefill?.base_url ?? '')
  const [authType, setAuthType] = useState<IntegrationAuthType>(
    integration?.auth_type ?? meta?.auth_types[0] ?? 'none'
  )
  const [creds, setCreds] = useState<IntegrationCredentials>({})
  // Values of the kind's own options, by key
  const [options, setOptions] = useState<Record<string, string>>(() =>
    Object.fromEntries(
      Object.entries(integration?.options ?? {}).filter(
        (entry): entry is [string, string] => typeof entry[1] === 'string'
      )
    )
  )
  const [verifyTls, setVerifyTls] = useState(integration?.verify_tls ?? true)
  const [refreshSeconds, setRefreshSeconds] = useState(integration?.refresh_seconds ?? 60)
  const [error, setError] = useState<string | null>(null)
  const [isSaving, setIsSaving] = useState(false)
  const [isTesting, setIsTesting] = useState(false)
  const [testResult, setTestResult] = useState<IntegrationTestResult | null>(null)

  const busy = isSaving || isTesting
  const hasSavedCredentials =
    isEdit && integration.has_credentials && integration.auth_type === authType
  const savedPlaceholder = hasSavedCredentials ? t('savedPlaceholder') : ''

  const selectKind = (next: string) => {
    const nextMeta = kinds.find((k) => k.kind === next)
    // Follow the kind's name unless the user typed their own
    if (!name.trim() || name === meta?.name) setName(nextMeta?.name ?? '')
    setKind(next)
    setAuthType(nextMeta?.auth_types[0] ?? 'none')
    setOptions({})
    // An error or a test result was about the other kind
    setError(null)
    setTestResult(null)
  }

  // The saved username is never sent back, so basic auth is changed as a pair
  const credentialsError = (): string | null => {
    if (authType !== 'basic' || !(creds.username || creds.password)) return null
    if (!creds.username) return t('usernameMissing')
    if (hasSavedCredentials && !creds.password) {
      return t('passwordMissing')
    }
    return null
  }

  // Every option the kind has, so that clearing one removes it. Kinds
  // without options send none.
  const optionsToSend = (): Record<string, string> | undefined => {
    if (!meta?.options?.length) return undefined
    return Object.fromEntries(
      meta.options.flatMap(({ key }) => {
        const value = options[key]?.trim()
        return value ? [[key, value]] : []
      })
    )
  }

  const buildRequest = (): IntegrationRequest => ({
    ...(isEdit ? {} : { kind }),
    name: name.trim(),
    base_url: baseUrl.trim(),
    auth_type: authType,
    credentials: credentialsFor(authType, creds),
    verify_tls: verifyTls,
    options: optionsToSend(),
    refresh_seconds: refreshSeconds,
  })

  // A new integration is tested from the form; a saved one as it is stored
  const handleTest = async () => {
    setError(null)
    setTestResult(null)
    const problem = integration ? null : credentialsError()
    if (problem) {
      setError(problem)
      return
    }
    setIsTesting(true)
    try {
      const response = integration
        ? await api.testSavedIntegration(integration.id)
        : await api.testIntegration(buildRequest())
      if (response.error || !response.data) {
        setError(response.error?.message || t('testFailed'))
        return
      }
      setTestResult(response.data)
    } finally {
      setIsTesting(false)
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    const problem = credentialsError()
    if (problem) {
      setError(problem)
      return
    }
    setIsSaving(true)
    try {
      const response = integration
        ? await api.updateIntegration(integration.id, buildRequest())
        : await api.createIntegration(buildRequest())
      if (response.error || !response.data) {
        setError(response.error?.message || t('saveFailed'))
        return
      }
      onSaved(response.data)
    } catch (err) {
      setError(err instanceof Error ? err.message : t('saveFailed'))
    } finally {
      setIsSaving(false)
    }
  }

  const port = meta?.default_port ? `:${meta.default_port}` : ''

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div
        className="absolute inset-0 bg-black/50"
        onClick={busy ? undefined : onClose}
        aria-hidden="true"
      />
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="integration-form-title"
        className="bg-card border-card-border relative z-10 flex max-h-[90vh] w-full max-w-lg flex-col rounded-lg border shadow-lg"
      >
        <div className="border-card-border flex items-center justify-between border-b p-4">
          <h3 id="integration-form-title" className="text-text-primary text-lg font-semibold">
            {isEdit ? t('editTitle', { name: integration.name }) : t('addTitle')}
          </h3>
          <button
            type="button"
            onClick={onClose}
            className="text-text-secondary hover:text-text-primary transition-colors"
            aria-label={t('close')}
            disabled={busy}
          >
            <XMarkIcon className="h-5 w-5" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="flex min-h-0 flex-col">
          <div className="space-y-4 overflow-y-auto p-4">
            {error && (
              <div className="rounded-md bg-red-50 p-3 text-sm text-red-600 dark:bg-red-900/20 dark:text-red-400">
                {error}
              </div>
            )}

            <div>
              <label htmlFor="integration-kind" className={labelClass}>
                {t('app')}
              </label>
              <select
                id="integration-kind"
                value={kind}
                onChange={(e) => selectKind(e.target.value)}
                className={inputClass}
                disabled={busy || isEdit}
              >
                {kinds.map((k) => (
                  <option key={k.kind} value={k.kind}>
                    {k.name}
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label htmlFor="integration-name" className={labelClass}>
                {t('name')}
              </label>
              <input
                id="integration-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                maxLength={100}
                className={inputClass}
                disabled={busy}
                required
              />
            </div>

            <div>
              <label htmlFor="integration-url" className={labelClass}>
                {t('url')}
              </label>
              <input
                id="integration-url"
                type="url"
                value={baseUrl}
                onChange={(e) => setBaseUrl(e.target.value)}
                placeholder={meta?.url_hint ?? `http://192.168.1.10${port}`}
                className={inputClass}
                disabled={busy}
                required
              />
              <p className="text-text-muted mt-1 text-xs break-words">
                {t('urlHelp')}
                {meta?.hint && ` ${meta.hint}`}
              </p>
            </div>

            {meta && meta.auth_types.length > 1 && (
              <div>
                <label htmlFor="integration-auth" className={labelClass}>
                  {t('signInWith')}
                </label>
                <select
                  id="integration-auth"
                  value={authType}
                  onChange={(e) => setAuthType(e.target.value as IntegrationAuthType)}
                  className={inputClass}
                  disabled={busy}
                >
                  {meta.auth_types.map((type) => (
                    <option key={type} value={type}>
                      {authLabels[type]}
                    </option>
                  ))}
                </select>
              </div>
            )}

            {authType === 'api_key' && (
              <div>
                <label htmlFor="integration-api-key" className={labelClass}>
                  {authLabels.api_key}
                </label>
                <input
                  id="integration-api-key"
                  type="password"
                  autoComplete="new-password"
                  value={creds.api_key ?? ''}
                  onChange={(e) => setCreds({ ...creds, api_key: e.target.value })}
                  placeholder={savedPlaceholder}
                  className={inputClass}
                  disabled={busy}
                />
              </div>
            )}
            {authType === 'token' && (
              <div>
                <label htmlFor="integration-token" className={labelClass}>
                  {authLabels.token}
                </label>
                <input
                  id="integration-token"
                  type="password"
                  autoComplete="new-password"
                  value={creds.token ?? ''}
                  onChange={(e) => setCreds({ ...creds, token: e.target.value })}
                  placeholder={savedPlaceholder}
                  className={inputClass}
                  disabled={busy}
                />
              </div>
            )}
            {authType === 'basic' && (
              <div className="grid grid-cols-2 gap-2">
                <div>
                  <label htmlFor="integration-username" className={labelClass}>
                    {t('username')}
                  </label>
                  <input
                    id="integration-username"
                    autoComplete="off"
                    value={creds.username ?? ''}
                    onChange={(e) => setCreds({ ...creds, username: e.target.value })}
                    placeholder={savedPlaceholder}
                    className={inputClass}
                    disabled={busy}
                  />
                </div>
                <div>
                  <label htmlFor="integration-password" className={labelClass}>
                    {t('password')}
                  </label>
                  <input
                    id="integration-password"
                    type="password"
                    autoComplete="new-password"
                    value={creds.password ?? ''}
                    onChange={(e) => setCreds({ ...creds, password: e.target.value })}
                    className={inputClass}
                    disabled={busy}
                  />
                </div>
              </div>
            )}
            <p className="text-text-muted -mt-2 text-xs">{t('credentialsNote')}</p>

            {meta?.options?.map((option) => {
              const id = `integration-option-${option.key}`
              return (
                <div key={`${kind}:${option.key}`}>
                  <label htmlFor={id} className={labelClass}>
                    {option.label}
                  </label>
                  <input
                    id={id}
                    type={option.type === 'number' ? 'number' : 'text'}
                    step="any"
                    value={options[option.key] ?? ''}
                    onChange={(e) => setOptions({ ...options, [option.key]: e.target.value })}
                    placeholder={option.placeholder}
                    maxLength={option.max_length}
                    pattern={option.pattern}
                    title={option.placeholder && t('forExample', { value: option.placeholder })}
                    required={option.required}
                    aria-describedby={option.help ? `${id}-help` : undefined}
                    autoComplete="off"
                    className={inputClass}
                    disabled={busy}
                  />
                  {option.help && (
                    <p id={`${id}-help`} className="text-text-muted mt-1 text-xs">
                      {option.help}
                    </p>
                  )}
                </div>
              )
            })}

            <div>
              <label htmlFor="integration-refresh" className={labelClass}>
                {t('refresh')}
              </label>
              <input
                id="integration-refresh"
                type="number"
                min={10}
                max={86400}
                value={refreshSeconds}
                onChange={(e) => setRefreshSeconds(Number(e.target.value))}
                className={inputClass}
                disabled={busy}
              />
            </div>

            <Toggle
              id="integration-verify-tls"
              enabled={verifyTls}
              onChange={setVerifyTls}
              label={t('verifyTls')}
              description={t('verifyTlsHint')}
              disabled={busy}
            />
            {!verifyTls && (
              <p role="status" className="text-warning -mt-2 text-xs">
                {t('verifyTlsWarning')}
              </p>
            )}

            {testResult && (
              <p
                role="status"
                className={`text-sm ${testResult.ok ? 'text-success' : 'text-error'}`}
              >
                {testResult.ok
                  ? t('connectedIn', { ms: testResult.latency_ms })
                  : t('connectionFailed', { error: testResult.error ?? '' })}
              </p>
            )}
          </div>

          <div className="border-card-border flex items-center justify-between gap-2 border-t p-4">
            <button
              type="button"
              onClick={handleTest}
              className="border-card-border text-text-primary hover:bg-card-hover rounded-md border px-4 py-2 text-sm font-medium transition-colors"
              disabled={busy}
            >
              {isTesting ? t('testing') : isEdit ? t('testSaved') : t('test')}
            </button>
            <div className="flex gap-2">
              <button
                type="button"
                onClick={onClose}
                className="text-text-secondary hover:text-text-primary hover:bg-card-hover rounded-md px-4 py-2 text-sm font-medium transition-colors"
                disabled={busy}
              >
                {t('cancel')}
              </button>
              <button
                type="submit"
                disabled={busy}
                className="bg-primary hover:bg-primary-hover disabled:bg-primary/50 rounded-md px-4 py-2 text-sm font-medium text-white transition-colors disabled:cursor-not-allowed"
              >
                {isSaving ? t('saving') : isEdit ? t('save') : t('add')}
              </button>
            </div>
          </div>
        </form>
      </div>
    </div>
  )
}
