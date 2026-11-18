'use client'

import { PencilIcon } from '@heroicons/react/24/outline'
import { Bars3Icon } from '@heroicons/react/24/solid'
import type { CardScale, Service } from '@/types'
import { getStatusBgColor } from '@/lib/status-utils'
import ServiceIcon from '@/components/ServiceIcon'
import { useServiceDetail } from '@/components/ServiceHoverDetail'

interface ServiceChipProps {
  service: Service
  openInNewTab: boolean
  cardScale?: CardScale
  // Effective monitoring state (accounts for the group's monitoring flag too).
  // Defaults to service.monitoring_enabled when not provided.
  isMonitored?: boolean
  isEditMode?: boolean
  // Shows a pencil in edit mode
  onEdit?: (service: Service) => void
  dragHandleProps?: Record<string, unknown>
  isDragging?: boolean
}

// Density follows the card size setting
const scaleClasses: Record<CardScale, string> = {
  small: 'gap-1.5 px-2 py-1 text-xs',
  medium: 'gap-2 px-3 py-1.5 text-sm',
  large: 'gap-2.5 px-4 py-2 text-base',
}

// One-line tile for quick-launch groups: icon, name and status dot
export default function ServiceChip({
  service,
  openInNewTab,
  cardScale = 'medium',
  isMonitored,
  isEditMode = false,
  onEdit,
  dragHandleProps,
  isDragging = false,
}: ServiceChipProps) {
  const monitored = isMonitored ?? service.monitoring_enabled
  const status = monitored ? service.status : 'not monitored'
  // URL, latency, uptime and last error, on hover, focus or a long press
  const detail = useServiceDetail(service, monitored, !isEditMode)

  const baseClasses = `glass-card border-card-border flex max-w-full items-center rounded-full border transition-all ${scaleClasses[cardScale]}`

  const content = (
    <>
      <ServiceIcon service={service} size="xs" className="shrink-0" />
      <span className="text-text-primary truncate font-medium">{service.name}</span>
      <span
        className={`h-2 w-2 shrink-0 rounded-full ${monitored ? getStatusBgColor(service.status) : 'bg-gray-400'}`}
        aria-hidden="true"
      />
      <span className="sr-only">{status}</span>
    </>
  )

  if (isEditMode) {
    return (
      <div
        className={`${baseClasses} border-dashed ${isDragging ? 'ring-primary shadow-lg ring-2' : ''}`}
      >
        <div
          {...dragHandleProps}
          className="cursor-grab touch-none active:cursor-grabbing"
          onClick={(e) => e.stopPropagation()}
          onTouchStart={(e) => e.stopPropagation()}
        >
          <Bars3Icon className="text-text-muted h-4 w-4" />
        </div>
        {content}
        {onEdit && (
          <button
            type="button"
            onClick={() => onEdit(service)}
            className="text-text-muted hover:text-text-primary shrink-0 transition-colors"
            aria-label={`Edit ${service.name}`}
          >
            <PencilIcon className="h-4 w-4" />
          </button>
        )}
      </div>
    )
  }

  return (
    <>
      <a
        href={service.url}
        target={openInNewTab ? '_blank' : '_self'}
        {...(openInNewTab && { rel: 'noopener noreferrer' })}
        {...detail.triggerProps}
        className={`${baseClasses} hover:border-primary hover:shadow-md`}
      >
        {content}
      </a>
      {detail.popover}
    </>
  )
}
