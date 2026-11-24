'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import { useLocale, useTranslations } from 'next-intl'
import NimbusLogo from '@/components/NimbusLogo'
import ThemeToggle from '@/components/ThemeToggle'
import UserMenu from '@/components/UserMenu'
import SearchButton from '@/components/SearchButton'
import { useTopBarWeather } from '@/lib/dashboard-store'
import { describeWeather } from '@/lib/weather'
import { useWeatherLabel } from '@/hooks/useWeatherLabel'
import BoardSwitcher from '@/components/boards/BoardSwitcher'

// Time and date, rendered after mount so server and browser agree
function TopBarClock() {
  const locale = useLocale()
  const [now, setNow] = useState<Date | null>(null)
  useEffect(() => {
    const tick = () => setNow(new Date())
    tick()
    const timer = setInterval(tick, 1000)
    return () => clearInterval(timer)
  }, [])
  if (!now) return null

  return (
    <time dateTime={now.toISOString()} className="text-center leading-tight">
      <span className="text-text-primary block text-lg font-semibold tabular-nums">
        {now.toLocaleTimeString(locale, { hour: '2-digit', minute: '2-digit' })}
      </span>
      <span className="text-text-muted hidden text-xs sm:block">
        {now.toLocaleDateString(locale, { weekday: 'long', day: 'numeric', month: 'long' })}
      </span>
    </time>
  )
}

// Current weather, when the dashboard has a weather widget
function TopBarWeather() {
  const t = useTranslations('layout.topBar')
  const weatherLabel = useWeatherLabel()
  const info = useTopBarWeather()
  if (!info) return null

  const { icon } = describeWeather(info.weather.weather_code, info.weather.is_day)
  const condition = weatherLabel(info.weather.weather_code)
  const label = info.location ? t('weatherIn', { condition, location: info.location }) : condition
  return (
    <p
      className="text-text-primary flex shrink-0 items-center gap-1.5 text-sm font-medium"
      title={label}
    >
      <span aria-hidden="true">{icon}</span>
      <span className="tabular-nums">{Math.round(info.weather.temperature)}°</span>
      <span className="sr-only">{label}</span>
      {info.location && (
        <span className="text-text-muted hidden truncate text-xs font-normal lg:inline">
          {info.location}
        </span>
      )}
    </p>
  )
}

// Top bar of the canvas layout; the user menu holds the navigation. It keeps
// a fixed translucency, so it stays readable whatever the card opacity is.
export default function CanvasTopBar() {
  const t = useTranslations('layout.topBar')
  return (
    <header
      data-canvas-chrome
      className="bg-card/85 border-card-border sticky top-0 z-30 border-b backdrop-blur-md"
    >
      <h1 className="sr-only">{t('heading')}</h1>
      <div className="mx-auto grid h-16 max-w-screen-2xl grid-cols-[1fr_auto_1fr] items-center gap-4 px-4 sm:px-6 lg:px-8">
        <div className="flex max-w-full min-w-0 items-center gap-3 justify-self-start sm:gap-4">
          <Link href="/dashboard" className="flex shrink-0 items-center gap-2">
            <NimbusLogo size={28} />
            <span className="text-text-primary hidden text-lg font-semibold sm:inline">Nimbus</span>
          </Link>
          <BoardSwitcher variant="menu" />
          <TopBarWeather />
        </div>
        <TopBarClock />
        <div className="flex items-center gap-2 justify-self-end">
          <SearchButton />
          <ThemeToggle />
          <UserMenu withNavigation />
        </div>
      </div>
    </header>
  )
}
