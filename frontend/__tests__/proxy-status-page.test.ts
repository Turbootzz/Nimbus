import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'
import { NextRequest } from 'next/server'

beforeAll(() => {
  process.env.JWT_SECRET = 'test-jwt-secret-key-for-middleware-minimum-32-characters'
})

afterEach(() => {
  vi.unstubAllGlobals()
})

async function requestStatusPage(backendStatus: number) {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('{}', { status: backendStatus })))
  const { default: proxy } = await import('@/proxy')
  return proxy(new NextRequest('http://localhost/status'))
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
})
