'use client'

import { ClockIcon, PencilIcon, TrashIcon } from '@heroicons/react/24/outline'
import Link from 'next/link'
import type { Service } from '@/types'
import { useTheme } from '@/contexts/ThemeContext'
import { getStatusColor, getStatusIcon, getResponseTimeColor } from '@/lib/status-utils'
import { formatRelativeTime } from '@/lib/date-utils'
import ServiceIcon from '@/components/ServiceIcon'

interface ServiceManagementCardPresentationProps {
  service: Service
  onDelete: (id: string, name: string) => void
  style?: React.CSSProperties
}

// Presentation-only component for DragOverlay (no useSortable hook)
export function ServiceManagementCardPresentation({
  service,
  onDelete,
  style,
}: ServiceManagementCardPresentationProps) {
  const { openInNewTab } = useTheme()

  return (
    <div
      style={style}
      className="bg-card border-card-border group relative rounded-lg border p-6 transition-all hover:shadow-lg"
    >
      {/* Service icon and status */}
      <div className="mb-4 flex items-start justify-between">
        <ServiceIcon service={service} size="md" />
        <div className="flex items-center gap-3">
          <div className={`order-1 flex items-center ${getStatusColor(service.status)}`}>
            {getStatusIcon(service.status)}
            <span className="ml-1 text-sm capitalize">{service.status}</span>
          </div>
        </div>
      </div>

      {/* Service info */}
      <h3 className="text-text-primary mb-1 text-lg font-semibold">{service.name}</h3>
      <p className="text-text-secondary mb-2 text-sm">{service.description || 'No description'}</p>
      <a
        href={service.url}
        target={openInNewTab ? '_blank' : '_self'}
        {...(openInNewTab && { rel: 'noopener noreferrer' })}
        className="text-primary hover:text-primary-hover mb-2 block truncate text-xs transition-colors"
      >
        {service.url}
      </a>

      {/* Response time and last checked */}
      <div className="text-text-muted mb-4 space-y-1 text-xs">
        {service.response_time !== undefined && service.response_time !== null && (
          <div className="flex items-center">
            <span className="mr-2">Response:</span>
            <span className={`${getResponseTimeColor(service.response_time)} font-medium`}>
              {service.response_time}ms
            </span>
          </div>
        )}
        {service.updated_at && service.status !== 'unknown' && (
          <div className="flex items-center">
            <ClockIcon className="mr-1 h-3 w-3" />
            Last checked: {formatRelativeTime(service.updated_at)}
          </div>
        )}
      </div>

      {/* Actions */}
      <div
        className="flex items-center gap-2 border-t pt-4"
        style={{ borderColor: 'var(--color-card-border)' }}
      >
        <Link
          href={`/services/${service.id}/edit`}
          className="hover:bg-card-border text-text-secondary hover:text-text-primary flex flex-1 items-center justify-center rounded-md px-3 py-2 text-sm font-medium transition-colors"
        >
          <PencilIcon className="mr-1 h-4 w-4" />
          Edit
        </Link>
        <button
          onClick={() => onDelete(service.id, service.name)}
          className="hover:bg-error text-error flex flex-1 items-center justify-center rounded-md px-3 py-2 text-sm font-medium transition-colors hover:text-white"
        >
          <TrashIcon className="mr-1 h-4 w-4" />
          Delete
        </button>
      </div>
    </div>
  )
}
