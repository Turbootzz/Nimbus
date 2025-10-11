'use client'

import { useState } from 'react'
import Link from 'next/link'

export default function LoginPage() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setIsLoading(true)

    try {
      console.log('Login attempt:', { email, password })

      // Call API
      const response = await fetch('http://localhost:8080/api/v1/auth/login', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ email, password }),
      })

      const data = await response.json()

      if (!response.ok) {
        setError(data.error || 'Invalid email or password')
        return
      }

      // Save token
      localStorage.setItem('auth_token', data.token)

      // Success!
      alert(`Welcome back ${data.user.name}!`)
      console.log('Logged in as:', data.user)

      // TODO: Redirect to dashboard
      // window.location.href = '/dashboard'
    } catch (err) {
      setError('Login failed. Please try again.')
      console.error('Login error:', err)
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="rounded-2xl bg-white p-8 shadow-xl">
      <div className="mb-6">
        <h2 className="text-2xl font-bold text-gray-900">Welcome back</h2>
        <p className="mt-1 text-gray-600">Sign in to your account</p>
      </div>

      {error && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">
          {error}
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-4">
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
          />
        </div>

        <div className="flex items-center justify-between text-sm">
          <label className="flex items-center">
            <input
              type="checkbox"
              className="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
            <span className="ml-2 text-gray-600">Remember me</span>
          </label>
          <a href="#" className="text-blue-600 hover:text-blue-700">
            Forgot password?
          </a>
        </div>

        <button
          type="submit"
          disabled={isLoading}
          className="w-full rounded-lg bg-blue-600 py-2.5 font-medium text-white transition hover:bg-blue-700 focus:ring-4 focus:ring-blue-200 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {isLoading ? 'Signing in...' : 'Sign in'}
        </button>
      </form>

      <div className="mt-6 text-center text-sm text-gray-600">
        Don&apos;t have an account?{' '}
        <Link href="/register" className="font-medium text-blue-600 hover:text-blue-700">
          Sign up
        </Link>
      </div>
    </div>
  )
}
