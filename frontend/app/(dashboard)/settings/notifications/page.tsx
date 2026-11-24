'use client'

import { useState, useEffect, useCallback } from 'react'
import { useTranslations } from 'next-intl'
import { api } from '@/lib/api'
import { WebhookForm, type WebhookFormData } from '@/components/WebhookForm'
import { WebhookCard } from '@/components/WebhookCard'
import type { Webhook } from '@/types'

export default function NotificationsPage() {
  const t = useTranslations('notifications.page')
  const [webhooks, setWebhooks] = useState<Webhook[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showForm, setShowForm] = useState(false)
  const [editingWebhook, setEditingWebhook] = useState<Webhook | null>(null)

  const fetchWebhooks = useCallback(async () => {
    try {
      const response = await api.getWebhooks()
      if (response.data) {
        setWebhooks(response.data)
        setError(null)
      } else if (response.error) {
        setError(response.error.message)
      }
    } catch {
      setError(t('loadFailed'))
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    fetchWebhooks()
  }, [fetchWebhooks])

  // Returns the API error so the form can show it, or null once saved
  const handleSubmit = async (data: WebhookFormData) => {
    const response = editingWebhook
      ? await api.updateWebhook(editingWebhook.id, data)
      : await api.createWebhook(data)
    const saved = response.data
    if (!saved) return response.error ?? { message: t('saveFailed') }

    if (editingWebhook) {
      setWebhooks(webhooks.map((w) => (w.id === saved.id ? saved : w)))
      setEditingWebhook(null)
    } else {
      setWebhooks([...webhooks, saved])
      setShowForm(false)
    }
    return null
  }

  const handleDelete = async (id: string) => {
    if (!confirm(t('confirmDelete'))) return
    const response = await api.deleteWebhook(id)
    if (!response.error) {
      setWebhooks(webhooks.filter((w) => w.id !== id))
    }
  }

  const handleToggle = async (webhook: Webhook) => {
    const response = await api.updateWebhook(webhook.id, { enabled: !webhook.enabled })
    if (response.data) {
      setWebhooks(webhooks.map((w) => (w.id === webhook.id ? response.data! : w)))
    }
  }

  const handleTest = async (id: string) => {
    const response = await api.testWebhook(id)
    if (response.data) {
      if (response.data.success) {
        alert(t('testSent'))
      } else {
        alert(t('testFailed', { error: response.data.error || t('unknownError') }))
      }
    } else if (response.error) {
      alert(t('testFailed', { error: response.error.message }))
    }
  }

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="border-primary h-8 w-8 animate-spin rounded-full border-4 border-t-transparent" />
      </div>
    )
  }

  return (
    <div className="max-w-4xl p-4 sm:p-6">
      <div className="mb-8 flex items-center justify-between">
        <div className="wallpaper-panel">
          <h1 className="text-text-primary mb-2 text-2xl font-bold sm:text-3xl">{t('title')}</h1>
          <p className="text-text-secondary text-sm sm:text-base">{t('description')}</p>
        </div>
        <button
          onClick={() => setShowForm(true)}
          className="bg-primary hover:bg-primary/90 rounded-lg px-4 py-2 text-sm font-medium text-white transition-colors"
        >
          {t('add')}
        </button>
      </div>

      {error && (
        <div className="mb-6 rounded-lg border border-red-500/20 bg-red-500/10 p-4 text-red-500">
          {error}
        </div>
      )}

      <div className="space-y-4">
        {webhooks.length === 0 ? (
          <div className="glass-card border-card-border rounded-lg border p-8 text-center">
            <svg
              className="text-text-muted mx-auto mb-4 h-12 w-12"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={1.5}
                d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"
              />
            </svg>
            <p className="text-text-secondary mb-4">{t('empty')}</p>
            <button onClick={() => setShowForm(true)} className="text-primary hover:underline">
              {t('createFirst')}
            </button>
          </div>
        ) : (
          webhooks.map((webhook) => (
            <WebhookCard
              key={webhook.id}
              webhook={webhook}
              onEdit={() => setEditingWebhook(webhook)}
              onDelete={() => handleDelete(webhook.id)}
              onTest={() => handleTest(webhook.id)}
              onToggle={() => handleToggle(webhook)}
            />
          ))
        )}
      </div>

      {(showForm || editingWebhook) && (
        <WebhookForm
          webhook={editingWebhook}
          onSubmit={handleSubmit}
          onClose={() => {
            setShowForm(false)
            setEditingWebhook(null)
          }}
        />
      )}
    </div>
  )
}
