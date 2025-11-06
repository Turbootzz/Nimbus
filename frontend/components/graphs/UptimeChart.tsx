'use client'

import { MetricsResponse } from '@/types'
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts'
import { format } from 'date-fns'
import { useEffect, useState } from 'react'

interface UptimeChartProps {
  metrics: MetricsResponse
  showResponseTime?: boolean
}

export default function UptimeChart({ metrics, showResponseTime = true }: UptimeChartProps) {
  const [isMobile, setIsMobile] = useState(false)

  useEffect(() => {
    const checkMobile = () => {
      setIsMobile(window.innerWidth < 768) // md breakpoint
    }

    checkMobile()
    window.addEventListener('resize', checkMobile)
    return () => window.removeEventListener('resize', checkMobile)
  }, [])

  // Transform data for Recharts
  const chartData = metrics.data_points.map((point) => ({
    timestamp: new Date(point.timestamp).getTime(),
    uptime: Number(point.uptime_percentage.toFixed(2)),
    responseTime: Number(point.avg_response_time.toFixed(2)),
    timeLabel: format(new Date(point.timestamp), 'MMM d, HH:mm'),
  }))

  return (
    <div className="w-full">
      {/* Mobile: Labels above chart */}
      {isMobile && (
        <div className="mb-2 flex items-center justify-between text-xs">
          <div className="flex items-center gap-1">
            <div className="h-2 w-2 rounded-full bg-green-500"></div>
            <span className="text-text-secondary">Uptime %</span>
          </div>
          {showResponseTime && (
            <div className="flex items-center gap-1">
              <div className="h-2 w-2 rounded-full bg-blue-500"></div>
              <span className="text-text-secondary">Response Time (ms)</span>
            </div>
          )}
        </div>
      )}

      <div className="h-80 w-full">
        <ResponsiveContainer width="100%" height="100%">
          <LineChart
            data={chartData}
            margin={
              isMobile
                ? { top: 5, right: 5, left: 5, bottom: 5 }
                : { top: 5, right: 30, left: 20, bottom: 5 }
            }
          >
            <CartesianGrid strokeDasharray="3 3" className="stroke-gray-200 dark:stroke-gray-700" />
            <XAxis
              dataKey="timeLabel"
              className="text-xs text-gray-600 dark:text-gray-400"
              tick={{ fontSize: isMobile ? 9 : 11 }}
            />
            <YAxis
              yAxisId="left"
              className="text-xs text-gray-600 dark:text-gray-400"
              label={
                !isMobile
                  ? { value: 'Uptime %', angle: -90, position: 'insideLeft', className: 'text-xs' }
                  : undefined
              }
              domain={[0, 100]}
              width={isMobile ? 30 : 60}
              tick={{ fontSize: isMobile ? 9 : 12 }}
            />
            {showResponseTime && (
              <YAxis
                yAxisId="right"
                orientation="right"
                className="text-xs text-gray-600 dark:text-gray-400"
                label={
                  !isMobile
                    ? {
                        value: 'Response Time (ms)',
                        angle: 90,
                        position: 'insideRight',
                        className: 'text-xs',
                      }
                    : undefined
                }
                width={isMobile ? 30 : 60}
                tick={{ fontSize: isMobile ? 9 : 12 }}
              />
            )}
            <Tooltip
              contentStyle={{
                backgroundColor: 'rgb(var(--background))',
                border: '1px solid rgb(var(--border))',
                borderRadius: '0.5rem',
              }}
              labelClassName="text-sm font-semibold text-gray-900 dark:text-gray-100"
              itemStyle={{ fontSize: '0.875rem' }}
            />
            {!isMobile && <Legend wrapperStyle={{ fontSize: '0.875rem' }} />}
            <Line
              yAxisId="left"
              type="monotone"
              dataKey="uptime"
              stroke="#10b981"
              strokeWidth={2}
              name="Uptime %"
              dot={false}
              activeDot={{ r: 4 }}
            />
            {showResponseTime && (
              <Line
                yAxisId="right"
                type="monotone"
                dataKey="responseTime"
                stroke="#3b82f6"
                strokeWidth={2}
                name="Avg Response Time (ms)"
                dot={false}
                activeDot={{ r: 4 }}
              />
            )}
          </LineChart>
        </ResponsiveContainer>
      </div>
    </div>
  )
}
