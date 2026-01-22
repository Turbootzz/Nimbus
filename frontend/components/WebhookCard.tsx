'use client'

import type { Webhook } from '@/types'

interface WebhookCardProps {
  webhook: Webhook
  onEdit: () => void
  onDelete: () => void
  onTest: () => void
  onToggle: () => void
}

const formatLabels: Record<string, string> = {
  generic: 'Generic',
  discord: 'Discord',
  slack: 'Slack',
}

export function WebhookCard({ webhook, onEdit, onDelete, onTest, onToggle }: WebhookCardProps) {
  const maskUrl = (url: string) => {
    try {
      const parsed = new URL(url)
      const pathParts = parsed.pathname.split('/')
      if (pathParts.length > 2) {
        const masked = pathParts.slice(0, 3).join('/') + '/...'
        return `${parsed.origin}${masked}`
      }
      return `${parsed.origin}${parsed.pathname.slice(0, 30)}${parsed.pathname.length > 30 ? '...' : ''}`
    } catch {
      return url.slice(0, 40) + (url.length > 40 ? '...' : '')
    }
  }

  const formatDate = (dateStr?: string) => {
    if (!dateStr) return 'Never'
    const date = new Date(dateStr)
    return (
      date.toLocaleDateString() +
      ' ' +
      date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
    )
  }

  return (
    <div className="bg-card border-card-border rounded-lg border p-4">
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0 flex-1">
          <div className="mb-1 flex items-center gap-2">
            <h3 className="text-text-primary truncate font-medium">{webhook.name}</h3>
            <span
              className={`rounded px-2 py-0.5 text-xs font-medium ${
                webhook.enabled ? 'bg-green-500/10 text-green-500' : 'bg-gray-500/10 text-gray-500'
              }`}
            >
              {webhook.enabled ? 'Active' : 'Disabled'}
            </span>
            <span className="bg-primary/10 text-primary rounded px-2 py-0.5 text-xs font-medium">
              {formatLabels[webhook.format] || webhook.format}
            </span>
          </div>

          <p className="text-text-muted mb-2 truncate font-mono text-xs">{maskUrl(webhook.url)}</p>

          <div className="text-text-secondary flex flex-wrap gap-x-4 gap-y-1 text-xs">
            <span>
              Triggers:{' '}
              {[webhook.triggers.on_offline && 'Offline', webhook.triggers.on_online && 'Online']
                .filter(Boolean)
                .join(', ') || 'None'}
            </span>
            <span>Sent: {webhook.total_sent}</span>
            {webhook.total_failed > 0 && (
              <span className="text-red-500">Failed: {webhook.total_failed}</span>
            )}
            <span>Last triggered: {formatDate(webhook.last_triggered_at)}</span>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={onToggle}
            className={`relative h-6 w-11 rounded-full transition-colors ${
              webhook.enabled ? 'bg-primary' : 'bg-gray-300 dark:bg-gray-600'
            }`}
            title={webhook.enabled ? 'Disable webhook' : 'Enable webhook'}
          >
            <span
              className={`absolute top-0.5 h-5 w-5 rounded-full bg-white shadow transition-transform ${
                webhook.enabled ? 'left-[22px]' : 'left-0.5'
              }`}
            />
          </button>
        </div>
      </div>

      <div className="border-card-border mt-4 flex gap-2 border-t pt-4">
        <button
          onClick={onTest}
          className="text-text-secondary hover:text-primary flex items-center gap-1 text-sm transition-colors"
        >
          <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M13 10V3L4 14h7v7l9-11h-7z"
            />
          </svg>
          Test
        </button>
        <button
          onClick={onEdit}
          className="text-text-secondary hover:text-primary flex items-center gap-1 text-sm transition-colors"
        >
          <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
            />
          </svg>
          Edit
        </button>
        <button
          onClick={onDelete}
          className="text-text-secondary flex items-center gap-1 text-sm transition-colors hover:text-red-500"
        >
          <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
            />
          </svg>
          Delete
        </button>
      </div>
    </div>
  )
}
