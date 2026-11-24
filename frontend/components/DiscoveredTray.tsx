'use client'

import { useCallback, useEffect, useRef, useState } from 'react'
import Image from 'next/image'
import Link from 'next/link'
import { ArrowPathIcon } from '@heroicons/react/24/outline'
import { useTranslations } from 'next-intl'
import { api } from '@/lib/api'
import { inputClass, labelClass } from '@/components/widgets/forms/fieldStyles'
import { INTEGRATION_PREFILL_KEY } from '@/types'
import type {
  DiscoveredContainer,
  DiscoveredIntegration,
  DiscoveryResponse,
  Service,
} from '@/types'

interface DiscoveredTrayProps {
  // 'link' is a small pill that leads to the tray on the services page
  variant?: 'tray' | 'link'
  onAdded?: (service: Service) => void
}

type Draft = Pick<DiscoveredContainer, 'key' | 'name' | 'url' | 'group'>

const buttonClass =
  'rounded-md px-3 py-1.5 text-sm font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-50 focus-visible:ring-2 focus-visible:ring-primary focus-visible:outline-none'
const primaryButton = `${buttonClass} bg-primary hover:bg-primary-hover text-white`
const quietButton = `${buttonClass} text-text-secondary hover:text-text-primary hover:bg-card-hover`
const badgeClass = 'rounded-full px-2 py-0.5 text-xs font-medium'

function ContainerIcon({ url }: { url: string }) {
  const [failed, setFailed] = useState(false)
  if (!url || failed) {
    return (
      <span className="flex h-8 w-8 shrink-0 items-center justify-center text-xl" aria-hidden>
        🔗
      </span>
    )
  }
  return (
    <Image
      src={url}
      alt=""
      width={32}
      height={32}
      className="h-8 w-8 shrink-0 object-contain"
      // The address comes from a container label: don't tell that host where Nimbus lives
      referrerPolicy="no-referrer"
      onError={() => setFailed(true)}
      unoptimized
    />
  )
}

/**
 * Containers found through the Docker integrations, to add as services.
 * Admins only; renders nothing for everyone else and without a Docker
 * integration. All label text is rendered as plain text.
 */
export default function DiscoveredTray({ variant = 'tray', onAdded }: DiscoveredTrayProps) {
  const t = useTranslations('dashboard.discovered')
  const [isAdmin, setIsAdmin] = useState(false)
  const [result, setResult] = useState<DiscoveryResponse | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [busyKey, setBusyKey] = useState<string | null>(null)
  const [draft, setDraft] = useState<Draft | null>(null)
  const [showIgnored, setShowIgnored] = useState(false)
  // The integration the last added container asked for with nimbus.widget
  const [suggestion, setSuggestion] = useState<DiscoveredIntegration | null>(null)
  // Bumped by every load and every completed action: an answer that started
  // before either is out of date and dropped
  const generation = useRef(0)

  // refresh asks Docker again; otherwise the server may reuse a recent answer
  const load = useCallback(
    async (refresh = false) => {
      const current = ++generation.current
      setIsLoading(true)
      try {
        const response = await api.getDockerDiscovery(refresh)
        if (current !== generation.current) return
        setError(response.error ? response.error.message : null)
        if (response.data) setResult(response.data)
      } catch (err) {
        if (current !== generation.current) return
        setError(err instanceof Error ? err.message : t('loadFailed'))
      } finally {
        setIsLoading(false)
      }
    },
    [t]
  )

  // Only admins may ask; everyone else never calls the endpoint
  useEffect(() => {
    let cancelled = false
    api
      .getCurrentUser()
      .then((response) => {
        if (cancelled || response.data?.role !== 'admin') return
        setIsAdmin(true)
        load()
      })
      .catch(() => {})
    return () => {
      cancelled = true
    }
  }, [load])

  // Runs one action on a container, then drops or changes it in the list
  const act = async (
    key: string,
    action: () => Promise<{ error?: { message: string } }>,
    update: (containers: DiscoveredContainer[]) => DiscoveredContainer[]
  ): Promise<boolean> => {
    setBusyKey(key)
    setError(null)
    try {
      const response = await action()
      if (response.error) {
        setError(response.error.message)
        return false
      }
      generation.current++
      setResult((prev) => prev && { ...prev, containers: update(prev.containers) })
      return true
    } catch (err) {
      setError(err instanceof Error ? err.message : t('actionFailed'))
      return false
    } finally {
      setBusyKey(null)
    }
  }

  const without = (key: string) => (containers: DiscoveredContainer[]) =>
    containers.filter((c) => c.key !== key)
  const setIgnored = (key: string, ignored: boolean) => (containers: DiscoveredContainer[]) =>
    containers.map((c) => (c.key === key ? { ...c, ignored } : c))

  const add = async (container: DiscoveredContainer, values: Draft) => {
    const added = await act(
      container.key,
      async () => {
        const response = await api.addDiscovered({
          key: container.key,
          name: values.name.trim(),
          url: values.url.trim(),
          group: values.group.trim(),
          icon_url: container.icon_url,
          description: container.description,
        })
        if (response.data) onAdded?.(response.data)
        return response
      },
      without(container.key)
    )
    if (added) {
      // Another container's form may be open by now
      setDraft((current) => (current?.key === container.key ? null : current))
      setSuggestion(container.integration ?? null)
    }
  }

  // The dashboard link leads to #discovered before the tray exists
  const loaded = result !== null
  useEffect(() => {
    if (loaded && variant === 'tray' && window.location.hash === '#discovered') {
      document.getElementById('discovered')?.scrollIntoView?.()
    }
  }, [loaded, variant])

  if (!isAdmin) return null
  if (!result) {
    // Without an answer there is no tray; say why instead of hiding it
    return variant === 'tray' && error ? (
      <p
        role="alert"
        className="glass-card border-card-border text-error mb-6 rounded-lg border p-4 text-sm"
      >
        {t('unavailable', { error })}
      </p>
    ) : null
  }
  if (result.integrations === 0) return null

  const offered = result.containers.filter((c) => !c.ignored)
  const ignored = result.containers.filter((c) => c.ignored)

  if (variant === 'link') {
    if (offered.length === 0) return null
    return (
      <Link
        href="/services#discovered"
        className="glass-card border-card-border text-text-secondary hover:text-text-primary hover:border-primary focus-visible:ring-primary mb-4 inline-flex items-center rounded-full border px-3 py-1 text-sm transition-colors focus-visible:ring-2 focus-visible:outline-none"
      >
        {t('link', { count: offered.length })}
      </Link>
    )
  }

  return (
    <section
      id="discovered"
      aria-labelledby="discovered-title"
      className="glass-card border-card-border mb-6 rounded-lg border p-4"
    >
      <div className="flex items-center justify-between gap-4">
        <div className="min-w-0">
          <h2 id="discovered-title" className="text-text-primary text-lg font-semibold">
            {t('title')}
          </h2>
          <p className="text-text-muted text-sm">{offered.length === 0 ? t('none') : t('intro')}</p>
        </div>
        <button
          onClick={() => load(true)}
          disabled={isLoading}
          className={`${quietButton} inline-flex shrink-0 items-center`}
        >
          <ArrowPathIcon
            className={`mr-1.5 h-4 w-4 ${isLoading ? 'animate-spin' : ''}`}
            aria-hidden
          />
          {t('refresh')}
        </button>
      </div>

      {error && (
        <p role="alert" className="text-error mt-3 text-sm">
          {error}
        </p>
      )}
      {result.errors.map((failed) => (
        <p key={failed.integration_id} role="alert" className="text-error mt-3 text-sm break-words">
          {failed.integration_name}: {failed.error}
        </p>
      ))}

      {suggestion && (
        <p className="text-text-secondary mt-3 text-sm">
          {t.rich('suggestion', {
            name: suggestion.name,
            link: (chunks) => (
              <Link
                href="/settings/integrations"
                onClick={() =>
                  sessionStorage.setItem(INTEGRATION_PREFILL_KEY, JSON.stringify(suggestion))
                }
                className="text-primary hover:text-primary-hover font-medium"
              >
                {chunks}
              </Link>
            ),
          })}
        </p>
      )}

      {offered.length > 0 && (
        <ul className="divide-card-border mt-3 divide-y">
          {offered.map((container) => {
            const busy = busyKey === container.key
            const editing = draft?.key === container.key
            // A guessed or missing address is reviewed before it is added
            const needsReview = container.url_guessed || !container.url
            const edit = () =>
              setDraft({
                key: container.key,
                name: container.name,
                url: container.url,
                group: container.group,
              })
            return (
              <li key={container.key} className="py-3">
                <div className="flex flex-wrap items-center gap-3">
                  <ContainerIcon url={container.icon_url} />
                  <div className="min-w-0 flex-1 basis-48">
                    <p className="text-text-primary flex flex-wrap items-center gap-2 font-medium">
                      <span className="break-all">{container.name}</span>
                      {container.enabled && (
                        <span className={`${badgeClass} bg-primary/10 text-primary`}>
                          {t('labelled')}
                        </span>
                      )}
                      {container.group && (
                        <span
                          className={`${badgeClass} border-card-border text-text-secondary border`}
                        >
                          {container.group}
                        </span>
                      )}
                      {container.state !== 'running' && (
                        <span className={`${badgeClass} border-card-border text-text-muted border`}>
                          {container.state}
                        </span>
                      )}
                    </p>
                    <p className="text-text-secondary text-sm break-all">
                      {container.url || t('noAddress')}
                      {container.url_guessed && (
                        <span className={`${badgeClass} bg-warning/10 text-warning ml-2`}>
                          {t('guess')}
                        </span>
                      )}
                    </p>
                    <p className="text-text-muted text-xs break-all">
                      {container.container} · {container.image} · {container.integration_name}
                    </p>
                    {container.description && (
                      <p className="text-text-muted mt-1 text-xs break-words">
                        {container.description}
                      </p>
                    )}
                  </div>
                  {!editing && (
                    <div className="flex shrink-0 gap-1">
                      {needsReview ? (
                        <button
                          onClick={edit}
                          disabled={busy}
                          className={primaryButton}
                          aria-label={t('reviewLabel', { name: container.name })}
                        >
                          {t('review')}
                        </button>
                      ) : (
                        <>
                          <button
                            onClick={() => add(container, container)}
                            disabled={busy}
                            className={primaryButton}
                            aria-label={t('addLabel', { name: container.name })}
                          >
                            {t('add')}
                          </button>
                          <button
                            onClick={edit}
                            disabled={busy}
                            className={quietButton}
                            aria-label={t('editLabel', { name: container.name })}
                          >
                            {t('edit')}
                          </button>
                        </>
                      )}
                      <button
                        onClick={() =>
                          act(
                            container.key,
                            () => api.ignoreDiscovered(container.key),
                            setIgnored(container.key, true)
                          )
                        }
                        disabled={busy}
                        className={quietButton}
                        aria-label={t('ignoreLabel', { name: container.name })}
                      >
                        {t('ignore')}
                      </button>
                    </div>
                  )}
                </div>

                {editing && draft && (
                  <form
                    className="mt-3 grid gap-3 sm:grid-cols-3"
                    onSubmit={(event) => {
                      event.preventDefault()
                      add(container, draft)
                    }}
                    onKeyDown={(event) => event.key === 'Escape' && setDraft(null)}
                  >
                    <div>
                      <label htmlFor="discovered-name" className={labelClass}>
                        {t('name')}
                      </label>
                      <input
                        id="discovered-name"
                        value={draft.name}
                        onChange={(event) => setDraft({ ...draft, name: event.target.value })}
                        className={inputClass}
                        maxLength={255}
                        required
                        autoFocus
                      />
                    </div>
                    <div>
                      <label htmlFor="discovered-url" className={labelClass}>
                        {t('address')}
                      </label>
                      <input
                        id="discovered-url"
                        type="url"
                        value={draft.url}
                        onChange={(event) => setDraft({ ...draft, url: event.target.value })}
                        className={inputClass}
                        placeholder="http://192.168.1.10:8080"
                        aria-describedby={container.url_guessed ? 'discovered-url-hint' : undefined}
                        required
                      />
                    </div>
                    <div>
                      <label htmlFor="discovered-group" className={labelClass}>
                        {t('group')}
                      </label>
                      <input
                        id="discovered-group"
                        value={draft.group}
                        onChange={(event) => setDraft({ ...draft, group: event.target.value })}
                        className={inputClass}
                        maxLength={35}
                        placeholder={t('groupPlaceholder')}
                      />
                    </div>
                    {container.url_guessed && (
                      <p id="discovered-url-hint" className="text-text-muted text-xs sm:col-span-3">
                        {t('guessHint')}
                      </p>
                    )}
                    <div className="flex gap-1 sm:col-span-3">
                      <button type="submit" disabled={busy} className={primaryButton}>
                        {t('addService')}
                      </button>
                      <button type="button" onClick={() => setDraft(null)} className={quietButton}>
                        {t('cancel')}
                      </button>
                    </div>
                  </form>
                )}
              </li>
            )
          })}
        </ul>
      )}

      {ignored.length > 0 && (
        <div className="mt-3">
          <button
            onClick={() => setShowIgnored(!showIgnored)}
            aria-expanded={showIgnored}
            aria-controls="discovered-ignored"
            className={`${quietButton} -ml-3`}
          >
            {t('ignored', { count: ignored.length })}
          </button>
          {showIgnored && (
            <ul id="discovered-ignored" className="mt-1 space-y-1">
              {ignored.map((container) => (
                <li key={container.key} className="flex items-center justify-between gap-3">
                  <span className="text-text-secondary min-w-0 text-sm break-all">
                    {container.name}
                    <span className="text-text-muted"> · {container.integration_name}</span>
                  </span>
                  <button
                    onClick={() =>
                      act(
                        container.key,
                        () => api.unignoreDiscovered(container.key),
                        setIgnored(container.key, false)
                      )
                    }
                    disabled={busyKey === container.key}
                    className={`${quietButton} shrink-0`}
                    aria-label={t('unignoreLabel', { name: container.name })}
                  >
                    {t('unignore')}
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </section>
  )
}
