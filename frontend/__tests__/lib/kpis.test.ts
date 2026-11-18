import { describe, expect, it } from 'vitest'
import { kpisForIntegration, tileKpis } from '@/lib/kpis'
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

  it('leaves out KPIs that only apply without an option', () => {
    const kuma: IntegrationKindMeta[] = [
      {
        kind: 'uptime_kuma',
        name: 'Uptime Kuma',
        auth_types: ['none'],
        kpis: [
          { key: 'up', label: 'Up' },
          { key: 'up_percent', label: 'Up share', without: 'slug' },
          { key: 'uptime', label: 'Uptime 24h', option: 'slug' },
        ],
      },
    ]
    const metrics = { id: 'k', kind: 'uptime_kuma', name: 'Kuma', options: {} } as Integration
    expect(kpisForIntegration(metrics, kuma).map((k) => k.key)).toEqual(['up', 'up_percent'])
    const page = { ...metrics, options: { slug: 'homelab' } } as Integration
    expect(kpisForIntegration(page, kuma).map((k) => k.key)).toEqual(['up', 'uptime'])
  })
})

describe('tileKpis', () => {
  const fixed = ['a', 'b', 'c', 'd', 'e'].map((key) => ({ key, label: key }))
  const chosen = ['x', 'y'].map((key) => ({ key, label: key, option: key }))
  const keys = (kpis: { key: string }[]) => kpis.map((k) => k.key).join('')

  it('keeps four, dropping KPIs without an option from the end first', () => {
    expect(keys(tileKpis(fixed))).toBe('abcd')
    expect(keys(tileKpis([...fixed.slice(0, 3), chosen[0]]))).toBe('abcx')
    expect(keys(tileKpis([...fixed.slice(0, 3), ...chosen]))).toBe('abxy')
    expect(keys(tileKpis([...chosen, ...chosen, ...chosen]))).toBe('xyxy')
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
