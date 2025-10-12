'use client'

import { useEffect } from 'react'
import { useRouter } from 'next/navigation'

export default function Home() {
  const router = useRouter()

  useEffect(() => {
    // Check if user is authenticated by calling the backend
    // The httpOnly cookie will be sent automatically
    const checkAuth = async () => {
      try {
        const response = await fetch('http://localhost:8080/api/v1/auth/me', {
          method: 'GET',
          credentials: 'include', // Required to send httpOnly cookie
        })

        if (response.ok) {
          router.push('/dashboard')
        } else {
          router.push('/login')
        }
      } catch (error) {
        console.error('Auth check error:', error)
        router.push('/login')
      }
    }

    checkAuth()
  }, [router])

  // Show loading state while checking authentication
  return (
    <div className="bg-background flex min-h-screen items-center justify-center">
      <div className="text-center">
        <div className="border-primary mx-auto mb-4 h-12 w-12 animate-spin rounded-full border-t-2 border-b-2"></div>
        <p className="text-text-secondary">Redirecting...</p>
      </div>
    </div>
  )
}
