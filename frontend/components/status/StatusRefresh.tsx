'use client'

import { useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { STATUS_REFRESH_SECONDS } from '@/lib/status-page'

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

  return (
    <p>
      {updatedAt && (
        <>
          Updated {/* The server formats in its own time zone; the browser's wins */}
          <time dateTime={updatedAt} suppressHydrationWarning>
            {new Date(updatedAt).toLocaleTimeString()}
          </time>
          {' · '}
        </>
      )}
      Refreshes every minute
    </p>
  )
}
