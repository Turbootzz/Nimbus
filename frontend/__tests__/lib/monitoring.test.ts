import { describe, it, expect } from 'vitest'
import type { Group } from '@/types'
import { buildGroupMonitoringMap, isServiceEffectivelyMonitored } from '@/lib/monitoring'

const group = (id: string, monitoring_enabled: boolean): Group => ({
  id,
  name: id,
  color: '#000',
  position: 0,
  is_default: false,
  monitoring_enabled,
  created_at: '2026-01-01T00:00:00Z',
})

describe('isServiceEffectivelyMonitored', () => {
  const map = buildGroupMonitoringMap([group('g-on', true), group('g-off', false)])

  it('returns false when the service itself opted out', () => {
    expect(
      isServiceEffectivelyMonitored({ monitoring_enabled: false, group_id: 'g-on' }, map)
    ).toBe(false)
  })

  it('returns false when the parent group opted out, even if the service did not', () => {
    expect(
      isServiceEffectivelyMonitored({ monitoring_enabled: true, group_id: 'g-off' }, map)
    ).toBe(false)
  })

  it('returns true when both flags are on', () => {
    expect(isServiceEffectivelyMonitored({ monitoring_enabled: true, group_id: 'g-on' }, map)).toBe(
      true
    )
  })

  it('returns true for an ungrouped service whose own flag is on', () => {
    expect(
      isServiceEffectivelyMonitored({ monitoring_enabled: true, group_id: undefined }, map)
    ).toBe(true)
  })

  it('falls back to the service flag when no group map is provided', () => {
    expect(isServiceEffectivelyMonitored({ monitoring_enabled: true, group_id: 'g-off' })).toBe(
      true
    )
    expect(isServiceEffectivelyMonitored({ monitoring_enabled: false, group_id: 'g-off' })).toBe(
      false
    )
  })

  it('treats an unknown group as monitored (data race tolerance)', () => {
    expect(
      isServiceEffectivelyMonitored({ monitoring_enabled: true, group_id: 'missing-from-map' }, map)
    ).toBe(true)
  })
})

describe('buildGroupMonitoringMap', () => {
  it('returns an empty map for null/undefined input', () => {
    expect(buildGroupMonitoringMap(null).size).toBe(0)
    expect(buildGroupMonitoringMap(undefined).size).toBe(0)
  })

  it('preserves per-group flags', () => {
    const map = buildGroupMonitoringMap([group('a', true), group('b', false)])
    expect(map.get('a')).toBe(true)
    expect(map.get('b')).toBe(false)
  })
})
