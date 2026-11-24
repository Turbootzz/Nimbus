/* global importScripts, route, networkFirst, cacheFirst, navigate, precacheOffline,
   staleCaches, staticCacheName, API_CACHE */
// Nimbus service worker: event wiring only, the rules live in sw-routing.js
importScripts('/sw-routing.js')

// Registered as /sw.js?v=<build id>, so each deploy gets its own static cache
const version = new URL(self.location.href).searchParams.get('v') || 'dev'
const openStatic = () => caches.open(staticCacheName(version))
const openApi = () => caches.open(API_CACHE)
const fetchFromNetwork = (request) => fetch(request)

const purge = () => caches.delete(API_CACHE)

self.addEventListener('install', (event) => {
  // Reload so the offline page is not taken from the HTTP cache
  const fetchFresh = (url) => fetch(url, { cache: 'reload', credentials: 'same-origin' })
  event.waitUntil(precacheOffline({ fetch: fetchFresh, openCache: openStatic }))
  // Safe to take over at once: nothing here depends on the page's build, and
  // no page reloads when its worker changes
  self.skipWaiting()
})

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((names) => Promise.all(staleCaches(names, version).map((name) => caches.delete(name))))
      .then(() => self.clients.claim())
  )
})

self.addEventListener('fetch', (event) => {
  const { request } = event
  switch (route(request, self.location.origin)) {
    case 'api':
      event.respondWith(
        networkFirst(request, { fetch: fetchFromNetwork, openCache: openApi, purge })
      )
      break
    case 'static':
      event.respondWith(cacheFirst(request, { fetch: fetchFromNetwork, openCache: openStatic }))
      break
    case 'navigate':
      event.respondWith(navigate(request, { fetch: fetchFromNetwork, openCache: openStatic }))
      break
    case 'purge':
      event.waitUntil(purge())
      break
    // default: not ours, the browser handles it (SSE stream, auth, mutations)
  }
})
