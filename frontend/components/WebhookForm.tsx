'use client'

import { useState } from 'react'
import { XMarkIcon } from '@heroicons/react/24/outline'
import type { Webhook, WebhookFormat } from '@/types'

export interface WebhookFormData {
  name: string
  url: string
  format: WebhookFormat
  triggers: {
    on_offline: boolean
    on_online: boolean
  }
  retry_count: number
  retry_delay_seconds: number
}

interface WebhookFormProps {
  webhook?: Webhook | null
  onSubmit: (data: WebhookFormData) => Promise<boolean>
  onClose: () => void
}

const formatOptions: { value: WebhookFormat; label: string; description: string }[] = [
  { value: 'generic', label: 'Generic', description: 'Standard JSON payload' },
  { value: 'discord', label: 'Discord', description: 'Discord-compatible embed format' },
  { value: 'slack', label: 'Slack', description: 'Slack Block Kit format' },
]

export function WebhookForm({ webhook, onSubmit, onClose }: WebhookFormProps) {
  const [name, setName] = useState(webhook?.name || '')
  const [url, setUrl] = useState(webhook?.url || '')
  const [format, setFormat] = useState<WebhookFormat>(webhook?.format || 'generic')
  const [onOffline, setOnOffline] = useState(webhook?.triggers?.on_offline ?? true)
  const [onOnline, setOnOnline] = useState(webhook?.triggers?.on_online ?? false)
  const [retryCount, setRetryCount] = useState(webhook?.retry_count ?? 0)
  const [retryDelaySeconds, setRetryDelaySeconds] = useState(webhook?.retry_delay_seconds ?? 30)
  const [error, setError] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(false)

  const isEditMode = !!webhook

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)

    const trimmedName = name.trim()
    const trimmedUrl = url.trim()

    if (!trimmedName) {
      setError('Name is required')
      return
    }
    if (trimmedName.length > 100) {
      setError('Name must be 100 characters or less')
      return
    }

    if (!trimmedUrl) {
      setError('URL is required')
      return
    }

    try {
      new URL(trimmedUrl)
    } catch {
      setError('Invalid URL format')
      return
    }

    if (!onOffline && !onOnline) {
      setError('Select at least one trigger event')
      return
    }

    setIsLoading(true)

    try {
      const data = {
        name: trimmedName,
        url: trimmedUrl,
        format,
        triggers: {
          on_offline: onOffline,
          on_online: onOnline,
        },
        retry_count: retryCount,
        retry_delay_seconds: retryDelaySeconds,
      }

      const success = await onSubmit(data)
      if (!success) {
        setError('Failed to save webhook')
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

      <div className="bg-card border-card-border relative z-10 w-full max-w-md rounded-lg border p-6 shadow-xl">
        <div className="mb-6 flex items-center justify-between">
          <h2 className="text-text-primary text-xl font-semibold">
            {isEditMode ? 'Edit Webhook' : 'Add Webhook'}
          </h2>
          <button
            onClick={onClose}
            disabled={isLoading}
            className="text-text-muted hover:text-text-primary disabled:opacity-50"
          >
            <XMarkIcon className="h-6 w-6" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          {error && (
            <div className="rounded-lg border border-red-500/20 bg-red-500/10 p-3 text-sm text-red-500">
              {error}
            </div>
          )}

          <div>
            <label htmlFor="name" className="text-text-primary mb-1 block text-sm font-medium">
              Name
            </label>
            <input
              type="text"
              id="name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="My Discord Webhook"
              disabled={isLoading}
              className="bg-background border-card-border text-text-primary placeholder:text-text-muted focus:border-primary w-full rounded-lg border px-4 py-2 focus:outline-none disabled:opacity-50"
            />
          </div>

          <div>
            <label htmlFor="url" className="text-text-primary mb-1 block text-sm font-medium">
              Webhook URL
            </label>
            <input
              type="url"
              id="url"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              placeholder="https://discord.com/api/webhooks/..."
              disabled={isLoading}
              className="bg-background border-card-border text-text-primary placeholder:text-text-muted focus:border-primary w-full rounded-lg border px-4 py-2 focus:outline-none disabled:opacity-50"
            />
          </div>

          <div>
            <label className="text-text-primary mb-2 block text-sm font-medium">Format</label>
            <div className="grid grid-cols-3 gap-2">
              {formatOptions.map((option) => (
                <button
                  key={option.value}
                  type="button"
                  onClick={() => setFormat(option.value)}
                  disabled={isLoading}
                  className={`rounded-lg border px-3 py-2 text-sm transition-colors ${
                    format === option.value
                      ? 'border-primary bg-primary/10 text-primary'
                      : 'border-card-border text-text-secondary hover:border-primary/50'
                  } disabled:opacity-50`}
                >
                  {option.label}
                </button>
              ))}
            </div>
            <p className="text-text-muted mt-1 text-xs">
              {formatOptions.find((o) => o.value === format)?.description}
            </p>
          </div>

          <div>
            <label className="text-text-primary mb-2 block text-sm font-medium">
              Trigger Events
            </label>
            <div className="space-y-2">
              <label className="flex cursor-pointer items-center gap-3">
                <input
                  type="checkbox"
                  checked={onOffline}
                  onChange={(e) => setOnOffline(e.target.checked)}
                  disabled={isLoading}
                  className="text-primary focus:ring-primary h-4 w-4 rounded border-gray-300"
                />
                <span className="text-text-primary text-sm">When service goes offline</span>
              </label>
              <label className="flex cursor-pointer items-center gap-3">
                <input
                  type="checkbox"
                  checked={onOnline}
                  onChange={(e) => setOnOnline(e.target.checked)}
                  disabled={isLoading}
                  className="text-primary focus:ring-primary h-4 w-4 rounded border-gray-300"
                />
                <span className="text-text-primary text-sm">When service comes back online</span>
              </label>
            </div>
          </div>

          {/* Notification Retries */}
          <div>
            <label className="text-text-primary mb-1 block text-sm font-medium">
              Notification Retries
            </label>
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
                  value={retryCount}
                  onChange={(e) =>
                    setRetryCount(Math.max(0, Math.min(5, parseInt(e.target.value) || 0)))
                  }
                  disabled={isLoading}
                  className="bg-background border-card-border text-text-primary focus:border-primary w-full rounded-lg border px-4 py-2 focus:outline-none disabled:opacity-50"
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
                  value={retryDelaySeconds}
                  onChange={(e) =>
                    setRetryDelaySeconds(
                      Math.max(10, Math.min(300, parseInt(e.target.value) || 30))
                    )
                  }
                  disabled={isLoading || retryCount === 0}
                  className="bg-background border-card-border text-text-primary focus:border-primary w-full rounded-lg border px-4 py-2 focus:outline-none disabled:opacity-50"
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
