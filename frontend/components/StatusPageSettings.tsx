'use client'

import { useEffect, useRef, useState } from 'react'
import { useTranslations } from 'next-intl'
import { ArrowTopRightOnSquareIcon, ArrowUpTrayIcon } from '@heroicons/react/24/outline'
import { api } from '@/lib/api'
import { Toggle } from '@/components/ui/Toggle'
import { ThemedInput } from '@/components/ui/ThemedInput'
import { StatusPageImage } from '@/components/status/StatusPageIcon'
import type { Group, StatusPageSettings as Settings } from '@/types'

const MAX_LOGO_BYTES = 512 * 1024 // Same limit as service icons
const MAX_TITLE_LENGTH = 80

const labelClass = 'text-text-secondary mb-1 block text-xs font-medium'
const buttonClass =
  'border-card-border text-text-primary hover:bg-card-border inline-flex items-center justify-center gap-1.5 rounded-lg border px-4 py-2 text-sm transition-colors disabled:cursor-not-allowed disabled:opacity-50'

const empty: Settings = { enabled: false, group_id: '', title: '', logo: '', show_links: false }

/** Admin settings of the public status page at /status */
export default function StatusPageSettings() {
  const t = useTranslations('admin.statusPage')
  const [form, setForm] = useState<Settings>(empty)
  const [saved, setSaved] = useState<Settings>(empty)
  const [groups, setGroups] = useState<Group[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [isSaving, setIsSaving] = useState(false)
  const [isUploading, setIsUploading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [message, setMessage] = useState<string | null>(null)
  const fileInput = useRef<HTMLInputElement>(null)

  useEffect(() => {
    let cancelled = false
    Promise.all([api.getStatusPageSettings(), api.getGroups()]).then(([settings, ownGroups]) => {
      if (cancelled) return
      // A form without both would save defaults or drop the stored group
      if (settings.data && !ownGroups.error) {
        setForm(settings.data)
        setSaved(settings.data)
        setGroups(ownGroups.data || [])
      } else {
        setLoadError(settings.error?.message || ownGroups.error?.message || t('loadFailed'))
      }
      setIsLoading(false)
    })
    return () => {
      cancelled = true
    }
  }, [t])

  // The stored group is not selectable when it was deleted, or when another
  // admin chose one of theirs
  const groupMissing = form.group_id !== '' && !groups.some((group) => group.id === form.group_id)

  const update = (changes: Partial<Settings>) => {
    setForm((current) => ({ ...current, ...changes }))
    setMessage(null)
  }

  const uploadLogo = async (file: File | undefined) => {
    if (!file) return
    if (fileInput.current) fileInput.current.value = ''
    if (file.size > MAX_LOGO_BYTES) {
      setError(t('logoTooLarge'))
      return
    }
    setError(null)
    setIsUploading(true)
    const response = await api.uploadServiceIcon(file)
    setIsUploading(false)
    if (response.data) {
      update({ logo: response.data.icon_image_path })
    } else {
      setError(response.error?.message || t('uploadFailed'))
    }
  }

  const save = async () => {
    setError(null)
    setMessage(null)
    setIsSaving(true)
    const response = await api.updateStatusPageSettings({
      ...form,
      group_id: groupMissing ? '' : form.group_id,
    })
    setIsSaving(false)
    if (response.data) {
      setForm(response.data)
      setSaved(response.data)
      setMessage(t('saved'))
    } else {
      setError(response.error?.message || t('saveFailed'))
    }
  }

  if (isLoading) {
    return <p className="text-text-muted text-sm">{t('loading')}</p>
  }
  if (loadError) {
    return (
      <p role="alert" className="text-error text-sm">
        {loadError}
      </p>
    )
  }

  return (
    // Disabled while saving, so no edit is lost when the response comes in
    <fieldset disabled={isSaving} className="min-w-0 space-y-4">
      <Toggle
        id="status-page-enabled"
        enabled={form.enabled}
        onChange={(enabled) => update({ enabled })}
        label={t('enable')}
        description={t('enableDescription')}
      />

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div>
          <label htmlFor="status-page-group" className={labelClass}>
            {t('group')}
          </label>
          <select
            id="status-page-group"
            value={groupMissing ? '' : form.group_id}
            onChange={(e) => update({ group_id: e.target.value })}
            className="border-card-border bg-background text-text-primary focus:ring-primary w-full rounded-lg border px-4 py-2 focus:ring-2 focus:outline-none"
          >
            <option value="">{t('chooseGroup')}</option>
            {groups.map((group) => (
              <option key={group.id} value={group.id}>
                {group.name}
              </option>
            ))}
          </select>
          {groupMissing && <p className="text-warning mt-1 text-xs">{t('groupMissing')}</p>}
        </div>
        <div>
          <label htmlFor="status-page-title" className={labelClass}>
            {t('title')}
          </label>
          <ThemedInput
            id="status-page-title"
            value={form.title}
            maxLength={MAX_TITLE_LENGTH}
            onChange={(e) => update({ title: e.target.value })}
            placeholder="Status"
          />
        </div>
      </div>
      <p className="text-text-muted text-xs">{t('groupsNote')}</p>

      <div>
        <span className={labelClass}>{t('logo')}</span>
        <div className="flex flex-wrap items-center gap-2">
          {form.logo && (
            <StatusPageImage
              file={form.logo}
              alt={t('currentLogo')}
              className="border-card-border h-10 w-10 rounded-lg border object-contain"
            />
          )}
          <input
            ref={fileInput}
            type="file"
            accept="image/jpeg,image/png,image/gif,image/webp"
            className="hidden"
            aria-label={t('logoFile')}
            onChange={(e) => uploadLogo(e.target.files?.[0])}
          />
          <button
            type="button"
            onClick={() => fileInput.current?.click()}
            disabled={isUploading}
            className={buttonClass}
          >
            <ArrowUpTrayIcon className="h-4 w-4" />
            {isUploading ? t('uploading') : form.logo ? t('replace') : t('upload')}
          </button>
          {form.logo && (
            <button type="button" onClick={() => update({ logo: '' })} className={buttonClass}>
              {t('remove')}
            </button>
          )}
        </div>
        <p className="text-text-muted mt-1 text-xs">{t('logoHint')}</p>
      </div>

      <Toggle
        id="status-page-show-links"
        enabled={form.show_links}
        onChange={(show_links) => update({ show_links })}
        label={t('showLinks')}
        description={t('showLinksDescription')}
      />

      {error && (
        <p role="alert" className="text-error text-sm">
          {error}
        </p>
      )}
      {message && (
        <p role="status" className="text-success text-sm">
          {message}
        </p>
      )}

      <div className="flex flex-wrap items-center gap-3">
        <button
          type="button"
          onClick={save}
          disabled={isSaving || isUploading}
          className="bg-primary rounded-lg px-4 py-2 text-sm font-medium text-white transition disabled:cursor-not-allowed disabled:opacity-50"
        >
          {isSaving ? t('saving') : t('save')}
        </button>
        {saved.enabled && (
          <a href="/status" target="_blank" rel="noopener noreferrer" className={buttonClass}>
            {t('open')}
            <ArrowTopRightOnSquareIcon className="h-4 w-4" />
          </a>
        )}
      </div>
    </fieldset>
  )
}
