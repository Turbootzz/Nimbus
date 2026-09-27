'use client'

import { useEffect, useState } from 'react'
import type { ClockWidgetConfig } from '@/types'
import type { WidgetRendererProps } from '@/components/widgets/registry'

// Formats the time and date, or returns null for a timezone the browser
// doesn't know
function formatClock(now: Date, config: ClockWidgetConfig) {
  const timeZone = config.timezone || undefined
  try {
    const time = new Intl.DateTimeFormat(undefined, {
      hour: '2-digit',
      minute: '2-digit',
      second: config.show_seconds ? '2-digit' : undefined,
      hour12: config.hour12,
      timeZone,
    }).format(now)
    const date =
      config.date_format === 'none'
        ? null
        : new Intl.DateTimeFormat(undefined, {
            dateStyle: config.date_format === 'long' ? 'full' : 'medium',
            timeZone,
          }).format(now)
    return { time, date }
  } catch {
    return null
  }
}

export default function ClockWidget({ config, cardSize }: WidgetRendererProps<ClockWidgetConfig>) {
  const [now, setNow] = useState(() => new Date())

  useEffect(() => {
    const interval = setInterval(() => setNow(new Date()), 1000)
    return () => clearInterval(interval)
  }, [])

  const clock = formatClock(now, config)
  if (!clock) {
    return <p className="text-error text-sm">Unknown timezone: {config.timezone}</p>
  }

  const timeSize = cardSize === '2x2' ? 'text-5xl' : cardSize === '1x1' ? 'text-2xl' : 'text-4xl'
  // Last part of the zone name, e.g. "New York" for America/New_York
  const place = config.timezone.split('/').pop()?.replaceAll('_', ' ')

  return (
    <div className="flex h-full flex-col items-center justify-center text-center">
      <p className={`text-text-primary font-semibold tabular-nums ${timeSize}`}>{clock.time}</p>
      {clock.date && <p className="text-text-secondary mt-1 text-sm">{clock.date}</p>}
      {place && <p className="text-text-muted mt-1 text-xs">{place}</p>}
    </div>
  )
}
