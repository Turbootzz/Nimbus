import type { Integration, IntegrationKindMeta, IntegrationPayload, Kpi } from '@/types'

// The KPIs of one integration: those of its kind that fit the options it
// has set
// Kind and options are all it reads, so a viewer's BoardIntegration works too
export function kpisForIntegration(
  integration: Pick<Integration, 'kind' | 'options'>,
  kinds: Pick<IntegrationKindMeta, 'kind' | 'kpis'>[]
): Kpi[] {
  const kpis = kinds.find((k) => k.kind === integration.kind)?.kpis ?? []
  const isSet = (option?: string) => !!option && !!integration.options?.[option]
  return kpis.filter(
    (kpi) => (!kpi.option || isSet(kpi.option)) && !(kpi.without && isSet(kpi.without))
  )
}

// A tile has four numbers. KPIs that follow an option were asked for, so
// the others give way first, from the end.
export function tileKpis(kpis: Kpi[]): Kpi[] {
  const shown = [...kpis]
  for (let i = shown.length - 1; i >= 0 && shown.length > 4; i--) {
    if (!shown[i].option) shown.splice(i, 1)
  }
  return shown.slice(0, 4)
}

// The KPI with the label and unit its integration last reported (kpi_defs)
export function liveKpi(kpi: Kpi, payload: unknown): Kpi {
  const live = (payload as IntegrationPayload | null | undefined)?.kpi_defs
  return (Array.isArray(live) && live.find((d) => d?.key === kpi.key)) || kpi
}
