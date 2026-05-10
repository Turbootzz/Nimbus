'use client'

import { useState } from 'react'
import Image from 'next/image'
import { UserCircleIcon } from '@heroicons/react/24/outline'
import { getApiUrl } from '@/lib/utils/api-url'

const resolveAvatarUrl = (avatarUrl: string | null | undefined): string | undefined => {
  if (!avatarUrl) return undefined
  // OAuth providers return a full URL; local uploads are stored as a
  // relative path that needs the API origin prepended.
  if (avatarUrl.startsWith('http')) return avatarUrl
  return getApiUrl() + avatarUrl
}

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
    return (
      <Image
        src={resolved}
        alt={name}
        width={size}
        height={size}
        className={sizeClass}
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
