import { closestCenter, pointerWithin, type CollisionDetection } from '@dnd-kit/core'
import { GROUP_DROP_PREFIX } from '@/lib/tiles'

const isGroupDrop = (id: string | number) => String(id).startsWith(GROUP_DROP_PREFIX)

/**
 * Collision detection for the dashboard. Group tabs only collide with other
 * tabs. A dragged tile prefers what is under the pointer: with tabs that is
 * a group tab, with sections (`sectionOfTile`: tile key to group id) a tile,
 * and a section itself only when the tile comes from another one.
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
      const ownSection = GROUP_DROP_PREFIX + sectionOfTile.get(activeId)
      if (groupCollision && groupCollision.id !== ownSection) return [groupCollision]
      // Between tiles: the nearest tile
      return closestCenter({
        ...args,
        droppableContainers: args.droppableContainers.filter((c) => !isGroupDrop(c.id)),
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
