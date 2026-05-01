'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { ThemedInput } from '@/components/ui/ThemedInput'
import { api } from '@/lib/api'

interface DangerZoneSectionProps {
  email: string
}

export default function DangerZoneSection({ email }: DangerZoneSectionProps) {
  const router = useRouter()
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [confirmText, setConfirmText] = useState('')
  const [isDeleting, setIsDeleting] = useState(false)
  const [error, setError] = useState('')

  const closeModal = () => {
    if (isDeleting) return
    setIsModalOpen(false)
    setConfirmText('')
    setError('')
  }

  const handleDelete = async () => {
    setError('')
    setIsDeleting(true)
    try {
      const response = await api.deleteAccount()
      if (response.error) {
        setError(response.error.message)
        setIsDeleting(false)
        return
      }
      router.push('/login')
    } catch {
      setError('Something went wrong. Please try again.')
      setIsDeleting(false)
    }
  }

  const isConfirmed = confirmText === email

  return (
    <div>
      <h3 className="mb-2 text-sm font-medium" style={{ color: 'var(--color-error)' }}>
        Danger Zone
      </h3>
      <p className="mb-3 text-sm" style={{ color: 'var(--color-text-secondary)' }}>
        Permanently delete your account and all associated data. This action cannot be undone.
      </p>
      <button
        type="button"
        onClick={() => setIsModalOpen(true)}
        className="rounded-lg px-4 py-2 text-sm font-medium text-white transition hover:opacity-90"
        style={{ backgroundColor: 'var(--color-error)' }}
      >
        Delete my account
      </button>

      {isModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <div className="absolute inset-0 bg-black/50" onClick={closeModal} />
          <div
            className="relative z-10 w-full max-w-md rounded-lg border p-6 shadow-lg"
            style={{
              backgroundColor: 'var(--color-card)',
              borderColor: 'var(--color-card-border)',
            }}
          >
            <h3
              className="mb-2 text-lg font-semibold"
              style={{ color: 'var(--color-text-primary)' }}
            >
              Delete account
            </h3>
            <p className="mb-4 text-sm" style={{ color: 'var(--color-text-secondary)' }}>
              This will permanently delete your account, services, groups, preferences, and all
              other associated data. This cannot be undone.
            </p>
            <p className="mb-2 text-sm" style={{ color: 'var(--color-text-secondary)' }}>
              Type <span className="font-mono font-semibold">{email}</span> to confirm.
            </p>
            <ThemedInput
              type="text"
              value={confirmText}
              onChange={(e) => setConfirmText(e.target.value)}
              placeholder={email}
              autoComplete="off"
              disabled={isDeleting}
            />

            {error && (
              <div
                className="mt-3 rounded-lg p-3 text-sm"
                style={{
                  backgroundColor: 'var(--color-error)',
                  color: 'white',
                  opacity: 0.9,
                }}
              >
                {error}
              </div>
            )}

            <div className="mt-4 flex justify-end gap-2">
              <button
                type="button"
                onClick={closeModal}
                disabled={isDeleting}
                className="rounded-lg px-4 py-2 text-sm font-medium transition disabled:cursor-not-allowed disabled:opacity-50"
                style={{ color: 'var(--color-text-secondary)' }}
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleDelete}
                disabled={!isConfirmed || isDeleting}
                className="rounded-lg px-4 py-2 text-sm font-medium text-white transition hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-50"
                style={{ backgroundColor: 'var(--color-error)' }}
              >
                {isDeleting ? 'Deleting...' : 'Delete account'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
