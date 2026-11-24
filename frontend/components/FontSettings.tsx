'use client'

import { useTranslations } from 'next-intl'
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
  const t = useTranslations('settings.theme.font')
  const { font, kpiMono, setFont, setKpiMono } = useTheme()
  // Inter and JetBrains Mono are names; these two are words
  const names: Partial<Record<ThemeFont, string>> = {
    default: t('names.default'),
    system: t('names.system'),
  }

  return (
    <div className="glass-card border-card-border rounded-lg border p-6">
      <h2 id="font-title" className="text-text-primary mb-2 text-xl font-semibold">
        {t('title')}
      </h2>
      <p className="text-text-secondary mb-4 text-sm">{t('description')}</p>

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
            {names[option.id] ?? option.name}
          </button>
        ))}
      </div>

      <div className="mt-6">
        <Toggle
          enabled={kpiMono}
          onChange={setKpiMono}
          label={t('kpiMono')}
          description={t('kpiMonoDescription')}
        />
      </div>
    </div>
  )
}
