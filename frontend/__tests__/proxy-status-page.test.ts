import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'
import { NextRequest } from 'next/server'

beforeAll(() => {
  process.env.JWT_SECRET = 'test-jwt-secret-key-for-middleware-minimum-32-characters'
})

afterEach(() => {
  vi.unstubAllGlobals()
})

async function requestStatusPage(backendStatus: number, path = '/status') {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('{}', { status: backendStatus })))
  const { default: proxy } = await import('@/proxy')
  // A client trying to pass the page as public itself
  return proxy(
    new NextRequest(`http://localhost${path}`, { headers: { 'x-nimbus-public-page': '1' } })
  )
}

describe('proxy on the public status page', () => {
  it('sends a switched off page to a route that does not exist', async () => {
    const response = await requestStatusPage(404)
    expect(response.headers.get('x-middleware-rewrite')).toBe('http://localhost/-')
  })

  it('lets an enabled or unreachable page through without asking for a login', async () => {
    for (const backendStatus of [200, 503]) {
      const response = await requestStatusPage(backendStatus)
      expect(response.headers.get('x-middleware-rewrite')).toBeNull()
      expect(response.headers.get('location')).toBeNull()
    }
  })

  it('marks only the status page as public, whatever the client sends', async () => {
    const status = await requestStatusPage(200)
    expect(status.headers.get('x-middleware-request-x-nimbus-public-page')).toBe('1')

    const login = await requestStatusPage(200, '/login')
    expect(login.headers.get('x-middleware-override-headers')).not.toContain('x-nimbus-public-page')
    expect(login.headers.get('x-middleware-request-x-nimbus-public-page')).toBeNull()
  })
})
