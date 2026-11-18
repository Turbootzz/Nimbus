'use client'

import { useId, useRef, useState } from 'react'
import { ArrowUpTrayIcon } from '@heroicons/react/24/outline'
import { api } from '@/lib/api'
import { SummaryTable, issuesOf } from '@/components/backup/DashboardImport'
import { canCreate, defaultTargets, mapGroups, type GroupTargets } from '@/lib/import-wizard'
import type {
  ApiError,
  ConvertResult,
  Group,
  ImportFormat,
  ImportIssue,
  ImportSummary,
} from '@/types'

// The server's limit for a config
const MAX_CONFIG_BYTES = 1024 * 1024

const formats: { value: ImportFormat | 'auto'; label: string; hint: string }[] = [
  { value: 'auto', label: 'Detect automatically', hint: '' },
  { value: 'homepage', label: 'Homepage', hint: 'services.yaml or bookmarks.yaml' },
  { value: 'homarr', label: 'Homarr', hint: 'a board config (JSON) of Homarr 0.x' },
  { value: 'homer', label: 'Homer', hint: 'config.yml' },
  { value: 'dashy', label: 'Dashy', hint: 'conf.yml' },
  { value: 'compose', label: 'docker-compose', hint: 'docker-compose.yml' },
]

const inputClass =
  'border-card-border bg-card text-text-primary focus-visible:ring-primary w-full rounded-lg border px-3 py-2 text-sm focus-visible:ring-2 focus-visible:outline-none'
const cellClass = 'py-2 pr-4 align-top'

// Imports the config of another dashboard: convert it, look at what comes
// out, choose the groups it goes into, import. The import itself is the
// normal one (merge), so nothing here can delete anything.
export default function ImportWizard() {
  const id = useId()
  const fileInput = useRef<HTMLInputElement>(null)
  const [config, setConfig] = useState('')
  const [format, setFormat] = useState<ImportFormat | 'auto'>('auto')
  // A compose file doesn't say where it runs; left empty, it is the host
  // this page is opened on
  const [host, setHost] = useState('')
  const [groups, setGroups] = useState<Group[]>([])
  const [converted, setConverted] = useState<ConvertResult | null>(null)
  const [targets, setTargets] = useState<GroupTargets>({})
  const [preview, setPreview] = useState<ImportSummary | null>(null)
  const [result, setResult] = useState<ImportSummary | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [issues, setIssues] = useState<ImportIssue[]>([])
  const [busy, setBusy] = useState(false)
  // Only the answer to the latest request counts
  const request = useRef(0)

  const fail = (apiError: ApiError | undefined, fallback: string) => {
    setError(apiError?.message ?? fallback)
    setIssues(apiError ? issuesOf(apiError) : [])
  }

  const reset = () => {
    request.current++
    setConverted(null)
    setPreview(null)
    setResult(null)
    setError(null)
    setIssues([])
  }

  const documentFor = (conversion: ConvertResult, chosen: GroupTargets, existing: Group[]) =>
    JSON.stringify(mapGroups(conversion.document, chosen, existing))

  // Asks the server what the import would do, without changing anything
  const loadPreview = async (
    conversion: ConvertResult,
    chosen: GroupTargets,
    existing: Group[]
  ) => {
    const current = ++request.current
    setBusy(true)
    setPreview(null)
    setError(null)
    setIssues([])
    const response = await api.importDashboard(
      documentFor(conversion, chosen, existing),
      'merge',
      true,
      true
    )
    if (current !== request.current) return
    setBusy(false)
    if (response.data) setPreview(response.data)
    else fail(response.error, 'The import could not be checked')
  }

  const convert = async () => {
    reset()
    const current = request.current
    setBusy(true)
    const [conversion, existing] = await Promise.all([
      api.convertConfig(config, format, host.trim() || window.location.hostname),
      api.getGroups(),
    ])
    if (current !== request.current) return
    if (!conversion.data) {
      setBusy(false)
      fail(conversion.error, 'The config could not be converted')
      return
    }
    const existingGroups = existing.data ?? []
    const chosen = defaultTargets(conversion.data.document, existingGroups)
    setGroups(existingGroups)
    setConverted(conversion.data)
    setTargets(chosen)
    await loadPreview(conversion.data, chosen, existingGroups)
  }

  const chooseFile = async (chosen?: File) => {
    if (!chosen) return
    reset()
    if (chosen.size > MAX_CONFIG_BYTES) {
      setError('The file is too large. A config is at most 1 MB.')
      return
    }
    setConfig(await chosen.text())
  }

  const chooseTarget = (ref: string, target: string) => {
    if (!converted) return
    const chosen = { ...targets, [ref]: target }
    setTargets(chosen)
    loadPreview(converted, chosen, groups)
  }

  const apply = async () => {
    if (!converted) return
    setBusy(true)
    setError(null)
    setIssues([])
    const response = await api.importDashboard(
      documentFor(converted, targets, groups),
      'merge',
      false,
      true
    )
    setBusy(false)
    if (!response.data) {
      fail(response.error, 'The import failed')
      return
    }
    setResult(response.data)
    setConverted(null)
    setPreview(null)
    setConfig('')
    if (fileInput.current) fileInput.current.value = ''
  }

  const doc = converted?.document
  const groupName = (ref?: string) => {
    const group = doc?.groups.find((g) => g.ref === ref)
    if (!group?.ref) return ''
    return groups.find((g) => g.id === targets[group.ref!])?.name ?? group.name
  }
  const kindOf = (ref?: string) => doc?.integrations.find((i) => i.ref && i.ref === ref)?.kind ?? ''
  const hint = formats.find((f) => f.value === format)?.hint

  return (
    <section className="glass-card border-card-border mt-6 rounded-lg border p-6">
      <h2 className="text-text-primary mb-1 text-lg font-semibold">
        Import from another dashboard
      </h2>
      <p className="text-text-secondary mb-4 text-sm">
        Switch from Homepage, Homarr, Homer or Dashy, or start from a docker-compose file. Paste the
        config or choose the file. It is added to your dashboard; nothing you have is changed. API
        keys and passwords in the config are not taken over.
      </p>

      <div className="mb-4 grid gap-4 sm:grid-cols-2">
        <div>
          <label
            htmlFor={`${id}-format`}
            className="text-text-primary mb-1 block text-sm font-medium"
          >
            Format
          </label>
          <select
            id={`${id}-format`}
            value={format}
            onChange={(e) => {
              reset()
              setFormat(e.target.value as ImportFormat | 'auto')
            }}
            disabled={busy}
            className={inputClass}
          >
            {formats.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
          {hint && <p className="text-text-muted mt-1 text-xs">Expects {hint}.</p>}
        </div>
        {(format === 'auto' || format === 'compose') && (
          <div>
            <label
              htmlFor={`${id}-host`}
              className="text-text-primary mb-1 block text-sm font-medium"
            >
              Docker host
            </label>
            <input
              id={`${id}-host`}
              type="text"
              value={host}
              onChange={(e) => {
                reset()
                setHost(e.target.value)
              }}
              disabled={busy}
              placeholder="This host"
              autoComplete="off"
              spellCheck={false}
              aria-describedby={`${id}-host-help`}
              className={inputClass}
            />
            <p id={`${id}-host-help`} className="text-text-muted mt-1 text-xs">
              Only for docker-compose: the host name or IP address its published ports are reached
              at. Empty means the host of this page.
            </p>
          </div>
        )}
      </div>

      <label htmlFor={`${id}-config`} className="text-text-primary mb-1 block text-sm font-medium">
        Config
      </label>
      <textarea
        id={`${id}-config`}
        value={config}
        onChange={(e) => {
          reset()
          setConfig(e.target.value)
        }}
        disabled={busy}
        rows={8}
        spellCheck={false}
        placeholder="Paste the YAML or JSON here"
        className={`${inputClass} font-mono`}
      />

      <input
        ref={fileInput}
        type="file"
        accept=".yaml,.yml,.json,text/yaml,application/json"
        className="hidden"
        aria-label="Config file"
        onChange={(e) => chooseFile(e.target.files?.[0])}
      />
      <div className="mt-3 mb-5 flex flex-wrap items-center gap-3">
        <button
          type="button"
          onClick={() => fileInput.current?.click()}
          disabled={busy}
          className="border-card-border text-text-primary hover:bg-card-hover focus-visible:ring-primary inline-flex items-center gap-2 rounded-lg border px-4 py-2 text-sm font-medium transition-colors focus-visible:ring-2 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50"
        >
          <ArrowUpTrayIcon className="h-4 w-4" aria-hidden="true" />
          Choose file
        </button>
        <button
          type="button"
          onClick={convert}
          disabled={busy || config.trim() === ''}
          className="bg-primary hover:bg-primary/90 focus-visible:ring-primary rounded-lg px-4 py-2 text-sm font-medium text-white transition-colors focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50"
        >
          Convert
        </button>
      </div>

      <div aria-live="polite">
        {busy && <p className="text-text-secondary text-sm">Working…</p>}

        {error && (
          <div
            role="alert"
            className="mb-4 rounded-lg border border-red-500/20 bg-red-500/10 p-4 text-sm text-red-500"
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

        {converted && doc && (
          <div className="space-y-6">
            <p className="text-text-secondary text-sm">
              Read as{' '}
              <span className="text-text-primary font-medium">
                {formats.find((f) => f.value === converted.format)?.label}
              </span>
              : {doc.services.length} services, {doc.groups.length} groups,{' '}
              {doc.integrations.length} integrations.
            </p>

            {converted.warnings.length > 0 && (
              <div className="border-warning/30 bg-warning/10 rounded-lg border p-4 text-sm">
                <h3 className="text-text-primary mb-2 font-medium">
                  Not everything could be taken over ({converted.warnings.length})
                </h3>
                <ul className="text-text-secondary max-h-48 list-disc space-y-1 overflow-y-auto pl-5">
                  {converted.warnings.map((warning, index) => (
                    <li key={index} className="break-words">
                      {warning}
                    </li>
                  ))}
                </ul>
              </div>
            )}

            {doc.groups.length > 0 && (
              <div>
                <h3 className="text-text-primary mb-2 text-sm font-medium">Groups</h3>
                <div className="overflow-x-auto">
                  <table className="w-full text-left text-sm">
                    <thead>
                      <tr className="text-text-muted border-card-border border-b">
                        <th scope="col" className="py-2 pr-4 font-medium">
                          Group in the config
                        </th>
                        <th scope="col" className="py-2 font-medium">
                          Goes into
                        </th>
                      </tr>
                    </thead>
                    <tbody>
                      {doc.groups.map((group) => (
                        <tr key={group.ref} className="border-card-border border-b last:border-b-0">
                          <th scope="row" className={`${cellClass} text-text-primary font-normal`}>
                            {group.name}
                          </th>
                          <td className="py-2">
                            <select
                              aria-label={`Group for ${group.name}`}
                              value={targets[group.ref ?? ''] ?? ''}
                              onChange={(e) => chooseTarget(group.ref ?? '', e.target.value)}
                              disabled={busy}
                              className={inputClass}
                            >
                              {canCreate(group, groups) && (
                                <option value="">New group “{group.name}”</option>
                              )}
                              {groups.map((existing) => (
                                <option key={existing.id} value={existing.id}>
                                  {existing.name}
                                  {existing.is_default ? ' (default)' : ''}
                                </option>
                              ))}
                            </select>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}

            <div>
              <h3 className="text-text-primary mb-2 text-sm font-medium">Services</h3>
              <div className="max-h-80 overflow-auto">
                <table className="w-full text-left text-sm">
                  <thead>
                    <tr className="text-text-muted border-card-border border-b">
                      <th scope="col" className="py-2 pr-4 font-medium">
                        Name
                      </th>
                      <th scope="col" className="py-2 pr-4 font-medium">
                        URL
                      </th>
                      <th scope="col" className="py-2 pr-4 font-medium">
                        Group
                      </th>
                      <th scope="col" className="py-2 font-medium">
                        Integration
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {doc.services.map((service, index) => (
                      <tr key={index} className="border-card-border border-b last:border-b-0">
                        <th scope="row" className={`${cellClass} text-text-primary font-normal`}>
                          {service.name}
                        </th>
                        <td className={`${cellClass} text-text-secondary break-all`}>
                          {service.url}
                        </td>
                        <td className={`${cellClass} text-text-secondary`}>
                          {groupName(service.group_ref) || '—'}
                        </td>
                        <td className="text-text-secondary py-2 align-top">
                          {kindOf(service.integration_ref) || '—'}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>

            {doc.integrations.length > 0 && (
              <div>
                <h3 className="text-text-primary mb-2 text-sm font-medium">Integrations</h3>
                <div className="overflow-x-auto">
                  <table className="w-full text-left text-sm">
                    <thead>
                      <tr className="text-text-muted border-card-border border-b">
                        <th scope="col" className="py-2 pr-4 font-medium">
                          Name
                        </th>
                        <th scope="col" className="py-2 pr-4 font-medium">
                          App
                        </th>
                        <th scope="col" className="py-2 font-medium">
                          Address
                        </th>
                      </tr>
                    </thead>
                    <tbody>
                      {doc.integrations.map((integration, index) => (
                        <tr key={index} className="border-card-border border-b last:border-b-0">
                          <th scope="row" className={`${cellClass} text-text-primary font-normal`}>
                            {integration.name}
                          </th>
                          <td className={`${cellClass} text-text-secondary`}>{integration.kind}</td>
                          <td className="text-text-secondary py-2 align-top break-all">
                            {integration.base_url}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
                <p className="text-text-muted mt-2 text-xs">
                  Credentials are not taken over. Enter them under Settings → Integrations after the
                  import.
                </p>
              </div>
            )}

            {preview && (
              <div>
                <SummaryTable summary={preview} />
                <button
                  type="button"
                  onClick={apply}
                  disabled={busy}
                  className="bg-primary hover:bg-primary/90 focus-visible:ring-primary mt-4 rounded-lg px-4 py-2 text-sm font-medium text-white transition-colors focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50"
                >
                  Import
                </button>
              </div>
            )}
          </div>
        )}

        {result && (
          <div className="border-success/30 bg-success/10 rounded-lg border p-4">
            <p className="text-text-primary mb-3 text-sm font-medium">Import finished.</p>
            <SummaryTable summary={result} />
            <a
              href="/dashboard"
              className="text-primary mt-3 inline-block text-sm font-medium hover:underline"
            >
              Open the dashboard
            </a>
          </div>
        )}
      </div>
    </section>
  )
}
