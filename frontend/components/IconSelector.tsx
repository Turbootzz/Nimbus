'use client'

import { useState, useRef, ChangeEvent } from 'react'
import type { IconType } from '@/types'

interface IconSelectorProps {
  icon: string
  iconType: IconType
  iconImagePath: string
  onIconChange: (icon: string) => void
  onIconTypeChange: (iconType: IconType) => void
  onIconImagePathChange: (path: string) => void
  onFileSelect: (file: File) => void
}

export default function IconSelector({
  icon,
  iconType,
  iconImagePath,
  onIconChange,
  onIconTypeChange,
  onIconImagePathChange,
  onFileSelect,
}: IconSelectorProps) {
  const [previewUrl, setPreviewUrl] = useState<string>('')
  const fileInputRef = useRef<HTMLInputElement>(null)

  const handleModeChange = (mode: IconType) => {
    onIconTypeChange(mode)
    setPreviewUrl('')
    if (mode === 'emoji') {
      onIconImagePathChange('')
    }
  }

  const handleFileChange = (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) {
      // Validate file type
      if (!file.type.startsWith('image/')) {
        alert('Please select an image file')
        return
      }
      // Validate file size (2MB)
      if (file.size > 2 * 1024 * 1024) {
        alert('Image must be less than 2MB')
        return
      }
      // Create preview
      const url = URL.createObjectURL(file)
      setPreviewUrl(url)
      onFileSelect(file)
    }
  }

  const handleRemoveImage = () => {
    setPreviewUrl('')
    onIconImagePathChange('')
    if (fileInputRef.current) {
      fileInputRef.current.value = ''
    }
  }

  const handleUrlChange = (e: ChangeEvent<HTMLInputElement>) => {
    const url = e.target.value
    onIconImagePathChange(url)
    setPreviewUrl(url)
  }

  return (
    <div className="space-y-4">
      {/* Mode selector */}
      <div>
        <label className="text-text-secondary mb-2 block text-sm font-medium">Icon Type</label>
        <div className="flex gap-2">
          <button
            type="button"
            onClick={() => handleModeChange('emoji')}
            className={`rounded-md px-4 py-2 text-sm font-medium transition-all ${
              iconType === 'emoji'
                ? 'bg-primary text-white shadow-md'
                : 'bg-background border-card-border text-text-primary hover:border-primary border-2'
            }`}
          >
            Emoji
          </button>
          <button
            type="button"
            onClick={() => handleModeChange('image_upload')}
            className={`rounded-md px-4 py-2 text-sm font-medium transition-all ${
              iconType === 'image_upload'
                ? 'bg-primary text-white shadow-md'
                : 'bg-background border-card-border text-text-primary hover:border-primary border-2'
            }`}
          >
            Upload Image
          </button>
          <button
            type="button"
            onClick={() => handleModeChange('image_url')}
            className={`rounded-md px-4 py-2 text-sm font-medium transition-all ${
              iconType === 'image_url'
                ? 'bg-primary text-white shadow-md'
                : 'bg-background border-card-border text-text-primary hover:border-primary border-2'
            }`}
          >
            Image URL
          </button>
        </div>
      </div>

      {/* Emoji input */}
      {iconType === 'emoji' && (
        <div>
          <label htmlFor="icon" className="text-text-secondary mb-1 block text-sm font-medium">
            Icon (Emoji)
          </label>
          <input
            type="text"
            id="icon"
            name="icon"
            value={icon}
            onChange={(e) => onIconChange(e.target.value)}
            placeholder="📺"
            maxLength={10}
            className="border-card-border bg-background text-text-primary focus:ring-primary w-full rounded-md border px-4 py-2 focus:ring-2 focus:outline-none"
          />
          <p className="text-text-muted mt-1 text-sm">
            Use an emoji to represent your service (default: 🔗)
          </p>
          {icon && (
            <div className="mt-2 flex items-center gap-2">
              <span className="text-text-secondary text-sm">Preview:</span>
              <span className="text-5xl">{icon}</span>
            </div>
          )}
        </div>
      )}

      {/* Upload input */}
      {iconType === 'image_upload' && (
        <div>
          <label
            htmlFor="icon-upload"
            className="text-text-secondary mb-1 block text-sm font-medium"
          >
            Upload Icon Image
          </label>
          <input
            ref={fileInputRef}
            type="file"
            id="icon-upload"
            accept="image/jpeg,image/png,image/gif,image/webp"
            onChange={handleFileChange}
            className="border-card-border bg-background text-text-primary focus:ring-primary file:bg-primary file:hover:bg-primary-hover w-full rounded-md border px-4 py-2 file:mr-4 file:rounded-md file:border-0 file:px-4 file:py-2 file:text-sm file:font-semibold file:text-white focus:ring-2 focus:outline-none"
          />
          <p className="text-text-muted mt-1 text-sm">
            Max size: 2MB. Formats: JPG, PNG, GIF, WEBP
          </p>
          {(previewUrl || iconImagePath) && (
            <div className="mt-3">
              <div className="flex items-center gap-3">
                <span className="text-text-secondary text-sm">Preview:</span>
                <img
                  src={previewUrl || `/api/v1/uploads/service-icons/${iconImagePath}`}
                  alt="Icon preview"
                  className="border-card-border h-16 w-16 rounded border object-contain"
                  onError={(e) => {
                    e.currentTarget.src =
                      'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="64" height="64"%3E%3Ctext x="32" y="32" font-size="32" text-anchor="middle" dy=".3em"%3E❌%3C/text%3E%3C/svg%3E'
                  }}
                />
                <button
                  type="button"
                  onClick={handleRemoveImage}
                  className="hover:bg-error rounded-md border px-3 py-1 text-sm transition-colors hover:text-white"
                  style={{
                    borderColor: 'var(--color-error)',
                    color: 'var(--color-error)',
                  }}
                >
                  Remove
                </button>
              </div>
            </div>
          )}
        </div>
      )}

      {/* URL input */}
      {iconType === 'image_url' && (
        <div>
          <label htmlFor="icon-url" className="text-text-secondary mb-1 block text-sm font-medium">
            Image URL
          </label>
          <input
            type="url"
            id="icon-url"
            name="icon-url"
            value={iconImagePath}
            onChange={handleUrlChange}
            placeholder="https://example.com/icon.png"
            className="border-card-border bg-background text-text-primary focus:ring-primary w-full rounded-md border px-4 py-2 focus:ring-2 focus:outline-none"
          />
          <p className="text-text-muted mt-1 text-sm">
            Enter the URL of an image to use as your service icon
          </p>
          {iconImagePath && (
            <div className="mt-3">
              <div className="flex items-center gap-3">
                <span className="text-text-secondary text-sm">Preview:</span>
                <img
                  src={iconImagePath}
                  alt="Icon preview"
                  className="border-card-border h-16 w-16 rounded border object-contain"
                  onError={(e) => {
                    e.currentTarget.src =
                      'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="64" height="64"%3E%3Ctext x="32" y="32" font-size="32" text-anchor="middle" dy=".3em"%3E❌%3C/text%3E%3C/svg%3E'
                  }}
                />
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
