'use client'

import { useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { CheckCircleIcon, ExclamationCircleIcon, ClockIcon } from '@heroicons/react/24/outline'
import { Bars3Icon } from '@heroicons/react/24/solid'
import type { Service } from '@/types'

interface DraggableServiceCardProps {
  service: Service
  isDragging?: boolean
}

export function DraggableServiceCard({ service, isDragging = false }: DraggableServiceCardProps) {
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

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'online':
        return 'text-success'
      case 'offline':
        return 'text-error'
      default:
        return 'text-warning'
    }
  }

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'online':
        return <CheckCircleIcon className="h-5 w-5" />
      case 'offline':
        return <ExclamationCircleIcon className="h-5 w-5" />
      default:
        return <ClockIcon className="h-5 w-5" />
    }
  }

  const getResponseTimeColor = (ms: number) => {
    if (ms < 200) return 'text-success'
    if (ms < 500) return 'text-warning'
    return 'text-error'
  }

  return (
    <div
      ref={setNodeRef}
      style={style}
      className="bg-card border-card-border hover:border-primary group relative rounded-lg border transition-all hover:shadow-lg"
    >
      {/* Drag handle */}
      <div
        {...attributes}
        {...listeners}
        className="bg-card-hover absolute top-2 right-2 cursor-grab rounded p-1 opacity-0 transition-opacity group-hover:opacity-100 active:cursor-grabbing"
        title="Drag to reorder"
      >
        <Bars3Icon className="text-text-muted h-5 w-5" />
      </div>

      {/* Service card content */}
      <a href={service.url} target="_blank" rel="noopener noreferrer" className="block p-6">
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
