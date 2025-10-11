'use client'

import { useEffect } from 'react'
import { useRouter } from 'next/navigation'

export default function Home() {
  const router = useRouter()

  useEffect(() => {
    // Check if user is authenticated
    const token = localStorage.getItem('auth_token')
    if (token) {
      // If authenticated, redirect to dashboard
      router.push('/dashboard')
    } else {
      // If not authenticated, redirect to login
      router.push('/login')
    }
  }, [router])

  // Show loading state while redirecting
  return (
    <div className="bg-background flex min-h-screen items-center justify-center">
      <div className="text-center">
        <div className="border-primary mx-auto mb-4 h-12 w-12 animate-spin rounded-full border-t-2 border-b-2"></div>
        <p className="text-text-secondary">Redirecting...</p>
      </div>
    </div>
  )
}
