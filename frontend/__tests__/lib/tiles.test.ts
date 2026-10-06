import { describe, it, expect } from 'vitest'
import {
  filterTilesByGroup,
  mergeTiles,
  reorderTiles,
  splitTiles,
  tileKey,
  toTilePositions,
} from '@/lib/tiles'
import type { Group, Service, Widget } from '@/types'

const service = (id: string, position: number, group_id?: string): Service => ({
  id,
  name: id,
  url: `https://${id}.test`,
  icon_type: 'emoji',
  status: 'online',
  position,
  card_size: '2x1',
  group_id,
  monitoring_enabled: true,
  created_at: '2026-01-01T00:00:00Z',
})

const widget = (id: string, position: number, group_id: string | null = null): Widget => ({
  id,
  type: 'clock',
  title: '',
  group_id,
  integration_id: null,
  config: {},
  card_size: '2x1',
  position,
  refresh_seconds: 300,
  enabled: true,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
})

const group = (id: string, is_default = false): Group => ({
  id,
  name: id,
  color: '#000000',
  position: 0,
  is_default,
  monitoring_enabled: true,
  created_at: '2026-01-01T00:00:00Z',
})

const keys = (tiles: { kind: 'service' | 'widget'; id: string }[]) => tiles.map(tileKey)

describe('mergeTiles', () => {
  it('sorts services and widgets by position', () => {
    const tiles = mergeTiles([service('a', 2), service('b', 0)], [widget('w', 1)])
    expect(keys(tiles)).toEqual(['service:b', 'widget:w', 'service:a'])
  })

  it('puts services first on equal positions', () => {
    const tiles = mergeTiles([service('a', 0)], [widget('w', 0)])
    expect(keys(tiles)).toEqual(['service:a', 'widget:w'])
  })
})

describe('filterTilesByGroup', () => {
  const groups = [group('default', true), group('media')]
  const tiles = mergeTiles(
    [service('ungrouped', 0), service('in-default', 1, 'default'), service('in-media', 2, 'media')],
    [widget('w-ungrouped', 3), widget('w-media', 4, 'media')]
  )

  it('shows everything without a selection', () => {
    expect(filterTilesByGroup(tiles, null, groups)).toHaveLength(5)
  })

  it('includes ungrouped tiles in the default group', () => {
    expect(keys(filterTilesByGroup(tiles, 'default', groups))).toEqual([
      'service:ungrouped',
      'service:in-default',
      'widget:w-ungrouped',
    ])
  })

  it('shows only exact matches for other groups', () => {
    expect(keys(filterTilesByGroup(tiles, 'media', groups))).toEqual([
      'service:in-media',
      'widget:w-media',
    ])
  })
})

describe('reorderTiles', () => {
  const all = mergeTiles(
    [service('a', 0, 'g'), service('b', 1), service('c', 3, 'g')],
    [widget('w', 2, 'g')]
  )
  const visible = all.filter((t) => (t.kind === 'service' ? t.service.group_id : t.widget.group_id))

  it('moves within the visible subset and keeps other tiles in their slots', () => {
    // visible: a, w, c; move c before a
    const next = reorderTiles(all, visible, 'service:c', 'service:a')
    expect(next && keys(next)).toEqual(['service:c', 'service:b', 'service:a', 'widget:w'])
  })

  it('moves a widget past a service', () => {
    const next = reorderTiles(all, all, 'widget:w', 'service:a')
    expect(next && keys(next)).toEqual(['widget:w', 'service:a', 'service:b', 'service:c'])
  })

  it('returns null when nothing moves or a key is unknown', () => {
    expect(reorderTiles(all, all, 'service:a', 'service:a')).toBeNull()
    expect(reorderTiles(all, all, 'service:x', 'service:a')).toBeNull()
    expect(reorderTiles(all, visible, 'service:b', 'service:a')).toBeNull()
  })
})

describe('toTilePositions and splitTiles', () => {
  it('number every tile by its index', () => {
    // Positions with a gap and a duplicate
    const tiles = mergeTiles([service('a', 0), service('b', 5)], [widget('w', 5)])
    expect(toTilePositions(tiles)).toEqual([
      { id: 'a', kind: 'service', position: 0 },
      { id: 'b', kind: 'service', position: 1 },
      { id: 'w', kind: 'widget', position: 2 },
    ])

    const { services, widgets } = splitTiles(tiles)
    expect(services.map((s) => [s.id, s.position])).toEqual([
      ['a', 0],
      ['b', 1],
    ])
    expect(widgets.map((w) => [w.id, w.position])).toEqual([['w', 2]])
  })
})
