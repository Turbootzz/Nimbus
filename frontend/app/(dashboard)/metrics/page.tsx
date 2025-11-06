'use client'

import { useState, useEffect } from 'react'
import Link from 'next/link'
import { ChartBarIcon, ServerIcon } from '@heroicons/react/24/outline'
import { CheckCircleIcon, ExclamationCircleIcon } from '@heroicons/react/24/solid'
import { Service } from '@/types'
import { api } from '@/lib/api'
import CombinedMetricsChart from '@/components/graphs/CombinedMetricsChart'

export default function MetricsPage() {
  const [services, setServices] = useState<Service[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const fetchServices = async () => {
      try {
        const response = await api.getServices()

        if (response.error) {
          throw new Error(response.error.message || 'Failed to fetch services')
        }

        if (response.data) {
          setServices(response.data)
        }
      } catch (err) {
        console.error('Error fetching services:', err)
        setError(err instanceof Error ? err.message : 'Failed to fetch services')
      } finally {
        setLoading(false)
      }
    }

    fetchServices()
  }, [])

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="text-text-secondary">Loading metrics...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center">
        <div className="text-error mb-4">{error}</div>
        <Link href="/dashboard" className="text-primary hover:text-primary-hover">
          Back to Dashboard
        </Link>
      </div>
    )
  }

  const onlineServices = services.filter((s) => s.status === 'online').length
  const offlineServices = services.filter((s) => s.status === 'offline').length

  const avgResponseTime = services
    .filter((s) => s.response_time !== null && s.response_time !== undefined)
    .reduce((sum, s) => sum + (s.response_time || 0), 0)

  const servicesWithResponse = services.filter(
    (s) => s.response_time !== null && s.response_time !== undefined
  ).length

  const avgResponse = servicesWithResponse > 0 ? avgResponseTime / servicesWithResponse : 0

  return (
    <div className="mx-auto max-w-7xl">
      {/* Header */}
      <div className="mb-6">
        <h1 className="text-text-primary text-3xl font-bold">Metrics Overview</h1>
        <p className="text-text-secondary mt-2">
          Monitor the health and performance of all your services
        </p>
      </div>

      {/* Overview Stats */}
      <div className="mb-6 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <div className="border-card-border bg-card rounded-lg border p-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-text-secondary text-sm">Total Services</p>
              <p className="text-text-primary mt-1 text-3xl font-bold">{services.length}</p>
            </div>
            <ServerIcon className="text-text-muted h-12 w-12" />
          </div>
        </div>

        <div className="border-card-border bg-card rounded-lg border p-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-text-secondary text-sm">Online</p>
              <p className="text-success mt-1 text-3xl font-bold">{onlineServices}</p>
            </div>
            <CheckCircleIcon className="text-success h-12 w-12" />
          </div>
        </div>

        <div className="border-card-border bg-card rounded-lg border p-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-text-secondary text-sm">Offline</p>
              <p className="text-error mt-1 text-3xl font-bold">{offlineServices}</p>
            </div>
            <ExclamationCircleIcon className="text-error h-12 w-12" />
          </div>
        </div>

        <div className="border-card-border bg-card rounded-lg border p-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-text-secondary text-sm">Avg Response</p>
              <p className="text-text-primary mt-1 text-3xl font-bold">
                {avgResponse.toFixed(0)}ms
              </p>
            </div>
            <ChartBarIcon className="text-text-muted h-12 w-12" />
          </div>
        </div>
      </div>

      {/* Combined Metrics Chart */}
      {services.length > 0 && (
        <div className="mb-6">
          <CombinedMetricsChart
            serviceIds={services.map((s) => s.id)}
            serviceNames={services.reduce(
              (acc, s) => {
                acc[s.id] = s.name
                return acc
              },
              {} as { [key: string]: string }
            )}
          />
        </div>
      )}

      {/* Services List */}
      <div className="border-card-border bg-card rounded-lg border p-6">
        <h2 className="text-text-primary mb-4 text-lg font-semibold">All Services</h2>

        {services.length === 0 ? (
          <div className="py-12 text-center">
            <ServerIcon className="text-text-muted mx-auto h-12 w-12" />
            <p className="text-text-secondary mt-2">No services configured yet</p>
            <Link
              href="/services/new"
              className="bg-primary mt-4 inline-block rounded px-4 py-2 text-white hover:opacity-90"
            >
              Add Your First Service
            </Link>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-card-border border-b">
                  <th className="text-text-secondary px-4 py-3 text-left text-sm font-medium">
                    Service
                  </th>
                  <th className="text-text-secondary px-4 py-3 text-left text-sm font-medium">
                    Status
                  </th>
                  <th className="text-text-secondary px-4 py-3 text-left text-sm font-medium">
                    Response Time
                  </th>
                  <th className="text-text-secondary px-4 py-3 text-left text-sm font-medium">
                    URL
                  </th>
                  <th className="text-text-secondary px-4 py-3 text-right text-sm font-medium">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="divide-card-border divide-y">
                {services.map((service) => (
                  <tr key={service.id} className="hover:bg-card-hover transition-colors">
                    <td className="px-4 py-4">
                      <div className="flex items-center gap-3">
                        <span className="text-2xl">{service.icon || '🔗'}</span>
                        <div>
                          <p className="text-text-primary font-medium">{service.name}</p>
                          {service.description && (
                            <p className="text-text-secondary text-sm">{service.description}</p>
                          )}
                        </div>
                      </div>
                    </td>
                    <td className="px-4 py-4">
                      <div className="flex items-center gap-2">
                        {service.status === 'online' ? (
                          <>
                            <CheckCircleIcon className="text-success h-5 w-5" />
                            <span className="text-success text-sm capitalize">Online</span>
                          </>
                        ) : service.status === 'offline' ? (
                          <>
                            <ExclamationCircleIcon className="text-error h-5 w-5" />
                            <span className="text-error text-sm capitalize">Offline</span>
                          </>
                        ) : (
                          <>
                            <div className="bg-warning h-5 w-5 rounded-full" />
                            <span className="text-warning text-sm capitalize">Unknown</span>
                          </>
                        )}
                      </div>
                    </td>
                    <td className="px-4 py-4">
                      {service.response_time !== null && service.response_time !== undefined ? (
                        <span
                          className={`text-sm font-medium ${
                            service.response_time < 200
                              ? 'text-success'
                              : service.response_time < 500
                                ? 'text-warning'
                                : 'text-error'
                          }`}
                        >
                          {service.response_time}ms
                        </span>
                      ) : (
                        <span className="text-text-muted text-sm">—</span>
                      )}
                    </td>
                    <td className="px-4 py-4">
                      <a
                        href={service.url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-primary hover:text-primary-hover text-sm"
                      >
                        {service.url}
                      </a>
                    </td>
                    <td className="px-4 py-4 text-right">
                      <Link
                        href={`/services/${service.id}`}
                        className="bg-primary inline-flex items-center gap-1 rounded px-3 py-1.5 text-sm text-white transition-opacity hover:opacity-90"
                      >
                        <ChartBarIcon className="h-4 w-4" />
                        View Metrics
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  )
}
