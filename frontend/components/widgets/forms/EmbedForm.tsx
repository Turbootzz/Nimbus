'use client'

import { useTranslations } from 'next-intl'
import type { EmbedWidgetConfig } from '@/types'
import type { WidgetFormProps } from '@/components/widgets/registry'
import { inputClass, labelClass } from '@/components/widgets/forms/fieldStyles'

export default function EmbedForm({
  config,
  onChange,
  disabled,
}: WidgetFormProps<EmbedWidgetConfig>) {
  const t = useTranslations('widgets.embed.form')
  return (
    <div className="space-y-4">
      <div>
        <label htmlFor="embed-url" className={labelClass}>
          URL
        </label>
        <input
          id="embed-url"
          type="url"
          value={config.url}
          onChange={(e) => onChange({ ...config, url: e.target.value })}
          placeholder="https://grafana.example.com/d/abc?kiosk"
          className={inputClass}
          disabled={disabled}
        />
        <p className="text-text-muted mt-1 text-xs">{t('hint')}</p>
      </div>
    </div>
  )
}
