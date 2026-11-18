'use client'

import { useState, useEffect } from 'react'
import { useRouter, useParams } from 'next/navigation'
import Link from 'next/link'
import { ArrowLeftIcon } from '@heroicons/react/24/outline'
import { api } from '@/lib/api'
import ServiceForm from '@/components/ServiceForm'
import type { Service } from '@/types'

export default function EditServicePage() {
  const router = useRouter()
  const params = useParams()
  const serviceId = params.id as string

  const [service, setService] = useState<Service | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')

  // Fetch service data
  useEffect(() => {
    const fetchService = async () => {
      setIsLoading(true)
      setError('')
      try {
        const response = await api.getService(serviceId)
        if (response.error || !response.data) {
          setError(response.error?.message || 'Service not found')
        } else {
          setService(response.data)
        }
      } catch (error) {
        setError(error instanceof Error ? error.message : 'Failed to load service')
      } finally {
        setIsLoading(false)
      }
    }

    fetchService()
  }, [serviceId])

  if (isLoading) {
    return (
      <div className="flex min-h-96 items-center justify-center">
        <div className="wallpaper-panel text-center">
          <div className="border-primary mx-auto mb-4 h-12 w-12 animate-spin rounded-full border-t-2 border-b-2"></div>
          <p className="text-text-secondary">Loading service...</p>
        </div>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-2xl">
      {/* Back button */}
      <Link
        href="/services"
        className="text-text-secondary hover:text-text-primary wallpaper-panel mb-6 inline-flex items-center text-sm transition-colors"
      >
        <ArrowLeftIcon className="mr-2 h-4 w-4" />
        Back to Services
      </Link>

      {/* Page header */}
      <div className="wallpaper-panel mb-6">
        <h1 className="text-text-primary text-3xl font-bold">Edit Service</h1>
        <p className="text-text-secondary mt-1">Update your service information</p>
      </div>

      {service ? (
        <ServiceForm
          service={service}
          cancelHref="/services"
          onSaved={() => router.push('/services')}
        />
      ) : (
        <div
          role="alert"
          className="bg-error border-error rounded-lg border p-3 text-sm text-white opacity-90"
        >
          {error}
        </div>
      )}
    </div>
  )
}
