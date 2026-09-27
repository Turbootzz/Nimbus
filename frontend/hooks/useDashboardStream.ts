'use client'

import { useEffect, useRef, useState } from 'react'
import { api } from '@/lib/api'
import { getApiUrl } from '@/lib/utils/api-url'
import type { ServiceStatusEvent, Snapshot } from '@/types'

const MIN_RETRY_MS = 1000
const MAX_RETRY_MS = 30000
// While the stream is down, poll like the dashboard did before SSE
const FALLBACK_POLL_MS = 30000
// EventSource hides status codes, so after this many failures in a row we
// ask /auth/me; the API client sends a logged out user to the login page
const AUTH_CHECK_AFTER_ERRORS = 3

export type SnapshotMap = Record<string, Snapshot>

export function snapshotKey(kind: Snapshot['source_kind'], id: string): string {
  return `${kind}:${id}`
}

function toMap(snapshots: Snapshot[]): SnapshotMap {
  return Object.fromEntries(snapshots.map((s) => [snapshotKey(s.source_kind, s.source_id), s]))
}

interface DashboardStreamOptions {
  // A health check finished for one of the user's services
  onServiceStatus: (event: ServiceStatusEvent) => void
  // Refetch services; called while the stream is down and after it recovers
  onResync: () => void
}

/**
 * Live dashboard data: loads GET /dashboard/data, then follows
 * /dashboard/stream. Reconnects with backoff (1s to 30s) and polls every 30s
 * while disconnected.
 */
export function useDashboardStream({ onServiceStatus, onResync }: DashboardStreamOptions) {
  const [snapshots, setSnapshots] = useState<SnapshotMap>({})
  const [connected, setConnected] = useState(false)

  // Keep the latest callbacks without reconnecting when they change
  const handlers = useRef({ onServiceStatus, onResync })
  useEffect(() => {
    handlers.current = { onServiceStatus, onResync }
  })

  useEffect(() => {
    let source: EventSource | null = null
    let retryTimer: ReturnType<typeof setTimeout> | undefined
    let pollTimer: ReturnType<typeof setInterval> | undefined
    let retryDelay = MIN_RETRY_MS
    let failures = 0
    let closed = false

    const loadData = async () => {
      try {
        const response = await api.getDashboardData()
        if (response.data && !closed) setSnapshots(toMap(response.data))
      } catch (error) {
        console.error('Failed to load dashboard data:', error)
      }
    }

    const resync = () => {
      handlers.current.onResync()
      loadData()
    }

    const stopPolling = () => {
      clearInterval(pollTimer)
      pollTimer = undefined
    }

    const connect = () => {
      if (closed) return
      source = new EventSource(`${getApiUrl()}/dashboard/stream`, { withCredentials: true })

      source.addEventListener('hello', () => {
        setConnected(true)
        // Catch up on what happened while we were away
        if (failures > 0) resync()
        failures = 0
        retryDelay = MIN_RETRY_MS
        stopPolling()
      })

      source.addEventListener('snapshot', (event) => {
        const snap: Snapshot = JSON.parse((event as MessageEvent).data)
        setSnapshots((prev) => ({ ...prev, [snapshotKey(snap.source_kind, snap.source_id)]: snap }))
      })

      source.addEventListener('service_status', (event) => {
        handlers.current.onServiceStatus(JSON.parse((event as MessageEvent).data))
      })

      source.onerror = () => {
        // Take over from EventSource's own retry to control the backoff
        source?.close()
        setConnected(false)
        failures++
        if (failures === AUTH_CHECK_AFTER_ERRORS) api.getCurrentUser()
        if (!pollTimer) pollTimer = setInterval(resync, FALLBACK_POLL_MS)
        retryTimer = setTimeout(connect, retryDelay)
        retryDelay = Math.min(retryDelay * 2, MAX_RETRY_MS)
      }
    }

    loadData()
    connect()

    return () => {
      closed = true
      source?.close()
      clearTimeout(retryTimer)
      stopPolling()
    }
  }, [])

  return { snapshots, connected }
}
