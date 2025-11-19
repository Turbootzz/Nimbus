import { renderHook, act, waitFor } from '@testing-library/react'
import { ThemeProvider, useTheme } from '@/contexts/ThemeContext'
import { ReactNode } from 'react'
import { describe, it, expect, beforeEach, vi } from 'vitest'

// Mock the API module
vi.mock('@/lib/api', () => ({
  api: {
    getPreferences: vi.fn(() =>
      Promise.resolve({
        data: {
          theme_mode: 'auto',
          theme_background: null,
          theme_accent_color: null,
          open_in_new_tab: true,
        },
      })
    ),
    updatePreferences: vi.fn(() => Promise.resolve({ data: {} })),
  },
}))

// Mock localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {}

  return {
    getItem: (key: string) => store[key] || null,
    setItem: (key: string, value: string) => {
      store[key] = value.toString()
    },
    removeItem: (key: string) => {
      delete store[key]
    },
    clear: () => {
      store = {}
    },
  }
})()

Object.defineProperty(window, 'localStorage', {
  value: localStorageMock,
})

// Store event listeners for testing
let matchMediaListeners: ((e: MediaQueryListEvent) => void)[] = []

// Mock matchMedia
const mockMatchMedia = (matches: boolean) => {
  matchMediaListeners = []
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: vi.fn().mockImplementation((query) => ({
      matches,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn((event: string, listener: (e: MediaQueryListEvent) => void) => {
        if (event === 'change') {
          matchMediaListeners.push(listener)
        }
      }),
      removeEventListener: vi.fn((event: string, listener: (e: MediaQueryListEvent) => void) => {
        if (event === 'change') {
          matchMediaListeners = matchMediaListeners.filter((l) => l !== listener)
        }
      }),
      dispatchEvent: vi.fn(),
    })),
  })
}

describe('ThemeContext', () => {
  beforeEach(() => {
    localStorageMock.clear()
    vi.clearAllMocks()
    mockMatchMedia(false) // Default to light system theme
  })

  const wrapper = ({ children }: { children: ReactNode }) => (
    <ThemeProvider>{children}</ThemeProvider>
  )

  describe('Initial state and system theme detection', () => {
    it('should initialize with auto theme mode', async () => {
      const { result } = renderHook(() => useTheme(), { wrapper })

      await waitFor(() => {
        expect(result.current.loading).toBe(false)
      })

      expect(result.current.theme).toBe('auto')
    })

    it('should detect system light theme when in auto mode', async () => {
      mockMatchMedia(false) // System is light
      const { result } = renderHook(() => useTheme(), { wrapper })

      await waitFor(() => {
        expect(result.current.loading).toBe(false)
      })

      expect(result.current.theme).toBe('auto')
      expect(result.current.effectiveTheme).toBe('light')
    })

    it('should detect system dark theme when in auto mode', async () => {
      mockMatchMedia(true) // System is dark
      const { result } = renderHook(() => useTheme(), { wrapper })

      await waitFor(() => {
        expect(result.current.loading).toBe(false)
      })

      expect(result.current.theme).toBe('auto')
      expect(result.current.effectiveTheme).toBe('dark')
    })
  })

  describe('Manual theme switching', () => {
    it('should switch to light theme manually', async () => {
      const { result } = renderHook(() => useTheme(), { wrapper })

      await waitFor(() => {
        expect(result.current.loading).toBe(false)
      })

      act(() => {
        result.current.setTheme('light')
      })

      expect(result.current.theme).toBe('light')
      expect(result.current.effectiveTheme).toBe('light')
      expect(localStorage.getItem('theme')).toBe('light')
    })

    it('should switch to dark theme manually', async () => {
      const { result } = renderHook(() => useTheme(), { wrapper })

      await waitFor(() => {
        expect(result.current.loading).toBe(false)
      })

      act(() => {
        result.current.setTheme('dark')
      })

      expect(result.current.theme).toBe('dark')
      expect(result.current.effectiveTheme).toBe('dark')
      expect(localStorage.getItem('theme')).toBe('dark')
    })

    it('should switch back to auto mode', async () => {
      mockMatchMedia(false) // System is light
      const { result } = renderHook(() => useTheme(), { wrapper })

      await waitFor(() => {
        expect(result.current.loading).toBe(false)
      })

      // First set to manual dark
      act(() => {
        result.current.setTheme('dark')
      })

      expect(result.current.theme).toBe('dark')
      expect(result.current.effectiveTheme).toBe('dark')

      // Then switch back to auto
      act(() => {
        result.current.setTheme('auto')
      })

      expect(result.current.theme).toBe('auto')
      expect(result.current.effectiveTheme).toBe('light') // System is light
      expect(localStorage.getItem('theme')).toBe('auto')
    })
  })

  describe('effectiveTheme resolution', () => {
    it('should resolve effectiveTheme correctly in auto mode with light system', async () => {
      mockMatchMedia(false) // System is light
      const { result } = renderHook(() => useTheme(), { wrapper })

      await waitFor(() => {
        expect(result.current.loading).toBe(false)
      })

      act(() => {
        result.current.setTheme('auto')
      })

      expect(result.current.effectiveTheme).toBe('light')
    })

    it('should resolve effectiveTheme correctly in auto mode with dark system', async () => {
      mockMatchMedia(true) // System is dark
      const { result } = renderHook(() => useTheme(), { wrapper })

      await waitFor(() => {
        expect(result.current.loading).toBe(false)
      })

      act(() => {
        result.current.setTheme('auto')
      })

      expect(result.current.effectiveTheme).toBe('dark')
    })

    it('should resolve effectiveTheme to light in manual light mode regardless of system', async () => {
      mockMatchMedia(true) // System is dark
      const { result } = renderHook(() => useTheme(), { wrapper })

      await waitFor(() => {
        expect(result.current.loading).toBe(false)
      })

      act(() => {
        result.current.setTheme('light')
      })

      expect(result.current.effectiveTheme).toBe('light')
    })

    it('should resolve effectiveTheme to dark in manual dark mode regardless of system', async () => {
      mockMatchMedia(false) // System is light
      const { result } = renderHook(() => useTheme(), { wrapper })

      await waitFor(() => {
        expect(result.current.loading).toBe(false)
      })

      act(() => {
        result.current.setTheme('dark')
      })

      expect(result.current.effectiveTheme).toBe('dark')
    })
  })

  describe('DOM class application', () => {
    it('should apply .dark class when theme is dark', async () => {
      const { result } = renderHook(() => useTheme(), { wrapper })

      await waitFor(() => {
        expect(result.current.loading).toBe(false)
      })

      act(() => {
        result.current.setTheme('dark')
      })

      await waitFor(() => {
        expect(document.documentElement.classList.contains('dark')).toBe(true)
        expect(document.documentElement.classList.contains('light')).toBe(false)
        expect(document.documentElement.classList.contains('auto')).toBe(false)
      })
    })

    it('should apply .light class when theme is light', async () => {
      const { result } = renderHook(() => useTheme(), { wrapper })

      await waitFor(() => {
        expect(result.current.loading).toBe(false)
      })

      act(() => {
        result.current.setTheme('light')
      })

      await waitFor(() => {
        expect(document.documentElement.classList.contains('light')).toBe(true)
        expect(document.documentElement.classList.contains('dark')).toBe(false)
        expect(document.documentElement.classList.contains('auto')).toBe(false)
      })
    })

    it('should apply .auto class when theme is auto', async () => {
      const { result } = renderHook(() => useTheme(), { wrapper })

      await waitFor(() => {
        expect(result.current.loading).toBe(false)
      })

      act(() => {
        result.current.setTheme('auto')
      })

      await waitFor(() => {
        expect(document.documentElement.classList.contains('auto')).toBe(true)
        expect(document.documentElement.classList.contains('dark')).toBe(false)
        expect(document.documentElement.classList.contains('light')).toBe(false)
      })
    })
  })

  describe('LocalStorage persistence', () => {
    it('should save theme preference to localStorage', async () => {
      const { result } = renderHook(() => useTheme(), { wrapper })

      await waitFor(() => {
        expect(result.current.loading).toBe(false)
      })

      act(() => {
        result.current.setTheme('dark')
      })

      expect(localStorage.getItem('theme')).toBe('dark')
    })

    it('should load theme from localStorage on mount', async () => {
      localStorage.setItem('theme', 'dark')

      const { result } = renderHook(() => useTheme(), { wrapper })

      // Should load from localStorage immediately
      expect(result.current.theme).toBe('dark')
    })
  })

  describe('Accent color', () => {
    it('should set and persist accent color', async () => {
      const { result } = renderHook(() => useTheme(), { wrapper })

      await waitFor(() => {
        expect(result.current.loading).toBe(false)
      })

      act(() => {
        result.current.setAccentColor('#FF5733')
      })

      expect(result.current.accentColor).toBe('#FF5733')
      expect(localStorage.getItem('accentColor')).toBe('#FF5733')
    })

    it('should clear accent color when set to undefined', async () => {
      localStorage.setItem('accentColor', '#FF5733')
      const { result } = renderHook(() => useTheme(), { wrapper })

      await waitFor(() => {
        expect(result.current.loading).toBe(false)
      })

      act(() => {
        result.current.setAccentColor(undefined)
      })

      expect(result.current.accentColor).toBeUndefined()
      expect(localStorage.getItem('accentColor')).toBeNull()
    })
  })

  describe('Background image', () => {
    it('should set and persist background', async () => {
      const { result } = renderHook(() => useTheme(), { wrapper })

      await waitFor(() => {
        expect(result.current.loading).toBe(false)
      })

      const bgUrl = 'https://example.com/bg.jpg'
      act(() => {
        result.current.setBackground(bgUrl)
      })

      expect(result.current.background).toBe(bgUrl)
      expect(localStorage.getItem('background')).toBe(bgUrl)
    })

    it('should clear background when set to undefined', async () => {
      localStorage.setItem('background', 'https://example.com/bg.jpg')
      const { result } = renderHook(() => useTheme(), { wrapper })

      await waitFor(() => {
        expect(result.current.loading).toBe(false)
      })

      act(() => {
        result.current.setBackground(undefined)
      })

      expect(result.current.background).toBeUndefined()
      expect(localStorage.getItem('background')).toBeNull()
    })
  })

  describe('Open in new tab preference', () => {
    it('should toggle openInNewTab setting', async () => {
      const { result } = renderHook(() => useTheme(), { wrapper })

      await waitFor(() => {
        expect(result.current.loading).toBe(false)
      })

      // Default is true
      expect(result.current.openInNewTab).toBe(true)

      act(() => {
        result.current.setOpenInNewTab(false)
      })

      expect(result.current.openInNewTab).toBe(false)
      expect(localStorage.getItem('openInNewTab')).toBe('false')

      act(() => {
        result.current.setOpenInNewTab(true)
      })

      expect(result.current.openInNewTab).toBe(true)
      expect(localStorage.getItem('openInNewTab')).toBe('true')
    })
  })

  describe('Runtime system theme changes', () => {
    it('should update effectiveTheme when system theme changes in auto mode', async () => {
      // Start with light system theme
      mockMatchMedia(false)
      const { result } = renderHook(() => useTheme(), { wrapper })

      await waitFor(() => {
        expect(result.current.loading).toBe(false)
      })

      // Ensure we're in auto mode
      act(() => {
        result.current.setTheme('auto')
      })

      expect(result.current.theme).toBe('auto')
      expect(result.current.effectiveTheme).toBe('light')

      // Simulate system theme change to dark by calling the registered listeners
      act(() => {
        const event = { matches: true } as MediaQueryListEvent
        matchMediaListeners.forEach((listener) => listener(event))
      })

      await waitFor(() => {
        expect(result.current.effectiveTheme).toBe('dark')
      })

      // Theme mode should still be auto
      expect(result.current.theme).toBe('auto')
    })

    it('should not affect manual theme when system theme changes', async () => {
      // Start with light system theme
      mockMatchMedia(false)
      const { result } = renderHook(() => useTheme(), { wrapper })

      await waitFor(() => {
        expect(result.current.loading).toBe(false)
      })

      // Set manual dark theme
      act(() => {
        result.current.setTheme('dark')
      })

      expect(result.current.theme).toBe('dark')
      expect(result.current.effectiveTheme).toBe('dark')

      // Simulate system theme change by calling the registered listeners
      act(() => {
        const event = { matches: true } as MediaQueryListEvent
        matchMediaListeners.forEach((listener) => listener(event))
      })

      // Manual theme should remain unchanged
      expect(result.current.theme).toBe('dark')
      expect(result.current.effectiveTheme).toBe('dark')
    })
  })
})
