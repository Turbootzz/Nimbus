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

  it('GETs the favicon endpoint with the URL encoded as a query param', async () => {
    fetchMock.mockResolvedValueOnce(okResponse({ icon_image_path: 'abc.png', message: 'ok' }))

    const result = await api.fetchServiceFavicon('https://github.com/owner/repo?x=1')

    expect(fetchMock).toHaveBeenCalledOnce()
    const [calledUrl, init] = fetchMock.mock.calls[0]
    expect(calledUrl).toBe(
      'http://localhost:8080/api/v1/services/favicon?url=https%3A%2F%2Fgithub.com%2Fowner%2Frepo%3Fx%3D1'
    )
    expect(init).toMatchObject({ credentials: 'include' })
    expect(result.data?.icon_image_path).toBe('abc.png')
    expect(result.error).toBeUndefined()
  })

  it('returns the backend error message on non-OK responses', async () => {
    fetchMock.mockResolvedValueOnce(
      errorResponse(502, { error: 'Could not fetch favicon: connection refused' })
    )

    const result = await api.fetchServiceFavicon('https://nope.invalid')

    expect(result.data).toBeUndefined()
    expect(result.error?.message).toBe('Could not fetch favicon: connection refused')
  })

  it('returns a network error message when fetch itself throws', async () => {
    fetchMock.mockRejectedValueOnce(new Error('boom'))

    const result = await api.fetchServiceFavicon('https://example.com')

    expect(result.error?.message).toBe('boom')
  })
})
