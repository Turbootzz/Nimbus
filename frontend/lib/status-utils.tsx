import React from 'react'
import { CheckCircleIcon, ExclamationCircleIcon, ClockIcon } from '@heroicons/react/24/solid'

/**
 * Returns the Tailwind CSS color class for a service status
 */
export const getStatusColor = (status: string): string => {
  switch (status) {
    case 'online':
      return 'text-success'
    case 'offline':
      return 'text-error'
    default:
      return 'text-warning'
  }
}

/**
 * Returns the appropriate icon component for a service status
 */
export const getStatusIcon = (status: string): React.ReactElement => {
  switch (status) {
    case 'online':
      return <CheckCircleIcon className="h-5 w-5" />
    case 'offline':
      return <ExclamationCircleIcon className="h-5 w-5" />
    default:
      return <ClockIcon className="h-5 w-5" />
  }
}

/**
 * Returns the Tailwind CSS color class based on response time
 * - Green: < 200ms
 * - Yellow: 200-500ms
 * - Red: > 500ms
 */
export const getResponseTimeColor = (ms: number): string => {
  if (ms < 200) return 'text-success'
  if (ms < 500) return 'text-warning'
  return 'text-error'
}
