import { createRequire } from 'module'
import { describe, expect, it, vi } from 'vitest'
import { NextRequest } from 'next/server'
import { API_CACHE, CACHED_AT_HEADER, clearApiCache } from '@/lib/pwa'
import proxy from '@/proxy'

// The worker's rules are a classic script (see public/sw-routing.js); require
// loads the very file the browser runs
const sw = createRequire(import.meta.url)('../../public/sw-routing.js')

const ORIGIN = 'https://nimbus.test'

type RouteInit = { method?: string; mode?: string; headers?: Record<string, string> }
const route = (path: string, init: RouteInit = {}) =>
  sw.route(
    {
      url: path.startsWith('http') ? path : ORIGIN + path,
      method: init.method ?? 'GET',
      mode: init.mode ?? 'cors',
      headers: new Headers(init.headers),
    },
    ORIGIN
  )

// A Cache Storage cache, as far as the worker uses it
function fakeCache(entries: Record<string, Response> = {}) {
  const store = new Map(Object.entries(entries))
  const key = (request: Request | string) => (typeof request === 'string' ? request : request.url)
  return {
    store,
    match: vi.fn(async (request: Request | string) => store.get(key(request))),
    put: vi.fn(async (request: Request | string, response: Response) => {
      store.set(key(request), response)
    }),
  }
}

// Cache Storage: a deleted cache keeps working for whoever opened it before,
// but is gone for everyone who opens the name afterwards
function fakeStorage() {
  const named = new Map<string, ReturnType<typeof fakeCache>>()
  return {
    open: async (name: string) => {
      if (!named.has(name)) named.set(name, fakeCache())
      return named.get(name)!
    },
    delete: async (name: string) => named.delete(name),
  }
}

const servicesRequest = () => new Request(`${ORIGIN}/api/v1/services`)
const json = (body: unknown, init?: ResponseInit) => new Response(JSON.stringify(body), init)

function apiDeps(cache: ReturnType<typeof fakeCache>, fetch: () => Promise<Response>) {
  return { fetch: vi.fn(fetch), openCache: async () => cache, purge: vi.fn() }
}

describe('service worker routing', () => {
  it('shares its constants with the page', () => {
    expect(sw.API_CACHE).toBe(API_CACHE)
    expect(sw.CACHED_AT_HEADER).toBe(CACHED_AT_HEADER)
  })

  it('handles the dashboard reads network-first', () => {
    for (const path of sw.CACHEABLE_API_PATHS) expect(route(path)).toBe('api')
    expect(route('/api/v1/services/')).toBe('api')
  })

  it('leaves the live stream alone, or it would hang', () => {
    expect(route('/api/v1/dashboard/stream')).toBeNull()
  })

  it('never handles mutations', () => {
    for (const method of ['POST', 'PUT', 'PATCH', 'DELETE']) {
      expect(route('/api/v1/services', { method })).toBeNull()
      expect(route('/dashboard', { method, mode: 'navigate' })).toBeNull()
    }
  })

  it('never handles auth, private or unlisted API reads', () => {
    for (const path of [
      '/api/v1/auth/me',
      '/api/v1/auth/oauth/providers',
      '/api/v1/api-tokens',
      '/api/v1/admin/users',
      '/api/v1/webhooks',
      '/api/v1/integrations',
      '/api/v1/setup/status',
      '/api/v1/services/5b0c3a1e-0000-4000-8000-000000000000',
      '/api/v1/uploads/avatars/a.png',
      '/api/v1/health',
      '/mcp',
    ]) {
      expect(route(path), path).toBeNull()
    }
  })

  it('never handles a URL that carries a query string', () => {
    expect(route('/api/v1/services?token=secret')).toBeNull()
    expect(route('/api/v1/services/favicon?url=https://example.com')).toBeNull()
  })

  it('never handles a range request', () => {
    const headers = { Range: 'bytes=0-99' }
    expect(route('/_next/static/media/intro-1a2b.mp4', { headers })).toBeNull()
    expect(route('/api/v1/services', { headers })).toBeNull()
  })

  it('purges the API cache when the session changes', () => {
    expect(route('/api/v1/auth/login', { method: 'POST' })).toBe('purge')
    expect(route('/api/v1/auth/logout', { method: 'POST' })).toBe('purge')
    expect(route('/api/v1/auth/register', { method: 'POST' })).toBe('purge')
    expect(route('/api/v1/auth/me', { method: 'DELETE' })).toBe('purge')
    expect(route('/api/v1/auth/change-password', { method: 'PUT' })).toBe('purge')
    expect(route('/api/v1/auth/reset-password', { method: 'POST' })).toBe('purge')
    expect(route('/api/v1/setup/admin', { method: 'POST' })).toBe('purge')
    expect(route('/api/v1/auth/oauth/github', { mode: 'navigate' })).toBe('purge')
    expect(route('/api/v1/auth/oauth/github/callback?code=x', { mode: 'navigate' })).toBe('purge')
  })

  it('serves build assets and icons cache-first', () => {
    expect(route('/_next/static/chunks/app-1a2b3c.js', { mode: 'no-cors' })).toBe('static')
    expect(route('/_next/static/media/font-1a2b.woff2')).toBe('static')
    expect(route('/favicon/android-chrome-192x192.png', { mode: 'no-cors' })).toBe('static')
  })

  it('does not cache unhashed or optimised files', () => {
    expect(route('/_next/image?url=%2Fimages%2Flogo.png&w=64&q=75')).toBeNull()
    expect(route('/sw.js')).toBeNull()
    expect(route('/manifest.webmanifest')).toBeNull()
    // RSC payloads of client-side navigations
    expect(route('/dashboard?_rsc=abc')).toBeNull()
  })

  it('answers page loads network-first, but not API URLs opened as a page', () => {
    expect(route('/dashboard', { mode: 'navigate' })).toBe('navigate')
    expect(route('/', { mode: 'navigate' })).toBe('navigate')
    expect(route('/api/v1/services', { mode: 'navigate' })).toBeNull()
  })

  it('ignores other origins, even with a matching path', () => {
    expect(route('https://other.test/api/v1/services')).toBeNull()
    expect(route('https://cdn.test/_next/static/chunks/a.js')).toBeNull()
    expect(route('https://other.test/', { mode: 'navigate' })).toBeNull()
  })
})

describe('network-first API strategy', () => {
  it('returns the fresh response while online and never reads the cache', async () => {
    const cache = fakeCache({ [`${ORIGIN}/api/v1/services`]: json([{ name: 'stale' }]) })
    const deps = apiDeps(cache, async () => json([{ name: 'fresh' }]))

    const response = await sw.networkFirst(servicesRequest(), deps)

    expect(await response.json()).toEqual([{ name: 'fresh' }])
    expect(cache.match).not.toHaveBeenCalled()
    // The cached copy is replaced and remembers when it was fetched
    const stored = cache.store.get(`${ORIGIN}/api/v1/services`)!
    expect(await stored.json()).toEqual([{ name: 'fresh' }])
    expect(Date.parse(stored.headers.get(CACHED_AT_HEADER)!)).not.toBeNaN()
  })

  it('passes server errors through instead of serving the cached copy', async () => {
    for (const status of [401, 403, 404, 500, 502, 503]) {
      const cache = fakeCache({ [`${ORIGIN}/api/v1/services`]: json([{ name: 'stale' }]) })
      const deps = apiDeps(cache, async () => json({ error: 'nope' }, { status }))

      const response = await sw.networkFirst(servicesRequest(), deps)

      expect(response.status).toBe(status)
      expect(cache.match).not.toHaveBeenCalled()
      expect(cache.put).not.toHaveBeenCalled()
    }
  })

  it('serves the cached copy only when the network fails', async () => {
    const cache = fakeCache({ [`${ORIGIN}/api/v1/services`]: json([{ name: 'cached' }]) })
    const deps = apiDeps(cache, async () => {
      throw new TypeError('Failed to fetch')
    })

    const response = await sw.networkFirst(servicesRequest(), deps)

    expect(await response.json()).toEqual([{ name: 'cached' }])
  })

  it('fails like the network did when offline with nothing cached', async () => {
    const deps = apiDeps(fakeCache(), async () => {
      throw new TypeError('Failed to fetch')
    })
    await expect(sw.networkFirst(servicesRequest(), deps)).rejects.toThrow('Failed to fetch')
  })

  it('never stores a response that sets a cookie', async () => {
    const cache = fakeCache()
    // Set-Cookie is hidden from fetch() in browsers and in Node, so fake the headers
    const response = json([])
    Object.defineProperty(response, 'headers', {
      value: { has: (n: string) => n === 'Set-Cookie' },
    })
    Object.defineProperty(response, 'clone', { value: () => response })

    await sw.networkFirst(
      servicesRequest(),
      apiDeps(cache, async () => response)
    )

    expect(cache.put).not.toHaveBeenCalled()
  })

  it('never stores a response the server marked no-store', async () => {
    const cache = fakeCache()
    const fresh = async () => json([], { headers: { 'Cache-Control': 'private, no-store' } })
    await sw.networkFirst(servicesRequest(), apiDeps(cache, fresh))
    expect(cache.put).not.toHaveBeenCalled()
  })

  it('still returns the server answer when the cache cannot be written', async () => {
    const cache = fakeCache()
    cache.put.mockRejectedValue(new DOMException('full', 'QuotaExceededError'))
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})

    const response = await sw.networkFirst(
      servicesRequest(),
      apiDeps(cache, async () => json([{ name: 'fresh' }]))
    )

    expect(await response.json()).toEqual([{ name: 'fresh' }])
    warn.mockRestore()
  })

  it('purges the cache on a 401', async () => {
    const deps = apiDeps(fakeCache(), async () => json({ error: 'expired' }, { status: 401 }))
    await sw.networkFirst(servicesRequest(), deps)
    expect(deps.purge).toHaveBeenCalledOnce()
  })

  it('still answers when the cache cannot be opened', async () => {
    const broken = async () => {
      throw new Error('broken')
    }
    const online = { ...apiDeps(fakeCache(), async () => json([])), openCache: broken }
    expect((await sw.networkFirst(servicesRequest(), online)).status).toBe(200)

    const offline = {
      ...apiDeps(fakeCache(), async () => {
        throw new TypeError('Failed to fetch')
      }),
      openCache: broken,
    }
    await expect(sw.networkFirst(servicesRequest(), offline)).rejects.toThrow('Failed to fetch')
  })
})

describe('a purge while an API response is in flight', () => {
  // The previous user's dashboard arrives after the purge, in a worker that
  // never heard of it (an older instance, or no worker controls the page)
  async function storedAfter(purge: (storage: ReturnType<typeof fakeStorage>) => Promise<unknown>) {
    const storage = fakeStorage()
    let respond!: (response: Response) => void
    const fetch = vi.fn(() => new Promise<Response>((resolve) => (respond = resolve)))
    const deps = { fetch, openCache: () => storage.open(API_CACHE), purge: vi.fn() }

    const pending = sw.networkFirst(servicesRequest(), deps)
    await vi.waitFor(() => expect(fetch).toHaveBeenCalled())
    await purge(storage)
    respond(json([{ name: 'previous user' }]))

    expect((await pending).status).toBe(200)
    return (await storage.open(API_CACHE)).store.size
  }

  it('stores the response when nothing purged', async () => {
    expect(await storedAfter(async () => {})).toBe(1)
  })

  it('drops it when another worker instance purged the cache', async () => {
    expect(await storedAfter((storage) => storage.delete(API_CACHE))).toBe(0)
  })

  it('drops it when the page purged without a controlling worker', async () => {
    const stored = await storedAfter(async (storage) => {
      vi.stubGlobal('caches', storage)
      await clearApiCache()
    })
    vi.unstubAllGlobals()
    expect(stored).toBe(0)
  })
})

describe('cache-first static strategy', () => {
  const request = () => new Request(`${ORIGIN}/_next/static/chunks/a.js`)

  it('serves from the cache without touching the network', async () => {
    const cache = fakeCache({ [request().url]: new Response('cached') })
    const fetch = vi.fn()
    const response = await sw.cacheFirst(request(), { fetch, openCache: async () => cache })
    expect(await response.text()).toBe('cached')
    expect(fetch).not.toHaveBeenCalled()
  })

  it('fetches and stores a miss, but not an error', async () => {
    const cache = fakeCache()
    const openCache = async () => cache
    await sw.cacheFirst(request(), { fetch: async () => new Response('js'), openCache })
    expect(await cache.store.get(request().url)!.text()).toBe('js')

    const missing = new Request(`${ORIGIN}/_next/static/chunks/gone.js`)
    const fetch404 = async () => new Response('', { status: 404 })
    const response = await sw.cacheFirst(missing, { fetch: fetch404, openCache })
    expect(response.status).toBe(404)
    expect(cache.store.has(missing.url)).toBe(false)
  })

  it('still returns the file when the cache cannot be written', async () => {
    const cache = fakeCache()
    cache.put.mockRejectedValue(new DOMException('full', 'QuotaExceededError'))
    const fetch = async () => new Response('js')
    const response = await sw.cacheFirst(request(), { fetch, openCache: async () => cache })
    expect(await response.text()).toBe('js')
  })

  it('fetches the file when the cache cannot be opened', async () => {
    const openCache = async () => {
      throw new Error('broken')
    }
    const response = await sw.cacheFirst(request(), {
      fetch: async () => new Response('js'),
      openCache,
    })
    expect(await response.text()).toBe('js')
  })

  it('fetches the file when the cache cannot be read', async () => {
    const cache = fakeCache()
    cache.match.mockRejectedValue(new Error('broken'))
    const fetch = async () => new Response('js')
    const response = await sw.cacheFirst(request(), { fetch, openCache: async () => cache })
    expect(await response.text()).toBe('js')
  })
})

describe('navigation strategy', () => {
  const request = () => new Request(`${ORIGIN}/dashboard`)

  it('returns the page from the network and never reads the cache while online', async () => {
    const cache = fakeCache({ [sw.OFFLINE_URL]: new Response('offline page') })
    const fetch = async () => new Response('dashboard')
    const response = await sw.navigate(request(), { fetch, openCache: async () => cache })
    expect(await response.text()).toBe('dashboard')
    expect(cache.match).not.toHaveBeenCalled()
  })

  it('falls back to the offline page when the network fails', async () => {
    const cache = fakeCache({ [sw.OFFLINE_URL]: new Response('offline page') })
    const fetch = async () => {
      throw new TypeError('Failed to fetch')
    }
    const response = await sw.navigate(request(), { fetch, openCache: async () => cache })
    expect(await response.text()).toBe('offline page')
  })
})

describe('offline page precache', () => {
  const html = `<link rel="stylesheet" href="/_next/static/chunks/app.css"/>
    <link rel="preload" href="/_next/static/media/geist.woff2" as="font"/>
    <script src="/_next/static/chunks/main-1a2b.js" async=""></script>
    <script>self.__next_f.push([1,"[\\"/_next/static/chunks/page-3c4d.js\\"]"])</script>
    <script src="/_next/static/chunks/main-1a2b.js"></script><a href="/dashboard">x</a>`

  it('finds the build assets of a page, each once', () => {
    expect(sw.extractAssetUrls(html)).toEqual([
      '/_next/static/chunks/app.css',
      '/_next/static/media/geist.woff2',
      '/_next/static/chunks/main-1a2b.js',
      '/_next/static/chunks/page-3c4d.js',
    ])
  })

  it('stores the page with its assets and survives a missing asset', async () => {
    const cache = fakeCache()
    const fetch = async (url: string) => {
      if (url === sw.OFFLINE_URL) return new Response(html)
      if (url.endsWith('.woff2')) return new Response('', { status: 404 })
      return new Response('asset')
    }
    await sw.precacheOffline({ fetch, openCache: async () => cache })
    expect([...cache.store.keys()].sort()).toEqual([
      '/_next/static/chunks/app.css',
      '/_next/static/chunks/main-1a2b.js',
      '/_next/static/chunks/page-3c4d.js',
      '/offline',
    ])
  })

  it('fails the install when the offline page cannot be fetched', async () => {
    const fetch = async () => new Response('', { status: 500 })
    const deps = { fetch, openCache: async () => fakeCache() }
    await expect(sw.precacheOffline(deps)).rejects.toThrow('offline page unavailable')
  })

  it('fails the install when the offline page is a redirect', async () => {
    const redirected = new Response(html)
    Object.defineProperty(redirected, 'redirected', { value: true })
    const deps = { fetch: async () => redirected, openCache: async () => fakeCache() }
    await expect(sw.precacheOffline(deps)).rejects.toThrow('offline page unavailable')
  })

  it('removes the caches of older builds only', () => {
    const names = ['nimbus-api-v1', 'nimbus-static-new', 'nimbus-static-old', 'nimbus-api-v0', 'x']
    expect(sw.staleCaches(names, 'new')).toEqual(['nimbus-static-old', 'nimbus-api-v0'])
  })
})

describe('auth proxy', () => {
  it('lets the worker, manifest and offline page through without a session', async () => {
    for (const path of ['/sw.js', '/sw-routing.js', '/manifest.webmanifest', '/offline']) {
      const response = await proxy(new NextRequest(`${ORIGIN}${path}`))
      expect(response.headers.get('location'), path).toBeNull()
    }
  })
})
