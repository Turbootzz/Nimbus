'use client'

import { useState, useEffect, useCallback } from 'react'
import { useParams, useRouter } from 'next/navigation'
import Link from 'next/link'
import { ArrowLeftIcon, ArrowPathIcon, PencilIcon, TrashIcon } from '@heroicons/react/24/outline'
import { CheckCircleIcon, ExclamationCircleIcon, ClockIcon } from '@heroicons/react/24/solid'
import { Service, MetricsResponse, TimeRangeOption } from '@/types'
import { api } from '@/lib/api'
import UptimeChart from '@/components/graphs/UptimeChart'
import MetricsCard from '@/components/graphs/MetricsCard'
import { getApiUrl } from '@/lib/utils/api-url'
import { getResponseTimeColor } from '@/lib/status-utils'

const API_URL = getApiUrl()

export default function ServiceDetailPage() {
  const params = useParams()
  const router = useRouter()
  const serviceId = params.id as string

  const [service, setService] = useState<Service | null>(null)
  const [metrics, setMetrics] = useState<MetricsResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [metricsLoading, setMetricsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [timeRange, setTimeRange] = useState<TimeRangeOption>('24h')
  const [isRefreshing, setIsRefreshing] = useState(false)

  const timeRangeOptions: { value: TimeRangeOption; label: string }[] = [
    { value: '1h', label: 'Last Hour' },
    { value: '6h', label: 'Last 6 Hours' },
    { value: '24h', label: 'Last 24 Hours' },
    { value: '7d', label: 'Last 7 Days' },
    { value: '30d', label: 'Last 30 Days' },
  ]

  const fetchService = useCallback(async () => {
    try {
      console.log('Fetching service:', serviceId)
      const response = await api.getService(serviceId)

      if (response.error) {
        throw new Error(response.error.message || 'Failed to fetch service')
      }

      if (!response.data) {
        throw new Error('Service not found')
      }

      console.log('Service data:', response.data)
      setService(response.data)
      setError(null)
    } catch (err) {
      console.error('Error fetching service:', err)
      setError(err instanceof Error ? err.message : 'Failed to fetch service')
    } finally {
      setLoading(false)
    }
  }, [serviceId])

  const fetchMetrics = useCallback(async () => {
    try {
      setMetricsLoading(true)
      // API_URL already includes /api/v1, so just append /metrics/:id
      const url = `${API_URL}/metrics/${serviceId}?range=${timeRange}&interval=5`
      console.log('Fetching metrics from:', url)
      const response = await fetch(url, {
        credentials: 'include',
      })

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}))
        console.error('Metrics fetch failed:', response.status, errorData)
        throw new Error(errorData.error || `Failed to fetch metrics (${response.status})`)
      }

      const data = await response.json()
      console.log('Metrics data:', data)
      setMetrics(data)
    } catch (err) {
      console.error('Failed to fetch metrics:', err)
      // Don't throw - just log the error and show "no data available" message
    } finally {
      setMetricsLoading(false)
    }
  }, [serviceId, timeRange])

  const handleRefresh = async () => {
    setIsRefreshing(true)
    try {
      await fetch(`${API_URL}/services/${serviceId}/check`, {
        method: 'POST',
        credentials: 'include',
      })

      // Refresh service data and metrics
      await Promise.all([fetchService(), fetchMetrics()])
    } catch (err) {
      console.error('Failed to refresh service:', err)
    } finally {
      setIsRefreshing(false)
    }
  }

  const handleDelete = async () => {
    if (!confirm(`Are you sure you want to delete ${service?.name}?`)) {
      return
    }

    try {
      const response = await api.deleteService(serviceId)

      if (response.error) {
        throw new Error(response.error.message || 'Failed to delete service')
      }

      router.push('/dashboard')
    } catch {
      alert('Failed to delete service')
    }
  }

  useEffect(() => {
    fetchService()
  }, [fetchService])

  useEffect(() => {
    if (service) {
      fetchMetrics()
    }
  }, [fetchMetrics, service])

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="text-text-secondary">Loading...</div>
      </div>
    )
  }

  if (error || !service) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center">
        <div className="text-error mb-4">{error || 'Service not found'}</div>
        <Link href="/dashboard" className="text-primary hover:text-primary-hover">
          Back to Dashboard
        </Link>
      </div>
    )
  }

  // Local status icon with larger size for detail page
  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'online':
        return <CheckCircleIcon className="h-6 w-6 text-green-500" />
      case 'offline':
        return <ExclamationCircleIcon className="h-6 w-6 text-red-500" />
      default:
        return <ClockIcon className="h-6 w-6 text-yellow-500" />
    }
  }

  return (
    <div className="mx-auto max-w-7xl p-4 sm:p-6">
      {/* Back Link */}
      <Link
        href="/dashboard"
        className="text-text-secondary hover:text-text-primary mb-4 inline-flex items-center"
      >
        <ArrowLeftIcon className="mr-2 h-5 w-5" />
        Back to Dashboard
      </Link>

      {/* Service Header */}
      <div className="border-card-border bg-card mb-6 rounded-lg border p-4 sm:p-6">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div className="flex items-start gap-3 sm:gap-4">
            <div className="text-3xl sm:text-4xl">{service.icon || '🔗'}</div>
            <div className="min-w-0 flex-1">
              <h1 className="text-text-primary text-2xl font-bold break-words sm:text-3xl">
                {service.name}
              </h1>
              <a
                href={service.url}
                target="_blank"
                rel="noopener noreferrer"
                className="text-primary hover:text-primary-hover block truncate text-sm"
              >
                {service.url}
              </a>
              {service.description && (
                <p className="text-text-secondary mt-2 text-sm sm:text-base">
                  {service.description}
                </p>
              )}
            </div>
          </div>

          <div className="flex shrink-0 gap-2 sm:self-start">
            <button
              onClick={handleRefresh}
              disabled={isRefreshing}
              className="text-text-secondary hover:text-text-primary p-2 disabled:opacity-50"
              title="Refresh"
            >
              <ArrowPathIcon className={`h-5 w-5 ${isRefreshing ? 'animate-spin' : ''}`} />
            </button>
            <Link
              href={`/services/${serviceId}/edit`}
              className="text-text-secondary hover:text-text-primary p-2"
              title="Edit"
            >
              <PencilIcon className="h-5 w-5" />
            </Link>
            <button
              onClick={handleDelete}
              className="text-error p-2 hover:opacity-80"
              title="Delete"
            >
              <TrashIcon className="h-5 w-5" />
            </button>
          </div>
        </div>
      </div>

      {/* Current Status */}
      <div className="border-card-border bg-card mb-6 rounded-lg border p-4 sm:p-6">
        <h2 className="text-text-primary mb-4 text-lg font-semibold">Current Status</h2>
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:gap-6">
          <div className="flex items-center gap-2">
            {getStatusIcon(service.status)}
            <span className="text-text-primary text-base font-medium capitalize sm:text-lg">
              {service.status}
            </span>
          </div>
          {service.response_time !== undefined && (
            <div className="flex flex-wrap items-baseline gap-1">
              <span className="text-text-secondary text-sm">Response Time: </span>
              <span className={`font-semibold ${getResponseTimeColor(service.response_time)}`}>
                {service.response_time}ms
              </span>
            </div>
          )}
        </div>
      </div>

      {/* Metrics Summary */}
      {metrics && !metricsLoading && (
        <div className="mb-6 grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
          <MetricsCard
            label="Uptime"
            value={`${metrics.uptime_percentage.toFixed(2)}%`}
            trend={
              metrics.uptime_percentage >= 99
                ? 'up'
                : metrics.uptime_percentage >= 95
                  ? 'neutral'
                  : 'down'
            }
          />
          <MetricsCard
            label="Avg Response Time"
            value={`${metrics.avg_response_time.toFixed(0)}ms`}
            subtext={`Min: ${metrics.min_response_time.toFixed(0)}ms / Max: ${metrics.max_response_time.toFixed(0)}ms`}
          />
          <MetricsCard
            label="Total Checks"
            value={metrics.total_checks}
            subtext={`${metrics.online_count} online / ${metrics.offline_count} offline`}
          />
          <MetricsCard
            label="Time Range"
            value={timeRangeOptions.find((o) => o.value === timeRange)?.label || ''}
            subtext={`${metrics.data_points.length} data points`}
          />
        </div>
      )}

      {/* Chart */}
      <div className="border-card-border bg-card mb-6 rounded-lg border p-4 sm:p-6">
        <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <h2 className="text-text-primary text-lg font-semibold">Uptime & Performance</h2>
          <div className="flex flex-wrap gap-2">
            {timeRangeOptions.map((option) => (
              <button
                key={option.value}
                onClick={() => setTimeRange(option.value)}
                className={`shrink-0 rounded px-2.5 py-1 text-xs sm:px-3 sm:text-sm ${
                  timeRange === option.value
                    ? 'bg-primary text-white'
                    : 'bg-background text-text-primary hover:bg-card-border'
                }`}
              >
                {option.label}
              </button>
            ))}
          </div>
        </div>

        {metricsLoading ? (
          <div className="flex h-80 items-center justify-center">
            <div className="text-text-secondary">Loading metrics...</div>
          </div>
        ) : metrics && metrics.data_points.length > 0 ? (
          <UptimeChart metrics={metrics} showResponseTime={true} />
        ) : (
          <div className="flex h-80 items-center justify-center">
            <div className="text-text-secondary">No metrics data available</div>
          </div>
        )}
      </div>
    </div>
  )
}
