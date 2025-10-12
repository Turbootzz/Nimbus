'use client'

import { useTheme } from '@/contexts/ThemeContext'

export default function ThemePage() {
  const { theme, accentColor, background, setTheme, setAccentColor, setBackground } = useTheme()

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
    <div className="max-w-4xl p-6">
      <h1 className="text-text-primary mb-2 text-3xl font-bold">Theme Settings</h1>
      <p className="text-text-secondary mb-2">Customize the appearance of your dashboard</p>
      <p className="text-text-muted mb-8 text-sm">Changes are saved automatically</p>

      <div className="space-y-6">
        {/* Theme Mode */}
        <div className="bg-card border-card-border rounded-lg border p-6">
          <h2 className="text-text-primary mb-2 text-xl font-semibold">Theme Mode</h2>
          <p className="text-text-secondary mb-4 text-sm">Choose between light and dark theme</p>

          <div className="flex gap-4">
            <button
              onClick={() => setTheme('light')}
              className={`flex flex-1 items-center justify-center gap-2 rounded-lg px-6 py-3 font-medium transition-all ${
                theme === 'light'
                  ? 'bg-primary text-white shadow-md'
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
              className={`flex flex-1 items-center justify-center gap-2 rounded-lg px-6 py-3 font-medium transition-all ${
                theme === 'dark'
                  ? 'bg-primary text-white shadow-md'
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

          <div className="mb-4 grid grid-cols-4 gap-3 md:grid-cols-8">
            {presetColors.map((color) => (
              <button
                key={color.value}
                onClick={() => setAccentColor(color.value)}
                className={`aspect-square rounded-lg border-2 transition-all hover:scale-110 ${
                  accentColor === color.value
                    ? 'border-text-primary ring-primary scale-110 ring-2 ring-offset-2'
                    : 'border-card-border'
                }`}
                style={{ backgroundColor: color.value }}
                title={color.name}
              />
            ))}
          </div>

          <div className="flex flex-wrap items-center gap-2">
            <label className="text-text-primary text-sm font-medium">Custom color:</label>
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
              className="border-card-border bg-background text-text-primary focus:ring-primary min-w-[150px] flex-1 rounded-lg border px-3 py-2 focus:ring-2 focus:outline-none"
              maxLength={7}
            />
            {accentColor && (
              <button
                onClick={() => setAccentColor(undefined)}
                className="text-text-secondary hover:text-text-primary px-4 py-2 text-sm transition-colors"
              >
                Reset
              </button>
            )}
          </div>
        </div>

        {/* Background Image */}
        <div className="bg-card border-card-border rounded-lg border p-6">
          <h2 className="text-text-primary mb-2 text-xl font-semibold">Background Image</h2>
          <p className="text-text-secondary mb-4 text-sm">Add a custom background image (URL)</p>

          <div className="flex flex-wrap gap-2">
            <input
              type="url"
              value={background || ''}
              onChange={(e) => setBackground(e.target.value)}
              placeholder="https://example.com/image.jpg"
              className="border-card-border bg-background text-text-primary focus:ring-primary min-w-[200px] flex-1 rounded-lg border px-3 py-2 focus:ring-2 focus:outline-none"
            />
            {background && (
              <button
                onClick={() => setBackground(undefined)}
                className="border-card-border text-text-primary hover:bg-card-border rounded-lg border px-4 py-2 transition-colors"
              >
                Clear
              </button>
            )}
          </div>

          {background && (
            <div className="border-card-border mt-4 overflow-hidden rounded-lg border">
              <img src={background} alt="Background preview" className="h-48 w-full object-cover" />
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
    </div>
  )
}
