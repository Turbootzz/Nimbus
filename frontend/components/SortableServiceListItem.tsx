'use client'

import { useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import type { Service } from '@/types'
import ServiceListItem from '@/components/ServiceListItem'

interface SortableServiceListItemProps {
  service: Service
  openInNewTab: boolean
}

export default function SortableServiceListItem({
  service,
  openInNewTab,
}: SortableServiceListItemProps) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: service.id,
  })

  // List items are uniform size, so CSS transforms work well for smooth animations
  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.4 : 1,
  }

  return (
    <div ref={setNodeRef} style={style}>
      <ServiceListItem
        service={service}
        openInNewTab={openInNewTab}
        isEditMode={true}
        dragHandleProps={{ ...attributes, ...listeners }}
        isDragging={isDragging}
      />
    </div>
  )
}
