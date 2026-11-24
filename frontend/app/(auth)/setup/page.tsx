'use client'

import { useState, useEffect, useCallback } from 'react'
import { useRouter } from 'next/navigation'
import { useTranslations } from 'next-intl'
import { CheckCircleIcon, ExclamationTriangleIcon } from '@heroicons/react/24/solid'
import { api } from '@/lib/api'
import { ThemedInput } from '@/components/ui/ThemedInput'
import { useHoverStyle, hoverStyles } from '@/hooks/useHoverStyle'

export default function SetupPage() {
  const t = useTranslations('auth.setup')
  const router = useRouter()
  const [step, setStep] = useState<'loading' | 'welcome' | 'create' | 'complete' | 'error'>(
    'loading'
  )
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')
  const buttonHover = useHoverStyle(hoverStyles.primaryButton, { disabled: isLoading })

  const checkSetupStatus = useCallback(async () => {
    try {
      const response = await api.getSetupStatus()
      if (response.data) {
        if (response.data.needs_setup) {
          setStep('welcome')
        } else {
          // Setup already complete, redirect to login
          router.push('/login')
        }
      } else {
        // API returned an error response
        setStep('error')
      }
    } catch {
      // Network error or API unavailable
      setStep('error')
    }
  }, [router])

  useEffect(() => {
    checkSetupStatus()
  }, [checkSetupStatus])

  const handleCreateAdmin = async (e: React.FormEvent) => {
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
      const response = await api.createInitialAdmin({ name, email, password })

      if (response.error) {
        setError(response.error.message || t('errors.failed'))
        return
      }

      // Success - show complete step briefly, then redirect
      setStep('complete')
      setTimeout(() => {
        window.location.href = '/dashboard'
      }, 1500)
    } catch (err) {
      setError(t('errors.failedRetry'))
      console.error('Setup error:', err)
    } finally {
      setIsLoading(false)
    }
  }

  if (step === 'loading') {
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
        <p className="mt-4" style={{ color: 'var(--color-text-secondary)' }}>
          {t('checking')}
        </p>
      </div>
    )
  }

  if (step === 'error') {
    return (
      <div
        className="rounded-2xl p-8 text-center shadow-xl"
        style={{
          backgroundColor: 'var(--color-card)',
          borderColor: 'var(--color-card-border)',
        }}
      >
        <ExclamationTriangleIcon
          className="mx-auto mb-4 h-16 w-16"
          style={{ color: 'var(--color-warning)' }}
        />
        <h2 className="text-2xl font-bold" style={{ color: 'var(--color-text-primary)' }}>
          {t('connectionError.title')}
        </h2>
        <p className="mt-2" style={{ color: 'var(--color-text-secondary)' }}>
          {t('connectionError.message')}
        </p>
        <p className="mt-1 text-sm" style={{ color: 'var(--color-text-muted)' }}>
          {t('connectionError.hint')}
        </p>
        <button
          onClick={() => {
            setStep('loading')
            checkSetupStatus()
          }}
          className="mt-6 rounded-lg px-6 py-2.5 font-medium text-white transition focus:ring-4"
          style={{
            backgroundColor: 'var(--color-primary)',
            color: 'var(--color-on-primary, white)',
          }}
          {...buttonHover}
        >
          {t('connectionError.retry')}
        </button>
      </div>
    )
  }

  if (step === 'welcome') {
    return (
      <div
        className="rounded-2xl p-8 shadow-xl"
        style={{
          backgroundColor: 'var(--color-card)',
          borderColor: 'var(--color-card-border)',
        }}
      >
        <div className="mb-6 text-center">
          <div className="mb-4 text-5xl">☁️</div>
          <h2 className="text-2xl font-bold" style={{ color: 'var(--color-text-primary)' }}>
            {t('welcome.title')}
          </h2>
          <p className="mt-2" style={{ color: 'var(--color-text-secondary)' }}>
            {t('welcome.subtitle')}
          </p>
        </div>

        <div className="mb-6 text-center" style={{ color: 'var(--color-text-secondary)' }}>
          <p className="text-sm">{t('welcome.intro')}</p>
        </div>

        <button
          onClick={() => setStep('create')}
          className="w-full rounded-lg py-2.5 font-medium text-white transition focus:ring-4"
          style={{
            backgroundColor: 'var(--color-primary)',
            color: 'var(--color-on-primary, white)',
          }}
          {...buttonHover}
        >
          {t('welcome.getStarted')}
        </button>
      </div>
    )
  }

  if (step === 'complete') {
    return (
      <div
        className="rounded-2xl p-8 text-center shadow-xl"
        style={{
          backgroundColor: 'var(--color-card)',
          borderColor: 'var(--color-card-border)',
        }}
      >
        <CheckCircleIcon
          className="mx-auto mb-4 h-16 w-16"
          style={{ color: 'var(--color-success)' }}
        />
        <h2 className="text-2xl font-bold" style={{ color: 'var(--color-text-primary)' }}>
          {t('complete.title')}
        </h2>
        <p className="mt-2" style={{ color: 'var(--color-text-secondary)' }}>
          {t('complete.redirecting')}
        </p>
      </div>
    )
  }

  // Step: create admin
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
          {t('create.title')}
        </h2>
        <p className="mt-1" style={{ color: 'var(--color-text-secondary)' }}>
          {t('create.subtitle')}
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

      <form onSubmit={handleCreateAdmin} className="space-y-4">
        <div>
          <label
            htmlFor="name"
            className="mb-1 block text-sm font-medium"
            style={{ color: 'var(--color-text-secondary)' }}
          >
            {t('create.fullName')}
          </label>
          <ThemedInput
            id="name"
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder={t('create.namePlaceholder')}
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
            {t('create.email')}
          </label>
          <ThemedInput
            id="email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="admin@example.com"
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
            {t('create.password')}
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
            {t('create.passwordHint')}
          </p>
        </div>

        <div>
          <label
            htmlFor="confirmPassword"
            className="mb-1 block text-sm font-medium"
            style={{ color: 'var(--color-text-secondary)' }}
          >
            {t('create.confirmPassword')}
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

        <div className="flex gap-3">
          <button
            type="button"
            onClick={() => setStep('welcome')}
            disabled={isLoading}
            className="flex-1 rounded-lg py-2.5 font-medium transition focus:ring-4 disabled:cursor-not-allowed disabled:opacity-50"
            style={{
              backgroundColor: 'var(--color-card-elevated)',
              color: 'var(--color-text-secondary)',
            }}
          >
            {t('create.back')}
          </button>
          <button
            type="submit"
            disabled={isLoading}
            className="flex-1 rounded-lg py-2.5 font-medium text-white transition focus:ring-4 disabled:cursor-not-allowed disabled:opacity-50"
            style={{
              backgroundColor: 'var(--color-primary)',
              color: 'var(--color-on-primary, white)',
            }}
            {...buttonHover}
          >
            {isLoading ? t('create.submitting') : t('create.submit')}
          </button>
        </div>
      </form>
    </div>
  )
}
