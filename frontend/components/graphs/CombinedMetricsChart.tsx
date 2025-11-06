'use client'

import { useEffect, useState } from 'react'
import { MetricsResponse, TimeRangeOption } from '@/types'
import UptimeChart from './UptimeChart'
import { getApiUrl } from '@/lib/utils/api-url'

const API_URL = getApiUrl()

interface CombinedMetricsChartProps {
  serviceIds: string[]
  serviceNames: { [key: string]: string }
}

export default function CombinedMetricsChart({
  serviceIds,
  serviceNames,
}: CombinedMetricsChartProps) {
  const [allMetrics, setAllMetrics] = useState<{ [key: string]: MetricsResponse }>({})
  const [loading, setLoading] = useState(true)
  const [timeRange, setTimeRange] = useState<TimeRangeOption>('24h')

  const timeRangeOptions: { value: TimeRangeOption; label: string }[] = [
    { value: '1h', label: 'Last Hour' },
    { value: '6h', label: 'Last 6 Hours' },
    { value: '24h', label: 'Last 24 Hours' },
    { value: '7d', label: 'Last 7 Days' },
    { value: '30d', label: 'Last 30 Days' },
  ]

  useEffect(() => {
    const fetchAllMetrics = async () => {
      setLoading(true)
      const metricsPromises = serviceIds.map(async (serviceId) => {
        try {
          const url = `${API_URL}/metrics/${serviceId}?range=${timeRange}&interval=5`
          const response = await fetch(url, {
            credentials: 'include',
          })

          if (!response.ok) {
            console.error(`Failed to fetch metrics for service ${serviceId}`)
            return { serviceId, metrics: null }
          }

          const data: MetricsResponse = await response.json()
          return { serviceId, metrics: data }
        } catch (err) {
          console.error(`Error fetching metrics for service ${serviceId}:`, err)
          return { serviceId, metrics: null }
        }
      })

      const results = await Promise.all(metricsPromises)
      const metricsMap: { [key: string]: MetricsResponse } = {}

      results.forEach(({ serviceId, metrics }) => {
        if (metrics) {
          metricsMap[serviceId] = metrics
        }
      })

      setAllMetrics(metricsMap)
      setLoading(false)
    }

    if (serviceIds.length > 0) {
      fetchAllMetrics()
    } else {
      setLoading(false)
    }
  }, [serviceIds, timeRange])

  // Calculate combined stats
  const combinedStats = Object.values(allMetrics).reduce(
    (acc, metrics) => {
      acc.totalChecks += metrics.total_checks
      acc.onlineCount += metrics.online_count
      acc.offlineCount += metrics.offline_count
      acc.totalResponseTime += metrics.avg_response_time * metrics.total_checks
      return acc
    },
    { totalChecks: 0, onlineCount: 0, offlineCount: 0, totalResponseTime: 0 }
  )

  const avgUptime =
    combinedStats.totalChecks > 0
      ? (combinedStats.onlineCount / combinedStats.totalChecks) * 100
      : 0

  const avgResponseTime =
    combinedStats.totalChecks > 0 ? combinedStats.totalResponseTime / combinedStats.totalChecks : 0

  if (loading) {
    return (
      <div className="border-card-border bg-card flex h-80 items-center justify-center rounded-lg border">
        <div className="text-text-secondary">Loading combined metrics...</div>
      </div>
    )
  }

  if (serviceIds.length === 0) {
    return (
      <div className="border-card-border bg-card flex h-80 items-center justify-center rounded-lg border">
        <div className="text-text-secondary">No services to display</div>
      </div>
    )
  }

  // Combine all data points from all services by timestamp
  const combinedDataPoints = new Map<
    string,
    { timestamp: Date; uptimeSum: number; responseTimeSum: number; count: number }
  >()

  Object.values(allMetrics).forEach((metrics) => {
    metrics.data_points.forEach((point) => {
      const timestamp = new Date(point.timestamp).toISOString()
      const existing = combinedDataPoints.get(timestamp)

      if (existing) {
        existing.uptimeSum += point.uptime_percentage
        existing.responseTimeSum += point.avg_response_time
        existing.count += 1
      } else {
        combinedDataPoints.set(timestamp, {
          timestamp: new Date(point.timestamp),
          uptimeSum: point.uptime_percentage,
          responseTimeSum: point.avg_response_time,
          count: 1,
        })
      }
    })
  })

  // Convert to array and calculate averages
  const aggregatedPoints = Array.from(combinedDataPoints.values())
    .map((point) => ({
      timestamp: point.timestamp.toISOString(),
      check_count: point.count,
      online_count: 0, // Not used in this view
      uptime_percentage: point.uptimeSum / point.count,
      avg_response_time: point.responseTimeSum / point.count,
    }))
    .sort((a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime())

  // Calculate actual time range from data points
  const timeRangeStart =
    aggregatedPoints.length > 0 ? aggregatedPoints[0].timestamp : new Date().toISOString()
  const timeRangeEnd =
    aggregatedPoints.length > 0
      ? aggregatedPoints[aggregatedPoints.length - 1].timestamp
      : new Date().toISOString()

  // Guard against empty metrics when computing min/max
  const allMetricsValues = Object.values(allMetrics)
  const minResponseTime =
    allMetricsValues.length > 0 ? Math.min(...allMetricsValues.map((m) => m.min_response_time)) : 0
  const maxResponseTime =
    allMetricsValues.length > 0 ? Math.max(...allMetricsValues.map((m) => m.max_response_time)) : 0

  const combinedMetrics: MetricsResponse = {
    service_id: 'combined',
    time_range: {
      start: timeRangeStart,
      end: timeRangeEnd,
    },
    uptime_percentage: avgUptime,
    total_checks: combinedStats.totalChecks,
    online_count: combinedStats.onlineCount,
    offline_count: combinedStats.offlineCount,
    avg_response_time: avgResponseTime,
    min_response_time: minResponseTime,
    max_response_time: maxResponseTime,
    data_points: aggregatedPoints,
  }

  return (
    <div className="border-card-border bg-card rounded-lg border p-4 sm:p-6">
      <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 className="text-text-primary text-lg font-semibold">Combined Performance Metrics</h2>
          <p className="text-text-secondary mt-1 text-sm">
            Aggregate view of all {serviceIds.length} service{serviceIds.length !== 1 ? 's' : ''}
          </p>
        </div>
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

      {/* Summary Stats */}
      <div className="mb-4 grid grid-cols-2 gap-3 sm:grid-cols-4">
        <div className="border-card-border bg-background rounded border p-3">
          <p className="text-text-secondary text-xs">Avg Uptime</p>
          <p className="text-text-primary mt-1 text-lg font-bold">{avgUptime.toFixed(2)}%</p>
        </div>
        <div className="border-card-border bg-background rounded border p-3">
          <p className="text-text-secondary text-xs">Avg Response</p>
          <p className="text-text-primary mt-1 text-lg font-bold">{avgResponseTime.toFixed(0)}ms</p>
        </div>
        <div className="border-card-border bg-background rounded border p-3">
          <p className="text-text-secondary text-xs">Total Checks</p>
          <p className="text-text-primary mt-1 text-lg font-bold">{combinedStats.totalChecks}</p>
        </div>
        <div className="border-card-border bg-background rounded border p-3">
          <p className="text-text-secondary text-xs">Data Points</p>
          <p className="text-text-primary mt-1 text-lg font-bold">{aggregatedPoints.length}</p>
        </div>
      </div>

      {/* Chart */}
      {aggregatedPoints.length > 0 ? (
        <UptimeChart metrics={combinedMetrics} showResponseTime={true} />
      ) : (
        <div className="flex h-80 items-center justify-center">
          <div className="text-text-secondary">No data available for selected time range</div>
        </div>
      )}

      {/* Individual Service Stats */}
      <div className="mt-6">
        <h3 className="text-text-primary mb-3 text-sm font-semibold">
          Individual Service Statistics
        </h3>
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {Object.entries(allMetrics).map(([serviceId, metrics]) => (
            <div key={serviceId} className="border-card-border bg-background rounded border p-3">
              <p className="text-text-primary truncate text-sm font-medium">
                {serviceNames[serviceId] || serviceId}
              </p>
              <div className="mt-2 grid grid-cols-2 gap-2 text-xs">
                <div>
                  <span className="text-text-secondary">Uptime: </span>
                  <span className="text-text-primary font-medium">
                    {metrics.uptime_percentage.toFixed(1)}%
                  </span>
                </div>
                <div>
                  <span className="text-text-secondary">Avg: </span>
                  <span className="text-text-primary font-medium">
                    {metrics.avg_response_time.toFixed(0)}ms
                  </span>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
