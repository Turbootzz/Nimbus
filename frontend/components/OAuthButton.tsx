'use client'

import { useState } from 'react'
import type { OAuthProvider } from '@/types'
import { api } from '@/lib/api'
import GoogleIcon from './icons/GoogleIcon'
import GitHubIcon from './icons/GitHubIcon'
import DiscordIcon from './icons/DiscordIcon'

interface OAuthButtonProps {
  provider: OAuthProvider
  redirectTo?: string
  className?: string
}

const providerConfig = {
  google: {
    name: 'Google',
    icon: GoogleIcon,
    bgColor: 'bg-white hover:bg-gray-50',
    textColor: 'text-gray-700',
    borderColor: 'border-gray-300',
  },
  github: {
    name: 'GitHub',
    icon: GitHubIcon,
    bgColor: 'bg-gray-900 hover:bg-gray-800',
    textColor: 'text-white',
    borderColor: 'border-gray-900',
  },
  discord: {
    name: 'Discord',
    icon: DiscordIcon,
    bgColor: 'bg-[#5865F2] hover:bg-[#4752C4]',
    textColor: 'text-white',
    borderColor: 'border-[#5865F2]',
  },
}

export default function OAuthButton({ provider, redirectTo, className = '' }: OAuthButtonProps) {
  const [isLoading, setIsLoading] = useState(false)
  const config = providerConfig[provider]
  const Icon = config.icon

  const handleClick = () => {
    setIsLoading(true)
    api.initiateOAuth(provider, redirectTo)
  }

  return (
    <button
      type="button"
      onClick={handleClick}
      disabled={isLoading}
      className={`flex w-full items-center justify-center gap-3 rounded-lg border px-4 py-2.5 font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-50 ${config.bgColor} ${config.textColor} ${config.borderColor} ${className}`}
    >
      <Icon className="h-5 w-5" />
      <span>{isLoading ? 'Connecting...' : `Continue with ${config.name}`}</span>
    </button>
  )
}
