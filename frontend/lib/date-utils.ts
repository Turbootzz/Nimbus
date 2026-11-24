export type TimeAgo = { unit: 'now' | 'minutes' | 'hours' | 'days'; count: number }

// How long ago a timestamp was, in its largest whole unit
export const timeAgo = (timestamp: string): TimeAgo => {
  const seconds = Math.floor((Date.now() - new Date(timestamp).getTime()) / 1000)

  if (seconds < 60) return { unit: 'now', count: 0 }
  if (seconds < 3600) return { unit: 'minutes', count: Math.floor(seconds / 60) }
  if (seconds < 86400) return { unit: 'hours', count: Math.floor(seconds / 3600) }
  return { unit: 'days', count: Math.floor(seconds / 86400) }
}

/**
 * Formats a timestamp as a relative time string (e.g., "2m ago", "3h ago");
 * English only, translated pages use hooks/useRelativeTime
 */
export const formatRelativeTime = (timestamp: string): string => {
  const { unit, count } = timeAgo(timestamp)
  if (unit === 'now') return 'just now'
  return `${count}${unit[0]} ago`
}
