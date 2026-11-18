import type { ReactNode } from 'react'
import { act, renderHook, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ThemeProvider, useTheme } from '@/contexts/ThemeContext'
import { api } from '@/lib/api'

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

  it('saves a new choice', async () => {
    preferences({ group_display: 'tabs' })
    const result = await loadTheme()
    act(() => result.current.setGroupDisplay('sections'))
    expect(result.current.groupDisplay).toBe('sections')
    expect(store.groupDisplay).toBe('sections')
    expect(api.updatePreferences).toHaveBeenCalledWith({ group_display: 'sections' })
  })
})
