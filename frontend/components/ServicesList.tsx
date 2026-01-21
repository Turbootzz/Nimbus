'use client'

import { ClockIcon } from '@heroicons/react/24/outline'
import type { Service } from '@/types'
import { getStatusColor, getStatusIcon } from '@/lib/status-utils'
import ServiceIcon from '@/components/ServiceIcon'

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
    <div className="space-y-2">
      {services.map((service) => (
        <ServiceListItem
          key={service.id}
          service={service}
          openInNewTab={openInNewTab}
          isEditMode={isEditMode}
        />
      ))}
    </div>
  )
}

interface ServiceListItemProps {
  service: Service
  openInNewTab: boolean
  isEditMode: boolean
}

function ServiceListItem({ service, openInNewTab, isEditMode }: ServiceListItemProps) {
  const linkProps = {
    href: service.url,
    target: openInNewTab ? '_blank' : '_self',
    ...(openInNewTab && { rel: 'noopener noreferrer' }),
  }

  const baseClasses =
    'bg-card border-card-border flex items-center gap-4 rounded-lg border p-4 transition-all'
  const hoverClasses = isEditMode ? '' : 'hover:border-primary hover:shadow-md'
  const editClasses = isEditMode ? 'border-dashed border-2' : ''

  const content = (
    <>
      <ServiceIcon service={service} size="md" />

      <div className="min-w-0 flex-1">
        <h3 className="text-text-primary truncate font-semibold">{service.name}</h3>
        {service.description && (
          <p className="text-text-secondary truncate text-sm">{service.description}</p>
        )}
      </div>

      <div className={`flex shrink-0 items-center gap-1 ${getStatusColor(service.status)}`}>
        {getStatusIcon(service.status)}
        <span className="text-sm capitalize">{service.status}</span>
      </div>

      {service.response_time !== undefined && service.response_time !== null && (
        <div className="text-text-muted flex shrink-0 items-center gap-1 text-sm">
          <ClockIcon className="h-4 w-4" />
          {service.response_time}ms
        </div>
      )}
    </>
  )

  if (isEditMode) {
    return <div className={`${baseClasses} ${editClasses}`}>{content}</div>
  }

  return (
    <a {...linkProps} className={`${baseClasses} ${hoverClasses}`}>
      {content}
    </a>
  )
}
