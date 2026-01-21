'use client'

import type { Service, CardSize } from '@/types'
import ServiceCard from '@/components/ServiceCard'
import SortableServiceCard from '@/components/SortableServiceCard'

interface ServicesGridProps {
  services: Service[]
  openInNewTab: boolean
  enableCardResizing: boolean
  isEditMode?: boolean
  onSizeChange?: (id: string, newSize: CardSize) => void
}

export default function ServicesGrid({
  services,
  openInNewTab,
  enableCardResizing,
  isEditMode = false,
  onSizeChange,
}: ServicesGridProps) {
  const gridContent = services.map((service) =>
    isEditMode && onSizeChange ? (
      <SortableServiceCard
        key={service.id}
        service={service}
        openInNewTab={openInNewTab}
        isEditMode={isEditMode}
        onSizeChange={onSizeChange}
        enableCardResizing={enableCardResizing}
      />
    ) : (
      <ServiceCard
        key={service.id}
        service={service}
        openInNewTab={openInNewTab}
        enableCardResizing={enableCardResizing}
      />
    )
  )

  return (
    <div
      className="grid grid-cols-2 gap-4 sm:grid-cols-4 xl:grid-cols-6 2xl:grid-cols-8"
      style={{ gridAutoFlow: 'dense' }}
    >
      {gridContent}
    </div>
  )
}
