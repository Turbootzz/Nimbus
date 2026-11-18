import { closestCenter, pointerWithin, type CollisionDetection } from '@dnd-kit/core'
import { GROUP_DROP_PREFIX } from '@/lib/tiles'

const isGroupDrop = (id: string | number) => String(id).startsWith(GROUP_DROP_PREFIX)

/**
 * Collision detection for the dashboard. Group tabs only collide with other
 * tabs. A dragged tile prefers what is under the pointer: with tabs that is
 * a group tab, with sections (`sectionOfTile`: tile key to group id) a tile,
 * and a section itself only when the tile comes from another one. In the
 * empty space of its own section the tile stays with that section's tiles.
 */
export function createCollisionDetection(
  isDraggingTab: boolean,
  groupIds: string[],
  sectionOfTile?: Map<string, string>
): CollisionDetection {
  return (args) => {
    const activeId = String(args.active.id)

    // If dragging a tab, only collide with other tabs
    if (isDraggingTab || groupIds.includes(activeId)) {
      return closestCenter(args)
    }

    const pointerCollisions = pointerWithin(args)
    const groupCollision = pointerCollisions.find((c) => isGroupDrop(c.id))

    if (sectionOfTile) {
      const tileCollision = pointerCollisions.find((c) => !isGroupDrop(c.id))
      if (tileCollision) return [tileCollision]
      const own = sectionOfTile.get(activeId)
      const inOwnSection = groupCollision?.id === GROUP_DROP_PREFIX + own
      if (groupCollision && !inOwnSection) return [groupCollision]
      // Between tiles: the nearest tile, of its own section while the pointer
      // is in it, or a tile of the next section could be nearer
      return closestCenter({
        ...args,
        droppableContainers: args.droppableContainers.filter(
          (c) => !isGroupDrop(c.id) && (!inOwnSection || sectionOfTile.get(String(c.id)) === own)
        ),
      })
    }

    // For tile drags: prioritize group tabs (pointer-based), then the grid (center-based)
    if (groupCollision) {
      return [groupCollision]
    }

    // Fall back to closestCenter for tile reordering
    return closestCenter(args)
  }
}
