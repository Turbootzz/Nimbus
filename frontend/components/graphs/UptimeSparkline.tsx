'use client'

import { MetricDataPoint } from '@/types'
import { LineChart, Line, ResponsiveContainer } from 'recharts'

interface UptimeSparklineProps {
  dataPoints: MetricDataPoint[]
  className?: string
}

export default function UptimeSparkline({ dataPoints, className = '' }: UptimeSparklineProps) {
  const chartData = dataPoints.map((point) => ({
    uptime: point.uptime_percentage,
  }))

  // Determine line color based on average uptime
  const avgUptime =
    dataPoints.length > 0
      ? dataPoints.reduce((sum, p) => sum + p.uptime_percentage, 0) / dataPoints.length
      : 0

  const lineColor = avgUptime >= 95 ? '#10b981' : avgUptime >= 80 ? '#f59e0b' : '#ef4444'

  return (
    <div className={`h-8 w-24 ${className}`}>
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={chartData}>
          <Line type="monotone" dataKey="uptime" stroke={lineColor} strokeWidth={2} dot={false} />
        </LineChart>
      </ResponsiveContainer>
    </div>
  )
}
