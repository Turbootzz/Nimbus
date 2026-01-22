'use client'

import { ClockIcon } from '@heroicons/react/24/outline'
import { Bars3Icon } from '@heroicons/react/24/solid'
import type { Service, CardSize, CardScale } from '@/types'
import {
  getStatusColor,
  getStatusBgColor,
  getStatusIcon,
  getResponseTimeColor,
} from '@/lib/status-utils'
import { sizeToGridSpan, getNextSize } from '@/lib/card-utils'
import ServiceIcon from '@/components/ServiceIcon'

interface ServiceCardProps {
  service: Service
  openInNewTab: boolean
  isEditMode?: boolean
  onSizeChange?: (id: string, newSize: CardSize) => void
  dragHandleProps?: Record<string, unknown>
  isDragging?: boolean
  enableCardResizing?: boolean
  cardScale?: CardScale
}

export default function ServiceCard({
  service,
  openInNewTab,
  isEditMode = false,
  onSizeChange,
  dragHandleProps,
  isDragging = false,
  enableCardResizing = true,
  cardScale = 'large',
}: ServiceCardProps) {
  // When card resizing is disabled, always use 2x1
  const cardSize = enableCardResizing ? service.card_size || '2x1' : '2x1'
  // In edit mode, the wrapper div handles grid spanning, so card just fills container
  const gridSpan = isEditMode ? '' : sizeToGridSpan[cardSize]

  const linkProps = {
    href: service.url,
    target: openInNewTab ? '_blank' : '_self',
    ...(openInNewTab && { rel: 'noopener noreferrer' }),
  }

  const handleClick = () => {
    // Only allow size change if resizing is enabled
    if (isEditMode && onSizeChange && enableCardResizing) {
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
    showSizeBadge: enableCardResizing,
    cardScale,
  }

  // When resizing is disabled, always use StandardCard (2x1)
  if (!enableCardResizing) {
    return <StandardCard {...variantProps} />
  }

  switch (cardSize) {
    case '1x1':
      return <CompactCard {...variantProps} />
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
  showSizeBadge: boolean
  cardScale: CardScale
}

// Icon sizes based on cardScale - icons shrink with denser grids
const scaleIconSizes: Record<
  CardScale,
  { standard: 'sm' | 'md' | 'lg'; large: 'lg' | 'xl' | '2xl' }
> = {
  small: { standard: 'sm', large: 'lg' },
  medium: { standard: 'sm', large: 'xl' },
  large: { standard: 'md', large: '2xl' },
}

// Padding classes based on cardScale
// Mobile always uses large padding, scale-specific padding kicks in at sm breakpoint
const scalePadding: Record<CardScale, { standard: string; compact: string }> = {
  small: { standard: 'p-6 sm:p-3', compact: 'p-2 sm:p-1.5' },
  medium: { standard: 'p-6 sm:p-4', compact: 'p-2' },
  large: { standard: 'p-6', compact: 'p-2' },
}

// Text sizes based on cardScale
const scaleText: Record<CardScale, { title: string; description: string }> = {
  small: { title: 'text-sm', description: 'text-xs' },
  medium: { title: 'text-base', description: 'text-sm' },
  large: { title: 'text-lg', description: 'text-sm' },
}

// Reusable edit mode overlay with drag handle and size badge
interface EditOverlayProps {
  dragHandleProps?: Record<string, unknown>
  cardSize: CardSize
  showSizeBadge: boolean
  compact?: boolean
}

function EditOverlay({
  dragHandleProps,
  cardSize,
  showSizeBadge,
  compact = false,
}: EditOverlayProps) {
  const position = compact ? 'top-1 right-1 left-1' : 'top-2 right-2 left-2'
  const iconSize = compact ? 'h-4 w-4' : 'h-5 w-5'

  return (
    <div className={`absolute ${position} z-10 flex items-center justify-between`}>
      <div
        {...dragHandleProps}
        className="bg-card/90 cursor-grab touch-none rounded p-1 active:cursor-grabbing"
        onClick={(e) => e.stopPropagation()}
        onTouchStart={(e) => e.stopPropagation()}
      >
        <Bars3Icon className={`text-text-muted ${iconSize}`} />
      </div>
      {showSizeBadge && (
        <span className="bg-primary rounded px-1.5 py-0.5 text-xs text-white">{cardSize}</span>
      )}
    </div>
  )
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
  showSizeBadge,
  cardScale,
}: CardVariantProps) {
  const padding = scalePadding[cardScale].compact
  const iconSize = scaleIconSizes[cardScale].large
  const titleSize = scaleText[cardScale].title
  const statusDotSize = cardScale === 'small' ? 'h-2 w-2' : 'h-3 w-3'

  const baseClasses = `${gridSpan} bg-card border-card-border flex h-full flex-col items-center justify-center rounded-lg border ${padding} transition-all relative`
  const editClasses = isEditMode
    ? 'border-dashed border-2 cursor-pointer hover:border-primary'
    : 'hover:border-primary hover:shadow-lg'
  const dragClasses = isDragging ? 'scale-105 shadow-2xl ring-2 ring-primary' : ''

  const content = (
    <>
      {isEditMode && (
        <EditOverlay
          dragHandleProps={dragHandleProps}
          cardSize={cardSize}
          showSizeBadge={showSizeBadge}
          compact
        />
      )}
      <div className="flex flex-1 items-center justify-center">
        <ServiceIcon service={service} size={iconSize} />
      </div>
      {/* Title with inline status indicator */}
      <div className="flex w-full items-center justify-center gap-1.5">
        <div
          className={`${statusDotSize} shrink-0 rounded-full ${getStatusBgColor(service.status)}`}
        />
        <h3 className={`text-text-primary ${titleSize} truncate font-semibold`}>{service.name}</h3>
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
  showSizeBadge,
  cardScale,
}: CardVariantProps) {
  const padding = scalePadding[cardScale].standard
  const iconSize = scaleIconSizes[cardScale].standard
  const titleSize = scaleText[cardScale].title
  const descSize = scaleText[cardScale].description
  const marginBottom = cardScale === 'small' ? 'mb-2' : 'mb-4'

  const baseClasses = `${gridSpan} bg-card border-card-border flex h-full flex-col rounded-lg border ${padding} transition-all relative`
  const editClasses = isEditMode
    ? 'border-dashed border-2 cursor-pointer hover:border-primary'
    : 'hover:border-primary hover:shadow-lg'
  const dragClasses = isDragging ? 'scale-105 shadow-2xl ring-2 ring-primary' : ''

  const content = (
    <>
      {isEditMode && (
        <EditOverlay
          dragHandleProps={dragHandleProps}
          cardSize={cardSize}
          showSizeBadge={showSizeBadge}
        />
      )}
      <div className={`${marginBottom} flex items-start justify-between`}>
        <ServiceIcon service={service} size={iconSize} />
        <div className={`flex items-center ${getStatusColor(service.status)}`}>
          {getStatusIcon(service.status)}
          <span className="ml-1 text-sm capitalize">{service.status}</span>
        </div>
      </div>

      <h3 className={`text-text-primary mb-1 truncate ${titleSize} font-semibold`}>
        {service.name}
      </h3>
      {service.description && (
        <p className={`text-text-secondary line-clamp-1 ${descSize}`}>{service.description}</p>
      )}

      <div
        className={`mt-auto flex items-center py-1 text-xs ${service.status === 'online' && service.response_time !== undefined && service.response_time !== null ? getResponseTimeColor(service.response_time) : 'text-transparent'}`}
      >
        <ClockIcon className="mr-1 h-3 w-3" />
        {service.status === 'online' &&
        service.response_time !== undefined &&
        service.response_time !== null
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
  showSizeBadge,
  cardScale,
}: CardVariantProps) {
  const padding = scalePadding[cardScale].standard
  const iconSize = scaleIconSizes[cardScale].large
  const largeTitleSizes: Record<CardScale, string> = {
    small: 'text-base',
    medium: 'text-xl',
    large: 'text-2xl',
  }
  const titleSize = largeTitleSizes[cardScale]
  const descSize = scaleText[cardScale].description
  const marginBottom = cardScale === 'small' ? 'mb-2' : 'mb-4'
  const marginTop = cardScale === 'small' ? 'mt-2' : 'mt-4'

  const baseClasses = `${gridSpan} bg-card border-card-border flex h-full flex-col rounded-lg border ${padding} transition-all relative`
  const editClasses = isEditMode
    ? 'border-dashed border-2 cursor-pointer hover:border-primary'
    : 'hover:border-primary hover:shadow-lg'
  const dragClasses = isDragging ? 'scale-105 shadow-2xl ring-2 ring-primary' : ''

  const content = (
    <>
      {isEditMode && (
        <EditOverlay
          dragHandleProps={dragHandleProps}
          cardSize={cardSize}
          showSizeBadge={showSizeBadge}
        />
      )}
      {/* Status in top right */}
      <div className={`${marginBottom} flex justify-end`}>
        <div className={`flex items-center text-sm ${getStatusColor(service.status)}`}>
          {getStatusIcon(service.status)}
          <span className="ml-1 capitalize">{service.status}</span>
        </div>
      </div>

      {/* Centered icon and title */}
      <div className="flex flex-1 flex-col items-center justify-center">
        <ServiceIcon service={service} size={iconSize} />
        <h3 className={`text-text-primary ${marginTop} text-center ${titleSize} font-semibold`}>
          {service.name}
        </h3>
        {service.description && (
          <p className={`text-text-secondary mt-2 line-clamp-2 text-center ${descSize}`}>
            {service.description}
          </p>
        )}
      </div>

      {/* Footer with URL and response time */}
      <div className={`${marginTop} space-y-2`}>
        <div className="text-text-muted truncate text-center text-xs">{service.url}</div>
        {service.status === 'online' &&
          service.response_time !== undefined &&
          service.response_time !== null && (
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
