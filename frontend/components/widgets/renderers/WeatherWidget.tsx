'use client'

import { useLocale, useTranslations } from 'next-intl'
import type { WeatherPayload, WeatherWidgetConfig } from '@/types'
import type { WidgetRendererProps } from '@/components/widgets/registry'
import { describeWeather, weekday } from '@/lib/weather'
import { useWeatherLabel } from '@/hooks/useWeatherLabel'

export default function WeatherWidget({
  config,
  cardSize,
  snapshot,
}: WidgetRendererProps<WeatherWidgetConfig>) {
  const t = useTranslations('widgets.weather')
  const locale = useLocale()
  const weatherLabel = useWeatherLabel()
  const weather = snapshot?.payload as WeatherPayload | null | undefined
  if (!weather) return null // WidgetCard shows loading and errors

  const now = describeWeather(weather.weather_code, weather.is_day)
  const nowLabel = weatherLabel(weather.weather_code)
  const temp = (value: number) => `${Math.round(value)}°`
  // A 2x1 tile is one row high, so the forecast goes next to the current
  // weather instead of below it
  const wide = cardSize === '2x1'
  const days = cardSize === '2x2' ? weather.daily : wide ? weather.daily.slice(1, 4) : []
  const detail = t('detail', { condition: nowLabel, temperature: temp(weather.feels_like) })
  const wind = t('wind', { speed: Math.round(weather.wind_speed), unit: weather.wind_unit })

  return (
    <div className={`flex h-full gap-3 ${wide ? 'items-center' : 'flex-col justify-between'}`}>
      <div className={`flex min-w-0 flex-col gap-2 ${wide ? 'flex-1' : ''}`}>
        <div className="flex items-center gap-3">
          <span className={cardSize === '1x1' ? 'text-3xl' : 'text-4xl'} aria-hidden="true">
            {now.icon}
          </span>
          <div className="min-w-0">
            <p className="text-text-primary text-2xl font-semibold tabular-nums">
              {temp(weather.temperature)}
            </p>
            <p className="text-text-secondary truncate text-xs">{config.location || nowLabel}</p>
          </div>
        </div>

        {wide && (
          <p className="text-text-muted truncate text-xs" title={`${detail} · ${wind}`}>
            {detail}
          </p>
        )}
        {cardSize === '2x2' && (
          <p className="text-text-muted text-xs">
            {detail} · {wind}
          </p>
        )}
      </div>

      {days.length > 0 && (
        <ul
          className={`grid gap-1 text-center text-xs ${wide ? 'flex-1' : ''} ${days.length > 3 ? 'grid-cols-4' : 'grid-cols-3'}`}
        >
          {days.map((day) => {
            const icon = describeWeather(day.weather_code).icon
            const label = weatherLabel(day.weather_code)
            return (
              <li key={day.date} className="bg-background/50 rounded px-1 py-1.5">
                <p className="text-text-muted">{weekday(day.date, locale)}</p>
                <p title={label} aria-label={label}>
                  {icon}
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
