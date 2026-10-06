'use client'

import type { ReactNode } from 'react'
import { useSortable } from '@dnd-kit/sortable'
import type { CardSize } from '@/types'
import { sizeToGridSpan } from '@/lib/card-utils'

interface SortableTileProps {
  id: string
  cardSize: CardSize
  children: (dragHandleProps: Record<string, unknown>) => ReactNode
}

// Makes any dashboard tile draggable in edit mode. The wrapper does the grid
// spanning, so the card inside just fills it.
export default function SortableTile({ id, cardSize, children }: SortableTileProps) {
  const { attributes, listeners, setNodeRef, isDragging } = useSortable({ id })

  // Don't apply transforms - with dense grid and variable sizes, transforms cause
  // weird stretching/compression. Tiles stay in place, only reorder after drop.
  const style = {
    opacity: isDragging ? 0.4 : 1,
    transition: 'opacity 150ms ease',
  }

  return (
    <div ref={setNodeRef} style={style} className={`${sizeToGridSpan[cardSize]} h-full`}>
      {children({ ...attributes, ...listeners })}
    </div>
  )
}
