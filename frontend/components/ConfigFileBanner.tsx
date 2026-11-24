'use client'

import { LockClosedIcon, DocumentTextIcon } from '@heroicons/react/24/outline'
import { useTranslations } from 'next-intl'
import { useConfigFile } from '@/hooks/useConfigFile'

// Tells the owner that their dashboard follows a config file
export default function ConfigFileBanner() {
  const t = useTranslations('layout.configFile')
  const configFile = useConfigFile()
  if (!configFile) return null

  const Icon = configFile.read_only ? LockClosedIcon : DocumentTextIcon
  return (
    <div
      role="status"
      className="glass-card border-card-border text-text-secondary mb-4 flex items-start gap-3 rounded-lg border p-3 text-sm"
    >
      <Icon className="text-primary mt-0.5 h-5 w-5 shrink-0" aria-hidden="true" />
      <p>
        <span className="text-text-primary font-medium">{t('title')}</span>{' '}
        {configFile.read_only ? t('readOnly') : t('editable')}
        {configFile.preferences && ` ${t('preferences')}`}
      </p>
    </div>
  )
}

// Explains a save button that a read-only dashboard disables
export function ReadOnlyNote() {
  const t = useTranslations('layout.configFile')
  return <p className="text-text-muted mt-2 text-sm">{t('readOnlyNote')}</p>
}
