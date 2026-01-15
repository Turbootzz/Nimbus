'use client'

import { useState, useEffect } from 'react'
import { useRouter, useParams } from 'next/navigation'
import Link from 'next/link'
import { ArrowLeftIcon } from '@heroicons/react/24/outline'
import { api } from '@/lib/api'
import IconSelector from '@/components/IconSelector'
import type { IconType, Group } from '@/types'
import { useTheme } from '@/contexts/ThemeContext'

export default function EditServicePage() {
  const router = useRouter()
  const params = useParams()
  const serviceId = params.id as string
  const { enableServiceGrouping } = useTheme()

  const [isLoading, setIsLoading] = useState(true)
  const [isSaving, setIsSaving] = useState(false)
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
  })

  // Fetch service data
  useEffect(() => {
    const fetchService = async () => {
      setIsLoading(true)
      setError('')

      const response = await api.getService(serviceId)

      if (response.error) {
        setError(response.error.message)
      } else if (response.data) {
        const service = response.data
        setFormData({
          name: service.name,
          url: service.url,
          icon: service.icon || '🔗',
          icon_type: service.icon_type || 'emoji',
          icon_image_path: service.icon_image_path || '',
          description: service.description || '',
          group_id: service.group_id || '',
        })
      }

      setIsLoading(false)
    }

    fetchService()
  }, [serviceId])

  // Fetch groups when grouping is enabled
  useEffect(() => {
    const fetchGroups = async () => {
      if (!enableServiceGrouping) {
        setGroupsLoading(false)
        return
      }

      try {
        const response = await api.getGroups()
        if (response.data) {
          setGroups(response.data)
        }
      } catch (error) {
        console.error('Failed to fetch groups:', error)
      } finally {
        setGroupsLoading(false)
      }
    }

    fetchGroups()
  }, [enableServiceGrouping])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setIsSaving(true)

    // Validation
    if (!formData.name.trim()) {
      setError('Service name is required')
      setIsSaving(false)
      return
    }

    if (!formData.url.trim()) {
      setError('Service URL is required')
      setIsSaving(false)
      return
    }

    // Basic URL validation
    try {
      new URL(formData.url)
    } catch {
      setError('Please enter a valid URL (e.g., https://example.com)')
      setIsSaving(false)
      return
    }

    // Upload image if needed
    let iconImagePath = formData.icon_image_path
    if (formData.icon_type === 'image_upload' && uploadedFile) {
      const uploadResponse = await api.uploadServiceIcon(uploadedFile)
      if (uploadResponse.error) {
        setError(`Image upload failed: ${uploadResponse.error.message}`)
        setIsSaving(false)
        return
      }
      iconImagePath = uploadResponse.data?.icon_image_path || ''
    }

    // Update service
    try {
      const response = await api.updateService(serviceId, {
        name: formData.name.trim(),
        url: formData.url.trim(),
        icon: formData.icon.trim() || '🔗',
        icon_type: formData.icon_type,
        icon_image_path: iconImagePath,
        description: formData.description.trim(),
        group_id: enableServiceGrouping ? formData.group_id || null : undefined,
      })

      if (response.error) {
        setError(response.error.message)
      } else {
        // Success - redirect to services list
        router.push('/services')
      }
    } catch (error) {
      console.error('Failed to update service:', error)
      const message =
        error instanceof Error ? error.message : 'Unable to update service. Please try again.'
      setError(message)
    } finally {
      setIsSaving(false)
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

  if (isLoading) {
    return (
      <div className="flex min-h-96 items-center justify-center">
        <div className="text-center">
          <div className="border-primary mx-auto mb-4 h-12 w-12 animate-spin rounded-full border-t-2 border-b-2"></div>
          <p className="text-text-secondary">Loading service...</p>
        </div>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-2xl">
      {/* Back button */}
      <Link
        href="/services"
        className="text-text-secondary hover:text-text-primary mb-6 inline-flex items-center text-sm transition-colors"
      >
        <ArrowLeftIcon className="mr-2 h-4 w-4" />
        Back to Services
      </Link>

      {/* Page header */}
      <div className="mb-6">
        <h1 className="text-text-primary text-3xl font-bold">Edit Service</h1>
        <p className="text-text-secondary mt-1">Update your service information</p>
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
              disabled={isSaving}
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
              disabled={isSaving}
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
            <div>
              <label
                htmlFor="group_id"
                className="text-text-secondary mb-2 block text-sm font-medium"
              >
                Group
              </label>
              <div className="relative">
                <select
                  id="group_id"
                  name="group_id"
                  value={formData.group_id}
                  onChange={handleChange}
                  className="border-card-border focus:border-primary focus:ring-opacity-50 w-full appearance-none rounded-md border px-4 py-2 pr-10 transition focus:ring-2 focus:outline-none"
                  style={{
                    backgroundColor: 'var(--color-background)',
                    color: 'var(--color-text-primary)',
                  }}
                  disabled={isSaving || groupsLoading}
                >
                  {groupsLoading ? (
                    <option value="">Loading groups...</option>
                  ) : groups.length === 0 ? (
                    <option value="">No groups available</option>
                  ) : (
                    <>
                      <option value="">No group</option>
                      {groups.map((group) => (
                        <option key={group.id} value={group.id}>
                          {group.name}
                          {group.is_default ? ' (Default)' : ''}
                        </option>
                      ))}
                    </>
                  )}
                </select>
                {/* Dropdown arrow */}
                <div className="pointer-events-none absolute inset-y-0 right-0 flex items-center pr-3">
                  <svg
                    className="text-text-muted h-4 w-4"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M19 9l-7 7-7-7"
                    />
                  </svg>
                </div>
              </div>
              <p className="text-text-muted mt-1 text-xs">
                Assign this service to a group for organization
              </p>
            </div>
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
              disabled={isSaving}
            />
          </div>

          {/* Form Actions */}
          <div
            className="flex items-center justify-end gap-3 border-t pt-6"
            style={{ borderColor: 'var(--color-card-border)' }}
          >
            <Link
              href="/services"
              className="hover:bg-card-border text-text-secondary hover:text-text-primary rounded-md px-4 py-2 text-sm font-medium transition-colors"
            >
              Cancel
            </Link>
            <button
              type="submit"
              disabled={isSaving}
              className="bg-primary hover:bg-primary-hover rounded-md px-6 py-2 text-sm font-medium text-white transition-colors disabled:cursor-not-allowed disabled:opacity-50"
            >
              {isSaving ? 'Saving...' : 'Save Changes'}
            </button>
          </div>
        </div>
      </form>
    </div>
  )
}
