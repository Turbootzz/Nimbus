import type { PublicStatusPage, StatusPageOverall } from '@/types'
import { getApiUrl } from '@/lib/utils/api-url'

export const STATUS_REFRESH_SECONDS = 60

export const overallLabels: Record<StatusPageOverall, string> = {
  operational: 'All systems operational',
  degraded: 'Some systems are down',
  outage: 'All systems are down',
  unknown: 'No status yet',
}

export const statusLabels: Record<string, string> = {
  online: 'Online',
  offline: 'Offline',
  unknown: 'Unknown',
}

export type BarTone = 'up' | 'partial' | 'down' | 'none'

// Uptime below this colours a bar as down rather than partly down
const DOWN_BELOW = 90

/** How a bar is coloured: null is a period without checks */
export function barTone(uptime: number | null): BarTone {
  if (uptime === null) return 'none'
  if (uptime >= 100) return 'up'
  return uptime >= DOWN_BELOW ? 'partial' : 'down'
}

/** "99.95%", without trailing zeros; "No data" for null */
export function formatUptime(uptime: number | null): string {
  return uptime === null ? 'No data' : `${Number(uptime.toFixed(2))}%`
}

/** When bar `index` of `count` was, with the newest bar last */
export function barPeriod(index: number, count: number, hoursPerBar: number): string {
  const hoursAgo = (count - 1 - index) * hoursPerBar
  if (hoursAgo === 0) return hoursPerBar === 1 ? 'This hour' : `Last ${hoursPerBar} hours`
  if (hoursAgo % 24 === 0) return hoursAgo === 24 ? '1 day ago' : `${hoursAgo / 24} days ago`
  return hoursAgo === 1 ? '1 hour ago' : `${hoursAgo} hours ago`
}

/** Text alternative of one bar, e.g. "3 hours ago: 98.5% uptime" */
export function barLabel(
  uptime: number | null,
  index: number,
  count: number,
  hoursPerBar: number
): string {
  const period = barPeriod(index, count, hoursPerBar)
  return uptime === null ? `${period}: no data` : `${period}: ${formatUptime(uptime)} uptime`
}

/**
 * API base URL for requests the Next.js server makes itself. In the unified
 * image the browser uses a relative path, which the server cannot fetch, so
 * that falls back to the backend next to it.
 */
export function serverApiUrl(): string {
  const url = process.env.INTERNAL_API_URL || getApiUrl()
  return url.startsWith('/') ? `http://127.0.0.1:8080${url}` : url
}

export type PublicStatusResult =
  { state: 'ok'; page: PublicStatusPage } | { state: 'disabled' } | { state: 'unavailable' }

/**
 * Loads the public status page on the server. Not cached here: the backend
 * caches it for 30 seconds, and a copy kept by Next.js would outlive an admin
 * turning the page off. visitorIP lets the backend rate limit per visitor.
 */
export async function fetchPublicStatus(visitorIP?: string | null): Promise<PublicStatusResult> {
  try {
    const response = await fetch(`${serverApiUrl()}/public/status`, {
      cache: 'no-store',
      headers: visitorIP ? { 'X-Real-IP': visitorIP } : undefined,
    })
    if (response.status === 404) return { state: 'disabled' }
    if (!response.ok) return { state: 'unavailable' }
    return { state: 'ok', page: (await response.json()) as PublicStatusPage }
  } catch {
    return { state: 'unavailable' }
  }
}
