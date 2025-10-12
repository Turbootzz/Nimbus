'use client'

import { useState } from 'react'
import Link from 'next/link'

export default function LoginPage() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [rememberMe, setRememberMe] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setIsLoading(true)

    try {
      // Call API with credentials to allow httpOnly cookies
      // Backend will set secure httpOnly cookie instead of returning token in response
      // Send rememberMe flag so backend can set appropriate cookie expiration
      const response = await fetch('http://localhost:8080/api/v1/auth/login', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include', // Required to receive and send httpOnly cookies
        body: JSON.stringify({ email, password, remember_me: rememberMe }),
      })

      const data = await response.json()

      if (!response.ok) {
        setError(data.error || 'Invalid email or password')
        return
      }

      // No need to store token - backend sets httpOnly cookie automatically
      // The cookie will be sent with all subsequent requests via credentials: 'include'
      // Cookie expiration is controlled by backend based on remember_me flag

      // Redirect to dashboard
      window.location.href = '/dashboard'
    } catch (err) {
      setError('Login failed. Please try again.')
      console.error('Login error:', err)
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
      <div className="mb-6">
        <h2 className="text-2xl font-bold" style={{ color: 'var(--color-text-primary)' }}>
          Welcome back
        </h2>
        <p className="mt-1" style={{ color: 'var(--color-text-secondary)' }}>
          Sign in to your account
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
            Email
          </label>
          <input
            id="email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className="w-full rounded-lg border px-4 py-2 transition focus:ring-2 focus:outline-none"
            style={{
              backgroundColor: 'var(--color-background)',
              borderColor: 'var(--color-card-border)',
              color: 'var(--color-text-primary)',
            }}
            onFocus={(e) => {
              e.currentTarget.style.borderColor = 'var(--color-primary)'
            }}
            onBlur={(e) => {
              e.currentTarget.style.borderColor = 'var(--color-card-border)'
            }}
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
          <input
            id="password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="w-full rounded-lg border px-4 py-2 transition focus:ring-2 focus:outline-none"
            style={{
              backgroundColor: 'var(--color-background)',
              borderColor: 'var(--color-card-border)',
              color: 'var(--color-text-primary)',
            }}
            onFocus={(e) => {
              e.currentTarget.style.borderColor = 'var(--color-primary)'
            }}
            onBlur={(e) => {
              e.currentTarget.style.borderColor = 'var(--color-card-border)'
            }}
            placeholder="••••••••"
            required
            disabled={isLoading}
          />
        </div>

        <div className="flex items-center justify-between text-sm">
          <label className="flex items-center cursor-pointer">
            <input
              type="checkbox"
              checked={rememberMe}
              onChange={(e) => setRememberMe(e.target.checked)}
              className="h-4 w-4 rounded border cursor-pointer"
              style={{
                borderColor: 'var(--color-card-border)',
                accentColor: 'var(--color-primary)',
              }}
              disabled={isLoading}
            />
            <span className="ml-2" style={{ color: 'var(--color-text-secondary)' }}>
              Remember me
            </span>
          </label>
          <a
            href="#"
            className="transition"
            style={{ color: 'var(--color-primary)' }}
            onMouseEnter={(e) => {
              e.currentTarget.style.color = 'var(--color-primary-hover)'
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.color = 'var(--color-primary)'
            }}
          >
            Forgot password?
          </a>
        </div>

        <button
          type="submit"
          disabled={isLoading}
          className="w-full rounded-lg py-2.5 font-medium text-white transition focus:ring-4 disabled:cursor-not-allowed disabled:opacity-50"
          style={{
            backgroundColor: 'var(--color-primary)',
          }}
          onMouseEnter={(e) => {
            if (!isLoading) {
              e.currentTarget.style.backgroundColor = 'var(--color-primary-hover)'
            }
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.backgroundColor = 'var(--color-primary)'
          }}
        >
          {isLoading ? 'Signing in...' : 'Sign in'}
        </button>
      </form>

      <div className="mt-6 text-center text-sm" style={{ color: 'var(--color-text-secondary)' }}>
        Don&apos;t have an account?{' '}
        <Link
          href="/register"
          className="font-medium transition"
          style={{ color: 'var(--color-primary)' }}
          onMouseEnter={(e) => {
            e.currentTarget.style.color = 'var(--color-primary-hover)'
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.color = 'var(--color-primary)'
          }}
        >
          Sign up
        </Link>
      </div>
    </div>
  )
}
