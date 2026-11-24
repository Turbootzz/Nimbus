'use client'

import { useTranslations } from 'next-intl'
import { useTheme } from '@/contexts/ThemeContext'
import { Toggle } from '@/components/ui/Toggle'
import WallpaperSettings from '@/components/WallpaperSettings'
import LayoutSettings from '@/components/LayoutSettings'
import StatusStripSettings from '@/components/StatusStripSettings'
import ThemeGallery from '@/components/ThemeGallery'
import FontSettings from '@/components/FontSettings'
import IconSourceSettings from '@/components/IconSourceSettings'
import CustomCssSettings from '@/components/CustomCssSettings'
import ThemeTransfer from '@/components/ThemeTransfer'
import { useLayoutMode } from '@/lib/layout-store'
import { resolveGroupDisplay } from '@/lib/tiles'

export default function ThemePage() {
  const t = useTranslations('settings.theme')
  const {
    theme,
    effectiveTheme,
    accentColor,
    openInNewTab,
    enableCardResizing,
    enableServiceGrouping,
    cardScale,
    viewMode,
    groupDisplay,
    setTheme,
    setAccentColor,
    setOpenInNewTab,
    setEnableCardResizing,
    setEnableServiceGrouping,
    setCardScale,
    setViewMode,
    setGroupDisplay,
  } = useTheme()
  // Without a choice, groups follow the layout
  const shownGroupDisplay = resolveGroupDisplay(groupDisplay, useLayoutMode())

  const presetColors = [
    { name: t('accent.colors.skyBlue'), value: '#0ea5e9' },
    { name: t('accent.colors.purple'), value: '#8B5CF6' },
    { name: t('accent.colors.pink'), value: '#EC4899' },
    { name: t('accent.colors.red'), value: '#EF4444' },
    { name: t('accent.colors.orange'), value: '#F97316' },
    { name: t('accent.colors.yellow'), value: '#EAB308' },
    { name: t('accent.colors.green'), value: '#10B981' },
    { name: t('accent.colors.teal'), value: '#14B8A6' },
  ]

  return (
    <div className="max-w-4xl p-4 sm:p-6 xl:max-w-none">
      <div className="wallpaper-panel mb-8">
        <h1 className="text-text-primary mb-2 text-2xl font-bold sm:text-3xl">{t('page.title')}</h1>
        <p className="text-text-secondary mb-2 text-sm sm:text-base">{t('page.subtitle')}</p>
        <p className="text-text-muted text-xs sm:text-sm">{t('page.autosave')}</p>
      </div>

      {/* Two-column layout on XL screens */}
      <div className="grid grid-cols-1 gap-6 xl:grid-cols-2">
        {/* Column 1: Appearance settings */}
        <div className="space-y-6">
          {/* Theme Mode */}
          <div className="glass-card border-card-border rounded-lg border p-6">
            <h2 className="text-text-primary mb-2 text-xl font-semibold">{t('mode.title')}</h2>
            <p className="text-text-secondary mb-4 text-sm">{t('mode.description')}</p>

            {/* Automatic Theme Toggle */}
            <div className="mb-4">
              <Toggle
                enabled={theme === 'auto'}
                onChange={(enabled) => setTheme(enabled ? 'auto' : effectiveTheme)}
                label={t('mode.auto')}
                description={
                  theme === 'auto'
                    ? t('mode.autoDescriptionCurrent', {
                        mode: t(`mode.current.${effectiveTheme}`),
                      })
                    : t('mode.autoDescription')
                }
              />
            </div>

            {/* Manual Theme Selection (disabled when auto is on) */}
            <div className="flex gap-4">
              <button
                onClick={() => setTheme('light')}
                disabled={theme === 'auto'}
                className={`flex flex-1 items-center justify-center gap-2 rounded-lg px-6 py-3 font-medium transition-all ${
                  theme === 'light'
                    ? 'bg-primary text-white shadow-md'
                    : theme === 'auto'
                      ? 'bg-background border-card-border text-text-muted cursor-not-allowed border-2 opacity-50'
                      : 'bg-background border-card-border text-text-primary hover:border-primary border-2'
                }`}
              >
                <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z"
                  />
                </svg>
                {t('mode.light')}
              </button>
              <button
                onClick={() => setTheme('dark')}
                disabled={theme === 'auto'}
                className={`flex flex-1 items-center justify-center gap-2 rounded-lg px-6 py-3 font-medium transition-all ${
                  theme === 'dark'
                    ? 'bg-primary text-white shadow-md'
                    : theme === 'auto'
                      ? 'bg-background border-card-border text-text-muted cursor-not-allowed border-2 opacity-50'
                      : 'bg-background border-card-border text-text-primary hover:border-primary border-2'
                }`}
              >
                <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z"
                  />
                </svg>
                {t('mode.dark')}
              </button>
            </div>
          </div>

          <ThemeGallery />

          {/* Accent Color */}
          <div className="glass-card border-card-border rounded-lg border p-6">
            <h2 className="text-text-primary mb-2 text-xl font-semibold">{t('accent.title')}</h2>
            <p className="text-text-secondary mb-4 text-sm">{t('accent.description')}</p>

            <div className="mb-4 flex flex-wrap gap-2">
              {presetColors.map((color) => (
                <button
                  key={color.value}
                  onClick={() => setAccentColor(color.value)}
                  aria-label={color.name}
                  aria-pressed={accentColor === color.value}
                  className={`h-12 w-12 rounded-lg border-2 transition-all hover:scale-110 sm:h-16 sm:w-16 ${
                    accentColor === color.value
                      ? 'border-text-primary ring-primary scale-110 ring-2 ring-offset-2'
                      : 'border-card-border'
                  }`}
                  style={{ backgroundColor: color.value }}
                  title={color.name}
                />
              ))}
            </div>

            <div className="flex flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-center sm:gap-2">
              <label className="text-text-primary text-sm font-medium">{t('accent.custom')}</label>
              <div className="flex flex-1 items-center gap-2">
                <input
                  type="color"
                  value={accentColor || '#0ea5e9'}
                  onChange={(e) => setAccentColor(e.target.value)}
                  className="border-card-border h-10 w-16 cursor-pointer rounded border"
                />
                <input
                  type="text"
                  value={accentColor || ''}
                  onChange={(e) => setAccentColor(e.target.value)}
                  placeholder="#0ea5e9"
                  className="border-card-border bg-background text-text-primary focus:ring-primary min-w-0 flex-1 rounded-lg border px-3 py-2 text-sm focus:ring-2 focus:outline-none sm:min-w-36"
                  maxLength={7}
                />
                {accentColor && (
                  <button
                    onClick={() => setAccentColor(undefined)}
                    className="text-text-secondary hover:text-text-primary px-3 py-2 text-sm whitespace-nowrap transition-colors sm:px-4"
                  >
                    {t('accent.reset')}
                  </button>
                )}
              </div>
            </div>
          </div>

          <FontSettings />

          <LayoutSettings />

          <StatusStripSettings />

          <WallpaperSettings />
        </div>

        {/* Column 2: Behavior & Layout settings */}
        <div className="space-y-6">
          {/* Link Behavior */}
          <div className="glass-card border-card-border rounded-lg border p-6">
            <h2 className="text-text-primary mb-2 text-xl font-semibold">{t('links.title')}</h2>
            <p className="text-text-secondary mb-4 text-sm">{t('links.description')}</p>

            <Toggle
              enabled={openInNewTab}
              onChange={setOpenInNewTab}
              label={t('links.newTab')}
              description={t('links.newTabDescription')}
            />
          </div>

          {/* Card Resizing */}
          <div className="glass-card border-card-border rounded-lg border p-6">
            <h2 className="text-text-primary mb-2 text-xl font-semibold">{t('resizing.title')}</h2>
            <p className="text-text-secondary mb-4 text-sm">{t('resizing.description')}</p>

            <Toggle
              enabled={enableCardResizing}
              onChange={setEnableCardResizing}
              label={t('resizing.toggle')}
              description={t('resizing.toggleDescription')}
            />
          </div>

          {/* Service Grouping */}
          <div className="glass-card border-card-border rounded-lg border p-6">
            <h2 className="text-text-primary mb-2 text-xl font-semibold">{t('grouping.title')}</h2>
            <p className="text-text-secondary mb-4 text-sm">{t('grouping.description')}</p>

            <Toggle
              enabled={enableServiceGrouping}
              onChange={setEnableServiceGrouping}
              label={t('grouping.toggle')}
              description={t('grouping.toggleDescription')}
            />

            {/* Tabs or sections - only matters with grouping on */}
            <div className={`mt-6 ${enableServiceGrouping ? '' : 'opacity-50'}`}>
              <p id="group-display-label" className="text-text-primary mb-2 text-sm font-medium">
                {t('grouping.showAs')}
              </p>
              <div role="group" aria-labelledby="group-display-label" className="flex gap-2">
                {(['tabs', 'sections'] as const).map((display) => (
                  <button
                    key={display}
                    type="button"
                    onClick={() => setGroupDisplay(display)}
                    disabled={!enableServiceGrouping}
                    aria-pressed={shownGroupDisplay === display}
                    className={`flex-1 rounded-lg px-4 py-2 text-sm font-medium capitalize transition-all ${
                      shownGroupDisplay === display
                        ? 'bg-primary text-white'
                        : 'bg-background border-card-border text-text-primary hover:border-primary border'
                    } ${enableServiceGrouping ? '' : 'cursor-not-allowed'}`}
                  >
                    {t(`grouping.display.${display}`)}
                  </button>
                ))}
              </div>
              <p className="text-text-muted mt-2 text-xs">
                {shownGroupDisplay === 'tabs' ? t('grouping.tabsHint') : t('grouping.sectionsHint')}
              </p>
            </div>
          </div>

          {/* Card Display */}
          <div className="glass-card border-card-border rounded-lg border p-6">
            <h2 className="text-text-primary mb-2 text-xl font-semibold">
              {t('cardDisplay.title')}
            </h2>
            <p className="text-text-secondary mb-4 text-sm">{t('cardDisplay.description')}</p>

            {/* View Mode */}
            <div className="mb-6">
              <Toggle
                enabled={viewMode === 'list'}
                onChange={(enabled) => setViewMode(enabled ? 'list' : 'grid')}
                label={t('cardDisplay.list')}
                description={t('cardDisplay.listDescription')}
              />
            </div>

            {/* Card Scale - disabled when list view is enabled */}
            <div className={viewMode === 'list' ? 'opacity-50' : ''}>
              <label className="text-text-primary mb-2 block text-sm font-medium">
                {t('cardDisplay.size')}
              </label>
              <div className="flex gap-2">
                {(['small', 'medium', 'large'] as const).map((scale) => (
                  <button
                    key={scale}
                    onClick={() => setCardScale(scale)}
                    disabled={viewMode === 'list'}
                    className={`flex-1 rounded-lg px-4 py-2 text-sm font-medium capitalize transition-all ${
                      cardScale === scale
                        ? 'bg-primary text-white'
                        : 'bg-background border-card-border text-text-primary hover:border-primary border'
                    } ${viewMode === 'list' ? 'cursor-not-allowed' : ''}`}
                  >
                    {t(`cardDisplay.scales.${scale}`)}
                  </button>
                ))}
              </div>
              <p className="text-text-muted mt-2 text-xs">
                {viewMode === 'list' ? t('cardDisplay.listHint') : t('cardDisplay.gridHint')}
              </p>
            </div>
          </div>

          <IconSourceSettings />

          <CustomCssSettings />

          <ThemeTransfer />
        </div>
      </div>
    </div>
  )
}
