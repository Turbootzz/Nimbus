'use client'

import { useState, Suspense } from 'react'
import Link from 'next/link'
import { useTranslations } from 'next-intl'
import { useSearchParams } from 'next/navigation'
import { ThemedInput } from '@/components/ui/ThemedInput'
import { useHoverStyle, hoverStyles } from '@/hooks/useHoverStyle'
import { api } from '@/lib/api'

function ResetPasswordForm() {
  const t = useTranslations('auth.resetPassword')
  const searchParams = useSearchParams()
  const token = searchParams.get('token')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState(false)
  const buttonHover = useHoverStyle(hoverStyles.primaryButton, { disabled: isLoading })
  const linkHover = useHoverStyle(hoverStyles.primaryLink)

  if (!token) {
    return (
      <div
        className="rounded-2xl p-8 shadow-xl"
        style={{
          backgroundColor: 'var(--color-card)',
          borderColor: 'var(--color-card-border)',
        }}
      >
        <div className="mb-4 text-center">
          <h2 className="text-2xl font-bold" style={{ color: 'var(--color-text-primary)' }}>
            {t('invalidLink.title')}
          </h2>
          <p className="mt-2" style={{ color: 'var(--color-text-secondary)' }}>
            {t('invalidLink.message')}
          </p>
        </div>
        <div className="text-center text-sm">
          <Link
            href="/forgot-password"
            className="font-medium transition"
            style={{ color: 'var(--color-primary)' }}
            {...linkHover}
          >
            {t('invalidLink.requestNew')}
          </Link>
        </div>
      </div>
    )
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')

    if (newPassword.length < 8) {
      setError(t('errors.passwordTooShort'))
      return
    }

    if (newPassword !== confirmPassword) {
      setError(t('errors.passwordMismatch'))
      return
    }

    setIsLoading(true)

    try {
      const response = await api.resetPassword({ token, new_password: newPassword })
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
          <div className="text-center text-sm">
            <Link
              href="/login"
              className="font-medium transition"
              style={{ color: 'var(--color-primary)' }}
              {...linkHover}
            >
              {t('signInWithNew')}
            </Link>
          </div>
        </div>
      ) : (
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label
              htmlFor="new-password"
              className="mb-1 block text-sm font-medium"
              style={{ color: 'var(--color-text-secondary)' }}
            >
              {t('newPassword')}
            </label>
            <ThemedInput
              id="new-password"
              type="password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              placeholder="••••••••"
              autoComplete="new-password"
              required
              disabled={isLoading}
            />
          </div>

          <div>
            <label
              htmlFor="confirm-password"
              className="mb-1 block text-sm font-medium"
              style={{ color: 'var(--color-text-secondary)' }}
            >
              {t('confirmPassword')}
            </label>
            <ThemedInput
              id="confirm-password"
              type="password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              placeholder="••••••••"
              autoComplete="new-password"
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
      )}
    </div>
  )
}

export default function ResetPasswordPage() {
  const t = useTranslations('auth.resetPassword')
  return (
    <Suspense fallback={<div>{t('loading')}</div>}>
      <ResetPasswordForm />
    </Suspense>
  )
}
