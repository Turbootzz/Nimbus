'use client'

import { useState } from 'react'
import Image from 'next/image'
import type { Service } from '@/types'
import { getApiUrl } from '@/lib/utils/api-url'

interface ServiceIconProps {
  service: Service
  size?: 'sm' | 'md' | 'lg' | 'xl' | '2xl' | '3xl'
  className?: string
}

// Container dimensions for all icon types
const sizeClasses = {
  sm: 'w-12 h-12',
  md: 'w-16 h-16',
  lg: 'w-20 h-20',
  xl: 'w-24 h-24',
  '2xl': 'w-32 h-32',
  '3xl': 'w-48 h-48',
}

// Emoji text sizes - slightly larger to fill container height
const emojiSizeClasses = {
  sm: 'text-4xl',
  md: 'text-6xl',
  lg: 'text-7xl',
  xl: 'text-8xl',
  '2xl': 'text-[7rem]',
  '3xl': 'text-[11rem]',
}

const sizeDimensions = {
  sm: 48,
  md: 64,
  lg: 80,
  xl: 96,
  '2xl': 128,
  '3xl': 192,
}

export default function ServiceIcon({ service, size = 'md', className = '' }: ServiceIconProps) {
  const [imageError, setImageError] = useState(false)
  const sizeClass = sizeClasses[size]
  const dimension = sizeDimensions[size]
  const apiUrl = getApiUrl()

  const emojiClass = emojiSizeClasses[size]

  // Fallback to emoji if image fails to load
  if (imageError) {
    return (
      <div className={`${sizeClass} flex items-center justify-center ${className}`}>
        <span className={emojiClass}>{service.icon || '🔗'}</span>
      </div>
    )
  }

  // Render uploaded image
  if (service.icon_type === 'image_upload' && service.icon_image_path) {
    const imageUrl = `${apiUrl}/uploads/service-icons/${service.icon_image_path}`
    // Use unoptimized for localhost to avoid Next.js blocking private IPs
    const isLocalhost = apiUrl.includes('localhost') || apiUrl.includes('127.0.0.1')
    return (
      <div className={`${sizeClass} relative overflow-hidden ${className}`}>
        <Image
          src={imageUrl}
          alt={`${service.name} icon`}
          width={dimension}
          height={dimension}
          className="h-full w-full object-contain"
          onError={() => setImageError(true)}
          unoptimized={isLocalhost}
        />
      </div>
    )
  }

  // Render image URL
  if (service.icon_type === 'image_url' && service.icon_image_path) {
    return (
      <div className={`${sizeClass} relative overflow-hidden ${className}`}>
        <Image
          src={service.icon_image_path}
          alt={`${service.name} icon`}
          width={dimension}
          height={dimension}
          className="h-full w-full object-contain"
          onError={() => setImageError(true)}
        />
      </div>
    )
  }

  // Render emoji (default) - wrap in div for consistent sizing with images
  return (
    <div className={`${sizeClass} flex items-center justify-center ${className}`}>
      <span className={emojiClass}>{service.icon || '🔗'}</span>
    </div>
  )
}
