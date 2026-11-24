'use client'

import { useTheme } from '@/contexts/ThemeContext'
import { Toggle } from '@/components/ui/Toggle'
import { themeFonts } from '@/lib/theme-presets'
import type { ThemeFont } from '@/types'

// Each button shows its own font; the variables are those of app/layout.tsx
const sample: Record<ThemeFont, string | undefined> = {
  default: undefined,
  system: 'system-ui, sans-serif',
  inter: 'var(--font-inter), sans-serif',
  'jetbrains-mono': 'var(--font-jetbrains-mono), monospace',
}

export default function FontSettings() {
  const { font, kpiMono, setFont, setKpiMono } = useTheme()

  return (
    <div className="glass-card border-card-border rounded-lg border p-6">
      <h2 id="font-title" className="text-text-primary mb-2 text-xl font-semibold">
        Font
      </h2>
      <p className="text-text-secondary mb-4 text-sm">
        The fonts are part of Nimbus: nothing is loaded from another site
      </p>

      <div role="group" aria-labelledby="font-title" className="grid grid-cols-2 gap-2">
        {themeFonts.map((option) => (
          <button
            key={option.id}
            type="button"
            onClick={() => setFont(option.id)}
            aria-pressed={font === option.id}
            style={{ fontFamily: sample[option.id] }}
            className={`rounded-lg px-4 py-2 text-sm font-medium transition-all ${
              font === option.id
                ? 'bg-primary text-white'
                : 'bg-background border-card-border text-text-primary hover:border-primary border'
            }`}
          >
            {option.name}
          </button>
        ))}
      </div>

      <div className="mt-6">
        <Toggle
          id="kpi-mono"
          enabled={kpiMono}
          onChange={setKpiMono}
          label="Monospace KPI labels"
          description="Show the small labels above the numbers on tiles, widgets and the status strip in JetBrains Mono"
        />
      </div>
    </div>
  )
}
