'use client'

import { useTranslations } from 'next-intl'
import { useTheme } from '@/contexts/ThemeContext'
import { isLanguage, languages } from '@/i18n/locale'

// The user's language; automatic follows the browser
export default function LanguageSettings() {
  const t = useTranslations('language')
  const { language, setLanguage } = useTheme()

  return (
    <div>
      <label
        htmlFor="language"
        className="mb-1 block text-sm font-medium"
        style={{ color: 'var(--color-text-secondary)' }}
      >
        {t('label')}
      </label>
      <select
        id="language"
        value={language ?? ''}
        disabled={language === undefined}
        onChange={(e) => setLanguage(isLanguage(e.target.value) ? e.target.value : null)}
        className="border-card-border bg-background text-text-primary focus:ring-primary w-full rounded-lg border px-3 py-2 text-sm focus:ring-2 focus:outline-none sm:w-64"
      >
        <option value="">{t('automatic')}</option>
        {languages.map((code) => (
          <option key={code} value={code} lang={code}>
            {t(`names.${code}`)}
          </option>
        ))}
      </select>
      <p className="mt-1 text-xs" style={{ color: 'var(--color-text-muted)' }}>
        {t('hint')}
      </p>
    </div>
  )
}
