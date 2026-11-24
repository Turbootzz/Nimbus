'use client'

import { useState, useEffect } from 'react'
import Link from 'next/link'
import { useTranslations } from 'next-intl'
import { getApiUrl } from '@/lib/utils/api-url'
import { ThemedInput } from '@/components/ui/ThemedInput'
import { useHoverStyle, hoverStyles } from '@/hooks/useHoverStyle'
import { api } from '@/lib/api'
import { clearApiCache } from '@/lib/pwa'

export default function RegisterPage() {
  const t = useTranslations('auth.register')
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')
  const [checkingStatus, setCheckingStatus] = useState(true)
  const [registrationEnabled, setRegistrationEnabled] = useState(true)
  const buttonHover = useHoverStyle(hoverStyles.primaryButton, { disabled: isLoading })
  const linkHover = useHoverStyle(hoverStyles.primaryLink)

  // Check if registration is enabled on mount
  useEffect(() => {
    const checkRegistrationStatus = async () => {
      try {
        const response = await api.getRegistrationStatus()
        if (response.data) {
          setRegistrationEnabled(response.data.enabled)
        }
      } catch (err) {
        console.error('Failed to check registration status:', err)
      } finally {
        setCheckingStatus(false)
      }
    }
    checkRegistrationStatus()
  }, [])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')

    // Validation
    if (password !== confirmPassword) {
      setError(t('errors.passwordMismatch'))
      return
    }

    if (password.length < 8) {
      setError(t('errors.passwordTooShort'))
      return
    }

    setIsLoading(true)

    try {
      // Call API with credentials to allow httpOnly cookies
      // Backend will set secure httpOnly cookie instead of returning token in response
      const response = await fetch(`${getApiUrl()}/auth/register`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include', // Required to receive and send httpOnly cookies
        body: JSON.stringify({ name, email, password }),
      })

      // Parse response as text first to handle non-JSON responses
      const text = await response.text()
      let data
      try {
        data = JSON.parse(text)
      } catch {
        if (text.includes('<!DOCTYPE') || text.includes('<html')) {
          setError(t('errors.apiUnreachable'))
        } else {
          setError(t('errors.invalidResponse'))
        }
        return
      }

      if (!response.ok) {
        // Check for registration disabled (403)
        if (response.status === 403) {
          setError(t('errors.disabled'))
          setRegistrationEnabled(false)
          return
        }
        setError(data.error || t('errors.failed'))
        return
      }

      // No need to store token - backend sets httpOnly cookie automatically
      // The cookie will be sent with all subsequent requests via credentials: 'include'

      // A different user may have used this device before
      await clearApiCache()

      // Redirect to dashboard
      window.location.href = '/dashboard'
    } catch (err) {
      setError(t('errors.failedRetry'))
      console.error('Registration error:', err)
    } finally {
      setIsLoading(false)
    }
  }

  // Show loading spinner while checking status
  if (checkingStatus) {
    return (
      <div
        className="rounded-2xl p-8 text-center shadow-xl"
        style={{
          backgroundColor: 'var(--color-card)',
          borderColor: 'var(--color-card-border)',
        }}
      >
        <div
          className="inline-block h-8 w-8 animate-spin rounded-full border-4 border-solid border-current border-r-transparent"
          style={{ color: 'var(--color-primary)' }}
        />
      </div>
    )
  }

  // Show message when registration is disabled
  if (!registrationEnabled) {
    return (
      <div
        className="rounded-2xl p-8 shadow-xl"
        style={{
          backgroundColor: 'var(--color-card)',
          borderColor: 'var(--color-card-border)',
        }}
      >
        <div className="mb-6 text-center">
          <h2 className="text-2xl font-bold" style={{ color: 'var(--color-text-primary)' }}>
            {t('disabled.title')}
          </h2>
          <p className="mt-4" style={{ color: 'var(--color-text-secondary)' }}>
            {t('disabled.message')}
          </p>
          <p className="mt-2" style={{ color: 'var(--color-text-muted)' }}>
            {t('disabled.contactAdmin')}
          </p>
        </div>

        <div className="mt-6 text-center text-sm" style={{ color: 'var(--color-text-secondary)' }}>
          {t('haveAccount')}{' '}
          <Link
            href="/login"
            className="font-medium transition"
            style={{ color: 'var(--color-primary)' }}
            {...linkHover}
          >
            {t('signIn')}
          </Link>
        </div>
      </div>
    )
  }

  return (
    <div
      className="rounded-2xl p-8 shadow-xl"
      style={{
        backgroundColor: 'var(--color-card)',
        borderColor: 'var(--color-card-border)',
      }}
    >
      <div className="mb-6 text-center">
        <h2 className="text-2xl font-bold" style={{ color: 'var(--color-text-primary)' }}>
          {t('title')}
        </h2>
        <p className="mt-1" style={{ color: 'var(--color-text-secondary)' }}>
          {t('subtitle')}
        </p>
      </div>

      {error && (
        <div
          className="mb-4 rounded-lg border p-3 text-sm"
          style={{
            backgroundColor: 'var(--color-error)',
            borderColor: 'var(--color-error)',
            color: 'white',
            opacity: 0.9,
          }}
        >
          {error}
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label
            htmlFor="name"
            className="mb-1 block text-sm font-medium"
            style={{ color: 'var(--color-text-secondary)' }}
          >
            {t('fullName')}
          </label>
          <ThemedInput
            id="name"
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder={t('namePlaceholder')}
            required
            disabled={isLoading}
          />
        </div>

        <div>
          <label
            htmlFor="email"
            className="mb-1 block text-sm font-medium"
            style={{ color: 'var(--color-text-secondary)' }}
          >
            {t('email')}
          </label>
          <ThemedInput
            id="email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="you@example.com"
            required
            disabled={isLoading}
          />
        </div>

        <div>
          <label
            htmlFor="password"
            className="mb-1 block text-sm font-medium"
            style={{ color: 'var(--color-text-secondary)' }}
          >
            {t('password')}
          </label>
          <ThemedInput
            id="password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="••••••••"
            required
            disabled={isLoading}
            minLength={8}
          />
          <p className="mt-1 text-xs" style={{ color: 'var(--color-text-muted)' }}>
            {t('passwordHint')}
          </p>
        </div>

        <div>
          <label
            htmlFor="confirmPassword"
            className="mb-1 block text-sm font-medium"
            style={{ color: 'var(--color-text-secondary)' }}
          >
            {t('confirmPassword')}
          </label>
          <ThemedInput
            id="confirmPassword"
            type="password"
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            placeholder="••••••••"
            required
            disabled={isLoading}
          />
        </div>

        <button
          type="submit"
          disabled={isLoading}
          className="w-full rounded-lg py-2.5 font-medium text-white transition focus:ring-4 disabled:cursor-not-allowed disabled:opacity-50"
          style={{
            backgroundColor: 'var(--color-primary)',
            color: 'var(--color-on-primary, white)',
          }}
          {...buttonHover}
        >
          {isLoading ? t('submitting') : t('submit')}
        </button>
      </form>

      <div className="mt-6 text-center text-sm" style={{ color: 'var(--color-text-secondary)' }}>
        {t('haveAccount')}{' '}
        <Link
          href="/login"
          className="font-medium transition"
          style={{ color: 'var(--color-primary)' }}
          {...linkHover}
        >
          {t('signIn')}
        </Link>
      </div>
    </div>
  )
}
