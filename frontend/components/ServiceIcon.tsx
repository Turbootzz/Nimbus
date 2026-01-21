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

// Check if a hostname is a local/private address
function isLocalHostname(url: string): boolean {
  try {
    const hostname = new URL(url).hostname

    // Check common local hostnames
    if (hostname === 'localhost' || hostname === '[::1]' || hostname.endsWith('.local')) {
      return true
    }

    // Check IPv4 loopback
    if (hostname.startsWith('127.')) {
      return true
    }

    // Check private IPv4 ranges
    const ipv4Match = hostname.match(/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/)
    if (ipv4Match) {
      const [, first, second] = ipv4Match.map(Number)
      // 10.0.0.0/8
      if (first === 10) return true
      // 172.16.0.0/12
      if (first === 172 && second >= 16 && second <= 31) return true
      // 192.168.0.0/16
      if (first === 192 && second === 168) return true
    }

    return false
  } catch {
    return false
  }
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
    // Use unoptimized for local addresses to avoid Next.js blocking private IPs
    const isLocalAddress = isLocalHostname(apiUrl)
    return (
      <div className={`${sizeClass} relative overflow-hidden ${className}`}>
        <Image
          src={imageUrl}
          alt={`${service.name} icon`}
          width={dimension}
          height={dimension}
          className="h-full w-full object-contain"
          onError={() => setImageError(true)}
          unoptimized={isLocalAddress}
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
