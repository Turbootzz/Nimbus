import { beforeEach, describe, expect, it, vi } from 'vitest'
import { API_CACHE, bindApiCacheTo, clearApiCache } from '@/lib/pwa'
import { api } from '@/lib/api'
import { getLastBoard, setLastBoard } from '@/lib/boards'

describe('offline cache owner', () => {
  const deleteCache = vi.fn(async () => true)

  beforeEach(() => {
    const store = new Map<string, string>()
    vi.stubGlobal('localStorage', {
      getItem: (key: string) => store.get(key) ?? null,
      setItem: (key: string, value: string) => store.set(key, value),
      removeItem: (key: string) => store.delete(key),
    })
    deleteCache.mockClear()
    vi.stubGlobal('caches', { delete: deleteCache })
  })

  it('drops the offline copy when another user turns out to be signed in', async () => {
    await bindApiCacheTo('user-a')
    deleteCache.mockClear()

    await bindApiCacheTo('user-b')

    expect(deleteCache).toHaveBeenCalledWith(API_CACHE)
  })

  it('forgets the board the last user opened', async () => {
    await bindApiCacheTo('user-a')
    setLastBoard('b-lab')
    await clearApiCache() // user B signs in
    expect(getLastBoard()).toBeNull()

    await bindApiCacheTo('user-b')
    setLastBoard('b-lab')
    await bindApiCacheTo('user-c') // a session change no purge saw
    expect(getLastBoard()).toBeNull()
  })

  it('keeps what a new user synced after signing in', async () => {
    await bindApiCacheTo('user-a')
    await clearApiCache() // user B signs in
    deleteCache.mockClear()

    await bindApiCacheTo('user-b')

    expect(deleteCache).not.toHaveBeenCalled()
  })

  it('keeps the offline copy for the same user', async () => {
    await bindApiCacheTo('user-a')
    deleteCache.mockClear()

    await bindApiCacheTo('user-a')

    expect(deleteCache).not.toHaveBeenCalled()
  })
})

describe('a 401 from the API', () => {
  it('drops the offline copy whatever the body, with the usual parse error', async () => {
    const deleteCache = vi.fn(async () => true)
    vi.stubGlobal('caches', { delete: deleteCache })
    const logError = vi.spyOn(console, 'error').mockImplementation(() => {})

    for (const [body, message] of [
      ['', 'invalid response'],
      ['<!DOCTYPE html><html>Unauthorized</html>', 'Cannot reach API server'],
    ]) {
      deleteCache.mockClear()
      vi.stubGlobal(
        'fetch',
        vi.fn(async () => new Response(body, { status: 401 }))
      )

      const result = await api.getCurrentUser()

      expect(result.error?.message).toContain(message)
      expect(deleteCache).toHaveBeenCalledWith(API_CACHE)
    }
    logError.mockRestore()
    vi.unstubAllGlobals()
  })
})
