'use client'

import { useMemo, useState } from 'react'
import type { CalendarPayload, CalendarWidgetConfig } from '@/types'
import type { WidgetRendererProps } from '@/components/widgets/registry'
import { dayKey, eventDay, monthWeeks, sourceColors } from '@/lib/calendar'

const weekdays = ['M', 'T', 'W', 'T', 'F', 'S', 'S']

function dayLabel(day: string, today: string, tomorrow: string): string {
  if (day === today) return 'Today'
  if (day === tomorrow) return 'Tomorrow'
  const [y, m, d] = day.split('-').map(Number)
  return new Date(y, m - 1, d).toLocaleDateString(undefined, {
    weekday: 'short',
    day: 'numeric',
    month: 'short',
  })
}

export default function CalendarWidget({
  config,
  cardSize,
  snapshot,
}: WidgetRendererProps<CalendarWidgetConfig>) {
  const calendar = snapshot?.payload as CalendarPayload | null | undefined
  const [selected, setSelected] = useState<string | null>(null)

  const now = new Date()
  const today = dayKey(now)
  const tomorrow = dayKey(new Date(now.getFullYear(), now.getMonth(), now.getDate() + 1))
  const lastDay = dayKey(new Date(now.getFullYear(), now.getMonth(), now.getDate() + config.days))

  // Sources with something on each day, for the dots in the grid
  const byDay = useMemo(() => {
    const map = new Map<string, Set<number>>()
    for (const event of calendar?.events ?? []) {
      const day = eventDay(event)
      if (!map.has(day)) map.set(day, new Set())
      map.get(day)!.add(event.source)
    }
    return map
  }, [calendar])

  if (!calendar) return null // WidgetCard shows loading and errors

  // The month grid needs room; smaller tiles only get the agenda (and drop
  // a day picked while the tile was bigger)
  const withGrid = cardSize === '2x2'
  const picked = withGrid ? selected : null
  const agenda = calendar.events
    .filter((event) => {
      const day = eventDay(event)
      return picked ? day === picked : day >= today && day <= lastDay
    })
    // By the day in this browser, all-day first, then by time
    .sort(
      (a, b) =>
        eventDay(a).localeCompare(eventDay(b)) ||
        Number(b.all_day) - Number(a.all_day) ||
        a.start.localeCompare(b.start)
    )
  const failed = calendar.failed ?? []

  return (
    <div className="flex h-full flex-col gap-2">
      {failed.length > 0 && (
        <p className="text-warning text-xs" title={failed.join('\n')}>
          {failed.length === 1 ? '1 source' : `${failed.length} sources`} could not be loaded
        </p>
      )}

      {withGrid && (
        <table className="w-full table-fixed text-center text-xs">
          <thead>
            <tr className="text-text-muted">
              {weekdays.map((day, i) => (
                <th key={i} className="font-medium">
                  {day}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {monthWeeks(calendar.from).map((week, i) => (
              <tr key={i}>
                {week.map((day, j) => (
                  <td key={j} className="p-0">
                    {day && (
                      <button
                        type="button"
                        onClick={() => setSelected(selected === day ? null : day)}
                        aria-pressed={selected === day}
                        aria-label={`${dayLabel(day, today, tomorrow)}${byDay.has(day) ? ', has events' : ''}`}
                        className={`flex w-full flex-col items-center rounded py-0.5 tabular-nums transition-colors ${
                          selected === day
                            ? 'bg-primary text-white'
                            : day === today
                              ? 'text-primary font-semibold'
                              : 'text-text-primary hover:bg-card-hover'
                        }`}
                      >
                        {Number(day.slice(8))}
                        <span className="flex h-1 gap-px">
                          {[...(byDay.get(day) ?? [])].slice(0, 3).map((source) => (
                            <span
                              key={source}
                              className={`h-1 w-1 rounded-full ${sourceColors[source % sourceColors.length]}`}
                            />
                          ))}
                        </span>
                      </button>
                    )}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {agenda.length === 0 ? (
        <p className="text-text-muted text-sm">
          {picked ? 'Nothing on this day' : 'Nothing coming up'}
        </p>
      ) : (
        <ul className="space-y-1">
          {agenda.map((event, i) => {
            const day = eventDay(event)
            const time = event.all_day
              ? null
              : new Date(event.start).toLocaleTimeString(undefined, {
                  hour: '2-digit',
                  minute: '2-digit',
                })
            return (
              <li
                key={`${i}-${event.start}-${event.title}`}
                className="flex items-start gap-2 text-sm"
              >
                <span
                  className={`mt-1.5 h-2 w-2 shrink-0 rounded-full ${sourceColors[event.source % sourceColors.length]}`}
                  title={calendar.sources[event.source]?.name}
                  aria-hidden="true"
                />
                <span className="min-w-0 flex-1">
                  <span className="text-text-primary block truncate">{event.title}</span>
                  <span className="text-text-muted block text-xs">
                    {dayLabel(day, today, tomorrow)}
                    {time && ` · ${time}`}
                  </span>
                </span>
              </li>
            )
          })}
        </ul>
      )}
    </div>
  )
}
