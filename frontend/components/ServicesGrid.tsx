'use client'

import type { Service, CardSize, CardScale, ViewMode } from '@/types'
import ServiceCard from '@/components/ServiceCard'
import SortableServiceCard from '@/components/SortableServiceCard'
import ServicesList from '@/components/ServicesList'
import { isServiceEffectivelyMonitored, type GroupMonitoringMap } from '@/lib/monitoring'

// Grid classes for different card scales
// Mobile always uses large layout (grid-cols-2) to avoid clutter
// Scale-specific columns apply from sm breakpoint and up
const gridClasses: Record<CardScale, string> = {
  small: 'grid-cols-2 gap-4 sm:grid-cols-6 sm:gap-2 xl:grid-cols-8 2xl:grid-cols-10',
  medium: 'grid-cols-2 gap-4 sm:grid-cols-4 sm:gap-3 xl:grid-cols-6 2xl:grid-cols-8',
  large: 'grid-cols-2 gap-4 sm:grid-cols-4 xl:grid-cols-6 2xl:grid-cols-8',
}

interface ServicesGridProps {
  services: Service[]
  openInNewTab: boolean
  enableCardResizing: boolean
  cardScale: CardScale
  viewMode: ViewMode
  isEditMode?: boolean
  onSizeChange?: (id: string, newSize: CardSize) => void
  groupMonitoringMap?: GroupMonitoringMap
}

export default function ServicesGrid({
  services,
  openInNewTab,
  enableCardResizing,
  cardScale,
  viewMode,
  isEditMode = false,
  onSizeChange,
  groupMonitoringMap,
}: ServicesGridProps) {
  // Render list view if viewMode is 'list'
  if (viewMode === 'list') {
    return (
      <ServicesList
        services={services}
        openInNewTab={openInNewTab}
        isEditMode={isEditMode}
        groupMonitoringMap={groupMonitoringMap}
      />
    )
  }

  // Render grid view
  const gridContent = services.map((service) => {
    const monitored = isServiceEffectivelyMonitored(service, groupMonitoringMap)
    return isEditMode && onSizeChange ? (
      <SortableServiceCard
        key={service.id}
        service={service}
        openInNewTab={openInNewTab}
        isEditMode={isEditMode}
        onSizeChange={onSizeChange}
        enableCardResizing={enableCardResizing}
        cardScale={cardScale}
        isMonitored={monitored}
      />
    ) : (
      <ServiceCard
        key={service.id}
        service={service}
        openInNewTab={openInNewTab}
        enableCardResizing={enableCardResizing}
        cardScale={cardScale}
        isMonitored={monitored}
      />
    )
  })

  return (
    <div className={`grid ${gridClasses[cardScale]}`} style={{ gridAutoFlow: 'dense' }}>
      {gridContent}
    </div>
  )
}
