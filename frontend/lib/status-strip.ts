import type {
  ChipOption,
  Integration,
  IntegrationKindMeta,
  IntegrationPayload,
  StatusChip,
  Widget,
} from '@/types'
import type { SnapshotMap } from '@/hooks/useDashboardStream'
import { snapshotKey } from '@/hooks/useDashboardStream'
import { kpisForIntegration } from '@/lib/kpis'

export const chipKey = (chip: StatusChip) => `${chip.source}:${chip.id}:${chip.kpi}`

// chipOptions lists every KPI of the user's integrations and every value of
// their enabled custom API widgets, keyed by chipKey. With snapshots, a
// KPI takes the label and unit its integration last reported (kpi_defs).
export function chipOptions(
  integrations: Integration[],
  kinds: IntegrationKindMeta[],
  widgets: Widget[],
  snapshots?: SnapshotMap
): Map<string, ChipOption> {
  const options: ChipOption[] = []
  for (const integration of integrations) {
    const payload = snapshots?.[snapshotKey('integration', integration.id)]?.payload
    const live = (payload as IntegrationPayload | null | undefined)?.kpi_defs
    for (const kindKpi of kpisForIntegration(integration, kinds)) {
      const kpi = (Array.isArray(live) && live.find((d) => d.key === kindKpi.key)) || kindKpi
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
