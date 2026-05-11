import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { api } from '@/lib/api'

describe('api.fetchServiceFavicon', () => {
  const fetchMock = vi.fn()
  const originalFetch = global.fetch

  beforeEach(() => {
    fetchMock.mockReset()
    global.fetch = fetchMock as unknown as typeof fetch
  })

  afterEach(() => {
    global.fetch = originalFetch
  })

  const okResponse = (body: object) =>
    new Response(JSON.stringify(body), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    })

  const errorResponse = (status: number, body: object) =>
    new Response(JSON.stringify(body), {
      status,
      headers: { 'Content-Type': 'application/json' },
    })

  it('sends both name and url as encoded query params when both are provided', async () => {
    fetchMock.mockResolvedValueOnce(okResponse({ icon_image_path: 'abc.png', message: 'ok' }))

    const result = await api.fetchServiceFavicon({
      name: 'Plex',
      url: 'https://plex.example.com',
    })

    expect(fetchMock).toHaveBeenCalledOnce()
    const [calledUrl, init] = fetchMock.mock.calls[0]
    expect(calledUrl).toBe(
      'http://localhost:8080/api/v1/services/favicon?name=Plex&url=https%3A%2F%2Fplex.example.com'
    )
    expect(init).toMatchObject({ credentials: 'include' })
    expect(result.data?.icon_image_path).toBe('abc.png')
  })

  it('omits absent or blank params', async () => {
    fetchMock.mockResolvedValueOnce(okResponse({ icon_image_path: 'x.svg', message: 'ok' }))

    await api.fetchServiceFavicon({ name: '  Sonarr  ', url: '' })

    const [calledUrl] = fetchMock.mock.calls[0]
    expect(calledUrl).toBe('http://localhost:8080/api/v1/services/favicon?name=Sonarr')
  })

  it('returns the backend error message on non-OK responses', async () => {
    fetchMock.mockResolvedValueOnce(
      errorResponse(502, { error: 'Could not fetch icon: connection refused' })
    )

    const result = await api.fetchServiceFavicon({ url: 'https://nope.invalid' })

    expect(result.data).toBeUndefined()
    expect(result.error?.message).toBe('Could not fetch icon: connection refused')
  })

  it('returns a network error message when fetch itself throws', async () => {
    fetchMock.mockRejectedValueOnce(new Error('boom'))

    const result = await api.fetchServiceFavicon({ url: 'https://example.com' })

    expect(result.error?.message).toBe('boom')
  })
})
