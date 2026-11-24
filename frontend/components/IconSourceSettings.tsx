'use client'

import { useTranslations } from 'next-intl'
import { useTheme } from '@/contexts/ThemeContext'
import { iconSources } from '@/lib/theme-presets'
import type { IconSource } from '@/types'

export default function IconSourceSettings() {
  const t = useTranslations('settings.theme.iconSource')
  const { iconSource, setIconSource } = useTheme()

  return (
    <div className="glass-card border-card-border rounded-lg border p-6">
      <h2 className="text-text-primary mb-2 text-xl font-semibold">{t('title')}</h2>
      <p className="text-text-secondary mb-4 text-sm">{t('description')}</p>

      <label htmlFor="icon-source" className="text-text-primary mb-2 block text-sm font-medium">
        {t('label')}
      </label>
      <select
        id="icon-source"
        value={iconSource}
        onChange={(e) => setIconSource(e.target.value as IconSource)}
        className="border-card-border bg-background text-text-primary focus:ring-primary w-full rounded-lg border px-3 py-2 text-sm focus:ring-2 focus:outline-none"
      >
        {iconSources.map((source) => (
          <option key={source.id} value={source.id}>
            {source.name}
          </option>
        ))}
      </select>
      <p className="text-text-muted mt-2 text-xs">{t('hint')}</p>
    </div>
  )
}
