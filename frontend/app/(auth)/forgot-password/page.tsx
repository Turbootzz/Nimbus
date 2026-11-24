'use client'

import { useState } from 'react'
import Link from 'next/link'
import { useTranslations } from 'next-intl'
import { ThemedInput } from '@/components/ui/ThemedInput'
import { useHoverStyle, hoverStyles } from '@/hooks/useHoverStyle'
import { api } from '@/lib/api'

export default function ForgotPasswordPage() {
  const t = useTranslations('auth.forgotPassword')
  const [email, setEmail] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState(false)
  const buttonHover = useHoverStyle(hoverStyles.primaryButton, { disabled: isLoading })
  const linkHover = useHoverStyle(hoverStyles.primaryLink)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setIsLoading(true)

    try {
      const response = await api.forgotPassword({ email })
      if (response.error) {
        setError(response.error.message)
      } else {
        setSuccess(true)
      }
    } catch {
      setError(t('errors.generic'))
    } finally {
      setIsLoading(false)
    }
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

      {success ? (
        <div>
          <div
            className="mb-4 rounded-lg border p-3 text-sm"
            style={{
              backgroundColor: 'var(--color-success, #22c55e)',
              borderColor: 'var(--color-success, #22c55e)',
              color: 'white',
              opacity: 0.9,
            }}
          >
            {t('success')}
          </div>
          <div className="text-center text-sm" style={{ color: 'var(--color-text-secondary)' }}>
            <Link
              href="/login"
              className="font-medium transition"
              style={{ color: 'var(--color-primary)' }}
              {...linkHover}
            >
              {t('backToLogin')}
            </Link>
          </div>
        </div>
      ) : (
        <>
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
                autoComplete="email"
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

          <div
            className="mt-6 text-center text-sm"
            style={{ color: 'var(--color-text-secondary)' }}
          >
            {t('rememberPassword')}{' '}
            <Link
              href="/login"
              className="font-medium transition"
              style={{ color: 'var(--color-primary)' }}
              {...linkHover}
            >
              {t('signIn')}
            </Link>
          </div>
        </>
      )}
    </div>
  )
}
