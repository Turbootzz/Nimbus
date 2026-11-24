import { act, renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useConfigFile } from '@/hooks/useConfigFile'
import { canEdit } from '@/lib/config-file'
import { api } from '@/lib/api'
import type { ConfigFileStatus } from '@/types'

vi.mock('@/lib/api', () => ({ api: { getCurrentUser: vi.fn() } }))

const me = (config_file?: ConfigFileStatus) => ({
  data: {
    id: 'u1',
    email: 'admin@example.com',
    name: 'Admin',
    role: 'admin' as const,
    provider: 'local' as const,
    email_verified: true,
    created_at: '2026-01-01T00:00:00Z',
    config_file,
  },
})
const failed = { error: { message: 'Network error' } }
const locked = { read_only: true, preferences: false }

const showTab = () => act(() => void document.dispatchEvent(new Event('visibilitychange')))

describe('useConfigFile', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('visible')
  })
  afterEach(() => {
    vi.restoreAllMocks()
    vi.useRealTimers()
  })

  it('is not known until the server answered, nor when that fails', async () => {
    vi.mocked(api.getCurrentUser).mockResolvedValue(failed)
    const { result } = renderHook(() => useConfigFile())
    expect(result.current).toBeUndefined()
    await waitFor(() => expect(api.getCurrentUser).toHaveBeenCalled())
    expect(result.current).toBeUndefined()
    expect(canEdit(result.current)).toBe(false)
  })

  it('asks again when the tab is shown and keeps what it knew when that fails', async () => {
    vi.mocked(api.getCurrentUser).mockResolvedValue(me())
    const { result } = renderHook(() => useConfigFile())
    await waitFor(() => expect(result.current).toBeNull())
    expect(canEdit(result.current)).toBe(true)

    // The file made the dashboard read-only meanwhile
    vi.mocked(api.getCurrentUser).mockResolvedValue(me(locked))
    showTab()
    await waitFor(() => expect(result.current).toEqual(locked))
    expect(canEdit(result.current)).toBe(false)

    vi.mocked(api.getCurrentUser).mockResolvedValue(failed)
    showTab()
    await waitFor(() => expect(api.getCurrentUser).toHaveBeenCalledTimes(3))
    expect(result.current).toEqual(locked)
  })

  it('asks again every minute', async () => {
    vi.useFakeTimers()
    vi.mocked(api.getCurrentUser).mockResolvedValue(me(locked))
    const { result } = renderHook(() => useConfigFile())
    await act(() => vi.advanceTimersByTimeAsync(0))
    expect(result.current).toEqual(locked)

    vi.mocked(api.getCurrentUser).mockResolvedValue(me())
    await act(() => vi.advanceTimersByTimeAsync(60_000))
    expect(api.getCurrentUser).toHaveBeenCalledTimes(2)
    expect(result.current).toBeNull()
  })
})
