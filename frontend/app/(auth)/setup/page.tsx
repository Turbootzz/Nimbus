'use client'

import { useState, useEffect, useCallback } from 'react'
import { useRouter } from 'next/navigation'
import { CheckCircleIcon, ExclamationTriangleIcon } from '@heroicons/react/24/solid'
import { api } from '@/lib/api'
import { ThemedInput } from '@/components/ui/ThemedInput'
import { useHoverStyle, hoverStyles } from '@/hooks/useHoverStyle'

export default function SetupPage() {
  const router = useRouter()
  const [step, setStep] = useState<'loading' | 'welcome' | 'create' | 'complete' | 'error'>('loading')
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
      setError('Passwords do not match')
      return
    }

    if (password.length < 8) {
      setError('Password must be at least 8 characters')
      return
    }

    setIsLoading(true)

    try {
      const response = await api.createInitialAdmin({ name, email, password })

      if (response.error) {
        setError(response.error.message || 'Failed to create admin account')
        return
      }

      // Success - show complete step briefly, then redirect
      setStep('complete')
      setTimeout(() => {
        window.location.href = '/dashboard'
      }, 1500)
    } catch (err) {
      setError('Failed to create admin account. Please try again.')
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
          Checking setup status...
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
          Connection Error
        </h2>
        <p className="mt-2" style={{ color: 'var(--color-text-secondary)' }}>
          Unable to connect to the Nimbus backend.
        </p>
        <p className="mt-1 text-sm" style={{ color: 'var(--color-text-muted)' }}>
          Please ensure the backend server is running and try again.
        </p>
        <button
          onClick={() => {
            setStep('loading')
            checkSetupStatus()
          }}
          className="mt-6 rounded-lg px-6 py-2.5 font-medium text-white transition focus:ring-4"
          style={{
            backgroundColor: 'var(--color-primary)',
          }}
          {...buttonHover}
        >
          Retry
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
            Welcome to Nimbus
          </h2>
          <p className="mt-2" style={{ color: 'var(--color-text-secondary)' }}>
            Let&apos;s set up your homelab dashboard
          </p>
        </div>

        <div className="mb-6 text-center" style={{ color: 'var(--color-text-secondary)' }}>
          <p className="text-sm">To get started, create an administrator account.</p>
        </div>

        <button
          onClick={() => setStep('create')}
          className="w-full rounded-lg py-2.5 font-medium text-white transition focus:ring-4"
          style={{
            backgroundColor: 'var(--color-primary)',
          }}
          {...buttonHover}
        >
          Get Started
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
          Setup Complete!
        </h2>
        <p className="mt-2" style={{ color: 'var(--color-text-secondary)' }}>
          Redirecting to your dashboard...
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
          Create Admin Account
        </h2>
        <p className="mt-1" style={{ color: 'var(--color-text-secondary)' }}>
          This will be the first administrator
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
            Full Name
          </label>
          <ThemedInput
            id="name"
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Admin"
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
            Back
          </button>
          <button
            type="submit"
            disabled={isLoading}
            className="flex-1 rounded-lg py-2.5 font-medium text-white transition focus:ring-4 disabled:cursor-not-allowed disabled:opacity-50"
            style={{
              backgroundColor: 'var(--color-primary)',
            }}
            {...buttonHover}
          >
            {isLoading ? 'Creating...' : 'Create Admin'}
          </button>
        </div>
      </form>
    </div>
  )
}
