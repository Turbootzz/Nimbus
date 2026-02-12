'use client'

import { useState } from 'react'
import Link from 'next/link'
import { ThemedInput } from '@/components/ui/ThemedInput'
import { useHoverStyle, hoverStyles } from '@/hooks/useHoverStyle'
import { api } from '@/lib/api'

export default function ForgotPasswordPage() {
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
      setError('Something went wrong. Please try again.')
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
          Reset your password
        </h2>
        <p className="mt-1" style={{ color: 'var(--color-text-secondary)' }}>
          Enter your email and we&apos;ll send you a reset link
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
            If an account with that email exists, a password reset link has been sent. Check your
            inbox.
          </div>
          <div className="text-center text-sm" style={{ color: 'var(--color-text-secondary)' }}>
            <Link
              href="/login"
              className="font-medium transition"
              style={{ color: 'var(--color-primary)' }}
              {...linkHover}
            >
              Back to login
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
                Email
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

            <button
              type="submit"
              disabled={isLoading}
              className="w-full rounded-lg py-2.5 font-medium text-white transition focus:ring-4 disabled:cursor-not-allowed disabled:opacity-50"
              style={{
                backgroundColor: 'var(--color-primary)',
              }}
              {...buttonHover}
            >
              {isLoading ? 'Sending...' : 'Send reset link'}
            </button>
          </form>

          <div
            className="mt-6 text-center text-sm"
            style={{ color: 'var(--color-text-secondary)' }}
          >
            Remember your password?{' '}
            <Link
              href="/login"
              className="font-medium transition"
              style={{ color: 'var(--color-primary)' }}
              {...linkHover}
            >
              Sign in
            </Link>
          </div>
        </>
      )}
    </div>
  )
}
