import { describe, it, expect } from 'vitest'
import { canCreate, defaultTargets, mapGroups } from '@/lib/import-wizard'
import type { ConvertedDocument, Group } from '@/types'

const group = (id: string, name: string, isDefault = false): Group => ({
  id,
  name,
  color: '#0ea5e9',
  position: 0,
  is_default: isDefault,
  monitoring_enabled: true,
  created_at: '2026-01-01T00:00:00Z',
})

const existing = [group('g-home', 'Home', true), group('g-media', 'Media')]

const doc = (): ConvertedDocument => ({
  version: 1,
  groups: [
    { ref: 'group-media', name: 'Media' },
    { ref: 'group-films', name: 'Films' },
    { ref: 'group-network', name: 'Network' },
  ],
  integrations: [],
  services: [
    { name: 'Sonarr', url: 'http://sonarr.lan', group_ref: 'group-media', position: 0 },
    { name: 'Radarr', url: 'http://radarr.lan', group_ref: 'group-films', position: 1 },
    { name: 'Pi-hole', url: 'http://pi.hole', group_ref: 'group-network', position: 2 },
    { name: 'Router', url: 'http://192.168.1.1', position: 3 },
  ],
  widgets: [{ type: 'markdown', group_ref: 'group-films', position: 4 }],
})

describe('import wizard group mapping', () => {
  it('suggests the existing group with the same name', () => {
    expect(defaultTargets(doc(), existing)).toEqual({
      'group-media': 'g-media',
      'group-films': '',
      'group-network': '',
    })
  })

  it('cannot create a group under a name that exists', () => {
    expect(canCreate({ name: 'Media' }, existing)).toBe(false)
    expect(canCreate({ name: 'Films' }, existing)).toBe(true)
  })

  it('leaves the document alone when every group is new', () => {
    const input = doc()
    const targets = { 'group-media': '', 'group-films': '', 'group-network': '' }
    expect(mapGroups(input, targets, [])).toEqual(input)
  })

  it('renames a group to the existing group it goes into', () => {
    const mapped = mapGroups(doc(), { 'group-films': 'g-media', 'group-media': '' }, [existing[1]])
    expect(mapped.groups.map((g) => g.name)).toEqual(['Media', 'Media', 'Network'])
  })

  it('makes one group of the groups that go into the same group', () => {
    const mapped = mapGroups(
      doc(),
      { 'group-media': 'g-media', 'group-films': 'g-media', 'group-network': '' },
      existing
    )
    expect(mapped.groups.map((g) => [g.ref, g.name])).toEqual([
      ['group-media', 'Media'],
      ['group-network', 'Network'],
    ])
    expect(mapped.services.map((s) => s.group_ref)).toEqual([
      'group-media',
      'group-media',
      'group-network',
      undefined,
    ])
    expect(mapped.widgets[0].group_ref).toBe('group-media')
  })

  it('marks the group that goes into the default group as the default', () => {
    const mapped = mapGroups(doc(), { 'group-network': 'g-home' }, existing)
    const network = mapped.groups.find((g) => g.ref === 'group-network')
    expect(network).toMatchObject({ name: 'Home', default: true })
    // The other groups carry no default flag into the JSON that is sent
    expect(
      JSON.parse(JSON.stringify(mapped)).groups.filter((g: object) => 'default' in g)
    ).toHaveLength(1)
  })

  it('does not change the document it is given', () => {
    const input = doc()
    mapGroups(input, { 'group-media': 'g-home', 'group-films': 'g-home' }, existing)
    expect(input).toEqual(doc())
  })

  it('ignores a target that is not one of the user’s groups', () => {
    const input = doc()
    expect(mapGroups(input, { 'group-media': 'someone-elses' }, existing)).toEqual(input)
  })
})
