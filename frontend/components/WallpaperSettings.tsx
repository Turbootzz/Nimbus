'use client'

import { useRef, useState } from 'react'
import { useTranslations } from 'next-intl'
import { ArrowUpTrayIcon } from '@heroicons/react/24/outline'
import { useTheme } from '@/contexts/ThemeContext'
import {
  type Glass,
  MAX_WALLPAPER_BYTES,
  MIN_DIM_FOR_GLASS,
  glassVariables,
  isUploadedWallpaper,
  wallpaperUrl,
} from '@/lib/wallpaper'

const sliders: { key: keyof Glass; max: number; unit: string }[] = [
  { key: 'wallpaperBlur', max: 20, unit: 'px' },
  { key: 'wallpaperDim', max: 80, unit: '%' },
  { key: 'cardOpacity', max: 100, unit: '%' },
  { key: 'cardBlur', max: 40, unit: 'px' },
]

const buttonClass =
  'border-card-border text-text-primary hover:bg-card-border rounded-lg border px-4 py-2 text-sm transition-colors disabled:opacity-50'

export default function WallpaperSettings() {
  const t = useTranslations('settings.theme.wallpaper')
  const { background, setBackground, glass, setGlass, uploadWallpaper } = useTheme()
  const uploaded = isUploadedWallpaper(background)
  // The URL field saves when you leave it, not on every key
  const [draft, setDraft] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [isUploading, setIsUploading] = useState(false)
  const fileInput = useRef<HTMLInputElement>(null)

  const imageUrl = wallpaperUrl(background)
  const urlValue = draft ?? (uploaded ? '' : (background ?? ''))
  const needsDim = glass.cardOpacity < 60

  // An emptied field changes nothing; Clear removes the wallpaper
  const saveUrl = () => {
    const url = draft?.trim()
    setDraft(null)
    if (url && url !== background) setBackground(url)
  }

  const upload = async (file: File | undefined) => {
    if (!file) return
    if (file.size > MAX_WALLPAPER_BYTES) {
      setError(t('tooLarge'))
      return
    }
    setError(null)
    setIsUploading(true)
    setError(await uploadWallpaper(file))
    setIsUploading(false)
    if (fileInput.current) fileInput.current.value = ''
  }

  // The preview gets the same variables as the page, scoped to itself
  const previewStyle = glassVariables(glass, imageUrl) as React.CSSProperties

  return (
    <div className="glass-card border-card-border rounded-lg border p-6">
      <h2 className="text-text-primary mb-2 text-xl font-semibold">{t('title')}</h2>
      <p className="text-text-secondary mb-4 text-sm">{t('description')}</p>

      <div className="flex flex-col gap-2 sm:flex-row sm:flex-wrap">
        <input
          type="url"
          aria-label={t('urlLabel')}
          value={urlValue}
          onChange={(e) => setDraft(e.target.value)}
          onBlur={saveUrl}
          onKeyDown={(e) => {
            if (e.key === 'Enter') saveUrl()
          }}
          placeholder={uploaded ? t('uploadedPlaceholder') : 'https://example.com/image.jpg'}
          className="border-card-border bg-background text-text-primary focus:ring-primary min-w-0 flex-1 rounded-lg border px-3 py-2 text-sm focus:ring-2 focus:outline-none sm:min-w-50"
        />
        <input
          ref={fileInput}
          type="file"
          accept="image/jpeg,image/png,image/gif,image/webp"
          className="hidden"
          aria-label={t('fileLabel')}
          onChange={(e) => upload(e.target.files?.[0])}
        />
        <button
          type="button"
          onClick={() => fileInput.current?.click()}
          disabled={isUploading}
          className={`${buttonClass} inline-flex items-center justify-center gap-1.5`}
        >
          <ArrowUpTrayIcon className="h-4 w-4" />
          {isUploading ? t('uploading') : t('upload')}
        </button>
        {background && (
          <button
            type="button"
            onClick={() => {
              setDraft(null)
              setBackground(undefined)
            }}
            className={buttonClass}
          >
            {t('clear')}
          </button>
        )}
      </div>
      {error && (
        <p role="alert" className="text-error mt-2 text-sm">
          {error}
        </p>
      )}

      <div
        data-testid="wallpaper-preview"
        className="border-card-border bg-background relative isolate mt-4 flex h-48 items-center justify-center overflow-hidden rounded-lg border"
        style={previewStyle}
      >
        <div
          className="absolute inset-0 -z-10"
          style={{
            background:
              'linear-gradient(rgb(0 0 0 / var(--wallpaper-dim)), rgb(0 0 0 / var(--wallpaper-dim))), var(--wallpaper-image) center / cover no-repeat',
            filter: 'blur(var(--wallpaper-blur))',
            transform: 'scale(1.1)',
          }}
        />
        <div className="glass-card border-card-border w-40 rounded-lg border p-3">
          <p className="text-text-primary text-sm font-semibold">{t('previewCard')}</p>
          <p className="text-text-secondary text-xs">{t('previewText')}</p>
        </div>
      </div>

      <div className="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
        {sliders.map((slider) => {
          const min = slider.key === 'wallpaperDim' && needsDim ? MIN_DIM_FOR_GLASS : 0
          return (
            <label key={slider.key} className="block">
              <span className="text-text-primary flex justify-between text-sm font-medium">
                {t(`sliders.${slider.key}`)}
                <span className="text-text-muted tabular-nums">
                  {glass[slider.key]}
                  {slider.unit}
                </span>
              </span>
              <input
                type="range"
                min={min}
                max={slider.max}
                value={glass[slider.key]}
                onChange={(e) => setGlass({ [slider.key]: Number(e.target.value) })}
                className="accent-primary mt-1 w-full"
              />
            </label>
          )
        })}
      </div>
      {needsDim && (
        <p className="text-text-muted mt-2 text-xs">{t('dimNote', { min: MIN_DIM_FOR_GLASS })}</p>
      )}
    </div>
  )
}
