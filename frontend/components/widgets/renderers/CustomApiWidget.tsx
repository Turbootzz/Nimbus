'use client'

import type { CustomApiPayload, CustomApiWidgetConfig } from '@/types'
import type { WidgetRendererProps } from '@/components/widgets/registry'
import { formatKpiValue, kpiColumns } from '@/components/KpiRow'

export default function CustomApiWidget({
  config,
  cardSize,
  snapshot,
}: WidgetRendererProps<CustomApiWidgetConfig>) {
  const payload = snapshot?.payload as CustomApiPayload | null | undefined
  if (!payload) return null // WidgetCard shows loading and errors

  const fields = config.fields.slice(0, 4)
  // Four values on a square tile read better as two rows
  const square = cardSize === '1x1' || cardSize === '2x2'
  const columns = square ? Math.min(fields.length, 2) : fields.length

  return (
    <dl className={`grid h-full content-center gap-3 ${kpiColumns[columns]}`}>
      {fields.map((field) => {
        const missing = payload.missing?.includes(field.label)
        const value = formatKpiValue(payload.kpis[field.label], field.unit)
        return (
          <div key={field.label} className="min-w-0">
            <dt className="text-text-muted truncate text-[10px] font-semibold tracking-wide uppercase">
              {field.label}
            </dt>
            <dd
              className={`truncate font-semibold tabular-nums ${cardSize === '1x1' ? 'text-lg' : 'text-2xl'} ${missing ? 'text-text-muted' : 'text-text-primary'}`}
              title={missing ? `Nothing found at ${field.path}` : value}
            >
              {value}
            </dd>
          </div>
        )
      })}
    </dl>
  )
}
