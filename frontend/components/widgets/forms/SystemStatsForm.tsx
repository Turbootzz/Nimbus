'use client'

import { useTranslations } from 'next-intl'
import type { SystemStatsWidgetConfig } from '@/types'
import type { WidgetFormProps } from '@/components/widgets/registry'
import { inputClass, labelClass } from '@/components/widgets/forms/fieldStyles'

export default function SystemStatsForm({
  config,
  onChange,
  disabled,
}: WidgetFormProps<SystemStatsWidgetConfig>) {
  const t = useTranslations('widgets.systemStats')
  return (
    <div>
      <label htmlFor="system-stats-disk" className={labelClass}>
        {t('disk')}
      </label>
      <input
        id="system-stats-disk"
        value={config.disk_path}
        onChange={(e) => onChange({ ...config, disk_path: e.target.value })}
        placeholder="/"
        maxLength={200}
        className={`${inputClass} font-mono`}
        disabled={disabled}
      />
      <p className="text-text-muted mt-1 text-xs">{t('form.diskHint')}</p>
    </div>
  )
}
