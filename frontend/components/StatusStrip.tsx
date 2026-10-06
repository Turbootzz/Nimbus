'use client'

import { useMemo } from 'react'
import type {
  Integration,
  IntegrationKindMeta,
  IntegrationPayload,
  StatusStrip as StatusStripConfig,
  Widget,
} from '@/types'
import type { SnapshotMap } from '@/hooks/useDashboardStream'
import { snapshotKey } from '@/hooks/useDashboardStream'
import { formatKpiValue } from '@/components/KpiRow'
import { chipKey, chipOptions } from '@/lib/status-strip'

interface StatusStripProps {
  strip: StatusStripConfig
  snapshots: SnapshotMap
  integrations: Integration[]
  kinds: IntegrationKindMeta[]
  widgets: Widget[]
  // Monitored services by status; the pill only shows once some are known
  up: number
  down: number
}

// A thin row of key numbers at the top of the dashboard. Chips whose
// integration, widget or value is gone are left out.
export default function StatusStrip({
  strip,
  snapshots,
  integrations,
  kinds,
  widgets,
  up,
  down,
}: StatusStripProps) {
  const options = useMemo(
    () => chipOptions(integrations, kinds, widgets),
    [integrations, kinds, widgets]
  )

  const chips = strip.chips.flatMap((chip) => {
    const option = options.get(chipKey(chip))
    if (!option) return []
    const snapshot = snapshots[snapshotKey(chip.source, chip.id)]
    const kpis = (snapshot?.payload as IntegrationPayload | null | undefined)?.kpis
    // A label like "constructor" must not find Object.prototype
    const value = kpis && Object.hasOwn(kpis, chip.kpi) ? kpis[chip.kpi] : undefined
    // Old data stays, muted, like on a KPI row
    const problem = snapshot?.error
      ? `Last update failed: ${snapshot.error}`
      : snapshot?.stale
        ? 'This data may be out of date'
        : undefined
    return [
      {
        key: chipKey(chip),
        label: option.label,
        value: formatKpiValue(value, option.unit),
        problem,
      },
    ]
  })

  return (
    <section aria-label="Status" className="mb-4 flex gap-2 overflow-x-auto pb-1">
      {up + down > 0 && (
        <span
          role="status"
          className={`glass-card flex shrink-0 items-center gap-1.5 rounded-full border px-3 py-1.5 text-sm font-medium ${
            down > 0 ? 'border-error/40 text-error' : 'border-success/40 text-success'
          }`}
        >
          <span
            className={`h-2 w-2 rounded-full ${down > 0 ? 'bg-error' : 'bg-success'}`}
            aria-hidden="true"
          />
          {down > 0 ? `${down} down` : 'Operational'}
        </span>
      )}
      {chips.map((chip) => (
        <span
          key={chip.key}
          title={chip.problem}
          className="glass-card border-card-border flex shrink-0 items-baseline gap-2 rounded-full border px-3 py-1.5"
        >
          <span className="text-text-muted text-[10px] font-semibold tracking-wide uppercase">
            {chip.label}
          </span>
          <span
            className={`text-sm font-semibold tabular-nums ${chip.problem ? 'text-text-muted' : 'text-text-primary'}`}
          >
            {chip.value}
          </span>
        </span>
      ))}
    </section>
  )
}
