'use client'

import { useEffect, useSyncExternalStore } from 'react'
import { useRouter } from 'next/navigation'
import { STATUS_REFRESH_SECONDS } from '@/lib/status-page'

const noSubscription = () => () => {}

/**
 * Shows when the page was last updated and reloads its data every minute
 * while the tab is visible.
 */
export default function StatusRefresh({ updatedAt }: { updatedAt?: string }) {
  const router = useRouter()

  useEffect(() => {
    const refresh = () => {
      if (document.visibilityState === 'visible') router.refresh()
    }
    const timer = setInterval(refresh, STATUS_REFRESH_SECONDS * 1000)
    document.addEventListener('visibilitychange', refresh)
    return () => {
      clearInterval(timer)
      document.removeEventListener('visibilitychange', refresh)
    }
  }, [router])

  // The server would format the time in its own zone, so only the browser does
  const inBrowser = useSyncExternalStore(
    noSubscription,
    () => true,
    () => false
  )

  return (
    <p>
      {updatedAt && inBrowser && (
        <>
          Updated <time dateTime={updatedAt}>{new Date(updatedAt).toLocaleTimeString()}</time>
          {' · '}
        </>
      )}
      Refreshes every minute
    </p>
  )
}
