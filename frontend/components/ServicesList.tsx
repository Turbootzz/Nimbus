'use client'

import type { Service } from '@/types'
import ServiceListItem from '@/components/ServiceListItem'
import SortableServiceListItem from '@/components/SortableServiceListItem'
import { isServiceEffectivelyMonitored, type GroupMonitoringMap } from '@/lib/monitoring'

interface ServicesListProps {
  services: Service[]
  openInNewTab: boolean
  isEditMode?: boolean
  groupMonitoringMap?: GroupMonitoringMap
}

export default function ServicesList({
  services,
  openInNewTab,
  isEditMode = false,
  groupMonitoringMap,
}: ServicesListProps) {
  return (
    <div className="grid grid-cols-1 gap-x-3 gap-y-1 lg:grid-cols-2 2xl:grid-cols-3">
      {services.map((service) => {
        const monitored = isServiceEffectivelyMonitored(service, groupMonitoringMap)
        return isEditMode ? (
          <SortableServiceListItem
            key={service.id}
            service={service}
            openInNewTab={openInNewTab}
            isMonitored={monitored}
          />
        ) : (
          <ServiceListItem
            key={service.id}
            service={service}
            openInNewTab={openInNewTab}
            isMonitored={monitored}
          />
        )
      })}
    </div>
  )
}
