'use client'

import { useState } from 'react'
import Link from 'next/link'
import { getApiUrl } from '@/lib/utils/api-url'
import { ThemedInput } from '@/components/ui/ThemedInput'
import { useHoverStyle, hoverStyles } from '@/hooks/useHoverStyle'

export default function RegisterPage() {
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')
  const buttonHover = useHoverStyle(hoverStyles.primaryButton, { disabled: isLoading })
  const linkHover = useHoverStyle(hoverStyles.primaryLink)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')

    // Validation
    if (password !== confirmPassword) {
      setError('Passwords do not match')
      return
    }

    if (password.length < 8) {
      setError('Password must be at least 8 characters')
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
          setError(
            'Cannot reach API server. If using Docker, ensure NEXT_PUBLIC_API_URL is set to your server IP (e.g., http://192.168.1.100:8080), not "http://backend:8080".'
          )
        } else {
          setError('API returned an invalid response. Check server configuration.')
        }
        return
      }

      if (!response.ok) {
        setError(data.error || 'Registration failed')
        return
      }

      // No need to store token - backend sets httpOnly cookie automatically
      // The cookie will be sent with all subsequent requests via credentials: 'include'

      // Redirect to dashboard
      window.location.href = '/dashboard'
    } catch (err) {
      setError('Registration failed. Please try again.')
      console.error('Registration error:', err)
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
          Create account
        </h2>
        <p className="mt-1" style={{ color: 'var(--color-text-secondary)' }}>
          Get started with Nimbus
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
            Full Name
          </label>
          <ThemedInput
            id="name"
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="John Doe"
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

        <div>
          <label
            htmlFor="password"
            className="mb-1 block text-sm font-medium"
            style={{ color: 'var(--color-text-secondary)' }}
          >
            Password
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
            Must be at least 8 characters
          </p>
        </div>

        <div>
          <label
            htmlFor="confirmPassword"
            className="mb-1 block text-sm font-medium"
            style={{ color: 'var(--color-text-secondary)' }}
          >
            Confirm Password
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
          }}
          {...buttonHover}
        >
          {isLoading ? 'Creating account...' : 'Create account'}
        </button>
      </form>

      <div className="mt-6 text-center text-sm" style={{ color: 'var(--color-text-secondary)' }}>
        Already have an account?{' '}
        <Link
          href="/login"
          className="font-medium transition"
          style={{ color: 'var(--color-primary)' }}
          {...linkHover}
        >
          Sign in
        </Link>
      </div>
    </div>
  )
}
