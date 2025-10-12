'use client'

import { useState, useEffect } from 'react'
import Link from 'next/link'
import {
  PlusIcon,
  CheckCircleIcon,
  ExclamationCircleIcon,
  ClockIcon,
  PencilIcon,
  TrashIcon,
} from '@heroicons/react/24/outline'
import { api } from '@/lib/api'
import type { Service } from '@/types'

export default function ServicesPage() {
  const [services, setServices] = useState<Service[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    fetchServices()
  }, [])

  const fetchServices = async () => {
    setIsLoading(true)
    setError('')

    const response = await api.getServices()

    if (response.error) {
      setError(response.error.message)
    } else if (response.data) {
      setServices(response.data)
    }

    setIsLoading(false)
  }

  const handleDelete = async (id: string, name: string) => {
    if (!confirm(`Are you sure you want to delete "${name}"?`)) {
      return
    }

    const response = await api.deleteService(id)

    if (response.error) {
      alert(`Failed to delete service: ${response.error.message}`)
    } else {
      // Remove from list
      setServices(services.filter((s) => s.id !== id))
    }
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'online':
        return 'text-success'
      case 'offline':
        return 'text-error'
      default:
        return 'text-text-muted'
    }
  }

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'online':
        return <CheckCircleIcon className="h-5 w-5" />
      case 'offline':
        return <ExclamationCircleIcon className="h-5 w-5" />
      default:
        return <ClockIcon className="h-5 w-5" />
    }
  }

  if (isLoading) {
    return (
      <div className="flex min-h-96 items-center justify-center">
        <div className="text-center">
          <div className="border-primary mx-auto mb-4 h-12 w-12 animate-spin rounded-full border-t-2 border-b-2"></div>
          <p className="text-text-secondary">Loading services...</p>
        </div>
      </div>
    )
  }

  return (
    <div>
      {/* Page header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-text-primary text-3xl font-bold">Services</h1>
          <p className="text-text-secondary mt-1">Manage your homelab services</p>
        </div>
        <Link
          href="/services/new"
          className="bg-primary hover:bg-primary-hover inline-flex items-center rounded-md px-4 py-2 text-sm font-medium text-white transition-colors"
        >
          <PlusIcon className="mr-2 h-4 w-4" />
          Add Service
        </Link>
      </div>

      {/* Error message */}
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

      {/* Services grid */}
      {services.length === 0 ? (
        <div className="bg-card border-card-border flex flex-col items-center justify-center rounded-lg border p-12 text-center">
          <div className="text-text-muted mb-4 text-6xl">🔗</div>
          <h3 className="text-text-primary mb-2 text-xl font-semibold">No services yet</h3>
          <p className="text-text-secondary mb-6 max-w-md">
            Get started by adding your first service. Services can be websites, apps, or any URL you
            want to monitor.
          </p>
          <Link
            href="/services/new"
            className="bg-primary hover:bg-primary-hover inline-flex items-center rounded-md px-4 py-2 text-sm font-medium text-white transition-colors"
          >
            <PlusIcon className="mr-2 h-4 w-4" />
            Add Your First Service
          </Link>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {services.map((service) => (
            <div
              key={service.id}
              className="bg-card border-card-border group relative rounded-lg border p-6 transition-all hover:shadow-lg"
            >
              {/* Service icon and status */}
              <div className="mb-4 flex items-start justify-between">
                <span className="text-3xl">{service.icon || '🔗'}</span>
                <div className={`flex items-center ${getStatusColor(service.status)}`}>
                  {getStatusIcon(service.status)}
                  <span className="ml-1 text-sm capitalize">{service.status}</span>
                </div>
              </div>

              {/* Service info */}
              <h3 className="text-text-primary mb-1 text-lg font-semibold">{service.name}</h3>
              <p className="text-text-secondary mb-2 text-sm">
                {service.description || 'No description'}
              </p>
              <a
                href={service.url}
                target="_blank"
                rel="noopener noreferrer"
                className="text-primary hover:text-primary-hover mb-4 block truncate text-xs transition-colors"
              >
                {service.url}
              </a>

              {/* Actions */}
              <div
                className="flex items-center gap-2 border-t pt-4"
                style={{ borderColor: 'var(--color-card-border)' }}
              >
                <Link
                  href={`/services/${service.id}/edit`}
                  className="hover:bg-card-border text-text-secondary hover:text-text-primary flex flex-1 items-center justify-center rounded-md px-3 py-2 text-sm font-medium transition-colors"
                >
                  <PencilIcon className="mr-1 h-4 w-4" />
                  Edit
                </Link>
                <button
                  onClick={() => handleDelete(service.id, service.name)}
                  className="hover:bg-error text-error flex flex-1 items-center justify-center rounded-md px-3 py-2 text-sm font-medium transition-colors hover:text-white"
                >
                  <TrashIcon className="mr-1 h-4 w-4" />
                  Delete
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
