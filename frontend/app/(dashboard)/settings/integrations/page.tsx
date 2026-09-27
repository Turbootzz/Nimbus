'use client'

import { useState } from 'react'
import Link from 'next/link'
import { PlusIcon } from '@heroicons/react/24/outline'
import { api } from '@/lib/api'
import { kindName, useIntegrations } from '@/hooks/useIntegrations'
import IntegrationForm from '@/components/integrations/IntegrationForm'
import ConfirmDialog from '@/components/ui/ConfirmDialog'
import type { Integration } from '@/types'

function StatusBadge({ integration }: { integration: Integration }) {
  if (integration.last_test_ok === null) {
    return <span className="text-text-muted text-xs">Not tested</span>
  }
  return integration.last_test_ok ? (
    <span className="text-success text-xs">Connected</span>
  ) : (
    <span className="text-error text-xs" title={integration.last_error ?? undefined}>
      Failed{integration.last_error ? `: ${integration.last_error}` : ''}
    </span>
  )
}

export default function IntegrationsPage() {
  const { integrations, setIntegrations, kinds, isLoading, error } = useIntegrations()
  const [showForm, setShowForm] = useState(false)
  const [editing, setEditing] = useState<Integration | null>(null)
  const [deleting, setDeleting] = useState<Integration | null>(null)
  const [testingId, setTestingId] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)

  const handleSaved = (saved: Integration) => {
    setIntegrations((prev) =>
      prev.some((i) => i.id === saved.id)
        ? prev.map((i) => (i.id === saved.id ? saved : i))
        : [...prev, saved]
    )
    setShowForm(false)
    setEditing(null)
  }

  const handleTest = async (integration: Integration) => {
    setTestingId(integration.id)
    setActionError(null)
    try {
      const response = await api.testSavedIntegration(integration.id)
      if (response.error || !response.data) {
        setActionError(response.error?.message || 'Test failed')
        return
      }
      const result = response.data
      setIntegrations((prev) =>
        prev.map((i) =>
          i.id === integration.id
            ? {
                ...i,
                last_test_ok: result.ok,
                last_error: result.error ?? null,
                last_test_at: new Date().toISOString(),
              }
            : i
        )
      )
    } finally {
      setTestingId(null)
    }
  }

  const handleDelete = async () => {
    if (!deleting) return
    const response = await api.deleteIntegration(deleting.id)
    if (response.error) {
      setActionError(response.error.message)
    } else {
      setIntegrations((prev) => prev.filter((i) => i.id !== deleting.id))
    }
    setDeleting(null)
  }

  if (isLoading) {
    return (
      <div className="flex min-h-96 items-center justify-center">
        <div className="border-primary h-8 w-8 animate-spin rounded-full border-4 border-t-transparent" />
      </div>
    )
  }

  return (
    <div className="max-w-4xl p-4 sm:p-6">
      <div className="mb-8 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h1 className="text-text-primary mb-2 text-2xl font-bold sm:text-3xl">Integrations</h1>
          <p className="text-text-secondary text-sm sm:text-base">
            Connect Nimbus to your apps. Link an integration to a service on its{' '}
            <Link href="/services" className="text-primary underline">
              edit page
            </Link>{' '}
            to show live numbers on the tile.
          </p>
        </div>
        {kinds.length > 0 && (
          <button
            onClick={() => {
              setEditing(null)
              setShowForm(true)
            }}
            className="bg-primary hover:bg-primary-hover inline-flex shrink-0 items-center rounded-md px-4 py-2 text-sm font-medium text-white transition-colors"
          >
            <PlusIcon className="mr-2 h-4 w-4" />
            Add Integration
          </button>
        )}
      </div>

      {(error || actionError) && (
        <div className="mb-6 rounded-lg border border-red-500/20 bg-red-500/10 p-4 text-red-500">
          {actionError || error}
        </div>
      )}

      {integrations.length === 0 ? (
        <div className="bg-card border-card-border rounded-lg border p-8 text-center">
          <p className="text-text-primary mb-1 font-medium">No integrations yet</p>
          <p className="text-text-secondary text-sm">
            {kinds.length > 0
              ? `Supported apps: ${kinds.map((k) => k.name).join(', ')}.`
              : 'This server has no integrations available.'}
          </p>
        </div>
      ) : (
        <ul className="bg-card border-card-border divide-card-border divide-y rounded-lg border">
          {integrations.map((integration) => (
            <li
              key={integration.id}
              className="flex flex-col gap-3 p-4 sm:flex-row sm:items-center sm:justify-between"
            >
              <div className="min-w-0">
                <p className="text-text-primary font-medium">
                  {integration.name}{' '}
                  <span className="text-text-muted text-xs font-normal">
                    {kindName(kinds, integration.kind)}
                  </span>
                </p>
                <p className="text-text-secondary truncate text-sm">{integration.base_url}</p>
                <StatusBadge integration={integration} />
              </div>
              <div className="flex shrink-0 gap-2">
                <button
                  onClick={() => handleTest(integration)}
                  disabled={testingId === integration.id}
                  className="border-card-border text-text-primary hover:bg-card-hover rounded-md border px-3 py-1.5 text-sm transition-colors"
                >
                  {testingId === integration.id ? 'Testing...' : 'Test'}
                </button>
                <button
                  onClick={() => {
                    setEditing(integration)
                    setShowForm(true)
                  }}
                  className="border-card-border text-text-primary hover:bg-card-hover rounded-md border px-3 py-1.5 text-sm transition-colors"
                >
                  Edit
                </button>
                <button
                  onClick={() => setDeleting(integration)}
                  className="text-error hover:bg-error/10 rounded-md px-3 py-1.5 text-sm transition-colors"
                >
                  Delete
                </button>
              </div>
            </li>
          ))}
        </ul>
      )}

      {showForm && (
        <IntegrationForm
          kinds={kinds}
          integration={editing}
          onClose={() => {
            setShowForm(false)
            setEditing(null)
          }}
          onSaved={handleSaved}
        />
      )}

      {deleting && (
        <ConfirmDialog
          title="Delete Integration"
          message={`Delete "${deleting.name}"? Services linked to it lose their live numbers.`}
          confirmLabel="Delete"
          onConfirm={handleDelete}
          onCancel={() => setDeleting(null)}
        />
      )}
    </div>
  )
}
