import { describe, it, expect } from 'vitest'
import {
  groupTilesBySection,
  mergeTiles,
  moveTile,
  resolveGroupDisplay,
  sortableOrder,
  tileGroupId,
  tileKey,
  toTilePositions,
  undoMove,
  withGroup,
} from '@/lib/tiles'
import type { Group, Service, Tile, Widget } from '@/types'

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

const keys = (tiles: Tile[]) => tiles.map(tileKey)

const groups = [group('home', true), group('media'), group('empty')]
// Order: a (ungrouped), m1, w (media), b (home), m2, gone (deleted group)
const all = mergeTiles(
  [
    service('a', 0),
    service('m1', 1, 'media'),
    service('b', 3, 'home'),
    service('m2', 4, 'media'),
    service('gone', 5, 'deleted'),
  ],
  [widget('w', 2, 'media')]
)
const sections = groupTilesBySection(all, groups)

describe('resolveGroupDisplay', () => {
  it('follows the layout until the user chooses', () => {
    expect(resolveGroupDisplay(null, 'classic')).toBe('tabs')
    expect(resolveGroupDisplay(undefined, 'canvas')).toBe('sections')
  })

  it('keeps the choice in either layout', () => {
    expect(resolveGroupDisplay('tabs', 'canvas')).toBe('tabs')
    expect(resolveGroupDisplay('sections', 'classic')).toBe('sections')
  })
})

describe('groupTilesBySection', () => {
  it('makes one section per group, in group order, also when empty', () => {
    expect(sections.map((s) => s.group.id)).toEqual(['home', 'media', 'empty'])
    expect(sections[2].tiles).toEqual([])
  })

  it('keeps services and widgets of a group together in list order', () => {
    expect(keys(sections[1].tiles)).toEqual(['service:m1', 'widget:w', 'service:m2'])
  })

  it('puts ungrouped tiles and tiles of a missing group in the default section', () => {
    expect(keys(sections[0].tiles)).toEqual(['service:a', 'service:b', 'service:gone'])
  })

  it('has no sections without groups', () => {
    expect(groupTilesBySection(all, [])).toEqual([])
  })
})

describe('moveTile', () => {
  it('reorders within a section and leaves the other sections in their slots', () => {
    // media: m1, w, m2; m2 goes before m1
    const move = moveTile(all, sections, 'service:m2', 'service:m1')
    expect(move?.regrouped).toBeUndefined()
    expect(move && keys(move.tiles)).toEqual([
      'service:a',
      'service:m2',
      'service:m1',
      'service:b',
      'widget:w',
      'service:gone',
    ])
  })

  it('moves a service to another section, before the tile it was dropped on', () => {
    const move = moveTile(all, sections, 'service:b', 'widget:w')
    expect(move && keys(move.tiles)).toEqual([
      'service:a',
      'service:m1',
      'service:b',
      'widget:w',
      'service:m2',
      'service:gone',
    ])
    expect(move?.regrouped?.group.id).toBe('media')
    expect(move?.regrouped && tileGroupId(move.regrouped.tile)).toBe('media')
    // The moved tile in the new order carries the group too
    expect(tileGroupId(move!.tiles[2])).toBe('media')
  })

  it('moves a widget to the default section, which clears its group', () => {
    const move = moveTile(all, sections, 'widget:w', 'service:b')
    expect(move && keys(move.tiles)).toEqual([
      'service:a',
      'service:m1',
      'widget:w',
      'service:b',
      'service:m2',
      'service:gone',
    ])
    expect(move?.regrouped?.group.id).toBe('home')
    expect(move?.regrouped && tileGroupId(move.regrouped.tile)).toBeNull()
  })

  it('appends after the last tile when dropped on a section itself', () => {
    const move = moveTile(all, sections, 'service:a', 'group-media')
    expect(move && keys(move.tiles)).toEqual([
      'service:m1',
      'widget:w',
      'service:b',
      'service:m2',
      'service:a',
      'service:gone',
    ])
    expect(move?.regrouped?.group.id).toBe('media')
  })

  it('keeps the slot when dropped on an empty section', () => {
    const move = moveTile(all, sections, 'service:m1', 'group-empty')
    expect(move && keys(move.tiles)).toEqual(keys(all))
    expect(move?.regrouped && tileGroupId(move.regrouped.tile)).toBe('empty')
  })

  it('keeps positions unique and complete after a move between sections', () => {
    const move = moveTile(all, sections, 'service:m2', 'service:a')
    const positions = toTilePositions(move!.tiles)
    expect(positions.map((p) => p.position)).toEqual([0, 1, 2, 3, 4, 5])
    expect(new Set(positions.map((p) => `${p.kind}:${p.id}`)).size).toBe(all.length)
  })

  it('only moves among the tiles that can be dragged', () => {
    // List view drags services only; the widget keeps its slot
    const services = all.filter((t) => t.kind === 'service')
    const move = moveTile(all, groupTilesBySection(services, groups), 'service:m2', 'service:m1')
    expect(move && keys(move.tiles)).toEqual([
      'service:a',
      'service:m2',
      'widget:w',
      'service:b',
      'service:m1',
      'service:gone',
    ])
  })

  it('returns null when nothing moves or an id is unknown', () => {
    expect(moveTile(all, sections, 'service:a', 'service:a')).toBeNull()
    expect(moveTile(all, sections, 'service:a', 'group-home')).toBeNull()
    expect(moveTile(all, sections, 'service:nope', 'service:a')).toBeNull()
    expect(moveTile(all, sections, 'service:a', 'service:nope')).toBeNull()
    expect(moveTile(all, sections, 'service:a', 'group-nope')).toBeNull()
  })
})

describe('undoMove', () => {
  // a and b both go to media; the save of a fails after b moved
  const afterA = moveTile(all, sections, 'service:a', 'group-media')!.tiles
  const afterB = moveTile(
    afterA,
    groupTilesBySection(afterA, groups),
    'service:b',
    'group-media'
  )!.tiles
  const groupOf = (tiles: Tile[], key: string) =>
    tileGroupId(tiles.find((t) => tileKey(t) === key)!)

  it.each([
    ['a reorder within a section', 'service:a', 'service:b'],
    ['a move to the front', 'service:gone', 'service:a'],
    ['a move to another section', 'service:a', 'group-media'],
  ])('puts every tile back after %s', (_, active, over) => {
    const shown = moveTile(all, sections, active, over)!.tiles
    const undone = undoMove(shown, shown, all, active)
    expect(keys(undone)).toEqual(keys(all))
    expect(groupOf(undone, active)).toBe(groupOf(all, active))
  })

  it('puts back only the tile whose save failed, after a newer move', () => {
    const undone = undoMove(afterB, afterA, all, 'service:a')
    expect(keys(undone)).toEqual([
      'service:a',
      'service:m1',
      'widget:w',
      'service:m2',
      'service:b',
      'service:gone',
    ])
    expect(groupOf(undone, 'service:a')).toBeUndefined()
    // b keeps its move
    expect(groupOf(undone, 'service:b')).toBe('media')
  })

  it('keeps the other changes to the tiles', () => {
    const resized = afterA.map((t) =>
      t.kind === 'widget' ? { ...t, widget: { ...t.widget, card_size: '2x2' as const } } : t
    )
    const undone = undoMove(resized, afterA, all, 'service:a')
    expect(keys(undone)).toEqual(keys(all))
    expect(undone.find((t) => t.kind === 'widget')).toMatchObject({ widget: { card_size: '2x2' } })
  })

  it('keeps a group that was saved when only the order failed', () => {
    // The tiles when b was dropped, with the group that was saved
    const saved = afterA.map((t) => (tileKey(t) === 'service:b' ? withGroup(t, groups[1]) : t))
    const undone = undoMove(afterB, afterB, saved, 'service:b')
    expect(keys(undone)).toEqual(keys(afterA))
    expect(groupOf(undone, 'service:b')).toBe('media')
  })

  it('leaves the tiles alone for an unknown tile', () => {
    expect(undoMove(afterA, afterA, all, 'service:nope')).toBe(afterA)
  })
})

describe('sortableOrder', () => {
  const tiles = mergeTiles([service('a', 0), service('b', 2)], [widget('w', 1)])
  const keys = (list: Tile[]) => list.map(tileKey)

  it('follows the render order of each view', () => {
    expect(keys(sortableOrder(tiles, 'grid'))).toEqual(['service:a', 'widget:w', 'service:b'])
    expect(keys(sortableOrder(tiles, 'list'))).toEqual(['service:a', 'service:b'])
    // Chips come first, the widgets are cards below them
    expect(keys(sortableOrder(tiles, 'grid', 'chips'))).toEqual([
      'service:a',
      'service:b',
      'widget:w',
    ])
    expect(keys(sortableOrder(tiles, 'list', 'chips'))).toEqual(['service:a', 'service:b'])
  })
})
