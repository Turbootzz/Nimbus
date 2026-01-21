'use client'

import { useSortable } from '@dnd-kit/sortable'
import type { Service, CardSize } from '@/types'
import { sizeToGridSpan } from '@/lib/card-utils'
import ServiceCard from '@/components/ServiceCard'

interface SortableServiceCardProps {
  service: Service
  openInNewTab: boolean
  isEditMode: boolean
  onSizeChange: (id: string, newSize: CardSize) => void
  enableCardResizing: boolean
}

export default function SortableServiceCard({
  service,
  openInNewTab,
  isEditMode,
  onSizeChange,
  enableCardResizing,
}: SortableServiceCardProps) {
  const { attributes, listeners, setNodeRef, isDragging } = useSortable({
    id: service.id,
  })

  // When card resizing is disabled, always use 2x1
  const cardSize = enableCardResizing ? service.card_size || '2x1' : '2x1'
  const gridSpan = sizeToGridSpan[cardSize]

  // Don't apply transforms - with dense grid and variable sizes, transforms cause
  // weird stretching/compression. Cards stay in place, only reorder after drop.
  const style = {
    opacity: isDragging ? 0.4 : 1,
    transition: 'opacity 150ms ease',
  }

  return (
    <div ref={setNodeRef} style={style} className={`${gridSpan} h-full`}>
      <ServiceCard
        service={service}
        openInNewTab={openInNewTab}
        isEditMode={isEditMode}
        onSizeChange={onSizeChange}
        dragHandleProps={{ ...attributes, ...listeners }}
        isDragging={false}
        enableCardResizing={enableCardResizing}
      />
    </div>
  )
}
