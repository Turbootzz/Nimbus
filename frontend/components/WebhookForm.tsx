'use client'

import { useEffect, useState } from 'react'
import { XMarkIcon } from '@heroicons/react/24/outline'
import {
  NTFY_PRIORITIES,
  WEBHOOK_TARGETS,
  buildWebhookRequest,
  getWebhookTarget,
  validateWebhookForm,
  type WebhookFieldErrors,
  type WebhookFormValues,
} from '@/lib/webhook-targets'
import type { ApiError, Webhook, WebhookCreateRequest } from '@/types'

export type WebhookFormData = WebhookCreateRequest

interface WebhookFormProps {
  webhook?: Webhook | null
  // Resolves to the API error, or null when the webhook was saved
  onSubmit: (data: WebhookFormData) => Promise<ApiError | null>
  onClose: () => void
}

const inputClass =
  'bg-background border-card-border text-text-primary placeholder:text-text-muted focus:border-primary focus-visible:ring-primary/40 w-full rounded-lg border px-4 py-2 focus:outline-none focus-visible:ring-2 disabled:opacity-50 aria-[invalid=true]:border-red-500'

interface FieldProps {
  id: string
  label: string
  help?: string
  error?: string
  children: (props: {
    id: string
    'aria-invalid': boolean
    'aria-describedby': string | undefined
  }) => React.ReactNode
}

// Field renders a label, the input and its help or validation message
function Field({ id, label, help, error, children }: FieldProps) {
  const noteId = error ? `${id}-error` : help ? `${id}-help` : undefined
  return (
    <div>
      <label htmlFor={id} className="text-text-primary mb-1 block text-sm font-medium">
        {label}
      </label>
      {children({ id, 'aria-invalid': !!error, 'aria-describedby': noteId })}
      {error ? (
        <p id={noteId} role="alert" className="mt-1 text-xs text-red-500">
          {error}
        </p>
      ) : (
        help && (
          <p id={noteId} className="text-text-muted mt-1 text-xs">
            {help}
          </p>
        )
      )}
    </div>
  )
}

export function WebhookForm({ webhook, onSubmit, onClose }: WebhookFormProps) {
  const [values, setValues] = useState<WebhookFormValues>({
    name: webhook?.name || '',
    format: webhook?.format || 'generic',
    // Telegram and Pushover store their fixed API endpoint as the URL
    url: webhook && getWebhookTarget(webhook.format).url ? webhook.url : '',
    token: '',
    userKey: '',
    chatId: webhook?.options?.chat_id || '',
    priority: webhook?.options?.priority || 3,
    removeToken: false,
    onOffline: webhook?.triggers?.on_offline ?? true,
    onOnline: webhook?.triggers?.on_online ?? false,
    retryCount: webhook?.retry_count ?? 0,
    retryDelaySeconds: webhook?.retry_delay_seconds ?? 30,
  })
  const [errors, setErrors] = useState<WebhookFieldErrors>({})
  const [error, setError] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(false)

  const isEditMode = !!webhook
  const target = getWebhookTarget(values.format)
  // Saved secrets only carry over while the type stays the same
  const keepsCredentials = !!webhook?.has_credentials && webhook.format === values.format
  const savedSecretPlaceholder = keepsCredentials ? 'Saved. Leave blank to keep it' : undefined

  const set = <K extends keyof WebhookFormValues>(key: K, value: WebhookFormValues[K]) =>
    setValues((current) => ({ ...current, [key]: value }))

  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !isLoading) onClose()
    }
    document.addEventListener('keydown', onKeyDown)
    return () => document.removeEventListener('keydown', onKeyDown)
  }, [isLoading, onClose])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)

    const found = validateWebhookForm(values, keepsCredentials)
    setErrors(found)
    if (Object.keys(found).length > 0) return

    setIsLoading(true)
    try {
      const failure = await onSubmit(buildWebhookRequest(values, keepsCredentials))
      if (failure) {
        // Field messages show at their input; the summary always shows on top
        setErrors(failure.fields ?? {})
        setError(failure.message || 'Failed to save webhook')
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save webhook')
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div
        className={`absolute inset-0 bg-black/50 ${isLoading ? 'cursor-not-allowed' : ''}`}
        onClick={isLoading ? undefined : onClose}
        aria-hidden="true"
      />

      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="webhook-form-title"
        className="bg-card border-card-border relative z-10 max-h-[90vh] w-full max-w-md overflow-y-auto rounded-lg border p-4 shadow-xl sm:p-6"
      >
        <div className="mb-6 flex items-center justify-between">
          <h2 id="webhook-form-title" className="text-text-primary text-xl font-semibold">
            {isEditMode ? 'Edit Webhook' : 'Add Webhook'}
          </h2>
          <button
            type="button"
            onClick={onClose}
            disabled={isLoading}
            aria-label="Close"
            className="text-text-muted hover:text-text-primary focus-visible:ring-primary/40 rounded focus:outline-none focus-visible:ring-2 disabled:opacity-50"
          >
            <XMarkIcon className="h-6 w-6" />
          </button>
        </div>

        <form onSubmit={handleSubmit} noValidate className="space-y-4">
          {error && (
            <div
              role="alert"
              className="rounded-lg border border-red-500/20 bg-red-500/10 p-3 text-sm text-red-500"
            >
              {error}
            </div>
          )}

          <Field id="name" label="Name" error={errors.name}>
            {(props) => (
              <input
                {...props}
                type="text"
                value={values.name}
                onChange={(e) => set('name', e.target.value)}
                placeholder="Homelab alerts"
                disabled={isLoading}
                className={inputClass}
              />
            )}
          </Field>

          <div>
            <span
              id="webhook-type-label"
              className="text-text-primary mb-2 block text-sm font-medium"
            >
              Type
            </span>
            <div
              role="group"
              aria-labelledby="webhook-type-label"
              className="grid grid-cols-2 gap-2 sm:grid-cols-4"
            >
              {WEBHOOK_TARGETS.map((option) => (
                <button
                  key={option.value}
                  type="button"
                  onClick={() => {
                    set('format', option.value)
                    setErrors({})
                  }}
                  disabled={isLoading}
                  aria-pressed={values.format === option.value}
                  className={`focus-visible:ring-primary/40 rounded-lg border px-3 py-2 text-sm transition-colors focus:outline-none focus-visible:ring-2 ${
                    values.format === option.value
                      ? 'border-primary bg-primary/10 text-primary'
                      : 'border-card-border text-text-secondary hover:border-primary/50'
                  } disabled:opacity-50`}
                >
                  {option.label}
                </button>
              ))}
            </div>
            <p className="text-text-muted mt-1 text-xs">{target.description}</p>
          </div>

          {target.url && (
            <Field id="url" label={target.url.label} help={target.url.help} error={errors.url}>
              {(props) => (
                <input
                  {...props}
                  type="url"
                  value={values.url}
                  onChange={(e) => set('url', e.target.value)}
                  placeholder={target.url?.placeholder}
                  disabled={isLoading}
                  className={inputClass}
                />
              )}
            </Field>
          )}

          {target.token && (
            <Field
              id="token"
              label={
                target.token.optional ? `${target.token.label} (optional)` : target.token.label
              }
              help={target.token.help}
              error={errors.token}
            >
              {(props) => (
                <input
                  {...props}
                  type="password"
                  autoComplete="off"
                  value={values.token}
                  onChange={(e) => set('token', e.target.value)}
                  placeholder={savedSecretPlaceholder ?? target.token?.placeholder}
                  disabled={isLoading || values.removeToken}
                  className={inputClass}
                />
              )}
            </Field>
          )}

          {target.token?.optional && keepsCredentials && (
            <label className="flex cursor-pointer items-center gap-3">
              <input
                type="checkbox"
                checked={values.removeToken}
                onChange={(e) => {
                  set('removeToken', e.target.checked)
                  if (e.target.checked) set('token', '')
                }}
                disabled={isLoading}
                className="text-primary focus:ring-primary h-4 w-4 rounded border-gray-300"
              />
              <span className="text-text-primary text-sm">Remove the saved access token</span>
            </label>
          )}

          {target.userKey && (
            <Field
              id="user_key"
              label={target.userKey.label}
              help={target.userKey.help}
              error={errors.user_key}
            >
              {(props) => (
                <input
                  {...props}
                  type="password"
                  autoComplete="off"
                  value={values.userKey}
                  onChange={(e) => set('userKey', e.target.value)}
                  placeholder={savedSecretPlaceholder}
                  disabled={isLoading}
                  className={inputClass}
                />
              )}
            </Field>
          )}

          {target.chatId && (
            <Field
              id="chat_id"
              label={target.chatId.label}
              help={target.chatId.help}
              error={errors.chat_id}
            >
              {(props) => (
                <input
                  {...props}
                  type="text"
                  value={values.chatId}
                  onChange={(e) => set('chatId', e.target.value)}
                  placeholder={target.chatId?.placeholder}
                  disabled={isLoading}
                  className={inputClass}
                />
              )}
            </Field>
          )}

          {target.priority && (
            <Field id="priority" label="Priority" error={errors.priority}>
              {(props) => (
                <select
                  {...props}
                  value={values.priority}
                  onChange={(e) => set('priority', Number(e.target.value))}
                  disabled={isLoading}
                  className={inputClass}
                >
                  {NTFY_PRIORITIES.map((priority) => (
                    <option key={priority.value} value={priority.value}>
                      {priority.label}
                    </option>
                  ))}
                </select>
              )}
            </Field>
          )}

          <fieldset aria-describedby={errors.triggers ? 'triggers-error' : undefined}>
            <legend className="text-text-primary mb-2 block text-sm font-medium">
              Trigger Events
            </legend>
            <div className="space-y-2">
              <label className="flex cursor-pointer items-center gap-3">
                <input
                  type="checkbox"
                  checked={values.onOffline}
                  onChange={(e) => set('onOffline', e.target.checked)}
                  disabled={isLoading}
                  className="text-primary focus:ring-primary h-4 w-4 rounded border-gray-300"
                />
                <span className="text-text-primary text-sm">When service goes offline</span>
              </label>
              <label className="flex cursor-pointer items-center gap-3">
                <input
                  type="checkbox"
                  checked={values.onOnline}
                  onChange={(e) => set('onOnline', e.target.checked)}
                  disabled={isLoading}
                  className="text-primary focus:ring-primary h-4 w-4 rounded border-gray-300"
                />
                <span className="text-text-primary text-sm">When service comes back online</span>
              </label>
            </div>
            {errors.triggers && (
              <p id="triggers-error" role="alert" className="mt-1 text-xs text-red-500">
                {errors.triggers}
              </p>
            )}
          </fieldset>

          {/* Notification Retries */}
          <div>
            <span className="text-text-primary mb-1 block text-sm font-medium">
              Notification Retries
            </span>
            <p className="text-text-muted mb-3 text-xs">
              For offline alerts: re-check before sending to prevent false positives.
            </p>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label htmlFor="retryCount" className="text-text-secondary mb-1 block text-xs">
                  Retries (0-5)
                </label>
                <input
                  type="number"
                  id="retryCount"
                  min={0}
                  max={5}
                  value={values.retryCount}
                  onChange={(e) =>
                    set('retryCount', Math.max(0, Math.min(5, parseInt(e.target.value) || 0)))
                  }
                  disabled={isLoading}
                  className={inputClass}
                />
              </div>
              <div>
                <label htmlFor="retryDelay" className="text-text-secondary mb-1 block text-xs">
                  Delay (10-300s)
                </label>
                <input
                  type="number"
                  id="retryDelay"
                  min={10}
                  max={300}
                  value={values.retryDelaySeconds}
                  onChange={(e) =>
                    set(
                      'retryDelaySeconds',
                      Math.max(10, Math.min(300, parseInt(e.target.value) || 30))
                    )
                  }
                  disabled={isLoading || values.retryCount === 0}
                  className={inputClass}
                />
              </div>
            </div>
          </div>

          <div className="flex gap-3 pt-4">
            <button
              type="button"
              onClick={onClose}
              disabled={isLoading}
              className="text-text-primary bg-background border-card-border hover:bg-card flex-1 rounded-lg border px-4 py-2 font-medium transition-colors disabled:opacity-50"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isLoading}
              className="bg-primary hover:bg-primary/90 flex-1 rounded-lg px-4 py-2 font-medium text-white transition-colors disabled:opacity-50"
            >
              {isLoading ? 'Saving...' : isEditMode ? 'Save Changes' : 'Create Webhook'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
