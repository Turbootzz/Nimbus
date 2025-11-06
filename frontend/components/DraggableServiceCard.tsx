'use client'

import { useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { ClockIcon } from '@heroicons/react/24/outline'
import { Bars3Icon, ChartBarIcon } from '@heroicons/react/24/solid'
import Link from 'next/link'
import type { Service } from '@/types'
import { useTheme } from '@/contexts/ThemeContext'
import { getStatusColor, getStatusIcon, getResponseTimeColor } from '@/lib/status-utils'

interface DraggableServiceCardProps {
  service: Service
  isDragging?: boolean
}

export function DraggableServiceCard({ service, isDragging = false }: DraggableServiceCardProps) {
  const { openInNewTab } = useTheme()
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging: isSortableDragging,
  } = useSortable({
    id: service.id,
  })

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isSortableDragging || isDragging ? 0.5 : 1,
  }

  return (
    <div
      ref={setNodeRef}
      style={style}
      className="bg-card border-card-border hover:border-primary group relative rounded-lg border transition-all hover:shadow-lg"
    >
      {/* Action buttons */}
      <div className="absolute top-2 right-2 flex gap-1 opacity-0 transition-opacity group-hover:opacity-100 focus-within:opacity-100 [@media(hover:none)]:opacity-100">
        {/* Metrics button */}
        <Link
          href={`/services/${service.id}`}
          className="bg-card-hover focus:ring-primary rounded p-1 transition-colors hover:bg-gray-300 focus:ring-2 focus:outline-none dark:hover:bg-gray-600"
          title="View metrics"
          aria-label={`View metrics for ${service.name}`}
          onClick={(e) => e.stopPropagation()}
        >
          <ChartBarIcon className="text-text-muted h-5 w-5" />
        </Link>

        {/* Drag handle */}
        <button
          {...attributes}
          {...listeners}
          className="bg-card-hover focus:ring-primary cursor-grab rounded p-1 transition-colors hover:bg-gray-300 focus:ring-2 focus:outline-none active:cursor-grabbing dark:hover:bg-gray-600"
          title="Drag to reorder"
          aria-label={`Drag to reorder ${service.name}`}
          type="button"
        >
          <Bars3Icon className="text-text-muted h-5 w-5" />
        </button>
      </div>

      {/* Service card content */}
      <a
        href={service.url}
        target={openInNewTab ? '_blank' : '_self'}
        {...(openInNewTab && { rel: 'noopener noreferrer' })}
        className="block p-6"
      >
        <div className="mb-4 flex items-start justify-between">
          <span className="text-3xl">{service.icon}</span>
          <div className={`flex items-center ${getStatusColor(service.status)}`}>
            {getStatusIcon(service.status)}
            <span className="ml-1 text-sm capitalize">{service.status}</span>
          </div>
        </div>

        <h3 className="text-text-primary mb-1 text-lg font-semibold">{service.name}</h3>
        <p className="text-text-secondary mb-3 text-sm">{service.description}</p>

        {service.response_time !== undefined && service.response_time !== null && (
          <div
            className={`flex items-center text-xs ${getResponseTimeColor(service.response_time)}`}
          >
            <ClockIcon className="mr-1 h-3 w-3" />
            {service.response_time}ms
          </div>
        )}
      </a>
    </div>
  )
}
