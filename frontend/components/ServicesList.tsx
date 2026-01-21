'use client'

import type { Service } from '@/types'
import ServiceListItem from '@/components/ServiceListItem'
import SortableServiceListItem from '@/components/SortableServiceListItem'

interface ServicesListProps {
  services: Service[]
  openInNewTab: boolean
  isEditMode?: boolean
}

export default function ServicesList({
  services,
  openInNewTab,
  isEditMode = false,
}: ServicesListProps) {
  return (
    <div className="grid grid-cols-1 gap-x-3 gap-y-1 lg:grid-cols-2 2xl:grid-cols-3">
      {services.map((service) =>
        isEditMode ? (
          <SortableServiceListItem key={service.id} service={service} openInNewTab={openInNewTab} />
        ) : (
          <ServiceListItem key={service.id} service={service} openInNewTab={openInNewTab} />
        )
      )}
    </div>
  )
}
