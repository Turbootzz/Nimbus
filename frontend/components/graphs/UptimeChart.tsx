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

interface UptimeChartProps {
  metrics: MetricsResponse
  showResponseTime?: boolean
}

export default function UptimeChart({ metrics, showResponseTime = true }: UptimeChartProps) {
  // Transform data for Recharts
  const chartData = metrics.data_points.map((point) => ({
    timestamp: new Date(point.timestamp).getTime(),
    uptime: Number(point.uptime_percentage.toFixed(2)),
    responseTime: Number(point.avg_response_time.toFixed(2)),
    timeLabel: format(new Date(point.timestamp), 'MMM d, HH:mm'),
  }))

  return (
    <div className="h-80 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={chartData} margin={{ top: 5, right: 30, left: 20, bottom: 5 }}>
          <CartesianGrid strokeDasharray="3 3" className="stroke-gray-200 dark:stroke-gray-700" />
          <XAxis
            dataKey="timeLabel"
            className="text-xs text-gray-600 dark:text-gray-400"
            tick={{ fontSize: 11 }}
          />
          <YAxis
            yAxisId="left"
            className="text-xs text-gray-600 dark:text-gray-400"
            label={{ value: 'Uptime %', angle: -90, position: 'insideLeft', className: 'text-xs' }}
            domain={[0, 100]}
          />
          {showResponseTime && (
            <YAxis
              yAxisId="right"
              orientation="right"
              className="text-xs text-gray-600 dark:text-gray-400"
              label={{
                value: 'Response Time (ms)',
                angle: 90,
                position: 'insideRight',
                className: 'text-xs',
              }}
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
          <Legend wrapperStyle={{ fontSize: '0.875rem' }} />
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
  )
}
