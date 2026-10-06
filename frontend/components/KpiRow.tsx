'use client'

import type { Kpi, Snapshot, IntegrationPayload } from '@/types'

// Tailwind needs full class names, so no template strings here
export const kpiColumns: Record<number, string> = {
  1: 'grid-cols-1',
  2: 'grid-cols-2',
  3: 'grid-cols-3',
  4: 'grid-cols-4',
}

export interface ServiceKpis {
  kpis: Kpi[] // from the integration kind, in display order
  snapshot?: Snapshot
}

export function formatKpiValue(value: unknown, unit?: string): string {
  if (value === undefined || value === null) return '-'
  const text = typeof value === 'number' ? value.toLocaleString() : String(value)
  return unit ? `${text} ${unit}` : text
}

// Up to four numbers from a linked integration, shown on a service tile
export default function KpiRow({ kpis, snapshot }: ServiceKpis) {
  const shown = kpis.slice(0, 4)
  if (shown.length === 0) return null

  const values = (snapshot?.payload as IntegrationPayload | null | undefined)?.kpis
  const loading = !snapshot
  const problem = snapshot?.error
    ? `Last update failed: ${snapshot.error}`
    : snapshot?.stale
      ? 'This data may be out of date'
      : undefined

  return (
    <dl className={`grid gap-2 ${kpiColumns[shown.length]}`} title={problem}>
      {shown.map((kpi) => (
        <div key={kpi.key} className="min-w-0">
          <dt className="text-text-muted truncate text-[10px] font-semibold tracking-wide uppercase">
            {kpi.label}
          </dt>
          <dd
            className={`text-sm font-semibold tabular-nums ${problem ? 'text-text-muted' : 'text-text-primary'}`}
          >
            {loading ? (
              <span className="bg-background inline-block h-4 w-8 animate-pulse rounded align-middle">
                <span className="sr-only">Loading</span>
              </span>
            ) : (
              formatKpiValue(values?.[kpi.key], kpi.unit)
            )}
          </dd>
        </div>
      ))}
    </dl>
  )
}
