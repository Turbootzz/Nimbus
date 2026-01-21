'use client'

import Image from 'next/image'
import { useTheme } from '@/contexts/ThemeContext'
import { Toggle } from '@/components/ui/Toggle'

export default function ThemePage() {
  const {
    theme,
    effectiveTheme,
    accentColor,
    background,
    openInNewTab,
    enableCardResizing,
    enableServiceGrouping,
    cardScale,
    viewMode,
    setTheme,
    setAccentColor,
    setBackground,
    setOpenInNewTab,
    setEnableCardResizing,
    setEnableServiceGrouping,
    setCardScale,
    setViewMode,
  } = useTheme()

  const presetColors = [
    { name: 'Sky Blue (Default)', value: '#0ea5e9' },
    { name: 'Purple', value: '#8B5CF6' },
    { name: 'Pink', value: '#EC4899' },
    { name: 'Red', value: '#EF4444' },
    { name: 'Orange', value: '#F97316' },
    { name: 'Yellow', value: '#EAB308' },
    { name: 'Green', value: '#10B981' },
    { name: 'Teal', value: '#14B8A6' },
  ]

  return (
    <div className="max-w-4xl p-4 sm:p-6 xl:max-w-none">
      <h1 className="text-text-primary mb-2 text-2xl font-bold sm:text-3xl">Theme Settings</h1>
      <p className="text-text-secondary mb-2 text-sm sm:text-base">
        Customize the appearance of your dashboard
      </p>
      <p className="text-text-muted mb-8 text-xs sm:text-sm">Changes are saved automatically</p>

      {/* Two-column layout on XL screens */}
      <div className="grid grid-cols-1 gap-6 xl:grid-cols-2">
        {/* Column 1: Appearance settings */}
        <div className="space-y-6">
          {/* Theme Mode */}
          <div className="bg-card border-card-border rounded-lg border p-6">
            <h2 className="text-text-primary mb-2 text-xl font-semibold">Theme Mode</h2>
            <p className="text-text-secondary mb-4 text-sm">Choose between light and dark theme</p>

            {/* Automatic Theme Toggle */}
            <div className="mb-4">
              <Toggle
                enabled={theme === 'auto'}
                onChange={(enabled) => setTheme(enabled ? 'auto' : effectiveTheme)}
                label="Automatic (follow system)"
                description={`Automatically switch between light and dark based on your system preferences${
                  theme === 'auto' ? ` (currently: ${effectiveTheme})` : ''
                }`}
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
                Light
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
                Dark
              </button>
            </div>
          </div>

          {/* Accent Color */}
          <div className="bg-card border-card-border rounded-lg border p-6">
            <h2 className="text-text-primary mb-2 text-xl font-semibold">Accent Color</h2>
            <p className="text-text-secondary mb-4 text-sm">Choose your preferred accent color</p>

            <div className="mb-4 flex flex-wrap gap-2">
              {presetColors.map((color) => (
                <button
                  key={color.value}
                  onClick={() => setAccentColor(color.value)}
                  className={`h-12 w-12 rounded-lg border-2 transition-all hover:scale-110 sm:h-18 sm:w-18 ${
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
              <label className="text-text-primary text-sm font-medium">Custom color:</label>
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
                    Reset
                  </button>
                )}
              </div>
            </div>
          </div>

          {/* Background Image */}
          <div className="bg-card border-card-border rounded-lg border p-6">
            <h2 className="text-text-primary mb-2 text-xl font-semibold">Background Image</h2>
            <p className="text-text-secondary mb-4 text-sm">Add a custom background image (URL)</p>

            <div className="flex flex-col gap-2 sm:flex-row sm:flex-wrap">
              <input
                type="url"
                value={background || ''}
                onChange={(e) => setBackground(e.target.value)}
                placeholder="https://example.com/image.jpg"
                className="border-card-border bg-background text-text-primary focus:ring-primary min-w-0 flex-1 rounded-lg border px-3 py-2 text-sm focus:ring-2 focus:outline-none sm:min-w-50"
              />
              {background && (
                <button
                  onClick={() => setBackground(undefined)}
                  className="border-card-border text-text-primary hover:bg-card-border rounded-lg border px-4 py-2 text-sm transition-colors"
                >
                  Clear
                </button>
              )}
            </div>

            {background && (
              <div className="border-card-border relative mt-4 h-48 overflow-hidden rounded-lg border">
                <Image
                  src={background}
                  alt="Background preview"
                  fill
                  className="object-cover"
                  unoptimized
                />
              </div>
            )}

            <div className="bg-info/10 border-info/30 mt-4 rounded-lg border p-4">
              <p className="text-info text-sm">
                <strong>Tip:</strong> For best quality, use high-resolution images (1920x1080 or
                higher). The background uses CSS cover which maintains aspect ratio and quality.
              </p>
            </div>
          </div>
        </div>

        {/* Column 2: Behavior & Layout settings */}
        <div className="space-y-6">
          {/* Link Behavior */}
          <div className="bg-card border-card-border rounded-lg border p-6">
            <h2 className="text-text-primary mb-2 text-xl font-semibold">Link Behavior</h2>
            <p className="text-text-secondary mb-4 text-sm">Choose how service links should open</p>

            <Toggle
              enabled={openInNewTab}
              onChange={setOpenInNewTab}
              label="Open services in new tab"
              description="When enabled, clicking a service will open it in a new tab"
            />
          </div>

          {/* Card Resizing */}
          <div className="bg-card border-card-border rounded-lg border p-6">
            <h2 className="text-text-primary mb-2 text-xl font-semibold">Card Resizing</h2>
            <p className="text-text-secondary mb-4 text-sm">
              Control whether service cards can have different sizes
            </p>

            <Toggle
              enabled={enableCardResizing}
              onChange={setEnableCardResizing}
              label="Enable card resizing"
              description="When enabled, you can resize cards in edit mode by clicking them. When disabled, all cards display as standard size (2x1)"
            />
          </div>

          {/* Service Grouping */}
          <div className="bg-card border-card-border rounded-lg border p-6">
            <h2 className="text-text-primary mb-2 text-xl font-semibold">Service Grouping</h2>
            <p className="text-text-secondary mb-4 text-sm">
              Organize your services into groups with a tabbed interface
            </p>

            <Toggle
              enabled={enableServiceGrouping}
              onChange={setEnableServiceGrouping}
              label="Enable service grouping"
              description="When enabled, you can organize services into groups displayed as tabs. Create, rename, and reorder groups in edit mode. When disabled, all services are shown in a single view"
            />
          </div>

          {/* Card Display */}
          <div className="bg-card border-card-border rounded-lg border p-6">
            <h2 className="text-text-primary mb-2 text-xl font-semibold">Card Display</h2>
            <p className="text-text-secondary mb-4 text-sm">
              Customize how service cards are displayed
            </p>

            {/* View Mode */}
            <div className="mb-6">
              <Toggle
                enabled={viewMode === 'list'}
                onChange={(enabled) => setViewMode(enabled ? 'list' : 'grid')}
                label="List view"
                description="Display services in a compact list instead of a grid"
              />
            </div>

            {/* Card Scale - disabled when list view is enabled */}
            <div className={viewMode === 'list' ? 'opacity-50' : ''}>
              <label className="text-text-primary mb-2 block text-sm font-medium">Card Size</label>
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
                    {scale}
                  </button>
                ))}
              </div>
              <p className="text-text-muted mt-2 text-xs">
                {viewMode === 'list'
                  ? 'Card size only applies to grid view'
                  : 'Adjust the grid density - smaller cards show more services per row'}
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
