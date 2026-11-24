'use client'

import { useState, useEffect, Suspense } from 'react'
import Link from 'next/link'
import { useSearchParams, useRouter } from 'next/navigation'
import { useTranslations } from 'next-intl'
import { getApiUrl } from '@/lib/utils/api-url'
import OAuthButton from '@/components/OAuthButton'
import { ThemedInput } from '@/components/ui/ThemedInput'
import { useHoverStyle, hoverStyles } from '@/hooks/useHoverStyle'
import { api } from '@/lib/api'
import { clearApiCache } from '@/lib/pwa'
import type { OAuthProvider } from '@/types'

function LoginForm() {
  const t = useTranslations('auth.login')
  const router = useRouter()
  const searchParams = useSearchParams()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [rememberMe, setRememberMe] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')
  const [oauthProviders, setOAuthProviders] = useState<OAuthProvider[]>([])
  const [checkingSetup, setCheckingSetup] = useState(true)
  const [registrationEnabled, setRegistrationEnabled] = useState(true)
  const buttonHover = useHoverStyle(hoverStyles.primaryButton, { disabled: isLoading })
  const linkHover = useHoverStyle(hoverStyles.primaryLink)

  // Check if setup is needed and registration status on mount
  useEffect(() => {
    const checkSetupAndRegistration = async () => {
      try {
        // Check setup status first
        const setupResponse = await api.getSetupStatus()
        if (setupResponse.data?.needs_setup) {
          router.push('/setup')
          return
        }

        // Check registration status
        const regResponse = await api.getRegistrationStatus()
        if (regResponse.data) {
          setRegistrationEnabled(regResponse.data.enabled)
        }
      } catch (err) {
        console.error('Failed to check status:', err)
      }
      setCheckingSetup(false)
    }
    checkSetupAndRegistration()
  }, [router])

  // Check for OAuth error in query params
  useEffect(() => {
    const oauthError = searchParams.get('error')
    if (oauthError) {
      setError(decodeURIComponent(oauthError))
    }
  }, [searchParams])

  // Whoever sees the login form has no session: drop the last user's offline copy
  useEffect(() => {
    clearApiCache()
  }, [])

  // Fetch available OAuth providers
  useEffect(() => {
    const fetchProviders = async () => {
      try {
        const response = await api.getOAuthProviders()
        if (response.data) {
          const enabled = response.data.providers
            .filter((p) => p.enabled)
            .map((p) => p.name as OAuthProvider)
          setOAuthProviders(enabled)
        } else if (response.error) {
          // Log error but don't show to user - OAuth is optional
          console.error('Failed to fetch OAuth providers:', response.error.message)
        }
      } catch (err) {
        // Log error but don't show to user - OAuth is optional
        console.error('Failed to fetch OAuth providers:', err)
      }
    }
    fetchProviders()
  }, [])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setIsLoading(true)

    try {
      // Call API with credentials to allow httpOnly cookies
      // Backend will set secure httpOnly cookie instead of returning token in response
      // Send rememberMe flag so backend can set appropriate cookie expiration
      const response = await fetch(`${getApiUrl()}/auth/login`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include', // Required to receive and send httpOnly cookies
        body: JSON.stringify({ email, password, remember_me: rememberMe }),
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
        setError(data.error || t('errors.invalidCredentials'))
        return
      }

      // No need to store token - backend sets httpOnly cookie automatically
      // The cookie will be sent with all subsequent requests via credentials: 'include'
      // Cookie expiration is controlled by backend based on remember_me flag

      // A different user may have used this device before
      await clearApiCache()

      // Redirect to dashboard
      window.location.href = '/dashboard'
    } catch (err) {
      setError(t('errors.failed'))
      console.error('Login error:', err)
    } finally {
      setIsLoading(false)
    }
  }

  // Show loading while checking setup status
  if (checkingSetup) {
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
          />
        </div>

        <div className="flex items-center justify-between text-sm">
          <label className="flex cursor-pointer items-center">
            <input
              type="checkbox"
              checked={rememberMe}
              onChange={(e) => setRememberMe(e.target.checked)}
              className="h-4 w-4 cursor-pointer rounded border"
              style={{
                borderColor: 'var(--color-card-border)',
                accentColor: 'var(--color-primary)',
              }}
              disabled={isLoading}
            />
            <span className="ml-2" style={{ color: 'var(--color-text-secondary)' }}>
              {t('rememberMe')}
            </span>
          </label>
          <Link
            href="/forgot-password"
            className="transition"
            style={{ color: 'var(--color-primary)' }}
            {...linkHover}
          >
            {t('forgotPassword')}
          </Link>
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

      {oauthProviders.length > 0 && (
        <>
          <div className="relative my-6">
            <div className="absolute inset-0 flex items-center">
              <div
                className="w-full border-t"
                style={{ borderColor: 'var(--color-card-border)' }}
              ></div>
            </div>
            <div className="relative flex justify-center text-sm">
              <span
                className="px-2"
                style={{
                  backgroundColor: 'var(--color-card)',
                  color: 'var(--color-text-secondary)',
                }}
              >
                {t('orContinueWith')}
              </span>
            </div>
          </div>

          <div className="space-y-3">
            {oauthProviders.includes('google') && (
              <OAuthButton provider="google" redirectTo="/dashboard" rememberMe={rememberMe} />
            )}
            {oauthProviders.includes('github') && (
              <OAuthButton provider="github" redirectTo="/dashboard" rememberMe={rememberMe} />
            )}
            {oauthProviders.includes('discord') && (
              <OAuthButton provider="discord" redirectTo="/dashboard" rememberMe={rememberMe} />
            )}
            {oauthProviders.includes('oidc') && (
              <OAuthButton provider="oidc" redirectTo="/dashboard" rememberMe={rememberMe} />
            )}
          </div>
        </>
      )}

      {registrationEnabled && (
        <div className="mt-6 text-center text-sm" style={{ color: 'var(--color-text-secondary)' }}>
          {t('noAccount')}{' '}
          <Link
            href="/register"
            className="font-medium transition"
            style={{ color: 'var(--color-primary)' }}
            {...linkHover}
          >
            {t('signUp')}
          </Link>
        </div>
      )}
    </div>
  )
}

export default function LoginPage() {
  const t = useTranslations('auth.login')
  return (
    <Suspense fallback={<div>{t('loading')}</div>}>
      <LoginForm />
    </Suspense>
  )
}
