'use client'

import { useState } from 'react'
import Link from 'next/link'

export default function RegisterPage() {
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')

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
      console.log('Register attempt:', { name, email, password })

      // Call API
      const response = await fetch('http://localhost:8080/api/v1/auth/register', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ name, email, password }),
      })

      const data = await response.json()

      if (!response.ok) {
        setError(data.error || 'Registration failed')
        return
      }

      // Save token
      localStorage.setItem('auth_token', data.token)

      // Success!
      alert(`Welcome ${data.user.name}! Registration successful!`)
      console.log('User created:', data.user)

      // TODO: Redirect to dashboard
      // window.location.href = '/dashboard'
    } catch (err) {
      setError('Registration failed. Please try again.')
      console.error('Registration error:', err)
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="rounded-2xl bg-white p-8 shadow-xl">
      <div className="mb-6">
        <h2 className="text-2xl font-bold text-gray-900">Create account</h2>
        <p className="mt-1 text-gray-600">Get started with Nimbus</p>
      </div>

      {error && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">
          {error}
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label htmlFor="name" className="mb-1 block text-sm font-medium text-gray-700">
            Full Name
          </label>
          <input
            id="name"
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="w-full rounded-lg border border-gray-300 px-4 py-2 text-gray-800 placeholder-gray-400 transition focus:border-transparent focus:ring-2 focus:ring-blue-500"
            placeholder="John Doe"
            required
            disabled={isLoading}
          />
        </div>

        <div>
          <label htmlFor="email" className="mb-1 block text-sm font-medium text-gray-700">
            Email
          </label>
          <input
            id="email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className="w-full rounded-lg border border-gray-300 px-4 py-2 text-gray-800 placeholder-gray-400 transition focus:border-transparent focus:ring-2 focus:ring-blue-500"
            placeholder="you@example.com"
            required
            disabled={isLoading}
          />
        </div>

        <div>
          <label htmlFor="password" className="mb-1 block text-sm font-medium text-gray-700">
            Password
          </label>
          <input
            id="password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="w-full rounded-lg border border-gray-300 px-4 py-2 text-gray-800 placeholder-gray-400 transition focus:border-transparent focus:ring-2 focus:ring-blue-500"
            placeholder="••••••••"
            required
            disabled={isLoading}
            minLength={8}
          />
          <p className="mt-1 text-xs text-gray-500">Must be at least 8 characters</p>
        </div>

        <div>
          <label htmlFor="confirmPassword" className="mb-1 block text-sm font-medium text-gray-700">
            Confirm Password
          </label>
          <input
            id="confirmPassword"
            type="password"
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            className="w-full rounded-lg border border-gray-300 px-4 py-2 text-gray-800 placeholder-gray-400 transition focus:border-transparent focus:ring-2 focus:ring-blue-500"
            placeholder="••••••••"
            required
            disabled={isLoading}
          />
        </div>

        <button
          type="submit"
          disabled={isLoading}
          className="w-full rounded-lg bg-blue-600 py-2.5 font-medium text-white transition hover:bg-blue-700 focus:ring-4 focus:ring-blue-200 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {isLoading ? 'Creating account...' : 'Create account'}
        </button>
      </form>

      <div className="mt-6 text-center text-sm text-gray-600">
        Already have an account?{' '}
        <Link href="/login" className="font-medium text-blue-600 hover:text-blue-700">
          Sign in
        </Link>
      </div>
    </div>
  )
}
