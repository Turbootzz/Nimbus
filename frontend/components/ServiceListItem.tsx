'use client'

import { ClockIcon } from '@heroicons/react/24/outline'
import { Bars3Icon } from '@heroicons/react/24/solid'
import type { Service } from '@/types'
import { getStatusBgColor, getResponseTimeColor } from '@/lib/status-utils'
import ServiceIcon from '@/components/ServiceIcon'

interface ServiceListItemProps {
  service: Service
  openInNewTab: boolean
  isEditMode?: boolean
  dragHandleProps?: Record<string, unknown>
  isDragging?: boolean
}

export default function ServiceListItem({
  service,
  openInNewTab,
  isEditMode = false,
  dragHandleProps,
  isDragging = false,
}: ServiceListItemProps) {
  const linkProps = {
    href: service.url,
    target: openInNewTab ? '_blank' : '_self',
    ...(openInNewTab && { rel: 'noopener noreferrer' }),
  }

  // Compact styling for list view
  const baseClasses =
    'bg-card border-card-border flex items-center gap-2 sm:gap-3 rounded-lg border p-2 sm:p-3 transition-all'
  const hoverClasses = isEditMode ? '' : 'hover:border-primary hover:shadow-md'
  const editClasses = isEditMode ? 'border-dashed border-2 cursor-pointer' : ''
  const dragClasses = isDragging ? 'ring-2 ring-primary' : ''

  const content = (
    <>
      {/* Drag handle - only visible in edit mode */}
      {isEditMode && (
        <div
          {...dragHandleProps}
          className="bg-card/90 cursor-grab touch-none rounded p-1 active:cursor-grabbing"
          onClick={(e) => e.stopPropagation()}
          onTouchStart={(e) => e.stopPropagation()}
        >
          <Bars3Icon className="text-text-muted h-4 w-4" />
        </div>
      )}

      {/* Service icon - smaller for compact view */}
      <ServiceIcon service={service} size="sm" />

      {/* Service name and description */}
      <div className="min-w-0 flex-1">
        <h3 className="text-text-primary truncate text-sm font-semibold">{service.name}</h3>
        {service.description && (
          <p className="text-text-secondary hidden truncate text-xs sm:block">
            {service.description}
          </p>
        )}
      </div>

      {/* Status indicator - compact dot style */}
      <div className="flex shrink-0 items-center gap-1.5">
        <div className={`h-2 w-2 rounded-full ${getStatusBgColor(service.status)}`} />
        <span className="text-text-secondary hidden text-xs capitalize sm:inline">
          {service.status}
        </span>
      </div>

      {/* Response time */}
      {service.response_time !== undefined && service.response_time !== null && (
        <div
          className={`flex shrink-0 items-center gap-1 text-xs ${getResponseTimeColor(service.response_time)}`}
        >
          <ClockIcon className="h-3 w-3" />
          <span className="hidden sm:inline">{service.response_time}ms</span>
          <span className="sm:hidden">{service.response_time}</span>
        </div>
      )}
    </>
  )

  if (isEditMode) {
    return <div className={`${baseClasses} ${editClasses} ${dragClasses}`}>{content}</div>
  }

  return (
    <a {...linkProps} className={`${baseClasses} ${hoverClasses}`}>
      {content}
    </a>
  )
}
