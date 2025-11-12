'use client'

import { useState, useEffect } from 'react'
import {
  ServerIcon,
  ClockIcon,
  CheckCircleIcon,
  ExclamationCircleIcon,
  PlusIcon,
} from '@heroicons/react/24/outline'
import Link from 'next/link'
import { api } from '@/lib/api'
import type { Service, Category } from '@/types'
import { useTheme } from '@/contexts/ThemeContext'
import { getStatusColor, getStatusIcon, getResponseTimeColor } from '@/lib/status-utils'
import CategorySection from '@/components/CategorySection'

export default function DashboardPage() {
  const { openInNewTab } = useTheme()
  const [services, setServices] = useState<Service[]>([])
  const [categories, setCategories] = useState<Category[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [stats, setStats] = useState({
    total: 0,
    online: 0,
    offline: 0,
    avgResponseTime: 0,
  })

  useEffect(() => {
    fetchData()
  }, [])

  useEffect(() => {
    // Calculate stats
    const online = services.filter((s) => s.status === 'online').length
    const offline = services.filter((s) => s.status === 'offline').length
    const responseTimes = services
      .filter((s) => s.response_time !== undefined && s.response_time !== null)
      .map((s) => s.response_time as number)
    const avgResponseTime =
      responseTimes.length > 0
        ? Math.round(responseTimes.reduce((a, b) => a + b, 0) / responseTimes.length)
        : 0

    setStats({
      total: services.length,
      online,
      offline,
      avgResponseTime,
    })
  }, [services])

  const fetchData = async () => {
    setIsLoading(true)
    try {
      const [servicesResponse, categoriesResponse] = await Promise.all([
        api.getServices(),
        api.getCategories(),
      ])

      if (servicesResponse.data) {
        setServices(servicesResponse.data)
      }

      if (categoriesResponse.data) {
        setCategories(categoriesResponse.data)
      }
    } catch (error) {
      console.error('Failed to fetch data:', error)
    } finally {
      setIsLoading(false)
    }
  }

  if (isLoading) {
    return (
      <div className="flex min-h-96 items-center justify-center">
        <div className="text-center">
          <div className="border-primary mx-auto mb-4 h-12 w-12 animate-spin rounded-full border-t-2 border-b-2"></div>
          <p className="text-text-secondary">Loading dashboard...</p>
        </div>
      </div>
    )
  }

  return (
    <div>
      {/* Page header */}
      <div className="mb-8">
        <h1 className="text-text-primary text-3xl font-bold">Welcome to Nimbus</h1>
        <p className="text-text-secondary mt-2">Monitor and manage your homelab services</p>
      </div>

      {/* Stats cards */}
      <div className="mb-8 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <div className="bg-card border-card-border rounded-lg border p-6">
          <div className="flex items-center">
            <ServerIcon className="text-primary h-8 w-8" />
            <div className="ml-4">
              <p className="text-text-muted text-sm">Total Services</p>
              <p className="text-text-primary text-2xl font-semibold">{stats.total}</p>
            </div>
          </div>
        </div>

        <div className="bg-card border-card-border rounded-lg border p-6">
          <div className="flex items-center">
            <CheckCircleIcon className="text-success h-8 w-8" />
            <div className="ml-4">
              <p className="text-text-muted text-sm">Online</p>
              <p className="text-text-primary text-2xl font-semibold">{stats.online}</p>
            </div>
          </div>
        </div>

        <div className="bg-card border-card-border rounded-lg border p-6">
          <div className="flex items-center">
            <ExclamationCircleIcon className="text-error h-8 w-8" />
            <div className="ml-4">
              <p className="text-text-muted text-sm">Offline</p>
              <p className="text-text-primary text-2xl font-semibold">{stats.offline}</p>
            </div>
          </div>
        </div>

        <div className="bg-card border-card-border rounded-lg border p-6">
          <div className="flex items-center">
            <ClockIcon className="text-info h-8 w-8" />
            <div className="ml-4">
              <p className="text-text-muted text-sm">Avg Response</p>
              <p className="text-text-primary text-2xl font-semibold">{stats.avgResponseTime}ms</p>
            </div>
          </div>
        </div>
      </div>

      {/* Services grid */}
      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-text-primary text-xl font-semibold">Services</h2>
        <Link
          href="/services/new"
          className="bg-primary hover:bg-primary-hover inline-flex items-center rounded-md px-4 py-2 text-sm font-medium text-white transition-colors"
        >
          <PlusIcon className="mr-2 h-4 w-4" />
          Add Service
        </Link>
      </div>

      {services.length === 0 ? (
        <div className="bg-card border-card-border flex flex-col items-center justify-center rounded-lg border p-12 text-center">
          <div className="text-text-muted mb-4 text-6xl">🔗</div>
          <h3 className="text-text-primary mb-2 text-xl font-semibold">No services yet</h3>
          <p className="text-text-secondary mb-6 max-w-md">
            Get started by adding your first service to monitor.
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
        <>
          {/* Render services grouped by category */}
          {categories.map((category) => {
            const categoryServices = services.filter((s) => s.category_id === category.id)
            return (
              <CategorySection
                key={category.id}
                category={category}
                services={categoryServices}
                renderService={(service) => (
                  <a
                    key={service.id}
                    href={service.url}
                    target={openInNewTab ? '_blank' : '_self'}
                    {...(openInNewTab && { rel: 'noopener noreferrer' })}
                    className="bg-card border-card-border hover:border-primary block rounded-lg border p-6 transition-all hover:shadow-lg"
                  >
                    <div className="mb-4 flex items-start justify-between">
                      <span className="text-3xl">{service.icon}</span>
                      <div className={`flex items-center ${getStatusColor(service.status)}`}>
                        {getStatusIcon(service.status)}
                        <span className="ml-1 text-sm capitalize">{service.status}</span>
                      </div>
                    </div>

                    <h3 className="text-text-primary mb-1 text-lg font-semibold">{service.name}</h3>
                    <p className="text-text-secondary mb-3 text-sm">{service.description}</p>

                    {service.response_time !== undefined && service.response_time !== null && (
                      <div
                        className={`flex items-center text-xs ${getResponseTimeColor(service.response_time)}`}
                      >
                        <ClockIcon className="mr-1 h-3 w-3" />
                        {service.response_time}ms
                      </div>
                    )}
                  </a>
                )}
              />
            )
          })}

          {/* Uncategorized services */}
          <CategorySection
            category={null}
            services={services.filter((s) => !s.category_id)}
            renderService={(service) => (
              <a
                key={service.id}
                href={service.url}
                target={openInNewTab ? '_blank' : '_self'}
                {...(openInNewTab && { rel: 'noopener noreferrer' })}
                className="bg-card border-card-border hover:border-primary block rounded-lg border p-6 transition-all hover:shadow-lg"
              >
                <div className="mb-4 flex items-start justify-between">
                  <span className="text-3xl">{service.icon}</span>
                  <div className={`flex items-center ${getStatusColor(service.status)}`}>
                    {getStatusIcon(service.status)}
                    <span className="ml-1 text-sm capitalize">{service.status}</span>
                  </div>
                </div>

                <h3 className="text-text-primary mb-1 text-lg font-semibold">{service.name}</h3>
                <p className="text-text-secondary mb-3 text-sm">{service.description}</p>

                {service.response_time !== undefined && service.response_time !== null && (
                  <div
                    className={`flex items-center text-xs ${getResponseTimeColor(service.response_time)}`}
                  >
                    <ClockIcon className="mr-1 h-3 w-3" />
                    {service.response_time}ms
                  </div>
                )}
              </a>
            )}
          />

          {/* Add new service card */}
          <div className="mt-6">
            <Link
              href="/services/new"
              className="bg-card border-card-border hover:border-primary hover:bg-primary-light flex cursor-pointer flex-col items-center justify-center rounded-lg border-2 border-dashed p-6 transition-all"
            >
              <PlusIcon className="text-text-muted mb-2 h-12 w-12" />
              <span className="text-text-secondary">Add Service</span>
            </Link>
          </div>
        </>
      )}
    </div>
  )
}
