'use client'

import type { WeatherPayload, WeatherWidgetConfig } from '@/types'
import type { WidgetRendererProps } from '@/components/widgets/registry'
import { describeWeather, weekday } from '@/lib/weather'

export default function WeatherWidget({
  config,
  cardSize,
  snapshot,
}: WidgetRendererProps<WeatherWidgetConfig>) {
  const weather = snapshot?.payload as WeatherPayload | null | undefined
  if (!weather) return null // WidgetCard shows loading and errors

  const now = describeWeather(weather.weather_code, weather.is_day)
  const temp = (value: number) => `${Math.round(value)}°`
  const days =
    cardSize === '2x2' ? weather.daily : cardSize === '2x1' ? weather.daily.slice(1, 4) : []

  return (
    <div className="flex h-full flex-col justify-between gap-2">
      <div className="flex items-center gap-3">
        <span className={cardSize === '1x1' ? 'text-3xl' : 'text-4xl'} aria-hidden="true">
          {now.icon}
        </span>
        <div className="min-w-0">
          <p className="text-text-primary text-2xl font-semibold tabular-nums">
            {temp(weather.temperature)}
          </p>
          <p className="text-text-secondary truncate text-xs">{config.location || now.label}</p>
        </div>
      </div>

      {cardSize !== '1x1' && (
        <p className="text-text-muted text-xs">
          {now.label} · feels like {temp(weather.feels_like)} · wind{' '}
          {Math.round(weather.wind_speed)} {weather.wind_unit}
        </p>
      )}

      {days.length > 0 && (
        <ul className="grid grid-cols-3 gap-1 text-center text-xs sm:grid-cols-4">
          {days.map((day) => {
            const condition = describeWeather(day.weather_code)
            return (
              <li key={day.date} className="bg-background/50 rounded px-1 py-1.5">
                <p className="text-text-muted">{weekday(day.date)}</p>
                <p title={condition.label} aria-label={condition.label}>
                  {condition.icon}
                </p>
                <p className="text-text-primary tabular-nums">
                  {temp(day.max)} <span className="text-text-muted">{temp(day.min)}</span>
                </p>
              </li>
            )
          })}
        </ul>
      )}
    </div>
  )
}
