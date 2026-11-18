import { describe, expect, it } from 'vitest'
import { kpisForIntegration } from '@/lib/kpis'
import { chipOptions } from '@/lib/status-strip'
import type { Integration, IntegrationKindMeta } from '@/types'

const kinds: IntegrationKindMeta[] = [
  {
    kind: 'home_assistant',
    name: 'Home Assistant',
    auth_types: ['token'],
    kpis: [
      { key: 'lights_on', label: 'Lights on' },
      { key: 'entity_1', label: 'Entity 1', option: 'entity_1' },
      { key: 'entity_2', label: 'Entity 2', option: 'entity_2' },
    ],
  },
]
const home = { id: 'ha', kind: 'home_assistant', name: 'Home' } as Integration
const withEntity = { ...home, options: { entity_1: 'sensor.temp' } } as Integration

describe('kpisForIntegration', () => {
  it('leaves out KPIs whose option is not set', () => {
    expect(kpisForIntegration(home, kinds).map((k) => k.key)).toEqual(['lights_on'])
    expect(kpisForIntegration(withEntity, kinds).map((k) => k.key)).toEqual([
      'lights_on',
      'entity_1',
    ])
    expect(kpisForIntegration({ ...home, kind: 'gone' }, kinds)).toEqual([])
  })
})

describe('chipOptions', () => {
  it('offers the KPIs that apply, with the live label and unit once known', () => {
    expect([...chipOptions([home], kinds, []).keys()]).toEqual(['integration:ha:lights_on'])

    const known = chipOptions([withEntity], kinds, [])
    expect(known.get('integration:ha:entity_1')?.label).toBe('Home · Entity 1')

    const live = chipOptions([withEntity], kinds, [], {
      'integration:ha': {
        source_kind: 'integration',
        source_id: 'ha',
        payload: { kpi_defs: [{ key: 'entity_1', label: 'Living room', unit: '°C' }] },
        fetched_at: '',
        stale: false,
      },
    })
    expect(live.get('integration:ha:entity_1')).toMatchObject({
      label: 'Home · Living room',
      unit: '°C',
    })
    expect(live.get('integration:ha:lights_on')?.label).toBe('Home · Lights on')
  })
})
