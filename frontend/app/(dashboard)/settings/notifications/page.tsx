'use client'

import { useState, useEffect, useCallback } from 'react'
import { api } from '@/lib/api'
import { WebhookForm, type WebhookFormData } from '@/components/WebhookForm'
import { WebhookCard } from '@/components/WebhookCard'
import type { Webhook } from '@/types'

export default function NotificationsPage() {
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
      setError('Failed to load webhooks')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchWebhooks()
  }, [fetchWebhooks])

  const handleSubmit = async (data: WebhookFormData) => {
    if (editingWebhook) {
      const response = await api.updateWebhook(editingWebhook.id, data)
      if (response.data) {
        setWebhooks(webhooks.map((w) => (w.id === editingWebhook.id ? response.data! : w)))
        setEditingWebhook(null)
        return true
      }
    } else {
      const response = await api.createWebhook(data)
      if (response.data) {
        setWebhooks([...webhooks, response.data])
        setShowForm(false)
        return true
      }
    }
    return false
  }

  const handleDelete = async (id: string) => {
    if (!confirm('Are you sure you want to delete this webhook?')) return
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
        alert('Test notification sent successfully!')
      } else {
        alert(`Test failed: ${response.data.error || 'Unknown error'}`)
      }
    } else if (response.error) {
      alert(`Test failed: ${response.error.message}`)
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
        <div>
          <h1 className="text-text-primary mb-2 text-2xl font-bold sm:text-3xl">Notifications</h1>
          <p className="text-text-secondary text-sm sm:text-base">
            Configure webhooks to receive alerts when services change status
          </p>
        </div>
        <button
          onClick={() => setShowForm(true)}
          className="bg-primary hover:bg-primary/90 rounded-lg px-4 py-2 text-sm font-medium text-white transition-colors"
        >
          Add Webhook
        </button>
      </div>

      {error && (
        <div className="mb-6 rounded-lg border border-red-500/20 bg-red-500/10 p-4 text-red-500">
          {error}
        </div>
      )}

      <div className="space-y-4">
        {webhooks.length === 0 ? (
          <div className="bg-card border-card-border rounded-lg border p-8 text-center">
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
            <p className="text-text-secondary mb-4">No webhooks configured yet</p>
            <button onClick={() => setShowForm(true)} className="text-primary hover:underline">
              Create your first webhook
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
