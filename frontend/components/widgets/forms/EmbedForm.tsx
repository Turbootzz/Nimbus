'use client'

import type { EmbedWidgetConfig } from '@/types'
import type { WidgetFormProps } from '@/components/widgets/registry'
import { inputClass, labelClass } from '@/components/widgets/forms/fieldStyles'

export default function EmbedForm({
  config,
  onChange,
  disabled,
}: WidgetFormProps<EmbedWidgetConfig>) {
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
        <p className="text-text-muted mt-1 text-xs">
          Some sites refuse to be embedded. Your browser loads the page, not the Nimbus server.
        </p>
      </div>
      <div>
        <label htmlFor="embed-height" className={labelClass}>
          Height (pixels)
        </label>
        <input
          id="embed-height"
          type="number"
          min={100}
          max={1200}
          step={10}
          value={config.height}
          onChange={(e) => onChange({ ...config, height: Number(e.target.value) })}
          className={inputClass}
          disabled={disabled}
        />
      </div>
    </div>
  )
}
