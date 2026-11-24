'use client'

import { useState, useRef, ChangeEvent } from 'react'
import { useTranslations } from 'next-intl'
import type { IconType } from '@/types'
import { api } from '@/lib/api'
import { getApiUrl } from '@/lib/utils/api-url'
import { isValidUrl } from '@/lib/utils/url'
import EmojiPicker from './EmojiPicker'

interface IconSelectorProps {
  icon: string
  iconType: IconType
  iconImagePath: string
  /** Current value of the service Name field. Primary lookup for the auto-fetch shortcut. */
  serviceName: string
  /** Current value of the service URL field. Used as fallback for the auto-fetch shortcut. */
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
  serviceName,
  serviceUrl,
  onIconChange,
  onIconTypeChange,
  onIconImagePathChange,
  onFileSelect,
}: IconSelectorProps) {
  const t = useTranslations('services.iconSelector')
  const [previewUrl, setPreviewUrl] = useState<string>('')
  const [showEmojiPicker, setShowEmojiPicker] = useState(false)
  const [autoFetching, setAutoFetching] = useState(false)
  const [autoFetchError, setAutoFetchError] = useState('')
  const fileInputRef = useRef<HTMLInputElement>(null)

  const trimmedName = serviceName.trim()
  const hasUsableUrl = isValidUrl(serviceUrl)
  // The backend tries the curated icon library by name first, then falls back
  // to scraping the URL's origin. Either input on its own is enough to attempt.
  const canAutoFetch = trimmedName.length > 0 || hasUsableUrl

  const handleAutoFetchIcon = async () => {
    if (!canAutoFetch || autoFetching) return
    setAutoFetchError('')
    setAutoFetching(true)
    try {
      const response = await api.fetchServiceFavicon({
        name: trimmedName || undefined,
        url: hasUsableUrl ? serviceUrl : undefined,
      })
      if (response.error || !response.data?.icon_image_path) {
        setAutoFetchError(response.error?.message || t('notFound'))
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
      setAutoFetchError(err instanceof Error ? err.message : t('fetchFailed'))
    } finally {
      setAutoFetching(false)
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
        alert(t('notAnImage'))
        return
      }
      // Validate file size (512KB)
      if (file.size > 512 * 1024) {
        alert(t('tooLarge'))
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
      {/* Mode selector + Fetch favicon shortcut */}
      <div>
        <label className="text-text-secondary mb-2 block text-sm font-medium">
          {t('iconType')}
        </label>
        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            onClick={() => handleModeChange('emoji')}
            className={`rounded-md px-4 py-2 text-sm font-medium transition-all ${
              iconType === 'emoji'
                ? 'bg-primary text-white shadow-md'
                : 'bg-background border-card-border text-text-primary hover:border-primary border-2'
            }`}
          >
            {t('emoji')}
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
            {t('uploadImage')}
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
            {t('imageUrl')}
          </button>
          <button
            type="button"
            onClick={handleAutoFetchIcon}
            disabled={!canAutoFetch || autoFetching}
            aria-busy={autoFetching}
            title={canAutoFetch ? t('autoFetchHint') : t('autoFetchNeedsInput')}
            className="bg-primary hover:bg-primary-hover rounded-md px-4 py-2 text-sm font-medium text-white transition-colors disabled:cursor-not-allowed disabled:opacity-50"
          >
            {autoFetching ? t('autoFetching') : t('autoFetch')}
          </button>
        </div>
        {autoFetchError && (
          <p className="text-error mt-2 text-sm" role="alert">
            {autoFetchError}
          </p>
        )}
      </div>

      {/* Emoji input */}
      {iconType === 'emoji' && (
        <div>
          <label htmlFor="icon" className="text-text-secondary mb-1 block text-sm font-medium">
            {t('emojiLabel')}
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
              {t('pickEmoji')}
            </button>
          </div>
          <p className="text-text-muted mt-1 text-sm">{t('emojiHint')}</p>
          {icon && (
            <div className="mt-2 flex items-center gap-2">
              <span className="text-text-secondary text-sm">{t('preview')}</span>
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
            {t('uploadLabel')}
          </label>
          <input
            ref={fileInputRef}
            type="file"
            id="icon-upload"
            accept="image/jpeg,image/png,image/gif,image/webp"
            onChange={handleFileChange}
            className="border-card-border bg-background text-text-primary focus:ring-primary file:bg-primary file:hover:bg-primary-hover w-full rounded-md border px-4 py-2 file:mr-4 file:rounded-md file:border-0 file:px-4 file:py-2 file:text-sm file:font-semibold file:text-white focus:ring-2 focus:outline-none"
          />
          <p className="text-text-muted mt-1 text-sm">{t('uploadHint')}</p>
          {(previewUrl || iconImagePath) && (
            <div className="mt-3">
              <div className="flex items-center gap-3">
                <span className="text-text-secondary text-sm">{t('preview')}</span>
                {/* Using <img> for blob URLs (temporary preview) - Next.js Image doesn't support blob: protocol */}
                {/* eslint-disable-next-line @next/next/no-img-element */}
                <img
                  src={previewUrl || `${getApiUrl()}/uploads/service-icons/${iconImagePath}`}
                  alt={t('previewAlt')}
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
                  {t('remove')}
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
            {t('imageUrl')}
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
          <p className="text-text-muted mt-1 text-sm">{t('urlHint')}</p>
          {iconImagePath && (
            <div className="mt-3">
              <div className="flex items-center gap-3">
                <span className="text-text-secondary text-sm">{t('preview')}</span>
                {/* Using <img> for user-provided URL preview with error fallback */}
                {/* eslint-disable-next-line @next/next/no-img-element */}
                <img
                  src={iconImagePath}
                  alt={t('previewAlt')}
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
