'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import { useTranslations } from 'next-intl'
import { PlusIcon } from '@heroicons/react/24/outline'
import { api } from '@/lib/api'
import { kindName, useIntegrations } from '@/hooks/useIntegrations'
import { useConfigFile } from '@/hooks/useConfigFile'
import IntegrationForm from '@/components/integrations/IntegrationForm'
import ConfirmDialog from '@/components/ui/ConfirmDialog'
import { INTEGRATION_PREFILL_KEY } from '@/types'
import type { DiscoveredIntegration, Integration } from '@/types'

// readPrefill returns what the Discovered tray left in this tab, if anything
function readPrefill(): DiscoveredIntegration | null {
  if (typeof window === 'undefined') return null
  try {
    const stored = JSON.parse(sessionStorage.getItem(INTEGRATION_PREFILL_KEY) ?? 'null')
    if (typeof stored?.kind !== 'string' || typeof stored?.base_url !== 'string') return null
    // Only a plain http(s) address: never one that carries a user or password
    const address = new URL(stored.base_url)
    if (!/^https?:$/.test(address.protocol) || address.username || address.password) return null
    return { kind: stored.kind, name: String(stored.name ?? ''), base_url: stored.base_url }
  } catch {
    return null
  }
}

function StatusBadge({ integration }: { integration: Integration }) {
  const t = useTranslations('integrations.page.status')
  // An imported integration: the file never carries credentials
  if (integration.auth_type !== 'none' && !integration.has_credentials) {
    return <span className="text-warning text-xs">{t('needsCredentials')}</span>
  }
  if (integration.last_test_ok === null) {
    return <span className="text-text-muted text-xs">{t('notTested')}</span>
  }
  return integration.last_test_ok ? (
    <span className="text-success text-xs">{t('connected')}</span>
  ) : (
    <span className="text-error text-xs" title={integration.last_error ?? undefined}>
      {integration.last_error ? t('failedWith', { error: integration.last_error }) : t('failed')}
    </span>
  )
}

export default function IntegrationsPage() {
  const t = useTranslations('integrations.page')
  // Locked by a config file: integrations can be tested, not changed
  const readOnly = useConfigFile()?.read_only ?? false
  const { integrations, setIntegrations, kinds, isLoading, error } = useIntegrations()
  const [showForm, setShowForm] = useState(false)
  const [editing, setEditing] = useState<Integration | null>(null)
  const [deleting, setDeleting] = useState<Integration | null>(null)
  const [testingId, setTestingId] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)
  // The Discovered tray leaves the integration a container label asked for;
  // the form opens filled in and the credentials are still entered here
  const [linked, setLinked] = useState<DiscoveredIntegration | null>(readPrefill)
  const linkedKind = kinds.find((k) => k.kind === linked?.kind)
  const prefill = linked && linkedKind ? { ...linked, name: linked.name || linkedKind.name } : null
  // Used once: leaving this page without closing the form must not bring it back
  useEffect(() => {
    try {
      sessionStorage.removeItem(INTEGRATION_PREFILL_KEY)
    } catch {
      // sessionStorage is blocked: there was nothing to read either
    }
  }, [])
  const closeForm = () => {
    setShowForm(false)
    setEditing(null)
    setLinked(null)
  }

  const handleSaved = (saved: Integration) => {
    setIntegrations((prev) =>
      prev.some((i) => i.id === saved.id)
        ? prev.map((i) => (i.id === saved.id ? saved : i))
        : [...prev, saved]
    )
    closeForm()
  }

  const handleTest = async (integration: Integration) => {
    setTestingId(integration.id)
    setActionError(null)
    try {
      const response = await api.testSavedIntegration(integration.id)
      if (response.error || !response.data) {
        setActionError(response.error?.message || t('testFailed'))
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
    const target = deleting
    setDeleting(null) // close first, so a double click can't delete twice
    const response = await api.deleteIntegration(target.id)
    if (response.error) {
      setActionError(response.error.message)
    } else {
      setIntegrations((prev) => prev.filter((i) => i.id !== target.id))
    }
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
        <div className="wallpaper-panel">
          <h1 className="text-text-primary mb-2 text-2xl font-bold sm:text-3xl">{t('title')}</h1>
          <p className="text-text-secondary text-sm sm:text-base">
            {t.rich('description', {
              link: (chunks) => (
                <Link href="/services" className="text-primary underline">
                  {chunks}
                </Link>
              ),
            })}
          </p>
        </div>
        {kinds.length > 0 && !readOnly && (
          <button
            onClick={() => {
              setEditing(null)
              setShowForm(true)
            }}
            className="bg-primary hover:bg-primary-hover inline-flex shrink-0 items-center rounded-md px-4 py-2 text-sm font-medium text-white transition-colors"
          >
            <PlusIcon className="mr-2 h-4 w-4" />
            {t('add')}
          </button>
        )}
      </div>

      {(error || actionError) && (
        <div className="mb-6 rounded-lg border border-red-500/20 bg-red-500/10 p-4 text-red-500">
          {actionError || error}
        </div>
      )}

      {integrations.length === 0 ? (
        <div className="glass-card border-card-border rounded-lg border p-8 text-center">
          <p className="text-text-primary mb-1 font-medium">{t('emptyTitle')}</p>
          <p className="text-text-secondary text-sm">
            {kinds.length > 0
              ? t('supportedApps', { apps: kinds.map((k) => k.name).join(', ') })
              : t('noneAvailable')}
          </p>
        </div>
      ) : (
        <ul className="glass-card border-card-border divide-card-border divide-y rounded-lg border">
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
                  {testingId === integration.id ? t('testing') : t('test')}
                </button>
                {!readOnly && (
                  <>
                    <button
                      onClick={() => {
                        setEditing(integration)
                        setShowForm(true)
                      }}
                      className="border-card-border text-text-primary hover:bg-card-hover rounded-md border px-3 py-1.5 text-sm transition-colors"
                    >
                      {t('edit')}
                    </button>
                    <button
                      onClick={() => setDeleting(integration)}
                      className="text-error hover:bg-error/10 rounded-md px-3 py-1.5 text-sm transition-colors"
                    >
                      {t('delete')}
                    </button>
                  </>
                )}
              </div>
            </li>
          ))}
        </ul>
      )}

      {(showForm || prefill) && (
        <IntegrationForm
          kinds={kinds}
          integration={editing}
          prefill={prefill}
          onClose={closeForm}
          onSaved={handleSaved}
        />
      )}

      {deleting && (
        <ConfirmDialog
          title={t('deleteTitle')}
          message={t('deleteMessage', { name: deleting.name })}
          confirmLabel={t('delete')}
          onConfirm={handleDelete}
          onCancel={() => setDeleting(null)}
        />
      )}
    </div>
  )
}
