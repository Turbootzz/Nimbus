import { beforeEach, describe, expect, it, vi } from 'vitest'
import { API_CACHE, bindApiCacheTo } from '@/lib/pwa'

describe('offline cache owner', () => {
  const deleteCache = vi.fn(async () => true)

  beforeEach(() => {
    const store = new Map<string, string>()
    vi.stubGlobal('localStorage', {
      getItem: (key: string) => store.get(key) ?? null,
      setItem: (key: string, value: string) => store.set(key, value),
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

  it('keeps the offline copy for the same user', async () => {
    await bindApiCacheTo('user-a')
    deleteCache.mockClear()

    await bindApiCacheTo('user-a')

    expect(deleteCache).not.toHaveBeenCalled()
  })
})
