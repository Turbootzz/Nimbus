'use client'

import { useState } from 'react'
import type { Service } from '@/types'
import { getApiUrl } from '@/lib/utils/api-url'

interface ServiceIconProps {
  service: Service
  size?: 'sm' | 'md' | 'lg' | 'xl'
  className?: string
}

const sizeClasses = {
  sm: 'text-3xl w-12 h-12',
  md: 'text-5xl w-16 h-16',
  lg: 'text-6xl w-20 h-20',
  xl: 'text-7xl w-24 h-24',
}

export default function ServiceIcon({ service, size = 'md', className = '' }: ServiceIconProps) {
  const [imageError, setImageError] = useState(false)
  const sizeClass = sizeClasses[size]
  const apiUrl = getApiUrl()

  // Fallback to emoji if image fails to load
  if (imageError) {
    return <span className={`${sizeClass} ${className}`}>{service.icon || '🔗'}</span>
  }

  // Render uploaded image
  if (service.icon_type === 'image_upload' && service.icon_image_path) {
    return (
      <img
        src={`${apiUrl}/uploads/service-icons/${service.icon_image_path}`}
        alt={`${service.name} icon`}
        className={`${sizeClass} object-contain ${className}`}
        onError={() => setImageError(true)}
      />
    )
  }

  // Render image URL
  if (service.icon_type === 'image_url' && service.icon_image_path) {
    return (
      <img
        src={service.icon_image_path}
        alt={`${service.name} icon`}
        className={`${sizeClass} object-contain ${className}`}
        onError={() => setImageError(true)}
      />
    )
  }

  // Render emoji (default)
  return <span className={`${sizeClass} ${className}`}>{service.icon || '🔗'}</span>
}
