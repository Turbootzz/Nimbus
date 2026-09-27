import { arrayMove } from '@dnd-kit/sortable'
import type { Group, Service, Tile, TilePosition, Widget } from '@/types'

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
