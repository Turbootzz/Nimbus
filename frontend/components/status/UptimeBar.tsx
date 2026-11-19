import { type BarTone, barLabel, barTone, formatUptime } from '@/lib/status-page'

// Height tells the bars apart as well, for who cannot go by colour
const toneClasses: Record<BarTone, string> = {
  up: 'h-full bg-success',
  partial: 'h-3/4 bg-warning',
  down: 'h-1/2 bg-error',
  none: 'h-1/4 bg-card-border',
}

interface UptimeBarProps {
  label: string // "24 hours" or "7 days"
  bars: (number | null)[] // Uptime per bar, oldest first
  hoursPerBar: number
  uptime: number | null // Uptime over the whole period
}

/**
 * A row of uptime bars. The row as a whole is one image for screen readers,
 * with the percentage as its text; each bar has its own tooltip.
 */
export default function UptimeBar({ label, bars, hoursPerBar, uptime }: UptimeBarProps) {
  const summary = uptime === null ? `${label}: no data` : `${label}: ${formatUptime(uptime)} uptime`

  return (
    <div>
      <div className="mb-1 flex items-baseline justify-between text-xs">
        <span className="text-text-secondary">{label}</span>
        <span className="text-text-primary font-medium tabular-nums">{formatUptime(uptime)}</span>
      </div>
      <div role="img" aria-label={summary} className="flex h-7 items-end gap-px sm:gap-0.5">
        {bars.map((bar, index) => (
          <span
            key={index}
            title={barLabel(bar, index, bars.length, hoursPerBar)}
            data-tone={barTone(bar)}
            className={`min-w-0 flex-1 rounded-sm ${toneClasses[barTone(bar)]}`}
          />
        ))}
      </div>
    </div>
  )
}
