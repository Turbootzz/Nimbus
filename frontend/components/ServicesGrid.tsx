'use client'

import type { Service, CardSize, CardScale, ViewMode } from '@/types'
import ServiceCard from '@/components/ServiceCard'
import SortableServiceCard from '@/components/SortableServiceCard'
import ServicesList from '@/components/ServicesList'

// Grid classes for different card scales
// Large = original layout, Medium = denser, Small = most dense
const gridClasses: Record<CardScale, string> = {
  small: 'grid-cols-4 sm:grid-cols-6 xl:grid-cols-8 2xl:grid-cols-10 gap-2',
  medium: 'grid-cols-3 sm:grid-cols-4 xl:grid-cols-6 2xl:grid-cols-8 gap-3',
  large: 'grid-cols-2 sm:grid-cols-4 xl:grid-cols-6 2xl:grid-cols-8 gap-4',
}

interface ServicesGridProps {
  services: Service[]
  openInNewTab: boolean
  enableCardResizing: boolean
  cardScale: CardScale
  viewMode: ViewMode
  isEditMode?: boolean
  onSizeChange?: (id: string, newSize: CardSize) => void
}

export default function ServicesGrid({
  services,
  openInNewTab,
  enableCardResizing,
  cardScale,
  viewMode,
  isEditMode = false,
  onSizeChange,
}: ServicesGridProps) {
  // Render list view if viewMode is 'list'
  if (viewMode === 'list') {
    return <ServicesList services={services} openInNewTab={openInNewTab} isEditMode={isEditMode} />
  }

  // Render grid view
  const gridContent = services.map((service) =>
    isEditMode && onSizeChange ? (
      <SortableServiceCard
        key={service.id}
        service={service}
        openInNewTab={openInNewTab}
        isEditMode={isEditMode}
        onSizeChange={onSizeChange}
        enableCardResizing={enableCardResizing}
        cardScale={cardScale}
      />
    ) : (
      <ServiceCard
        key={service.id}
        service={service}
        openInNewTab={openInNewTab}
        enableCardResizing={enableCardResizing}
        cardScale={cardScale}
      />
    )
  )

  return (
    <div className={`grid ${gridClasses[cardScale]}`} style={{ gridAutoFlow: 'dense' }}>
      {gridContent}
    </div>
  )
}
