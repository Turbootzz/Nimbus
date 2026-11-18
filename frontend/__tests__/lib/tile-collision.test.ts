import { describe, expect, it } from 'vitest'
import type { CollisionDetection } from '@dnd-kit/core'
import { createCollisionDetection } from '@/lib/tile-collision'

type Args = Parameters<CollisionDetection>[0]
type Box = [left: number, top: number, width: number, height: number]

const rect = ([left, top, width, height]: Box) => ({
  left,
  top,
  width,
  height,
  right: left + width,
  bottom: top + height,
})

// Two sections under each other, each with a header strip and two tiles;
// media also has room under its tiles
const boxes: Record<string, Box> = {
  'group-home': [0, 100, 400, 200],
  'service:a': [0, 140, 190, 100],
  'service:b': [210, 140, 190, 100],
  'group-media': [0, 320, 400, 300],
  'service:m': [0, 360, 190, 100],
  'widget:w': [210, 360, 190, 100],
}
const sectionOfTile = new Map([
  ['service:a', 'home'],
  ['service:b', 'home'],
  ['service:m', 'media'],
  ['widget:w', 'media'],
])

// A drag of `active` with the pointer (and the dragged tile's centre) at x, y
function drag(active: string, x: number, y: number, ids = Object.keys(boxes)): Args {
  const droppableRects = new Map(ids.map((id) => [id, rect(boxes[id])]))
  return {
    active: { id: active },
    collisionRect: rect([x - 95, y - 50, 190, 100]),
    droppableRects,
    droppableContainers: ids.map((id) => ({ id, rect: { current: droppableRects.get(id) } })),
    pointerCoordinates: { x, y },
  } as unknown as Args
}

const hit = (detect: CollisionDetection, args: Args) => detect(args)[0]?.id

describe('createCollisionDetection with sections', () => {
  const detect = createCollisionDetection(false, ['home', 'media'], sectionOfTile)

  it('picks the tile under the pointer, in the same or another section', () => {
    expect(hit(detect, drag('service:a', 300, 190))).toBe('service:b')
    expect(hit(detect, drag('service:a', 300, 400))).toBe('widget:w')
  })

  it('picks another section on its header or empty space', () => {
    expect(hit(detect, drag('service:a', 200, 330))).toBe('group-media')
    expect(hit(detect, drag('service:a', 200, 580))).toBe('group-media')
  })

  it('never picks the own section: between its tiles the nearest tile wins', () => {
    // On the header of home, closest to tile b
    expect(hit(detect, drag('service:a', 320, 120))).toBe('service:b')
    // In the gap between a and b, slightly towards b
    expect(hit(detect, drag('service:a', 205, 190))).toBe('service:b')
  })

  it('stays with its own tiles in the empty space of its own section', () => {
    // Bottom right of home, with tile b gone: widget w below is nearer than a
    const ids = Object.keys(boxes).filter((id) => id !== 'service:b')
    expect(hit(detect, drag('service:a', 300, 290, ids))).toBe('service:a')
  })

  it('falls back to the nearest tile outside every section', () => {
    expect(hit(detect, drag('service:m', 100, 60))).toBe('service:a')
  })
})

describe('createCollisionDetection with tabs', () => {
  const ids = ['group-home', 'service:a', 'service:b']
  const tabs: Record<string, Box> = { ...boxes, 'group-home': [0, 0, 100, 30] }
  const dragOnTabs = (active: string, x: number, y: number, over = ids) => {
    const args = drag(active, x, y, over)
    const droppableRects = new Map(over.map((id) => [id, rect(tabs[id] ?? boxes[id])]))
    return {
      ...args,
      droppableRects,
      droppableContainers: over.map((id) => ({ id, rect: { current: droppableRects.get(id) } })),
    } as unknown as Args
  }
  const detect = createCollisionDetection(false, ['home', 'media'])

  it('prefers the group tab under the pointer for a tile', () => {
    expect(hit(detect, dragOnTabs('service:a', 50, 15))).toBe('group-home')
  })

  it('reorders by the nearest tile otherwise', () => {
    expect(hit(detect, dragOnTabs('service:a', 300, 190))).toBe('service:b')
  })

  it('lets a dragged tab only meet other tabs', () => {
    const tabIds = ['home', 'media', 'service:a']
    const tabBoxes: Record<string, Box> = {
      home: [0, 0, 100, 30],
      media: [110, 0, 100, 30],
      'service:a': boxes['service:a'],
    }
    const droppableRects = new Map(tabIds.map((id) => [id, rect(tabBoxes[id])]))
    const args = {
      active: { id: 'home' },
      collisionRect: rect([100, 0, 100, 30]),
      droppableRects,
      droppableContainers: tabIds.map((id) => ({ id, rect: { current: droppableRects.get(id) } })),
      pointerCoordinates: { x: 150, y: 15 },
    } as unknown as Args
    expect(hit(createCollisionDetection(true, ['home', 'media']), args)).toBe('media')
  })
})
