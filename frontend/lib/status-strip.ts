import type { ChipOption, Integration, IntegrationKindMeta, StatusChip, Widget } from '@/types'

export const chipKey = (chip: StatusChip) => `${chip.source}:${chip.id}:${chip.kpi}`

// chipOptions lists every KPI of the user's integrations and every value of
// their enabled custom API widgets, keyed by chipKey
export function chipOptions(
  integrations: Integration[],
  kinds: IntegrationKindMeta[],
  widgets: Widget[]
): Map<string, ChipOption> {
  const options: ChipOption[] = []
  for (const integration of integrations) {
    const kpis = kinds.find((k) => k.kind === integration.kind)?.kpis ?? []
    for (const kpi of kpis) {
      options.push({
        chip: { source: 'integration', id: integration.id, kpi: kpi.key },
        label: `${integration.name} · ${kpi.label}`,
        unit: kpi.unit,
      })
    }
  }
  for (const widget of widgets) {
    if (widget.type !== 'custom_api' || !widget.enabled) continue
    const fields = (widget.config.fields ?? []) as { label: string; unit?: string }[]
    for (const field of fields) {
      options.push({
        chip: { source: 'widget', id: widget.id, kpi: field.label },
        label: `${widget.title || 'Custom API'} · ${field.label}`,
        unit: field.unit || undefined,
      })
    }
  }
  return new Map(options.map((option) => [chipKey(option.chip), option]))
}
