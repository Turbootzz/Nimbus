'use client'

import { useState } from 'react'
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
  onClose: () => void
  onSaved: (integration: Integration) => void
}

const authLabels: Record<IntegrationAuthType, string> = {
  none: 'None',
  api_key: 'API key',
  token: 'Token',
  basic: 'Username and password',
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
  onClose,
  onSaved,
}: IntegrationFormProps) {
  const isEdit = !!integration
  const [kind, setKind] = useState(integration?.kind ?? kinds[0]?.kind ?? '')
  const meta = kinds.find((k) => k.kind === kind)
  const [name, setName] = useState(integration?.name ?? meta?.name ?? '')
  const [baseUrl, setBaseUrl] = useState(integration?.base_url ?? '')
  const [authType, setAuthType] = useState<IntegrationAuthType>(
    integration?.auth_type ?? meta?.auth_types[0] ?? 'none'
  )
  const [creds, setCreds] = useState<IntegrationCredentials>({})
  const [verifyTls, setVerifyTls] = useState(integration?.verify_tls ?? true)
  const [refreshSeconds, setRefreshSeconds] = useState(integration?.refresh_seconds ?? 60)
  const [error, setError] = useState<string | null>(null)
  const [isSaving, setIsSaving] = useState(false)
  const [isTesting, setIsTesting] = useState(false)
  const [testResult, setTestResult] = useState<IntegrationTestResult | null>(null)

  const busy = isSaving || isTesting
  const hasSavedCredentials =
    isEdit && integration.has_credentials && integration.auth_type === authType
  const savedPlaceholder = hasSavedCredentials ? 'Saved; leave empty to keep' : ''

  const selectKind = (next: string) => {
    const nextMeta = kinds.find((k) => k.kind === next)
    // Follow the kind's name unless the user typed their own
    if (!name.trim() || name === meta?.name) setName(nextMeta?.name ?? '')
    setKind(next)
    setAuthType(nextMeta?.auth_types[0] ?? 'none')
    setTestResult(null)
  }

  // The saved username is never sent back, so basic auth is changed as a pair
  const credentialsError = (): string | null => {
    if (authType !== 'basic' || !(creds.username || creds.password)) return null
    if (!creds.username) return 'Enter the username too'
    if (hasSavedCredentials && !creds.password) {
      return 'Enter the password too, or leave both empty to keep the saved ones'
    }
    return null
  }

  const buildRequest = (): IntegrationRequest => ({
    ...(isEdit ? {} : { kind }),
    name: name.trim(),
    base_url: baseUrl.trim(),
    auth_type: authType,
    credentials: credentialsFor(authType, creds),
    verify_tls: verifyTls,
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
        setError(response.error?.message || 'Test failed')
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
        setError(response.error?.message || 'Failed to save integration')
        return
      }
      onSaved(response.data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save integration')
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
            {isEdit ? `Edit ${integration.name}` : 'Add Integration'}
          </h3>
          <button
            type="button"
            onClick={onClose}
            className="text-text-secondary hover:text-text-primary transition-colors"
            aria-label="Close"
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
                App
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
                Name
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
                URL
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
              <p className="text-text-muted mt-1 text-xs">
                The address Nimbus uses to reach the app, including any sub path.
                {meta?.unix_socket &&
                  ' Use unix:// with the socket the server allows in DOCKER_SOCKET, or the http:// address of a socket proxy.'}
              </p>
            </div>

            {meta && meta.auth_types.length > 1 && (
              <div>
                <label htmlFor="integration-auth" className={labelClass}>
                  Sign in with
                </label>
                <select
                  id="integration-auth"
                  value={authType}
                  onChange={(e) => setAuthType(e.target.value as IntegrationAuthType)}
                  className={inputClass}
                  disabled={busy}
                >
                  {meta.auth_types.map((t) => (
                    <option key={t} value={t}>
                      {authLabels[t]}
                    </option>
                  ))}
                </select>
              </div>
            )}

            {authType === 'api_key' && (
              <div>
                <label htmlFor="integration-api-key" className={labelClass}>
                  API key
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
                  Token
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
                    Username
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
                    Password
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
            <p className="text-text-muted -mt-2 text-xs">
              Credentials are stored encrypted and never sent back to the browser.
            </p>

            <div>
              <label htmlFor="integration-refresh" className={labelClass}>
                Refresh every (seconds)
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
              label="Verify TLS certificate"
              description="Turn off for apps with a self-signed certificate"
              disabled={busy}
            />

            {testResult && (
              <p
                role="status"
                className={`text-sm ${testResult.ok ? 'text-success' : 'text-error'}`}
              >
                {testResult.ok
                  ? `Connected in ${testResult.latency_ms} ms`
                  : `Connection failed: ${testResult.error}`}
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
              {isTesting ? 'Testing...' : isEdit ? 'Test saved connection' : 'Test connection'}
            </button>
            <div className="flex gap-2">
              <button
                type="button"
                onClick={onClose}
                className="text-text-secondary hover:text-text-primary hover:bg-card-hover rounded-md px-4 py-2 text-sm font-medium transition-colors"
                disabled={busy}
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={busy}
                className="bg-primary hover:bg-primary-hover disabled:bg-primary/50 rounded-md px-4 py-2 text-sm font-medium text-white transition-colors disabled:cursor-not-allowed"
              >
                {isSaving ? 'Saving...' : isEdit ? 'Save Changes' : 'Add Integration'}
              </button>
            </div>
          </div>
        </form>
      </div>
    </div>
  )
}
