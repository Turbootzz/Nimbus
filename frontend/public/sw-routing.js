// Routing and caching rules of the service worker. A classic script: sw.js
// loads it with importScripts, the tests with require, so the tested code is
// the code that ships. Keep lib/pwa.ts in sync with the constants.

const API_CACHE = 'nimbus-api-v1'
const STATIC_CACHE_PREFIX = 'nimbus-static-'
const OFFLINE_URL = '/offline'
const CACHED_AT_HEADER = 'X-Nimbus-Cached-At'
const CLEAR_API_CACHE = 'CLEAR_API_CACHE'

// The only API responses kept for offline use: what the dashboard reads.
// Everything else (tokens, admin, webhooks, integrations, ...) is never stored.
const CACHEABLE_API_PATHS = [
  '/api/v1/services',
  '/api/v1/groups',
  '/api/v1/widgets',
  '/api/v1/dashboard/data',
  '/api/v1/users/me/preferences',
  '/api/v1/boards',
]

function staticCacheName(version) {
  return STATIC_CACHE_PREFIX + version
}

// A request after which the cached data may belong to someone else: login,
// logout, register, password and account changes, first-run setup, OAuth
function changesSession(method, path) {
  if (path.startsWith('/api/v1/auth/oauth/')) return path !== '/api/v1/auth/oauth/providers'
  if (method === 'GET') return false
  return path.startsWith('/api/v1/auth/') || path.startsWith('/api/v1/setup')
}

/**
 * Decides what the worker does with a request:
 * - 'api': network-first, cached copy only when the network fails
 * - 'static': cache-first (hashed build assets and icons)
 * - 'navigate': network-first, offline page when the network fails
 * - 'purge': drop the API cache, then let the browser handle the request
 * - null: not ours; the worker must not call respondWith (SSE stream, auth,
 *   mutations, other origins, anything with a query string)
 */
function route(request, origin) {
  const url = new URL(request.url)
  if (url.origin !== origin) return null
  // A cached full file is no answer to a request for part of it
  if (request.headers.has('range')) return null
  const path = url.pathname.replace(/\/+$/, '') || '/'

  if (path === '/api' || path.startsWith('/api/')) {
    if (changesSession(request.method, path)) return 'purge'
    const cacheable =
      request.method === 'GET' &&
      request.mode !== 'navigate' &&
      url.search === '' &&
      CACHEABLE_API_PATHS.includes(path)
    return cacheable ? 'api' : null
  }

  if (request.method !== 'GET' || path === '/mcp') return null
  if (request.mode === 'navigate') return 'navigate'
  if (path.startsWith('/_next/static/') || path.startsWith('/favicon/')) return 'static'
  return null
}

// Only complete, successful, same-origin responses that set no cookie and that
// the server allows to be stored. Workers cannot see Set-Cookie in browsers, so
// route() keeping auth out is the real guard.
function isCacheable(response) {
  return (
    response.status === 200 &&
    response.type !== 'opaque' &&
    !response.headers.has('Set-Cookie') &&
    !/no-store/i.test(response.headers.get('Cache-Control') || '')
  )
}

// A copy of the response that remembers when it was fetched
function withCachedAt(response, now) {
  const headers = new Headers(response.headers)
  headers.set(CACHED_AT_HEADER, now.toISOString())
  return new Response(response.body, {
    status: response.status,
    statusText: response.statusText,
    headers,
  })
}

/**
 * Network-first. Any answer from the server, errors included, is returned as
 * is; the cache is read only when fetch itself fails (offline).
 * deps: { fetch, openCache, generation, purge }. generation() changes when the
 * cache is purged, so a response that was in flight is not stored afterwards.
 */
async function networkFirst(request, deps) {
  const generation = deps.generation()
  let response
  try {
    response = await deps.fetch(request)
  } catch (error) {
    const cached = await (await deps.openCache()).match(request)
    if (cached) return cached
    throw error
  }
  try {
    if (response.status === 401) {
      // Session ended: the cached data has no owner anymore
      await deps.purge()
    } else if (isCacheable(response)) {
      const copy = withCachedAt(response.clone(), new Date())
      const cache = await deps.openCache()
      if (generation === deps.generation()) await cache.put(request, copy)
    }
  } catch (error) {
    // A full or broken cache must not cost the page the server's answer
    console.warn('Could not update the offline cache:', error)
  }
  return response
}

// Cache-first, for files whose URL changes when their content does
async function cacheFirst(request, deps) {
  const cache = await deps.openCache()
  const cached = await cache.match(request)
  if (cached) return cached
  const response = await deps.fetch(request)
  // A full cache must not cost the page the file
  if (isCacheable(response)) await cache.put(request, response.clone()).catch(() => {})
  return response
}

// Page loads always go to the network; the offline page is the fallback
async function navigate(request, deps) {
  try {
    return await deps.fetch(request)
  } catch (error) {
    const offline = await (await deps.openCache()).match(OFFLINE_URL)
    if (offline) return offline
    throw error
  }
}

// The build assets a page needs to render without network
function extractAssetUrls(html) {
  return [...new Set(html.match(/\/_next\/static\/[^"'\\\s<>)]+/g) || [])]
}

// Stores the offline page and its assets. Fails (and with it the install)
// when the page itself cannot be fetched; a missing asset is not fatal.
async function precacheOffline(deps) {
  const response = await deps.fetch(OFFLINE_URL)
  // A redirected response cannot answer a page load
  if (!isCacheable(response) || response.redirected) {
    throw new Error('offline page unavailable: ' + response.status)
  }
  const html = await response.clone().text()
  const cache = await deps.openCache()
  await cache.put(OFFLINE_URL, response)
  await Promise.allSettled(
    extractAssetUrls(html).map(async (url) => {
      const asset = await deps.fetch(url)
      if (isCacheable(asset)) await cache.put(url, asset)
    })
  )
}

// Caches of older builds and cache versions
function staleCaches(names, version) {
  const keep = [API_CACHE, staticCacheName(version)]
  return names.filter((name) => name.startsWith('nimbus-') && !keep.includes(name))
}

if (typeof module === 'object') {
  module.exports = {
    API_CACHE,
    OFFLINE_URL,
    CACHED_AT_HEADER,
    CLEAR_API_CACHE,
    CACHEABLE_API_PATHS,
    staticCacheName,
    route,
    isCacheable,
    networkFirst,
    cacheFirst,
    navigate,
    extractAssetUrls,
    precacheOffline,
    staleCaches,
  }
}
