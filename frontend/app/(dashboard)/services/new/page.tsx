'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import { ArrowLeftIcon } from '@heroicons/react/24/outline'
import { api } from '@/lib/api'
import IconSelector from '@/components/IconSelector'
import type { IconType } from '@/types'

export default function NewServicePage() {
  const router = useRouter()
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')
  const [uploadedFile, setUploadedFile] = useState<File | null>(null)

  const [formData, setFormData] = useState({
    name: '',
    url: '',
    icon: '🔗',
    icon_type: 'emoji' as IconType,
    icon_image_path: '',
    description: '',
  })

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

    // Basic URL validation
    try {
      new URL(formData.url)
    } catch {
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
      })

      if (response.error) {
        setError(response.error.message)
      } else {
        // Success - redirect to services list
        router.push('/services')
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

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
    setFormData({
      ...formData,
      [e.target.name]: e.target.value,
    })
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
            onIconChange={(icon) => setFormData({ ...formData, icon })}
            onIconTypeChange={(icon_type) => setFormData({ ...formData, icon_type })}
            onIconImagePathChange={(icon_image_path) =>
              setFormData({ ...formData, icon_image_path })
            }
            onFileSelect={(file) => setUploadedFile(file)}
          />

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
