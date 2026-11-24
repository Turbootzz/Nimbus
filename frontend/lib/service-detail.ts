import { api } from '@/lib/api'
import type { MetricsResponse } from '@/types'

const TTL_MS = 60_000

const cache = new Map<string, { at: number; request: Promise<MetricsResponse | null> }>()

/**
 * Last 24 hours of a service, in hourly points, for the hover detail of its
 * tile. Loaded on first use and kept for a minute, so moving over the tiles
 * doesn't refetch. Resolves to null when it can't be loaded; a failure is
 * not kept.
 */
export function loadServiceDetail(serviceId: string): Promise<MetricsResponse | null> {
  const cached = cache.get(serviceId)
  if (cached && Date.now() - cached.at < TTL_MS) return cached.request

  const request = api
    .getServiceMetrics(serviceId, '24h', 60)
    .then((response) => response.data ?? null)
    .catch(() => null)
  cache.set(serviceId, { at: Date.now(), request })
  request.then((detail) => {
    if (!detail && cache.get(serviceId)?.request === request) cache.delete(serviceId)
  })
  return request
}

export function clearServiceDetailCache() {
  cache.clear()
}
