'use client'

import { useEffect, useRef, useState } from 'react'
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
  const [form, setForm] = useState<Settings>(empty)
  const [saved, setSaved] = useState<Settings>(empty)
  const [groups, setGroups] = useState<Group[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [isSaving, setIsSaving] = useState(false)
  const [isUploading, setIsUploading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [message, setMessage] = useState<string | null>(null)
  const fileInput = useRef<HTMLInputElement>(null)

  useEffect(() => {
    let cancelled = false
    Promise.all([api.getStatusPageSettings(), api.getGroups()]).then(([settings, ownGroups]) => {
      if (cancelled) return
      if (settings.data) {
        setForm(settings.data)
        setSaved(settings.data)
      } else {
        setError(settings.error?.message || 'Failed to load the status page settings')
      }
      setGroups(ownGroups.data || [])
      setIsLoading(false)
    })
    return () => {
      cancelled = true
    }
  }, [])

  const update = (changes: Partial<Settings>) => {
    setForm((current) => ({ ...current, ...changes }))
    setMessage(null)
  }

  const uploadLogo = async (file: File | undefined) => {
    if (!file) return
    if (fileInput.current) fileInput.current.value = ''
    if (file.size > MAX_LOGO_BYTES) {
      setError('The logo is larger than 512 KB')
      return
    }
    setError(null)
    setIsUploading(true)
    const response = await api.uploadServiceIcon(file)
    setIsUploading(false)
    if (response.data) {
      update({ logo: response.data.icon_image_path })
    } else {
      setError(response.error?.message || 'Failed to upload the logo')
    }
  }

  const save = async () => {
    setError(null)
    setMessage(null)
    setIsSaving(true)
    const response = await api.updateStatusPageSettings(form)
    setIsSaving(false)
    if (response.data) {
      setForm(response.data)
      setSaved(response.data)
      setMessage('Status page saved')
    } else {
      setError(response.error?.message || 'Failed to save the status page')
    }
  }

  if (isLoading) {
    return <p className="text-text-muted text-sm">Loading...</p>
  }

  // The stored group is gone when it was deleted after it was chosen
  const groupMissing = form.group_id !== '' && !groups.some((group) => group.id === form.group_id)

  return (
    <div className="space-y-4">
      <Toggle
        id="status-page-enabled"
        enabled={form.enabled}
        onChange={(enabled) => update({ enabled })}
        label="Enable the status page"
        description="Anyone who can reach this Nimbus can open /status without signing in. While off, the page does not exist."
      />

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div>
          <label htmlFor="status-page-group" className={labelClass}>
            Group to show
          </label>
          <select
            id="status-page-group"
            value={groupMissing ? '' : form.group_id}
            onChange={(e) => update({ group_id: e.target.value })}
            className="border-card-border bg-background text-text-primary focus:ring-primary w-full rounded-lg border px-4 py-2 focus:ring-2 focus:outline-none"
          >
            <option value="">Choose a group</option>
            {groups.map((group) => (
              <option key={group.id} value={group.id}>
                {group.name}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label htmlFor="status-page-title" className={labelClass}>
            Title
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
      <p className="text-text-muted text-xs">
        Groups are personal, so the page shows one of your own groups. Only its monitored services
        appear: name, icon, status and uptime. Nothing else about them is shared.
      </p>

      <div>
        <span className={labelClass}>Logo</span>
        <div className="flex flex-wrap items-center gap-2">
          {form.logo && (
            <StatusPageImage
              file={form.logo}
              alt="Current logo"
              className="border-card-border h-10 w-10 rounded-lg border object-contain"
            />
          )}
          <input
            ref={fileInput}
            type="file"
            accept="image/jpeg,image/png,image/gif,image/webp"
            className="hidden"
            aria-label="Logo file"
            onChange={(e) => uploadLogo(e.target.files?.[0])}
          />
          <button
            type="button"
            onClick={() => fileInput.current?.click()}
            disabled={isUploading}
            className={buttonClass}
          >
            <ArrowUpTrayIcon className="h-4 w-4" />
            {isUploading ? 'Uploading...' : form.logo ? 'Replace' : 'Upload'}
          </button>
          {form.logo && (
            <button type="button" onClick={() => update({ logo: '' })} className={buttonClass}>
              Remove
            </button>
          )}
        </div>
        <p className="text-text-muted mt-1 text-xs">JPG, PNG, GIF or WEBP, up to 512 KB</p>
      </div>

      <Toggle
        id="status-page-show-links"
        enabled={form.show_links}
        onChange={(show_links) => update({ show_links })}
        label="Show links"
        description="Makes each service name a link to its URL, and shows icons loaded from a URL. Leave off to keep your addresses private."
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
          {isSaving ? 'Saving...' : 'Save status page'}
        </button>
        {saved.enabled && (
          <a href="/status" target="_blank" rel="noopener noreferrer" className={buttonClass}>
            Open status page
            <ArrowTopRightOnSquareIcon className="h-4 w-4" />
          </a>
        )}
      </div>
    </div>
  )
}
