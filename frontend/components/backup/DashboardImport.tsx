'use client'

import { useEffect, useRef, useState } from 'react'
import { useTranslations } from 'next-intl'
import { ArrowUpTrayIcon } from '@heroicons/react/24/outline'
import { api } from '@/lib/api'
import ConfirmDialog from '@/components/ui/ConfirmDialog'
import { useConfigFile } from '@/hooks/useConfigFile'
import { ReadOnlyNote } from '@/components/ConfigFileBanner'
import type { ApiError, ImportIssue, ImportMode, ImportSummary } from '@/types'

// The server's body limit
const MAX_FILE_BYTES = 10 * 1024 * 1024

const modes: ImportMode[] = ['merge', 'replace']

const entities = ['groups', 'services', 'widgets', 'integrations'] as const

export function issuesOf(error: ApiError): ImportIssue[] {
  return Array.isArray(error.details) ? (error.details as ImportIssue[]) : []
}

type ImportT = ReturnType<typeof useTranslations<'backup.import'>>

// What a replace does to the public status page, which shows one group
export function statusPageNote({ status_page: page }: ImportSummary, t: ImportT): string | null {
  if (!page) return null
  return t(page.kept ? 'statusPage.kept' : 'statusPage.offline', { group: page.group })
}

// What an import does (or did), per kind of entity
export function SummaryTable({ summary }: { summary: ImportSummary }) {
  const t = useTranslations('backup.import')
  const replace = summary.mode === 'replace'
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-left text-sm">
        <thead>
          <tr className="text-text-muted border-card-border border-b">
            <th scope="col" className="py-2 pr-4 font-medium">
              {summary.dry_run ? t('summary.willDo') : t('summary.hasDone')}
            </th>
            <th scope="col" className="py-2 pr-4 font-medium">
              {summary.dry_run ? t('summary.add') : t('summary.added')}
            </th>
            {replace && (
              <th scope="col" className="py-2 font-medium">
                {summary.dry_run ? t('summary.delete') : t('summary.deleted')}
              </th>
            )}
          </tr>
        </thead>
        <tbody>
          {entities.map((entity) => (
            <tr key={entity} className="border-card-border border-b last:border-b-0">
              <th scope="row" className="text-text-primary py-2 pr-4 font-normal capitalize">
                {t(`summary.entities.${entity}`)}
              </th>
              <td className="text-text-primary py-2 pr-4">{summary[entity].created}</td>
              {replace && (
                <td className={summary[entity].removed > 0 ? 'text-error py-2' : 'py-2'}>
                  {summary[entity].removed}
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
      <ul className="text-text-secondary mt-3 list-disc space-y-1 pl-5 text-sm">
        <li>
          {summary.preferences ? t('summary.preferencesFromFile') : t('summary.preferencesKept')}
        </li>
        {summary.needs_credentials > 0 && (
          <li>{t('summary.needsCredentials', { count: summary.needs_credentials })}</li>
        )}
        {summary.status_page && <li>{statusPageNote(summary, t)}</li>}
        {summary.needs_secrets > 0 && (
          <li>{t('summary.needsSecrets', { count: summary.needs_secrets })}</li>
        )}
      </ul>
    </div>
  )
}

export default function DashboardImport() {
  const t = useTranslations('backup.import')
  // Locked by a config file: only the preview works
  const readOnly = useConfigFile()?.read_only ?? false
  const fileInput = useRef<HTMLInputElement>(null)
  const [file, setFile] = useState<{ name: string; text: string } | null>(null)
  const [mode, setMode] = useState<ImportMode>('merge')
  const [preview, setPreview] = useState<ImportSummary | null>(null)
  const [result, setResult] = useState<ImportSummary | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [issues, setIssues] = useState<ImportIssue[]>([])
  const [busy, setBusy] = useState(false)
  const [confirming, setConfirming] = useState(false)
  // Only the answer to the latest file read or preview request counts
  const request = useRef(0)
  // The mode a file is previewed in once it is read
  const modeRef = useRef(mode)

  useEffect(() => {
    if (!confirming) return
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setConfirming(false)
    }
    document.addEventListener('keydown', onKeyDown)
    return () => document.removeEventListener('keydown', onKeyDown)
  }, [confirming])

  const fail = (apiError?: ApiError) => {
    setError(apiError?.message ?? t('failed'))
    setIssues(apiError ? issuesOf(apiError) : [])
  }

  // Asks the server what importing text would do, without changing anything
  const loadPreview = async (text: string, nextMode: ImportMode) => {
    const current = ++request.current
    setBusy(true)
    setPreview(null)
    setResult(null)
    setError(null)
    setIssues([])
    const response = await api.importDashboard(text, nextMode, true)
    if (current !== request.current) return
    setBusy(false)
    if (response.data) setPreview(response.data)
    else fail(response.error)
  }

  const chooseFile = async (chosen?: File) => {
    if (!chosen) return
    // So choosing the same file again, after fixing it, is noticed
    if (fileInput.current) fileInput.current.value = ''
    // Whatever the previous file was doing no longer counts
    const current = ++request.current
    setFile(null)
    setPreview(null)
    setResult(null)
    setError(null)
    setIssues([])
    if (chosen.size > MAX_FILE_BYTES) {
      setError(t('tooLarge'))
      return
    }
    setBusy(true)
    let text: string
    try {
      text = await chosen.text()
    } catch {
      if (current !== request.current) return
      setBusy(false)
      setError(t('unreadable'))
      return
    }
    if (current !== request.current) return
    setFile({ name: chosen.name, text })
    await loadPreview(text, modeRef.current)
  }

  const chooseMode = (next: ImportMode) => {
    setMode(next)
    modeRef.current = next
    if (file) loadPreview(file.text, next)
  }

  // A preview of another mode tells nothing about this one
  const ready = preview?.mode === mode ? preview : null

  const apply = async () => {
    if (!file || !ready) return
    setConfirming(false)
    setBusy(true)
    setError(null)
    setIssues([])
    const response = await api.importDashboard(file.text, ready.mode)
    setBusy(false)
    if (!response.data) {
      fail(response.error)
      return
    }
    setResult(response.data)
    setPreview(null)
    setFile(null)
    if (fileInput.current) fileInput.current.value = ''
  }

  const willDelete = ready ? entities.reduce((n, e) => n + ready[e].removed, 0) : 0

  return (
    <section className="glass-card border-card-border rounded-lg border p-6">
      <h2 className="text-text-primary mb-1 text-lg font-semibold">{t('title')}</h2>
      <p className="text-text-secondary mb-4 text-sm">{t('description')}</p>

      <input
        ref={fileInput}
        type="file"
        accept="application/json,.json"
        className="hidden"
        aria-label={t('fileLabel')}
        onChange={(e) => chooseFile(e.target.files?.[0])}
      />
      <div className="mb-5 flex flex-wrap items-center gap-3">
        <button
          type="button"
          onClick={() => fileInput.current?.click()}
          disabled={busy}
          className="border-card-border text-text-primary hover:bg-card-hover focus-visible:ring-primary inline-flex items-center gap-2 rounded-lg border px-4 py-2 text-sm font-medium transition-colors focus-visible:ring-2 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50"
        >
          <ArrowUpTrayIcon className="h-4 w-4" aria-hidden="true" />
          {t('chooseFile')}
        </button>
        <span className="text-text-secondary min-w-0 text-sm break-all">
          {file ? file.name : t('noFile')}
        </span>
      </div>

      <fieldset className="mb-5">
        <legend className="text-text-primary mb-2 text-sm font-medium">{t('howToImport')}</legend>
        <div className="grid gap-3 sm:grid-cols-2">
          {modes.map((option) => (
            <label
              key={option}
              className={`flex cursor-pointer items-start gap-3 rounded-lg border p-3 transition-colors ${
                mode === option ? 'border-primary bg-primary/5' : 'border-card-border'
              }`}
            >
              <input
                type="radio"
                name="import-mode"
                value={option}
                checked={mode === option}
                onChange={() => chooseMode(option)}
                disabled={busy}
                className="accent-primary mt-1"
              />
              <span>
                <span className="text-text-primary block text-sm font-medium">
                  {t(`modes.${option}.label`)}
                </span>
                <span className="text-text-secondary block text-sm">
                  {t(`modes.${option}.description`)}
                </span>
              </span>
            </label>
          ))}
        </div>
      </fieldset>

      <div aria-live="polite">
        {busy && <p className="text-text-secondary text-sm">{t('checking')}</p>}

        {error && (
          <div
            role="alert"
            className="rounded-lg border border-red-500/20 bg-red-500/10 p-4 text-sm text-red-500"
          >
            <p className="font-medium">{error}</p>
            {issues.length > 0 && (
              <ul className="mt-2 list-disc space-y-1 pl-5">
                {issues.map((issue, index) => (
                  <li key={index}>
                    <code className="break-all">{issue.path}</code>: {issue.message}
                  </li>
                ))}
              </ul>
            )}
          </div>
        )}

        {ready && (
          <>
            {mode === 'replace' && (
              <div
                role="alert"
                className="mb-4 rounded-lg border border-red-500/20 bg-red-500/10 p-4 text-sm text-red-500"
              >
                <p className="font-medium">{t('replaceWarning.title')}</p>
                <p>{t('replaceWarning.body')}</p>
              </div>
            )}
            <SummaryTable summary={ready} />
            {ready.custom_css && (
              <div className="mt-4">
                <h3 className="text-text-primary mb-1 text-sm font-medium">
                  {t('customCss.title')}
                </h3>
                <p className="text-text-secondary mb-2 text-sm">{t('customCss.description')}</p>
                <pre
                  tabIndex={0}
                  aria-label={t('customCss.label')}
                  className="bg-background border-card-border text-text-primary max-h-48 overflow-auto rounded border p-2 font-mono text-xs whitespace-pre-wrap"
                >
                  {ready.custom_css}
                </pre>
              </div>
            )}
            <button
              type="button"
              onClick={() => (mode === 'replace' ? setConfirming(true) : apply())}
              disabled={busy || readOnly}
              className={`focus-visible:ring-primary mt-4 rounded-lg px-4 py-2 text-sm font-medium text-white transition-colors focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50 ${
                mode === 'replace' ? 'bg-error hover:bg-error/80' : 'bg-primary hover:bg-primary/90'
              }`}
            >
              {mode === 'replace' ? t('replaceButton') : t('importButton')}
            </button>
            {readOnly && <ReadOnlyNote />}
          </>
        )}

        {result && (
          <div className="border-success/30 bg-success/10 rounded-lg border p-4">
            <p className="text-text-primary mb-3 text-sm font-medium">{t('finished')}</p>
            <SummaryTable summary={result} />
            {/* A full page load, so imported preferences are picked up */}
            <a
              href="/dashboard"
              className="text-primary mt-3 inline-block text-sm font-medium hover:underline"
            >
              {t('openDashboard')}
            </a>
          </div>
        )}
      </div>

      {confirming && ready && (
        <ConfirmDialog
          title={t('confirm.title')}
          message={`${t('confirm.message', { count: willDelete })} ${statusPageNote(ready, t) ?? ''}`.trim()}
          confirmLabel={t('confirm.confirm')}
          onConfirm={apply}
          onCancel={() => setConfirming(false)}
        />
      )}
    </section>
  )
}
