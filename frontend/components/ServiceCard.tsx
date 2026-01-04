'use client'

import { ClockIcon } from '@heroicons/react/24/outline'
import { Bars3Icon } from '@heroicons/react/24/solid'
import type { Service, CardSize } from '@/types'
import { getStatusColor, getStatusIcon, getResponseTimeColor } from '@/lib/status-utils'
import ServiceIcon from '@/components/ServiceIcon'

interface ServiceCardProps {
  service: Service
  openInNewTab: boolean
  isEditMode?: boolean
  onSizeChange?: (id: string, newSize: CardSize) => void
  dragHandleProps?: Record<string, unknown>
  isDragging?: boolean
}

const sizeOrder: CardSize[] = ['1x1', '2x1', '1x2', '2x2']

function getNextSize(current: CardSize): CardSize {
  const idx = sizeOrder.indexOf(current)
  return sizeOrder[(idx + 1) % sizeOrder.length]
}

// CSS Grid span classes for each size
// Grid uses 2/4/6/8 columns at different breakpoints
// 1x1 = 1 col (half width), 2x1 = 2 cols (standard), 1x2 = 1 col + 2 rows, 2x2 = 2 cols + 2 rows
const sizeToGridSpan: Record<CardSize, string> = {
  '1x1': 'col-span-1 row-span-1',
  '2x1': 'col-span-2 row-span-1',
  '1x2': 'col-span-1 row-span-2',
  '2x2': 'col-span-2 row-span-2',
}

export default function ServiceCard({
  service,
  openInNewTab,
  isEditMode = false,
  onSizeChange,
  dragHandleProps,
  isDragging = false,
}: ServiceCardProps) {
  const cardSize = service.card_size || '2x1'
  // In edit mode, the wrapper div handles grid spanning, so card just fills container
  const gridSpan = isEditMode ? '' : sizeToGridSpan[cardSize]

  const linkProps = {
    href: service.url,
    target: openInNewTab ? '_blank' : '_self',
    ...(openInNewTab && { rel: 'noopener noreferrer' }),
  }

  const handleClick = () => {
    if (isEditMode && onSizeChange) {
      onSizeChange(service.id, getNextSize(cardSize))
    }
  }

  const variantProps = {
    service,
    gridSpan,
    linkProps,
    isEditMode,
    onSizeChange: handleClick,
    dragHandleProps,
    isDragging,
    cardSize,
  }

  switch (cardSize) {
    case '1x1':
      return <CompactCard {...variantProps} />
    case '1x2':
      return <TallCard {...variantProps} />
    case '2x2':
      return <LargeCard {...variantProps} />
    default:
      return <StandardCard {...variantProps} />
  }
}

interface CardVariantProps {
  service: Service
  gridSpan: string
  linkProps: {
    href: string
    target: string
    rel?: string
  }
  isEditMode: boolean
  onSizeChange: () => void
  dragHandleProps?: Record<string, unknown>
  isDragging: boolean
  cardSize: CardSize
}

// 1x1 - Compact: large icon centered, name below, status indicator dot
function CompactCard({
  service,
  gridSpan,
  linkProps,
  isEditMode,
  onSizeChange,
  dragHandleProps,
  isDragging,
  cardSize,
}: CardVariantProps) {
  const baseClasses = `${gridSpan} bg-card border-card-border flex h-full flex-col items-center justify-center rounded-lg border p-2 transition-all relative`
  const editClasses = isEditMode
    ? 'border-dashed border-2 cursor-pointer hover:border-primary'
    : 'hover:border-primary hover:shadow-lg'
  const dragClasses = isDragging ? 'opacity-50' : ''

  const content = (
    <>
      {isEditMode && (
        <div className="absolute top-1 right-1 left-1 z-10 flex items-center justify-between">
          <div
            {...dragHandleProps}
            className="bg-card/90 cursor-grab rounded p-1 active:cursor-grabbing"
            onClick={(e) => e.stopPropagation()}
          >
            <Bars3Icon className="text-text-muted h-4 w-4" />
          </div>
          <span className="bg-primary rounded px-1.5 py-0.5 text-xs text-white">{cardSize}</span>
        </div>
      )}
      <div className="flex flex-1 items-center justify-center">
        <div className="relative inline-block">
          <ServiceIcon service={service} size="xl" />
          <div
            className={`absolute right-0 bottom-0 h-4 w-4 rounded-full border-2 ${
              service.status === 'online'
                ? 'border-card bg-success'
                : service.status === 'offline'
                  ? 'border-card bg-error'
                  : 'border-card bg-warning'
            }`}
          />
        </div>
      </div>
      <h3 className="text-text-primary w-full truncate text-center text-xs font-semibold">
        {service.name}
      </h3>
    </>
  )

  if (isEditMode) {
    return (
      <div onClick={onSizeChange} className={`${baseClasses} ${editClasses} ${dragClasses}`}>
        {content}
      </div>
    )
  }

  return (
    <a {...linkProps} className={`${baseClasses} ${editClasses}`}>
      {content}
    </a>
  )
}

// 2x1 - Standard (current layout): icon, status, name, description, response time
function StandardCard({
  service,
  gridSpan,
  linkProps,
  isEditMode,
  onSizeChange,
  dragHandleProps,
  isDragging,
  cardSize,
}: CardVariantProps) {
  const baseClasses = `${gridSpan} bg-card border-card-border flex h-full flex-col rounded-lg border p-6 transition-all relative`
  const editClasses = isEditMode
    ? 'border-dashed border-2 cursor-pointer hover:border-primary'
    : 'hover:border-primary hover:shadow-lg'
  const dragClasses = isDragging ? 'opacity-50' : ''

  const content = (
    <>
      {isEditMode && (
        <div className="absolute top-2 right-2 left-2 z-10 flex items-center justify-between">
          <div
            {...dragHandleProps}
            className="bg-card/90 cursor-grab rounded p-1 active:cursor-grabbing"
            onClick={(e) => e.stopPropagation()}
          >
            <Bars3Icon className="text-text-muted h-5 w-5" />
          </div>
          <span className="bg-primary rounded px-1.5 py-0.5 text-xs text-white">{cardSize}</span>
        </div>
      )}
      <div className="mb-4 flex items-start justify-between">
        <ServiceIcon service={service} size="md" />
        <div className={`flex items-center ${getStatusColor(service.status)}`}>
          {getStatusIcon(service.status)}
          <span className="ml-1 text-sm capitalize">{service.status}</span>
        </div>
      </div>

      <h3 className="text-text-primary mb-1 truncate text-lg font-semibold">{service.name}</h3>
      {service.description && (
        <p className="text-text-secondary line-clamp-1 text-sm">{service.description}</p>
      )}

      <div
        className={`mt-auto flex items-center py-1 text-xs ${service.response_time !== undefined && service.response_time !== null ? getResponseTimeColor(service.response_time) : 'text-transparent'}`}
      >
        <ClockIcon className="mr-1 h-3 w-3" />
        {service.response_time !== undefined && service.response_time !== null
          ? `${service.response_time}ms`
          : '-'}
      </div>
    </>
  )

  if (isEditMode) {
    return (
      <div onClick={onSizeChange} className={`${baseClasses} ${editClasses} ${dragClasses}`}>
        {content}
      </div>
    )
  }

  return (
    <a {...linkProps} className={`${baseClasses} ${editClasses}`}>
      {content}
    </a>
  )
}

// 1x2 - Tall: large icon centered, name, description, status stacked vertically
function TallCard({
  service,
  gridSpan,
  linkProps,
  isEditMode,
  onSizeChange,
  dragHandleProps,
  isDragging,
  cardSize,
}: CardVariantProps) {
  const baseClasses = `${gridSpan} bg-card border-card-border flex h-full flex-col items-center rounded-lg border p-4 transition-all relative`
  const editClasses = isEditMode
    ? 'border-dashed border-2 cursor-pointer hover:border-primary'
    : 'hover:border-primary hover:shadow-lg'
  const dragClasses = isDragging ? 'opacity-50' : ''

  const content = (
    <>
      {isEditMode && (
        <div className="absolute top-2 right-2 left-2 z-10 flex items-center justify-between">
          <div
            {...dragHandleProps}
            className="bg-card/90 cursor-grab rounded p-1 active:cursor-grabbing"
            onClick={(e) => e.stopPropagation()}
          >
            <Bars3Icon className="text-text-muted h-4 w-4" />
          </div>
          <span className="bg-primary rounded px-1.5 py-0.5 text-xs text-white">{cardSize}</span>
        </div>
      )}
      <div className="flex flex-1 items-center justify-center py-4">
        <ServiceIcon service={service} size="lg" />
      </div>

      <div className="w-full text-center">
        <h3 className="text-text-primary mb-1 text-base font-semibold">{service.name}</h3>
        <p className="text-text-secondary mb-3 line-clamp-3 text-xs">{service.description}</p>

        <div
          className={`flex items-center justify-center text-xs ${getStatusColor(service.status)}`}
        >
          {getStatusIcon(service.status)}
          <span className="ml-1 capitalize">{service.status}</span>
        </div>
        {service.response_time !== undefined && service.response_time !== null && (
          <div
            className={`mt-1 flex items-center justify-center text-xs ${getResponseTimeColor(service.response_time)}`}
          >
            <ClockIcon className="mr-1 h-3 w-3" />
            {service.response_time}ms
          </div>
        )}
      </div>
    </>
  )

  if (isEditMode) {
    return (
      <div onClick={onSizeChange} className={`${baseClasses} ${editClasses} ${dragClasses}`}>
        {content}
      </div>
    )
  }

  return (
    <a {...linkProps} className={`${baseClasses} ${editClasses}`}>
      {content}
    </a>
  )
}

// 2x2 - Large: centered icon and title, status in corner, description and details below
function LargeCard({
  service,
  gridSpan,
  linkProps,
  isEditMode,
  onSizeChange,
  dragHandleProps,
  isDragging,
  cardSize,
}: CardVariantProps) {
  const baseClasses = `${gridSpan} bg-card border-card-border flex h-full flex-col rounded-lg border p-6 transition-all relative`
  const editClasses = isEditMode
    ? 'border-dashed border-2 cursor-pointer hover:border-primary'
    : 'hover:border-primary hover:shadow-lg'
  const dragClasses = isDragging ? 'opacity-50' : ''

  const content = (
    <>
      {isEditMode && (
        <div className="absolute top-2 right-2 left-2 z-10 flex items-center justify-between">
          <div
            {...dragHandleProps}
            className="bg-card/90 cursor-grab rounded p-1 active:cursor-grabbing"
            onClick={(e) => e.stopPropagation()}
          >
            <Bars3Icon className="text-text-muted h-5 w-5" />
          </div>
          <span className="bg-primary rounded px-1.5 py-0.5 text-xs text-white">{cardSize}</span>
        </div>
      )}
      {/* Status in top right */}
      <div className="mb-4 flex justify-end">
        <div className={`flex items-center text-sm ${getStatusColor(service.status)}`}>
          {getStatusIcon(service.status)}
          <span className="ml-1 capitalize">{service.status}</span>
        </div>
      </div>

      {/* Centered icon and title */}
      <div className="flex flex-1 flex-col items-center justify-center">
        <ServiceIcon service={service} size="2xl" />
        <h3 className="text-text-primary mt-4 text-center text-2xl font-semibold">
          {service.name}
        </h3>
        <p className="text-text-secondary mt-2 line-clamp-2 text-center text-sm">
          {service.description}
        </p>
      </div>

      {/* Footer with URL and response time */}
      <div className="mt-4 space-y-2">
        <div className="text-text-muted truncate text-center text-xs">{service.url}</div>
        {service.response_time !== undefined && service.response_time !== null && (
          <div
            className={`flex items-center justify-center text-sm ${getResponseTimeColor(service.response_time)}`}
          >
            <ClockIcon className="mr-1 h-4 w-4" />
            {service.response_time}ms
          </div>
        )}
      </div>
    </>
  )

  if (isEditMode) {
    return (
      <div onClick={onSizeChange} className={`${baseClasses} ${editClasses} ${dragClasses}`}>
        {content}
      </div>
    )
  }

  return (
    <a {...linkProps} className={`${baseClasses} ${editClasses}`}>
      {content}
    </a>
  )
}
