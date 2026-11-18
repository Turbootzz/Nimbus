import type { Integration, IntegrationKindMeta, Kpi } from '@/types'

// The KPIs of one integration: those of its kind, without the ones that
// need an option it has not set
export function kpisForIntegration(integration: Integration, kinds: IntegrationKindMeta[]): Kpi[] {
  const kpis = kinds.find((k) => k.kind === integration.kind)?.kpis ?? []
  return kpis.filter((kpi) => !kpi.option || !!integration.options?.[kpi.option])
}
