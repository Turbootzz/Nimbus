'use client'

import { useLocale, useTranslations } from 'next-intl'
import type { SystemStatsPayload, SystemStatsWidgetConfig } from '@/types'
import type { WidgetRendererProps } from '@/components/widgets/registry'

const GB = 1024 ** 3

type T = ReturnType<typeof useTranslations<'widgets.systemStats'>>

// Like toFixed, with the locale's decimal separator and no grouping
function gigabytes(bytes: number, locale: string): string {
  const digits = bytes < 10 * GB ? 1 : 0
  return (bytes / GB).toLocaleString(locale, {
    minimumFractionDigits: digits,
    maximumFractionDigits: digits,
    useGrouping: false,
  })
}

// "3d 5h", "5h 2m" or "2m"; in the user's language when given t
export function formatUptime(seconds: number, t?: T): string {
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  if (days > 0) return t ? t('uptime.days', { days, hours }) : `${days}d ${hours}h`
  if (hours > 0) return t ? t('uptime.hours', { hours, minutes }) : `${hours}h ${minutes}m`
  return t ? t('uptime.minutes', { minutes }) : `${minutes}m`
}

// 0-100, and 0 for anything that isn't a number
function clamp(percent: number): number {
  return Number.isFinite(percent) ? Math.min(Math.max(percent, 0), 100) : 0
}

// Fill color by how full the bar is
function barColor(percent: number): string {
  if (percent >= 90) return 'bg-error'
  if (percent >= 75) return 'bg-warning'
  return 'bg-primary'
}

export default function SystemStatsWidget({
  cardSize,
  snapshot,
}: WidgetRendererProps<SystemStatsWidgetConfig>) {
  const t = useTranslations('widgets.systemStats')
  const locale = useLocale()
  const stats = snapshot?.payload as SystemStatsPayload | null | undefined
  if (!stats) return null // WidgetCard shows loading and errors

  // A 1x1 tile only has room for the percentages; a 2x1 tile is one row
  // high, so its bars go side by side with the numbers below them
  const small = cardSize === '1x1'
  const wide = cardSize === '2x1'
  const size = (used: number, total: number) =>
    t('gigabytes', { used: gigabytes(used, locale), total: gigabytes(total, locale) })
  const bars = [
    {
      label: 'CPU',
      percent: clamp(stats.cpu_percent),
      detail: `${Math.round(stats.cpu_percent)}%`,
    },
    {
      label: 'RAM',
      percent: clamp(stats.memory_percent),
      detail: size(stats.memory_used, stats.memory_total),
    },
    {
      label: t('disk'),
      percent: clamp(stats.disk_percent),
      detail: size(stats.disk_used, stats.disk_total),
    },
  ]

  return (
    <div className={`flex h-full flex-col justify-center ${small ? 'gap-1.5' : 'gap-2'}`}>
      <div
        className={wide ? 'grid grid-cols-3 gap-3' : `flex flex-col ${small ? 'gap-1.5' : 'gap-2'}`}
      >
        {bars.map((bar) => {
          const detail = (
            <span className="text-text-secondary truncate text-xs tabular-nums">
              {small ? `${Math.round(bar.percent)}%` : bar.detail}
            </span>
          )
          return (
            <div key={bar.label} className="min-w-0">
              <div className="flex items-baseline justify-between gap-2">
                <span className="kpi-label text-text-muted text-[10px] font-semibold tracking-wide uppercase">
                  {bar.label}
                </span>
                {!wide && detail}
              </div>
              <div
                className="bg-background mt-1 h-1.5 overflow-hidden rounded-full"
                role="meter"
                aria-label={bar.label}
                aria-valuemin={0}
                aria-valuemax={100}
                aria-valuenow={Math.round(bar.percent)}
              >
                <div
                  className={`h-full rounded-full ${barColor(bar.percent)}`}
                  style={{ width: `${bar.percent}%` }}
                />
              </div>
              {wide && <p className="mt-1 truncate">{detail}</p>}
            </div>
          )
        })}
      </div>
      {!small && (
        <p className="text-text-muted text-xs">
          {t('up', { uptime: formatUptime(stats.uptime_seconds, t) })}
        </p>
      )}
    </div>
  )
}
