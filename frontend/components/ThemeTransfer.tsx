'use client'

import { useRef, useState } from 'react'
import { useTranslations } from 'next-intl'
import { useTheme } from '@/contexts/ThemeContext'
import { exportTheme, parseThemeFile } from '@/lib/theme-file'
import type { PreferencesUpdateRequest } from '@/types'

// Far above any real theme file (the CSS alone is capped at 20 KB)
const MAX_THEME_FILE_BYTES = 100 * 1024

const buttonClass =
  'border-card-border text-text-primary hover:bg-card-border rounded-lg border px-4 py-2 text-sm transition-colors disabled:opacity-50'

export default function ThemeTransfer() {
  const t = useTranslations('settings.theme.transfer')
  const { theme, accentColor, preset, font, kpiMono, iconSource, glass, customCss, saveTheme } =
    useTheme()
  // The theme of a chosen file, shown before anything is applied
  const [pending, setPending] = useState<PreferencesUpdateRequest | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [applying, setApplying] = useState(false)
  const fileInput = useRef<HTMLInputElement>(null)

  const download = () => {
    const file = exportTheme({
      preset,
      mode: theme,
      accentColor,
      font,
      kpiMono,
      iconSource,
      glass,
      customCss,
    })
    const url = URL.createObjectURL(
      new Blob([JSON.stringify(file, null, 2)], { type: 'application/json' })
    )
    const link = document.createElement('a')
    link.href = url
    link.download = 'nimbus-theme.json'
    link.click()
    URL.revokeObjectURL(url)
  }

  const read = async (file: File | undefined) => {
    if (fileInput.current) fileInput.current.value = ''
    if (!file) return
    setPending(null)
    if (file.size > MAX_THEME_FILE_BYTES) {
      setError(t('tooLarge'))
      return
    }
    const parsed = parseThemeFile(await file.text(), {
      notJson: t('errors.notJson'),
      notTheme: t('errors.notTheme'),
      newerVersion: t('errors.newerVersion'),
      invalidField: (field) => t('errors.invalidField', { field }),
    })
    if ('error' in parsed) {
      setError(parsed.error)
      return
    }
    setError(null)
    setPending(parsed.update)
  }

  const apply = async () => {
    if (!pending) return
    setApplying(true)
    const problem = await saveTheme(pending)
    setApplying(false)
    setError(problem)
    if (!problem) setPending(null)
  }

  const { custom_css: pendingCss, ...pendingSettings } = pending ?? {}
  // The fields a theme file can set, besides the CSS
  const fieldNames: Record<string, string> = {
    theme_preset: t('fields.themePreset'),
    theme_mode: t('fields.themeMode'),
    theme_accent_color: t('fields.themeAccentColor'),
    theme_font: t('fields.themeFont'),
    kpi_mono: t('fields.kpiMono'),
    icon_source: t('fields.iconSource'),
    wallpaper_blur: t('fields.wallpaperBlur'),
    wallpaper_dim: t('fields.wallpaperDim'),
    card_opacity: t('fields.cardOpacity'),
    card_blur: t('fields.cardBlur'),
  }

  return (
    <div className="glass-card border-card-border rounded-lg border p-6">
      <h2 className="text-text-primary mb-2 text-xl font-semibold">{t('title')}</h2>
      <p className="text-text-secondary mb-4 text-sm">{t('description')}</p>

      <div className="flex flex-wrap gap-2">
        <button type="button" onClick={download} className={buttonClass}>
          {t('export')}
        </button>
        <button type="button" onClick={() => fileInput.current?.click()} className={buttonClass}>
          {t('import')}
        </button>
        <input
          ref={fileInput}
          type="file"
          accept="application/json,.json"
          className="hidden"
          aria-label={t('fileLabel')}
          onChange={(e) => read(e.target.files?.[0])}
        />
      </div>

      {pending && (
        <div className="border-card-border mt-4 rounded-lg border p-4">
          <h3 className="text-text-primary mb-2 text-sm font-semibold">{t('willSet')}</h3>
          <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-xs">
            {Object.entries(pendingSettings).map(([name, value]) => (
              <div key={name} className="contents">
                <dt className="text-text-muted">{fieldNames[name] ?? name.replaceAll('_', ' ')}</dt>
                <dd className="text-text-primary break-all">{String(value ?? t('none'))}</dd>
              </div>
            ))}
          </dl>
          {pendingCss !== undefined && (
            <>
              <h4 className="text-text-primary mt-3 mb-1 text-xs font-semibold">
                {pendingCss ? t('cssReview') : t('cssClears')}
              </h4>
              {pendingCss && (
                <pre
                  tabIndex={0}
                  aria-label={t('cssLabel')}
                  className="bg-background border-card-border text-text-primary max-h-48 overflow-auto rounded border p-2 font-mono text-xs whitespace-pre-wrap"
                >
                  {pendingCss}
                </pre>
              )}
            </>
          )}
          <div className="mt-3 flex gap-2">
            <button
              type="button"
              onClick={apply}
              disabled={applying}
              className="bg-primary hover:bg-primary-hover rounded-lg px-4 py-2 text-sm font-medium text-white transition-colors disabled:opacity-50"
            >
              {applying ? t('applying') : t('apply')}
            </button>
            <button type="button" onClick={() => setPending(null)} className={buttonClass}>
              {t('cancel')}
            </button>
          </div>
        </div>
      )}
      {error && (
        <p role="alert" className="text-error mt-2 text-sm">
          {error}
        </p>
      )}
    </div>
  )
}
