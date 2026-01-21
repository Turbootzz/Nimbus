'use client'

import { useSortable } from '@dnd-kit/sortable'
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
  const { attributes, listeners, setNodeRef, isDragging } = useSortable({
    id: service.id,
  })

  const style = {
    opacity: isDragging ? 0.4 : 1,
    transition: 'opacity 150ms ease',
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
