'use client'

import { useEffect, useState } from 'react'
import { api } from '@/lib/api'
import { getApiUrl } from '@/lib/utils/api-url'
import type { User } from '@/types'
import { UserCircleIcon } from '@heroicons/react/24/outline'
import GoogleIcon from '@/components/icons/GoogleIcon'
import GitHubIcon from '@/components/icons/GitHubIcon'
import DiscordIcon from '@/components/icons/DiscordIcon'

export default function ProfilePage() {
  const [user, setUser] = useState<User | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [isUploading, setIsUploading] = useState(false)
  const [uploadError, setUploadError] = useState('')

  useEffect(() => {
    const fetchUser = async () => {
      try {
        const response = await api.getCurrentUser()
        if (response.data) {
          setUser(response.data)
        }
      } catch (error) {
        console.error('Failed to fetch user:', error)
        // Consider adding error state to show user-friendly message
      } finally {
        setIsLoading(false)
      }
    }
    fetchUser()
  }, [])

  const handleAvatarUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    // Validate file type
    if (!file.type.startsWith('image/')) {
      setUploadError('Please select an image file')
      return
    }

    // Validate file size (max 5MB)
    if (file.size > 5 * 1024 * 1024) {
      setUploadError('Image size must be less than 5MB')
      return
    }

    setIsUploading(true)
    setUploadError('')

    const formData = new FormData()
    formData.append('avatar', file)

    const response = await api.uploadAvatar(formData)
    if (response.data) {
      setUser(response.data)
    } else if (response.error) {
      setUploadError(response.error)
    }

    setIsUploading(false)
  }

  const getProviderIcon = (provider: string) => {
    switch (provider) {
      case 'google':
        return <GoogleIcon className="h-5 w-5" />
      case 'github':
        return <GitHubIcon className="h-5 w-5" />
      case 'discord':
        return <DiscordIcon className="h-5 w-5" />
      default:
        return null
    }
  }

  const getProviderName = (provider: string) => {
    switch (provider) {
      case 'google':
        return 'Google'
      case 'github':
        return 'GitHub'
      case 'discord':
        return 'Discord'
      case 'local':
        return 'Email & Password'
      default:
        return provider
    }
  }

  const getAvatarUrl = (avatarUrl: string | undefined) => {
    if (!avatarUrl) return undefined
    // If it's a full URL (OAuth provider), return as-is
    if (avatarUrl.startsWith('http')) return avatarUrl
    // If it's a relative path (local upload), prepend API URL
    return getApiUrl() + avatarUrl
  }

  if (isLoading) {
    return (
      <div className="p-4 sm:p-6">
        <h1
          className="mb-4 text-2xl font-bold sm:text-3xl"
          style={{ color: 'var(--color-text-primary)' }}
        >
          Profile
        </h1>
        <p style={{ color: 'var(--color-text-secondary)' }}>Loading...</p>
      </div>
    )
  }

  return (
    <div className="p-4 sm:p-6">
      <h1
        className="mb-4 text-2xl font-bold sm:text-3xl"
        style={{ color: 'var(--color-text-primary)' }}
      >
        Profile
      </h1>

      <div
        className="max-w-2xl rounded-lg border p-6"
        style={{
          backgroundColor: 'var(--color-card)',
          borderColor: 'var(--color-card-border)',
        }}
      >
        <div className="space-y-6">
          {/* Profile Picture Section */}
          <div>
            <label
              className="mb-3 block text-sm font-medium"
              style={{ color: 'var(--color-text-secondary)' }}
            >
              Profile Picture
            </label>
            <div className="flex items-center space-x-4">
              {user?.avatar_url ? (
                <img
                  src={getAvatarUrl(user.avatar_url)}
                  alt={user.name}
                  className="h-20 w-20 rounded-full object-cover"
                />
              ) : (
                <UserCircleIcon
                  className="h-20 w-20"
                  style={{ color: 'var(--color-text-secondary)' }}
                />
              )}

              {user?.provider === 'local' ? (
                <div className="flex-1">
                  <label
                    htmlFor="avatar-upload"
                    className="inline-flex cursor-pointer items-center rounded-lg border px-4 py-2 text-sm font-medium transition-colors"
                    style={{
                      backgroundColor: 'var(--color-card)',
                      borderColor: 'var(--color-card-border)',
                      color: 'var(--color-text-primary)',
                    }}
                  >
                    {isUploading ? 'Uploading...' : 'Upload Picture'}
                  </label>
                  <input
                    id="avatar-upload"
                    type="file"
                    accept="image/*"
                    onChange={handleAvatarUpload}
                    disabled={isUploading}
                    className="hidden"
                  />
                  <p className="mt-1 text-xs" style={{ color: 'var(--color-text-muted)' }}>
                    JPG, PNG or GIF (max 5MB)
                  </p>
                  {uploadError && (
                    <p className="mt-1 text-xs" style={{ color: 'var(--color-error)' }}>
                      {uploadError}
                    </p>
                  )}
                </div>
              ) : (
                <div className="flex-1">
                  <p className="text-sm" style={{ color: 'var(--color-text-secondary)' }}>
                    Profile picture synced from {getProviderName(user?.provider || '')}
                  </p>
                  <p className="mt-1 text-xs" style={{ color: 'var(--color-text-muted)' }}>
                    Update your picture in your {getProviderName(user?.provider || '')} account
                  </p>
                </div>
              )}
            </div>
          </div>

          <div className="border-t" style={{ borderColor: 'var(--color-card-border)' }} />

          {/* Account Provider */}
          <div>
            <label
              className="mb-1 block text-sm font-medium"
              style={{ color: 'var(--color-text-secondary)' }}
            >
              Sign-in Method
            </label>
            <div className="flex items-center space-x-2">
              {getProviderIcon(user?.provider || 'local')}
              <p className="text-lg" style={{ color: 'var(--color-text-primary)' }}>
                {getProviderName(user?.provider || 'local')}
              </p>
            </div>
          </div>

          {/* Name */}
          <div>
            <label
              className="mb-1 block text-sm font-medium"
              style={{ color: 'var(--color-text-secondary)' }}
            >
              Name
            </label>
            <p className="text-lg" style={{ color: 'var(--color-text-primary)' }}>
              {user?.name}
            </p>
          </div>

          {/* Email */}
          <div>
            <label
              className="mb-1 block text-sm font-medium"
              style={{ color: 'var(--color-text-secondary)' }}
            >
              Email
            </label>
            <p className="text-lg" style={{ color: 'var(--color-text-primary)' }}>
              {user?.email}
            </p>
          </div>

          {/* Role */}
          <div>
            <label
              className="mb-1 block text-sm font-medium"
              style={{ color: 'var(--color-text-secondary)' }}
            >
              Role
            </label>
            <p className="text-lg capitalize" style={{ color: 'var(--color-text-primary)' }}>
              {user?.role}
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}
