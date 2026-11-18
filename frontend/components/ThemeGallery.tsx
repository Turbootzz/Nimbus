'use client'

import { useTheme } from '@/contexts/ThemeContext'
import { themePresets } from '@/lib/theme-presets'

const dots = ['bg-primary', 'bg-success', 'bg-warning', 'bg-error', 'bg-info']

export default function ThemeGallery() {
  const { preset, setPreset } = useTheme()

  return (
    <div className="glass-card border-card-border rounded-lg border p-6">
      <h2 id="theme-gallery-title" className="text-text-primary mb-2 text-xl font-semibold">
        Theme
      </h2>
      <p className="text-text-secondary mb-4 text-sm">
        A theme sets every colour, light or dark, and its own accent. You can still pick another
        accent colour on top. Choosing a mode the theme doesn&apos;t have goes back to Default.
      </p>

      <div
        role="group"
        aria-labelledby="theme-gallery-title"
        className="grid grid-cols-2 gap-3 sm:grid-cols-4"
      >
        {themePresets.map((option) => (
          <button
            key={option.id}
            type="button"
            onClick={() => setPreset(option.id)}
            aria-pressed={preset === option.id}
            className={`focus-visible:ring-primary rounded-lg border-2 p-1.5 text-left transition-colors focus-visible:ring-2 focus-visible:outline-none ${
              preset === option.id ? 'border-primary' : 'border-card-border hover:border-primary'
            }`}
          >
            {option.mode ? (
              // The swatch takes the preset's own tokens (styles/themes.css)
              <span data-preset={option.id} className="bg-background block rounded-md p-2">
                <span className="bg-card border-card-border block rounded border p-2">
                  <span className="text-text-primary block text-xs font-semibold">Aa</span>
                  <span className="text-text-muted block text-[10px]">Muted</span>
                  <span className="mt-1.5 flex gap-1">
                    {dots.map((dot) => (
                      <span key={dot} className={`${dot} h-2.5 w-2.5 rounded-full`} />
                    ))}
                  </span>
                </span>
              </span>
            ) : (
              // Default follows the mode, so it shows both
              <span
                className="block h-[4.75rem] rounded-md"
                style={{ background: 'linear-gradient(135deg, #f9fafb 50%, #0a0a0a 50%)' }}
              />
            )}
            <span className="text-text-primary mt-1.5 block truncate text-xs font-medium">
              {option.name}
            </span>
            <span className="text-text-muted block text-[10px] capitalize">
              {option.mode ?? 'Light or dark'}
            </span>
          </button>
        ))}
      </div>
    </div>
  )
}
