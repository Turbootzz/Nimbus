'use client'

import { useEffect, useState } from 'react'
import { SignalSlashIcon } from '@heroicons/react/24/outline'
import { API_CACHE, CACHED_AT_HEADER } from '@/lib/pwa'
import { getApiUrl } from '@/lib/utils/api-url'
import type { Service } from '@/types'

interface CachedServices {
  services: Service[]
  // When the list was last fetched from the server
  syncedAt: Date | null
}

// Reads the services list the service worker stored on the last sync
async function readCachedServices(): Promise<CachedServices | null> {
  if (!('caches' in window) || !(await caches.has(API_CACHE))) return null
  const cache = await caches.open(API_CACHE)
  const response = await cache.match(`${getApiUrl()}/services`)
  if (!response) return null
  const services = await response.json()
  if (!Array.isArray(services)) return null
  const cachedAt = Date.parse(response.headers.get(CACHED_AT_HEADER) ?? '')
  return { services, syncedAt: Number.isNaN(cachedAt) ? null : new Date(cachedAt) }
}

// Leaves the offline page; a page the worker answered for reloads in place
function backToApp() {
  if (window.location.pathname === '/offline') window.location.replace('/dashboard')
  else window.location.reload()
}

/**
 * Read-only list of the last synced services, shown while the server cannot be
 * reached. Returns to the app as soon as the browser is back online.
 */
export default function OfflineServices() {
  // undefined while the cache is being read
  const [cached, setCached] = useState<CachedServices | null | undefined>(undefined)

  useEffect(() => {
    readCachedServices()
      .catch(() => null)
      .then(setCached)
    window.addEventListener('online', backToApp)
    return () => window.removeEventListener('online', backToApp)
  }, [])

  return (
    <div className="bg-card border-card-border rounded-lg border p-6 shadow-sm">
      <div className="flex items-center gap-3">
        <SignalSlashIcon className="text-warning h-8 w-8 shrink-0" aria-hidden="true" />
        <div>
          <h1 className="text-text-primary text-xl font-semibold">You are offline</h1>
          <p className="text-text-secondary text-sm" role="status">
            {cached?.syncedAt
              ? `Showing your services as of ${cached.syncedAt.toLocaleString()}. Statuses are not updated.`
              : 'The server cannot be reached right now.'}
          </p>
        </div>
      </div>

      {cached === null && (
        <p className="text-text-muted mt-6 text-sm">No services were saved on this device yet.</p>
      )}

      {cached && cached.services.length > 0 && (
        <ul className="divide-card-border mt-6 divide-y" aria-label="Last synced services">
          {cached.services.map((service) => (
            <li key={service.id} className="flex items-center justify-between gap-4 py-2">
              <span className="text-text-primary min-w-0 truncate text-sm font-medium">
                {service.name}
              </span>
              <a
                href={service.url}
                target="_blank"
                rel="noopener noreferrer"
                className="text-primary min-w-0 truncate text-sm hover:underline focus-visible:underline"
              >
                {service.url}
              </a>
            </li>
          ))}
        </ul>
      )}

      <button
        type="button"
        onClick={backToApp}
        className="bg-primary hover:bg-primary-hover focus-visible:ring-primary mt-6 rounded-md px-4 py-2 text-sm font-medium text-white focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none"
      >
        Try again
      </button>
    </div>
  )
}
