'use client'

import { useMemo } from 'react'
import { Toggle } from '@/components/ui/Toggle'
import type { ClockWidgetConfig } from '@/types'
import type { WidgetFormProps } from '@/components/widgets/registry'
import { inputClass, labelClass } from '@/components/widgets/forms/fieldStyles'

// Timezone suggestions; older browsers lack supportedValuesOf
function timezones(): string[] {
  return typeof Intl.supportedValuesOf === 'function' ? Intl.supportedValuesOf('timeZone') : []
}

export default function ClockForm({
  config,
  onChange,
  disabled,
}: WidgetFormProps<ClockWidgetConfig>) {
  const zones = useMemo(() => timezones(), [])

  return (
    <div className="space-y-4">
      <div>
        <label htmlFor="clock-timezone" className={labelClass}>
          Timezone
        </label>
        <input
          id="clock-timezone"
          type="text"
          list="clock-timezones"
          value={config.timezone}
          onChange={(e) => onChange({ ...config, timezone: e.target.value })}
          placeholder="Your browser's timezone"
          className={inputClass}
          disabled={disabled}
        />
        <datalist id="clock-timezones">
          {zones.map((zone) => (
            <option key={zone} value={zone} />
          ))}
        </datalist>
        <p className="text-text-muted mt-1 text-xs">For example Europe/Amsterdam</p>
      </div>

      <div>
        <label htmlFor="clock-date-format" className={labelClass}>
          Date
        </label>
        <select
          id="clock-date-format"
          value={config.date_format}
          onChange={(e) =>
            onChange({
              ...config,
              date_format: e.target.value as ClockWidgetConfig['date_format'],
            })
          }
          className={inputClass}
          disabled={disabled}
        >
          <option value="short">Short</option>
          <option value="long">Long</option>
          <option value="none">Hidden</option>
        </select>
      </div>

      <Toggle
        id="clock-hour12"
        enabled={config.hour12}
        onChange={(hour12) => onChange({ ...config, hour12 })}
        label="12-hour clock"
        disabled={disabled}
      />
      <Toggle
        id="clock-seconds"
        enabled={config.show_seconds}
        onChange={(show_seconds) => onChange({ ...config, show_seconds })}
        label="Show seconds"
        disabled={disabled}
      />
    </div>
  )
}
