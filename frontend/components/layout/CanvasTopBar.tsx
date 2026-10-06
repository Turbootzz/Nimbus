'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import NimbusLogo from '@/components/NimbusLogo'
import ThemeToggle from '@/components/ThemeToggle'
import UserMenu from '@/components/UserMenu'
import SearchButton from '@/components/SearchButton'

// Time and date, rendered after mount so server and browser agree
function TopBarClock() {
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
        {now.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' })}
      </span>
      <span className="text-text-muted hidden text-xs sm:block">
        {now.toLocaleDateString(undefined, { weekday: 'long', day: 'numeric', month: 'long' })}
      </span>
    </time>
  )
}

// Top bar of the canvas layout; the user menu holds the navigation. It keeps
// a fixed translucency, so it stays readable whatever the card opacity is.
export default function CanvasTopBar() {
  return (
    <header className="bg-card/85 border-card-border sticky top-0 z-30 border-b backdrop-blur-md">
      <h1 className="sr-only">Dashboard</h1>
      <div className="mx-auto grid h-16 max-w-screen-2xl grid-cols-[1fr_auto_1fr] items-center gap-4 px-4 sm:px-6 lg:px-8">
        <Link href="/dashboard" className="flex items-center gap-2 justify-self-start">
          <NimbusLogo size={28} />
          <span className="text-text-primary hidden text-lg font-semibold sm:inline">Nimbus</span>
        </Link>
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
