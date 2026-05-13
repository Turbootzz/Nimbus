'use client'

import { useState, useEffect, Suspense } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import Link from 'next/link'
import { ArrowLeftIcon } from '@heroicons/react/24/outline'
import { api } from '@/lib/api'
import { isValidUrl } from '@/lib/utils/url'
import IconSelector from '@/components/IconSelector'
import GroupSelector from '@/components/GroupSelector'
import { Toggle } from '@/components/ui/Toggle'
import type { IconType, Group } from '@/types'
import { useTheme } from '@/contexts/ThemeContext'
import { useGroupMonitoringLock } from '@/hooks/useGroupMonitoringLock'

// Whitelist of valid `from` values to prevent open-redirect via crafted URLs.
// href and label live together so adding a target can't leave the back-link
// pointing one place while the label says another.
const RETURN_TARGETS: Record<string, { href: string; label: string }> = {
  dashboard: { href: '/dashboard', label: 'Dashboard' },
}
const DEFAULT_RETURN = { href: '/services', label: 'Services' }

function NewServiceContent() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const preselectedGroupId = searchParams.get('group')
  const returnTarget = RETURN_TARGETS[searchParams.get('from') ?? ''] ?? DEFAULT_RETURN
  const { enableServiceGrouping } = useTheme()
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')
  const [uploadedFile, setUploadedFile] = useState<File | null>(null)
  const [groups, setGroups] = useState<Group[]>([])
  const [groupsLoading, setGroupsLoading] = useState(true)

  const [formData, setFormData] = useState({
    name: '',
    url: '',
    icon: '🔗',
    icon_type: 'emoji' as IconType,
    icon_image_path: '',
    description: '',
    group_id: '' as string,
    monitoring_enabled: true,
  })

  const { groupMonitoringDisabled, monitoringDescription } = useGroupMonitoringLock({
    groups,
    selectedGroupId: formData.group_id,
    enableServiceGrouping,
    monitoringEnabled: formData.monitoring_enabled,
    setMonitoringEnabled: (next) => setFormData((prev) => ({ ...prev, monitoring_enabled: next })),
  })

  // Fetch groups when grouping is enabled
  useEffect(() => {
    const fetchGroups = async () => {
      if (!enableServiceGrouping) {
        setGroupsLoading(false)
        return
      }

      setGroupsLoading(true)
      try {
        const response = await api.getGroups()
        if (response.error) {
          setError(`Failed to load groups: ${response.error.message}`)
          return
        }
        if (response.data) {
          setGroups(response.data)
          // Use preselected group from URL if valid, otherwise use default group
          const preselectedGroup = preselectedGroupId
            ? response.data.find((g) => g.id === preselectedGroupId)
            : null
          const defaultGroup = response.data.find((g) => g.is_default)
          const groupToSelect = preselectedGroup || defaultGroup
          if (groupToSelect) {
            setFormData((prev) => ({ ...prev, group_id: groupToSelect.id }))
          }
        }
      } catch (error) {
        console.error('Failed to fetch groups:', error)
        setError('Failed to load groups. Please try again.')
      } finally {
        setGroupsLoading(false)
      }
    }

    fetchGroups()
  }, [enableServiceGrouping, preselectedGroupId])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setIsLoading(true)

    // Validation
    if (!formData.name.trim()) {
      setError('Service name is required')
      setIsLoading(false)
      return
    }

    if (!formData.url.trim()) {
      setError('Service URL is required')
      setIsLoading(false)
      return
    }

    if (!isValidUrl(formData.url)) {
      setError('Please enter a valid URL (e.g., https://example.com)')
      setIsLoading(false)
      return
    }

    // Upload image if needed
    let iconImagePath = formData.icon_image_path
    if (formData.icon_type === 'image_upload' && uploadedFile) {
      const uploadResponse = await api.uploadServiceIcon(uploadedFile)
      if (uploadResponse.error) {
        setError(`Image upload failed: ${uploadResponse.error.message}`)
        setIsLoading(false)
        return
      }
      iconImagePath = uploadResponse.data?.icon_image_path || ''
    }

    // Create service
    try {
      const response = await api.createService({
        name: formData.name.trim(),
        url: formData.url.trim(),
        icon: formData.icon.trim() || '🔗',
        icon_type: formData.icon_type,
        icon_image_path: iconImagePath,
        description: formData.description.trim(),
        group_id: enableServiceGrouping && formData.group_id ? formData.group_id : undefined,
        monitoring_enabled: formData.monitoring_enabled,
      })

      if (response.error) {
        setError(response.error.message)
      } else {
        router.push(returnTarget.href)
      }
    } catch (error) {
      console.error('Failed to create service:', error)
      const message =
        error instanceof Error ? error.message : 'Unable to create service. Please try again.'
      setError(message)
    } finally {
      setIsLoading(false)
    }
  }

  const handleChange = (
    e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>
  ) => {
    setFormData({
      ...formData,
      [e.target.name]: e.target.value,
    })
  }

  return (
    <div className="mx-auto max-w-2xl">
      {/* Back button */}
      <Link
        href={returnTarget.href}
        className="text-text-secondary hover:text-text-primary mb-6 inline-flex items-center text-sm transition-colors"
      >
        <ArrowLeftIcon className="mr-2 h-4 w-4" />
        Back to {returnTarget.label}
      </Link>

      {/* Page header */}
      <div className="mb-6">
        <h1 className="text-text-primary text-3xl font-bold">Add New Service</h1>
        <p className="text-text-secondary mt-1">
          Add a service to monitor in your homelab dashboard
        </p>
      </div>

      {/* Error message */}
      {error && (
        <div
          className="mb-4 rounded-lg border p-3 text-sm"
          style={{
            backgroundColor: 'var(--color-error)',
            borderColor: 'var(--color-error)',
            color: 'white',
            opacity: 0.9,
          }}
        >
          {error}
        </div>
      )}

      {/* Form */}
      <form onSubmit={handleSubmit} className="bg-card border-card-border rounded-lg border p-6">
        <div className="space-y-6">
          {/* Service Name */}
          <div>
            <label htmlFor="name" className="text-text-secondary mb-2 block text-sm font-medium">
              Service Name <span className="text-error">*</span>
            </label>
            <input
              type="text"
              id="name"
              name="name"
              value={formData.name}
              onChange={handleChange}
              className="border-card-border focus:border-primary focus:ring-opacity-50 w-full rounded-md border px-4 py-2 transition focus:ring-2 focus:outline-none"
              style={{
                backgroundColor: 'var(--color-background)',
                color: 'var(--color-text-primary)',
              }}
              placeholder="e.g., Plex Media Server"
              required
              disabled={isLoading}
            />
          </div>

          {/* Service URL */}
          <div>
            <label htmlFor="url" className="text-text-secondary mb-2 block text-sm font-medium">
              Service URL <span className="text-error">*</span>
            </label>
            <input
              type="url"
              id="url"
              name="url"
              value={formData.url}
              onChange={handleChange}
              className="border-card-border focus:border-primary focus:ring-opacity-50 w-full rounded-md border px-4 py-2 transition focus:ring-2 focus:outline-none"
              style={{
                backgroundColor: 'var(--color-background)',
                color: 'var(--color-text-primary)',
              }}
              placeholder="https://plex.example.com"
              required
              disabled={isLoading}
            />
            <p className="text-text-muted mt-1 text-xs">
              The URL where your service can be accessed
            </p>
          </div>

          {/* Service Icon */}
          <IconSelector
            icon={formData.icon}
            iconType={formData.icon_type}
            iconImagePath={formData.icon_image_path}
            serviceName={formData.name}
            serviceUrl={formData.url}
            onIconChange={(icon) => setFormData({ ...formData, icon })}
            onIconTypeChange={(icon_type) => {
              setFormData((prev) => ({ ...prev, icon_type }))
              setUploadedFile(null) // Clear uploaded file when switching icon type
            }}
            onIconImagePathChange={(icon_image_path) =>
              setFormData((prev) => ({ ...prev, icon_image_path }))
            }
            onFileSelect={(file) => setUploadedFile(file)}
          />

          {/* Group Selector (only when grouping is enabled) */}
          {enableServiceGrouping && (
            <GroupSelector
              value={formData.group_id}
              onChange={(value) => setFormData((prev) => ({ ...prev, group_id: value }))}
              groups={groups}
              isLoading={groupsLoading}
              disabled={isLoading}
            />
          )}

          {/* Service Description */}
          <div>
            <label
              htmlFor="description"
              className="text-text-secondary mb-2 block text-sm font-medium"
            >
              Description
            </label>
            <textarea
              id="description"
              name="description"
              value={formData.description}
              onChange={handleChange}
              rows={3}
              className="border-card-border focus:border-primary focus:ring-opacity-50 w-full rounded-md border px-4 py-2 transition focus:ring-2 focus:outline-none"
              style={{
                backgroundColor: 'var(--color-background)',
                color: 'var(--color-text-primary)',
              }}
              placeholder="Brief description of what this service does"
              disabled={isLoading}
            />
          </div>

          {/* Monitoring Toggle */}
          <Toggle
            id="monitoring_enabled"
            enabled={formData.monitoring_enabled}
            onChange={(enabled) =>
              setFormData((prev) => ({ ...prev, monitoring_enabled: enabled }))
            }
            label="Enable Monitoring"
            description={monitoringDescription}
            disabled={isLoading || groupMonitoringDisabled}
          />

          {/* Form Actions */}
          <div
            className="flex items-center justify-end gap-3 border-t pt-6"
            style={{ borderColor: 'var(--color-card-border)' }}
          >
            <Link
              href={returnTarget.href}
              className="hover:bg-card-border text-text-secondary hover:text-text-primary rounded-md px-4 py-2 text-sm font-medium transition-colors"
            >
              Cancel
            </Link>
            <button
              type="submit"
              disabled={isLoading}
              className="bg-primary hover:bg-primary-hover rounded-md px-6 py-2 text-sm font-medium text-white transition-colors disabled:cursor-not-allowed disabled:opacity-50"
            >
              {isLoading ? 'Creating...' : 'Create Service'}
            </button>
          </div>
        </div>
      </form>
    </div>
  )
}

export default function NewServicePage() {
  return (
    <Suspense fallback={<div className="text-text-secondary">Loading...</div>}>
      <NewServiceContent />
    </Suspense>
  )
}
