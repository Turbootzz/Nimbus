'use client'

import { ClockIcon } from '@heroicons/react/24/outline'
import type { Service, CardSize } from '@/types'
import { getStatusColor, getStatusIcon, getResponseTimeColor } from '@/lib/status-utils'
import ServiceIcon from '@/components/ServiceIcon'

interface ServiceCardProps {
  service: Service
  openInNewTab: boolean
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

export default function ServiceCard({ service, openInNewTab }: ServiceCardProps) {
  const cardSize = service.card_size || '2x1'
  const gridSpan = sizeToGridSpan[cardSize]

  const linkProps = {
    href: service.url,
    target: openInNewTab ? '_blank' : '_self',
    ...(openInNewTab && { rel: 'noopener noreferrer' }),
  }

  switch (cardSize) {
    case '1x1':
      return <CompactCard service={service} gridSpan={gridSpan} linkProps={linkProps} />
    case '1x2':
      return <TallCard service={service} gridSpan={gridSpan} linkProps={linkProps} />
    case '2x2':
      return <LargeCard service={service} gridSpan={gridSpan} linkProps={linkProps} />
    default:
      return <StandardCard service={service} gridSpan={gridSpan} linkProps={linkProps} />
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
}

// 1x1 - Compact: large icon centered, name below, status indicator dot
function CompactCard({ service, gridSpan, linkProps }: CardVariantProps) {
  return (
    <a
      {...linkProps}
      className={`${gridSpan} bg-card border-card-border hover:border-primary flex h-full flex-col items-center justify-center rounded-lg border p-2 transition-all hover:shadow-lg`}
    >
      <div className="flex flex-1 items-center justify-center">
        <div className="relative inline-block">
          <ServiceIcon service={service} size="2xl" />
          <div
            className={`absolute right-0 bottom-0 h-5 w-5 rounded-full border-2 ${
              service.status === 'online'
                ? 'border-card bg-success'
                : service.status === 'offline'
                  ? 'border-card bg-error'
                  : 'border-card bg-warning'
            }`}
          />
        </div>
      </div>
      <h3 className="text-text-primary w-full truncate text-center text-sm font-semibold">
        {service.name}
      </h3>
    </a>
  )
}

// 2x1 - Standard (current layout): icon, status, name, description, response time
function StandardCard({ service, gridSpan, linkProps }: CardVariantProps) {
  return (
    <a
      {...linkProps}
      className={`${gridSpan} bg-card border-card-border hover:border-primary block rounded-lg border p-6 transition-all hover:shadow-lg`}
    >
      <div className="mb-4 flex items-start justify-between">
        <ServiceIcon service={service} size="md" />
        <div className={`flex items-center ${getStatusColor(service.status)}`}>
          {getStatusIcon(service.status)}
          <span className="ml-1 text-sm capitalize">{service.status}</span>
        </div>
      </div>

      <h3 className="text-text-primary mb-1 text-lg font-semibold">{service.name}</h3>
      <p className="text-text-secondary mb-3 text-sm">{service.description}</p>

      {service.response_time !== undefined && service.response_time !== null && (
        <div className={`flex items-center text-xs ${getResponseTimeColor(service.response_time)}`}>
          <ClockIcon className="mr-1 h-3 w-3" />
          {service.response_time}ms
        </div>
      )}
    </a>
  )
}

// 1x2 - Tall: large icon centered, name, description, status stacked vertically
function TallCard({ service, gridSpan, linkProps }: CardVariantProps) {
  return (
    <a
      {...linkProps}
      className={`${gridSpan} bg-card border-card-border hover:border-primary flex flex-col items-center rounded-lg border p-4 transition-all hover:shadow-lg`}
    >
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
    </a>
  )
}

// 2x2 - Large: centered icon and title, status in corner, description and details below
function LargeCard({ service, gridSpan, linkProps }: CardVariantProps) {
  return (
    <a
      {...linkProps}
      className={`${gridSpan} bg-card border-card-border hover:border-primary flex h-full flex-col rounded-lg border p-6 transition-all hover:shadow-lg`}
    >
      {/* Status in top right */}
      <div className={`mb-4 flex justify-end text-sm ${getStatusColor(service.status)}`}>
        {getStatusIcon(service.status)}
        <span className="ml-1 capitalize">{service.status}</span>
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
    </a>
  )
}
