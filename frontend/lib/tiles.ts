import { arrayMove } from '@dnd-kit/sortable'
import type { Group, GroupDisplay, LayoutMode, Service, Tile, TilePosition, Widget } from '@/types'

/**
 * Unique key for a tile across services and widgets, used as the dnd-kit id
 */
export function tileKey(tile: Pick<Tile, 'kind' | 'id'>): string {
  return `${tile.kind}:${tile.id}`
}

export function tileGroupId(tile: Tile): string | null | undefined {
  return tile.kind === 'service' ? tile.service.group_id : tile.widget.group_id
}

/**
 * Puts services and widgets in one list sorted by position. Sorting is
 * stable, so on equal positions services come first, then input order.
 */
export function mergeTiles(services: Service[], widgets: Widget[]): Tile[] {
  const tiles: Tile[] = [
    ...services.map((service) => ({
      kind: 'service' as const,
      id: service.id,
      position: service.position,
      service,
    })),
    ...widgets.map((widget) => ({
      kind: 'widget' as const,
      id: widget.id,
      position: widget.position,
      widget,
    })),
  ]
  return tiles.sort((a, b) => a.position - b.position)
}

/**
 * Tiles in the selected group. The default group also shows ungrouped tiles,
 * like services always did. No selection shows everything.
 */
export function filterTilesByGroup(
  tiles: Tile[],
  selectedGroupId: string | null,
  groups: Group[]
): Tile[] {
  if (!selectedGroupId) return tiles
  const isDefault = groups.find((g) => g.id === selectedGroupId)?.is_default ?? false
  return tiles.filter((tile) => {
    const groupId = tileGroupId(tile)
    return groupId === selectedGroupId || (isDefault && !groupId)
  })
}

/**
 * Moves the tile `activeKey` to the place of `overKey` within `visible`, a
 * subset of `all` in the same order. Tiles outside `visible` keep their
 * slots. Returns the new full order, or null when nothing moves.
 */
export function reorderTiles(
  all: Tile[],
  visible: Tile[],
  activeKey: string,
  overKey: string
): Tile[] | null {
  const from = visible.findIndex((t) => tileKey(t) === activeKey)
  const to = visible.findIndex((t) => tileKey(t) === overKey)
  if (from === -1 || to === -1 || from === to) return null

  const moved = arrayMove(visible, from, to)
  const visibleKeys = new Set(visible.map(tileKey))
  let next = 0
  return all.map((tile) => (visibleKeys.has(tileKey(tile)) ? moved[next++] : tile))
}

/**
 * Reorder payload that numbers every tile 0..n-1 in list order. Sending all
 * tiles keeps positions unique, even if older data has gaps or duplicates.
 */
export function toTilePositions(tiles: Tile[]): TilePosition[] {
  return tiles.map((tile, position) => ({ id: tile.id, kind: tile.kind, position }))
}

/**
 * Splits tiles back into services and widgets, with positions set to their
 * index so they match what toTilePositions sends.
 */
export function splitTiles(tiles: Tile[]): { services: Service[]; widgets: Widget[] } {
  const services: Service[] = []
  const widgets: Widget[] = []
  tiles.forEach((tile, position) => {
    if (tile.kind === 'service') {
      services.push({ ...tile.service, position })
    } else {
      widgets.push({ ...tile.widget, position })
    }
  })
  return { services, widgets }
}

/**
 * How groups show: the user's choice, or else what fits the layout (tabs in
 * classic, sections in canvas).
 */
export function resolveGroupDisplay(
  choice: GroupDisplay | null | undefined,
  layoutMode: LayoutMode
): GroupDisplay {
  return choice ?? (layoutMode === 'canvas' ? 'sections' : 'tabs')
}

export interface TileSection {
  group: Group
  tiles: Tile[]
}

/** Droppable id of a group: its tab, or its section */
export const GROUP_DROP_PREFIX = 'group-'

/**
 * One section per group, in group order, each with its tiles in list order.
 * Ungrouped tiles, and tiles of a group that is gone, go to the default
 * group's section.
 */
export function groupTilesBySection(tiles: Tile[], groups: Group[]): TileSection[] {
  const sections: TileSection[] = groups.map((group) => ({ group, tiles: [] }))
  const byGroup = new Map(sections.map((section) => [section.group.id, section]))
  const fallback = sections.find((section) => section.group.is_default) ?? sections[0]
  for (const tile of tiles) {
    const section = byGroup.get(tileGroupId(tile) ?? '') ?? fallback
    section?.tiles.push(tile)
  }
  return sections
}

// The default group holds the ungrouped tiles, so moving there clears the group
function withGroup(tile: Tile, group: Group): Tile {
  const groupId = group.is_default ? undefined : group.id
  return tile.kind === 'service'
    ? { ...tile, service: { ...tile.service, group_id: groupId } }
    : { ...tile, widget: { ...tile.widget, group_id: groupId ?? null } }
}

export interface TileMove {
  // New full order; the moved tile already has its new group
  tiles: Tile[]
  // Set when the tile changed section: the tile, and the group it joined
  regrouped?: { tile: Tile; group: Group }
}

/**
 * Drops tile `activeKey` on `overId` in sections mode. `overId` is another
 * tile, or a section (GROUP_DROP_PREFIX + group id) when the drop was on its
 * header or empty space. `sections` hold the tiles that can be dragged, a
 * subset of `all` in the same order.
 *
 * Within a section this reorders like reorderTiles. In another section the
 * tile joins that group, before the tile it was dropped on or after the
 * section's last tile. Everything else keeps its place in the one order that
 * services and widgets share. Returns null when nothing moves.
 */
export function moveTile(
  all: Tile[],
  sections: TileSection[],
  activeKey: string,
  overId: string
): TileMove | null {
  const sectionOf = (key: string) =>
    sections.find((section) => section.tiles.some((tile) => tileKey(tile) === key))
  const from = sectionOf(activeKey)
  const active = from?.tiles.find((tile) => tileKey(tile) === activeKey)
  if (!from || !active) return null

  const onSection = overId.startsWith(GROUP_DROP_PREFIX)
  const to = onSection
    ? sections.find((section) => section.group.id === overId.slice(GROUP_DROP_PREFIX.length))
    : sectionOf(overId)
  if (!to) return null

  if (to === from) {
    const tiles = onSection ? null : reorderTiles(all, from.tiles, activeKey, overId)
    return tiles && { tiles }
  }

  const moved = withGroup(active, to.group)
  const rest = all.filter((tile) => tileKey(tile) !== activeKey)
  const last = to.tiles.at(-1)
  let index: number
  if (!onSection) {
    index = rest.findIndex((tile) => tileKey(tile) === overId)
  } else if (last) {
    index = rest.findIndex((tile) => tileKey(tile) === tileKey(last)) + 1
  } else {
    // An empty section has no neighbours, so the tile keeps its slot
    index = all.findIndex((tile) => tileKey(tile) === activeKey)
  }
  return {
    tiles: [...rest.slice(0, index), moved, ...rest.slice(index)],
    regrouped: { tile: moved, group: to.group },
  }
}
