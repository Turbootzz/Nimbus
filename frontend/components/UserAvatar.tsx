'use client'

import { useState } from 'react'
import Image from 'next/image'
import { UserCircleIcon } from '@heroicons/react/24/outline'
import { uploadUrl } from '@/lib/utils/api-url'

// OAuth providers return a full URL; local uploads a path on the API server
const resolveAvatarUrl = (avatarUrl: string | null | undefined): string | undefined =>
  avatarUrl ? uploadUrl(avatarUrl) : undefined

interface UserAvatarProps {
  avatarUrl: string | null | undefined
  name: string
  size: number
  className?: string
}

export default function UserAvatar({ avatarUrl, name, size, className = '' }: UserAvatarProps) {
  const resolved = resolveAvatarUrl(avatarUrl)
  const [failedUrl, setFailedUrl] = useState<string | null>(null)
  const failed = failedUrl !== null && failedUrl === resolved

  const sizeClass = `rounded-full object-cover ${className}`.trim()

  if (resolved && !failed) {
    // Skip the Next.js optimizer, as service icons do: it refuses an API on a
    // private IP, cannot reach a same-origin API path, and logs an upstream
    // 404 for a stale OAuth avatar. The browser loads it directly instead.
    return (
      <Image
        src={resolved}
        alt={name}
        width={size}
        height={size}
        className={sizeClass}
        unoptimized
        onError={() => setFailedUrl(resolved)}
      />
    )
  }

  return (
    <UserCircleIcon
      className={className || undefined}
      style={{
        color: 'var(--color-text-secondary)',
        width: size,
        height: size,
      }}
    />
  )
}
