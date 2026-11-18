import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  barLabel,
  barPeriod,
  barTone,
  fetchPublicStatus,
  formatUptime,
  serverApiUrl,
} from '@/lib/status-page'

afterEach(() => {
  vi.unstubAllEnvs()
  vi.unstubAllGlobals()
})

describe('barTone', () => {
  it('separates no data from downtime', () => {
    expect(barTone(null)).toBe('none')
    expect(barTone(0)).toBe('down')
  })

  it('is only up at a full 100%', () => {
    expect(barTone(100)).toBe('up')
    expect(barTone(99.99)).toBe('partial')
    expect(barTone(90)).toBe('partial')
    expect(barTone(89.99)).toBe('down')
  })
})

describe('formatUptime', () => {
  it('drops trailing zeros and keeps two decimals', () => {
    expect(formatUptime(100)).toBe('100%')
    expect(formatUptime(99.5)).toBe('99.5%')
    expect(formatUptime(33.333)).toBe('33.33%')
    expect(formatUptime(0)).toBe('0%')
    expect(formatUptime(null)).toBe('No data')
  })
})

describe('barPeriod', () => {
  it('counts back from the newest bar in hours', () => {
    expect(barPeriod(23, 24, 1)).toBe('This hour')
    expect(barPeriod(22, 24, 1)).toBe('1 hour ago')
    expect(barPeriod(0, 24, 1)).toBe('23 hours ago')
  })

  it('uses days for the six-hour bars where they fit', () => {
    expect(barPeriod(27, 28, 6)).toBe('Last 6 hours')
    expect(barPeriod(26, 28, 6)).toBe('6 hours ago')
    expect(barPeriod(23, 28, 6)).toBe('1 day ago')
    expect(barPeriod(19, 28, 6)).toBe('2 days ago')
  })
})

describe('barLabel', () => {
  it('is the text alternative of a bar', () => {
    expect(barLabel(98.5, 20, 24, 1)).toBe('3 hours ago: 98.5% uptime')
    expect(barLabel(null, 23, 24, 1)).toBe('This hour: no data')
  })
})

describe('serverApiUrl', () => {
  it('prefers INTERNAL_API_URL', () => {
    vi.stubEnv('INTERNAL_API_URL', 'http://backend:8080/api/v1')
    expect(serverApiUrl()).toBe('http://backend:8080/api/v1')
  })

  it('reaches the bundled backend when the browser uses a relative path', () => {
    vi.stubEnv('INTERNAL_API_URL', '')
    vi.stubEnv('NEXT_PUBLIC_API_URL', 'same-origin')
    expect(serverApiUrl()).toBe('http://127.0.0.1:8080/api/v1')
  })

  it('uses an absolute public URL as it is', () => {
    vi.stubEnv('INTERNAL_API_URL', '')
    vi.stubEnv('NEXT_PUBLIC_API_URL', 'https://nimbus.example.com')
    expect(serverApiUrl()).toBe('https://nimbus.example.com/api/v1')
  })
})

describe('fetchPublicStatus', () => {
  const page = { title: 'Lab', overall: 'operational', updated_at: '', services: [] }

  it('returns the page and passes the forwarding headers on, uncached', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify(page)))
    vi.stubGlobal('fetch', fetchMock)
    const visitor = new Headers({
      'x-forwarded-for': '198.51.100.4, 172.18.0.2',
      'x-real-ip': '172.18.0.2',
      cookie: 'auth_token=secret',
    })

    expect(await fetchPublicStatus(visitor)).toEqual({ state: 'ok', page })
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toMatch(/\/api\/v1\/public\/status$/)
    expect(init).toEqual({
      cache: 'no-store',
      headers: { 'x-forwarded-for': '198.51.100.4, 172.18.0.2', 'x-real-ip': '172.18.0.2' },
    })
  })

  it('reports a 404 as disabled', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('{}', { status: 404 })))
    expect(await fetchPublicStatus()).toEqual({ state: 'disabled' })
  })

  it('reports rate limits, errors and an unreachable backend as unavailable', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('{}', { status: 429 })))
    expect(await fetchPublicStatus()).toEqual({ state: 'unavailable' })
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('ECONNREFUSED')))
    expect(await fetchPublicStatus()).toEqual({ state: 'unavailable' })
  })
})
