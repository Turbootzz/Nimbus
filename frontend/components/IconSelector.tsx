'use client'

import { useState, useRef, ChangeEvent } from 'react'
import type { IconType } from '@/types'
import { api } from '@/lib/api'
import { isValidUrl } from '@/lib/utils/url'
import EmojiPicker from './EmojiPicker'

interface IconSelectorProps {
  icon: string
  iconType: IconType
  iconImagePath: string
  /** Current value of the service URL field. Used to enable the "Fetch favicon" shortcut. */
  serviceUrl: string
  onIconChange: (icon: string) => void
  onIconTypeChange: (iconType: IconType) => void
  onIconImagePathChange: (path: string) => void
  onFileSelect: (file: File | null) => void
}

// Helper function to check if a string contains only emojis
function isEmoji(str: string): boolean {
  if (!str) return true // Allow empty string
  // Regex to match emoji characters
  const emojiRegex = /^[\p{Emoji}\p{Emoji_Component}\s]+$/u
  return emojiRegex.test(str)
}

export default function IconSelector({
  icon,
  iconType,
  iconImagePath,
  serviceUrl,
  onIconChange,
  onIconTypeChange,
  onIconImagePathChange,
  onFileSelect,
}: IconSelectorProps) {
  const [previewUrl, setPreviewUrl] = useState<string>('')
  const [showEmojiPicker, setShowEmojiPicker] = useState(false)
  const [faviconFetching, setFaviconFetching] = useState(false)
  const [faviconError, setFaviconError] = useState('')
  const fileInputRef = useRef<HTMLInputElement>(null)

  const canFetchFavicon = isValidUrl(serviceUrl)

  const handleFetchFavicon = async () => {
    if (!canFetchFavicon || faviconFetching) return
    setFaviconError('')
    setFaviconFetching(true)
    try {
      const response = await api.fetchServiceFavicon(serviceUrl)
      if (response.error || !response.data?.icon_image_path) {
        setFaviconError(
          response.error?.message || "Couldn't fetch favicon — try uploading manually"
        )
        return
      }
      // Switch into image_upload mode pointing at the fetched-and-stored file.
      // The server already persisted it, so we clear any pending File so the
      // parent's submit flow doesn't re-upload on top of it.
      onIconTypeChange('image_upload')
      onIconImagePathChange(response.data.icon_image_path)
      onFileSelect(null)
      setPreviewUrl('')
      if (fileInputRef.current) {
        fileInputRef.current.value = ''
      }
    } catch (err) {
      setFaviconError(err instanceof Error ? err.message : 'Failed to fetch favicon')
    } finally {
      setFaviconFetching(false)
    }
  }

  const handleModeChange = (mode: IconType) => {
    // No-op if clicking the already-selected mode
    if (mode === iconType) {
      return
    }

    onIconTypeChange(mode)
    setPreviewUrl('')
    onIconImagePathChange('')
    // Clear file input when switching modes
    if (fileInputRef.current) {
      fileInputRef.current.value = ''
    }
  }

  const handleEmojiInputChange = (e: ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value
    // Only allow emojis
    if (isEmoji(value) || value === '') {
      onIconChange(value)
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
      // Validate file size (512KB)
      if (file.size > 512 * 1024) {
        alert('Image must be less than 512KB')
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
      {/* Fetch favicon shortcut */}
      <div>
        <div className="flex flex-wrap items-center gap-3">
          <button
            type="button"
            onClick={handleFetchFavicon}
            disabled={!canFetchFavicon || faviconFetching}
            aria-busy={faviconFetching}
            title={canFetchFavicon ? 'Fetch the favicon from the service URL' : 'Enter a URL first'}
            className="bg-primary hover:bg-primary-hover rounded-md px-4 py-2 text-sm font-medium text-white transition-colors disabled:cursor-not-allowed disabled:opacity-50"
          >
            {faviconFetching ? 'Fetching favicon...' : 'Fetch favicon from URL'}
          </button>
          {faviconError && (
            <span className="text-error text-sm" role="alert">
              {faviconError}
            </span>
          )}
        </div>
        <p className="text-text-muted mt-1 text-xs">
          Automatically grab the site&apos;s favicon and save it as the service icon.
        </p>
      </div>

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
          <div className="flex gap-2">
            <input
              type="text"
              id="icon"
              name="icon"
              value={icon}
              onChange={handleEmojiInputChange}
              placeholder="📺"
              maxLength={10}
              className="border-card-border bg-background text-text-primary focus:ring-primary flex-1 rounded-md border px-4 py-2 focus:ring-2 focus:outline-none"
            />
            <button
              type="button"
              onClick={() => setShowEmojiPicker(true)}
              className="bg-primary hover:bg-primary-hover rounded-md px-4 py-2 text-sm font-medium whitespace-nowrap text-white transition-colors"
            >
              Pick Emoji
            </button>
          </div>
          <p className="text-text-muted mt-1 text-sm">
            Click &quot;Pick Emoji&quot; or paste an emoji (letters and numbers are not allowed)
          </p>
          {icon && (
            <div className="mt-2 flex items-center gap-2">
              <span className="text-text-secondary text-sm">Preview:</span>
              <span className="text-5xl">{icon}</span>
            </div>
          )}
        </div>
      )}

      {/* Emoji Picker Modal */}
      {showEmojiPicker && (
        <EmojiPicker
          onSelect={(emoji) => onIconChange(emoji)}
          onClose={() => setShowEmojiPicker(false)}
        />
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
            Max size: 512KB. Formats: JPG, PNG, GIF, WEBP
          </p>
          {(previewUrl || iconImagePath) && (
            <div className="mt-3">
              <div className="flex items-center gap-3">
                <span className="text-text-secondary text-sm">Preview:</span>
                {/* Using <img> for blob URLs (temporary preview) - Next.js Image doesn't support blob: protocol */}
                {/* eslint-disable-next-line @next/next/no-img-element */}
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
                {/* Using <img> for user-provided URL preview with error fallback */}
                {/* eslint-disable-next-line @next/next/no-img-element */}
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
