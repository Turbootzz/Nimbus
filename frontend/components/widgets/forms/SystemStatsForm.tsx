'use client'

import type { SystemStatsWidgetConfig } from '@/types'
import type { WidgetFormProps } from '@/components/widgets/registry'
import { inputClass, labelClass } from '@/components/widgets/forms/fieldStyles'

export default function SystemStatsForm({
  config,
  onChange,
  disabled,
}: WidgetFormProps<SystemStatsWidgetConfig>) {
  return (
    <div>
      <label htmlFor="system-stats-disk" className={labelClass}>
        Disk
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
      <p className="text-text-muted mt-1 text-xs">
        A path on the Nimbus server; the tile shows the use of the disk it is on. In Docker, mount a
        host disk into the container to see it.
      </p>
    </div>
  )
}
