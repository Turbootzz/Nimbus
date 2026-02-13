'use client'

import { useState } from 'react'
import { ThemedInput } from '@/components/ui/ThemedInput'
import { api } from '@/lib/api'

export default function ChangePasswordSection() {
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setSuccess('')

    if (newPassword.length < 8) {
      setError('New password must be at least 8 characters')
      return
    }

    if (newPassword !== confirmPassword) {
      setError('New passwords do not match')
      return
    }

    if (currentPassword === newPassword) {
      setError('New password must be different from current password')
      return
    }

    setIsLoading(true)

    try {
      const response = await api.changePassword({
        current_password: currentPassword,
        new_password: newPassword,
      })

      if (response.error) {
        setError(response.error.message)
      } else {
        setSuccess('Password changed successfully')
        setCurrentPassword('')
        setNewPassword('')
        setConfirmPassword('')
      }
    } catch {
      setError('Something went wrong. Please try again.')
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div>
      <h3 className="mb-3 text-sm font-medium" style={{ color: 'var(--color-text-secondary)' }}>
        Change Password
      </h3>

      {error && (
        <div
          className="mb-3 rounded-lg p-3 text-sm"
          style={{
            backgroundColor: 'var(--color-error)',
            color: 'white',
            opacity: 0.9,
          }}
        >
          {error}
        </div>
      )}

      {success && (
        <div
          className="mb-3 rounded-lg p-3 text-sm"
          style={{
            backgroundColor: 'var(--color-success, #22c55e)',
            color: 'white',
            opacity: 0.9,
          }}
        >
          {success}
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-3">
        <div>
          <label htmlFor="current-password" className="sr-only">
            Current password
          </label>
          <ThemedInput
            id="current-password"
            type="password"
            value={currentPassword}
            onChange={(e) => setCurrentPassword(e.target.value)}
            placeholder="Current password"
            autoComplete="current-password"
            required
            disabled={isLoading}
          />
        </div>
        <div>
          <label htmlFor="new-password" className="sr-only">
            New password
          </label>
          <ThemedInput
            id="new-password"
            type="password"
            value={newPassword}
            onChange={(e) => setNewPassword(e.target.value)}
            placeholder="New password"
            autoComplete="new-password"
            required
            disabled={isLoading}
          />
        </div>
        <div>
          <label htmlFor="confirm-new-password" className="sr-only">
            Confirm new password
          </label>
          <ThemedInput
            id="confirm-new-password"
            type="password"
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            placeholder="Confirm new password"
            autoComplete="new-password"
            required
            disabled={isLoading}
          />
        </div>
        <button
          type="submit"
          disabled={isLoading}
          className="rounded-lg px-4 py-2 text-sm font-medium text-white transition disabled:cursor-not-allowed disabled:opacity-50"
          style={{ backgroundColor: 'var(--color-primary)' }}
        >
          {isLoading ? 'Changing...' : 'Change password'}
        </button>
      </form>
    </div>
  )
}
