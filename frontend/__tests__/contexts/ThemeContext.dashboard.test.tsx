import type { ReactNode } from 'react'
import { act, renderHook, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ThemeProvider, useTheme } from '@/contexts/ThemeContext'
import { api } from '@/lib/api'
import { getLayoutPendingSnapshot, getLayoutSnapshot } from '@/lib/layout-store'

vi.mock('@/lib/api', () => ({
  api: { getPreferences: vi.fn(), updatePreferences: vi.fn(() => Promise.resolve({ data: {} })) },
}))

// happy-dom's localStorage has no clear()
const store: Record<string, string> = {}
Object.defineProperty(window, 'localStorage', {
  value: {
    getItem: (key: string) => store[key] ?? null,
    setItem: (key: string, value: string) => {
      store[key] = String(value)
    },
    removeItem: (key: string) => {
      delete store[key]
    },
  },
})
Object.defineProperty(window, 'matchMedia', {
  value: () => ({ matches: false, addEventListener: vi.fn(), removeEventListener: vi.fn() }),
})

const wrapper = ({ children }: { children: ReactNode }) => <ThemeProvider>{children}</ThemeProvider>

const preferences = (data: object) =>
  vi
    .mocked(api.getPreferences)
    .mockResolvedValue({ data: { theme_mode: 'auto', ...data } } as never)

async function loadTheme() {
  const { result } = renderHook(() => useTheme(), { wrapper })
  await waitFor(() => expect(result.current.loading).toBe(false))
  return result
}

// First: once the wait for the layout has ended it stays ended
describe('ThemeContext layout of a new browser', () => {
  beforeEach(() => {
    Object.keys(store).forEach((key) => delete store[key])
    vi.clearAllMocks()
  })

  it('waits for the preferences, then knows the layout', async () => {
    let answer: (response: never) => void = () => {}
    vi.mocked(api.getPreferences).mockReturnValue(new Promise((resolve) => (answer = resolve)))
    const { result } = renderHook(() => useTheme(), { wrapper })
    await waitFor(() => expect(api.getPreferences).toHaveBeenCalled())
    expect(getLayoutPendingSnapshot()).toBe(true)

    await act(async () => answer({ data: { theme_mode: 'auto', layout_mode: 'canvas' } } as never))
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(getLayoutPendingSnapshot()).toBe(false)
    expect(getLayoutSnapshot()).toBe('canvas')
  })

  it('stops waiting when the preferences cannot be loaded', async () => {
    vi.mocked(api.getPreferences).mockResolvedValue({ error: { message: 'offline' } })
    await loadTheme()
    expect(store['nimbus-layout']).toBeUndefined()
    expect(getLayoutPendingSnapshot()).toBe(false)
  })
})

describe('ThemeContext group display', () => {
  beforeEach(() => {
    Object.keys(store).forEach((key) => delete store[key])
    vi.clearAllMocks()
  })

  it('has no choice for a new user, so groups follow the layout', async () => {
    store.groupDisplay = 'sections' // stale cache of another account
    preferences({ group_display: null })
    const result = await loadTheme()
    expect(result.current.groupDisplay).toBeNull()
    expect(store.groupDisplay).toBeUndefined()
  })

  it('loads the choice and caches it', async () => {
    preferences({ group_display: 'sections' })
    const result = await loadTheme()
    expect(result.current.groupDisplay).toBe('sections')
    expect(store.groupDisplay).toBe('sections')
  })

  it('keeps a choice made in another tab while the preferences load', async () => {
    let answer: (response: never) => void = () => {}
    vi.mocked(api.getPreferences).mockReturnValue(new Promise((resolve) => (answer = resolve)))
    const { result } = renderHook(() => useTheme(), { wrapper })
    await waitFor(() => expect(api.getPreferences).toHaveBeenCalled())

    store.groupDisplay = 'sections'
    act(() => {
      window.dispatchEvent(
        new StorageEvent('storage', {
          key: 'groupDisplay',
          newValue: 'sections',
          storageArea: localStorage,
        })
      )
    })
    await act(async () => answer({ data: { theme_mode: 'auto', group_display: 'tabs' } } as never))
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.groupDisplay).toBe('sections')
    expect(store.groupDisplay).toBe('sections')
  })

  it('saves a new choice', async () => {
    preferences({ group_display: 'tabs' })
    const result = await loadTheme()
    act(() => result.current.setGroupDisplay('sections'))
    expect(result.current.groupDisplay).toBe('sections')
    expect(store.groupDisplay).toBe('sections')
    expect(api.updatePreferences).toHaveBeenCalledWith({ group_display: 'sections' })
  })
})
