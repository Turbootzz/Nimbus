'use client'

import {
  createContext,
  useContext,
  useEffect,
  useState,
  useCallback,
  useMemo,
  useRef,
  ReactNode,
} from 'react'
import { api } from '@/lib/api'
import type { PreferencesUpdateRequest } from '@/types'

interface ThemeContextType {
  theme: 'light' | 'dark'
  accentColor?: string
  background?: string
  openInNewTab: boolean
  setTheme: (theme: 'light' | 'dark') => void
  setAccentColor: (color: string | undefined) => void
  setBackground: (background: string | undefined) => void
  setOpenInNewTab: (openInNewTab: boolean) => void
  loading: boolean
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined)

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [theme, setThemeState] = useState<'light' | 'dark'>('light')
  const [accentColor, setAccentColorState] = useState<string | undefined>()
  const [background, setBackgroundState] = useState<string | undefined>()
  const [openInNewTab, setOpenInNewTabState] = useState<boolean>(true)
  const [loading, setLoading] = useState(true)
  const [syncing, setSyncing] = useState(false)

  // Ref to store pending updates that arrive while syncing
  const pendingUpdatesRef = useRef<PreferencesUpdateRequest | null>(null)

  // Load preferences on mount (from localStorage first, then API)
  useEffect(() => {
    const loadPreferences = async () => {
      // Step 1: Load from localStorage immediately (fast, prevents flash)
      const savedTheme = localStorage.getItem('theme') as 'light' | 'dark' | null
      const savedAccent = localStorage.getItem('accentColor')
      const savedBackground = localStorage.getItem('background')
      const savedOpenInNewTab = localStorage.getItem('openInNewTab')

      if (savedTheme) setThemeState(savedTheme)
      if (savedAccent) setAccentColorState(savedAccent)
      if (savedBackground) setBackgroundState(savedBackground)
      if (savedOpenInNewTab !== null) setOpenInNewTabState(savedOpenInNewTab === 'true')

      // Step 2: Try to load from API and sync
      try {
        const response = await api.getPreferences()

        if (response.data) {
          // API data is the source of truth - update state and localStorage
          const apiTheme = response.data.theme_mode || 'light'
          const apiAccent = response.data.theme_accent_color
          const apiBackground = response.data.theme_background
          const apiOpenInNewTab = response.data.open_in_new_tab ?? true

          setThemeState(apiTheme)
          setAccentColorState(apiAccent)
          setBackgroundState(apiBackground)
          setOpenInNewTabState(apiOpenInNewTab)

          // Update localStorage cache with API data
          localStorage.setItem('theme', apiTheme)
          localStorage.setItem('openInNewTab', String(apiOpenInNewTab))
          if (apiAccent) {
            localStorage.setItem('accentColor', apiAccent)
          } else {
            localStorage.removeItem('accentColor')
          }
          if (apiBackground) {
            localStorage.setItem('background', apiBackground)
          } else {
            localStorage.removeItem('background')
          }
        }
        // If response.error (e.g., 401 Unauthorized), silently use localStorage values
      } catch (error) {
        // Network error or other issue - fall back to localStorage
        if (error instanceof Error) {
          console.warn('Failed to load preferences from API, using localStorage:', error.message)
        } else {
          // Unexpected error type - log and re-throw in development
          console.error('Unexpected error loading preferences:', error)
          if (process.env.NODE_ENV === 'development') throw error
        }
      } finally {
        setLoading(false)
      }
    }

    loadPreferences()
  }, [])

  // Apply theme to document
  useEffect(() => {
    const root = document.documentElement

    // Set theme mode
    if (theme === 'dark') {
      root.classList.add('dark')
      root.classList.remove('light')
    } else {
      root.classList.add('light')
      root.classList.remove('dark')
    }

    // Set accent color
    if (accentColor) {
      root.style.setProperty('--color-primary', accentColor)
      root.style.setProperty('--color-primary-hover', accentColor)
      root.style.setProperty('--dark-primary', accentColor)
      root.style.setProperty('--dark-primary-hover', accentColor)
    } else {
      root.style.removeProperty('--color-primary')
      root.style.removeProperty('--color-primary-hover')
      root.style.removeProperty('--dark-primary')
      root.style.removeProperty('--dark-primary-hover')
    }

    // Set background image with XSS protection
    if (background) {
      try {
        const parsedURL = new URL(background, window.location.href)
        if (parsedURL.protocol === 'http:' || parsedURL.protocol === 'https:') {
          document.body.style.backgroundImage = `url("${parsedURL.href}")`
          document.body.style.backgroundSize = 'cover'
          document.body.style.backgroundPosition = 'center'
          document.body.style.backgroundAttachment = 'fixed'
        } else {
          console.warn(`Background URL rejected: only HTTP(S) URLs are allowed`)
        }
      } catch {
        console.warn('Invalid background URL:', background)
      }
    } else {
      document.body.style.backgroundImage = ''
      document.body.style.backgroundSize = ''
      document.body.style.backgroundPosition = ''
      document.body.style.backgroundAttachment = ''
    }
  }, [theme, accentColor, background])

  // Helper to update localStorage for a set of preference updates
  const updateLocalStorage = (updates: PreferencesUpdateRequest) => {
    if (updates.theme_mode) localStorage.setItem('theme', updates.theme_mode)
    if (updates.open_in_new_tab !== undefined) {
      localStorage.setItem('openInNewTab', String(updates.open_in_new_tab))
    }
    if (updates.theme_accent_color !== undefined) {
      if (updates.theme_accent_color) {
        localStorage.setItem('accentColor', updates.theme_accent_color)
      } else {
        localStorage.removeItem('accentColor')
      }
    }
    if (updates.theme_background !== undefined) {
      if (updates.theme_background) {
        localStorage.setItem('background', updates.theme_background)
      } else {
        localStorage.removeItem('background')
      }
    }
  }

  // Save preferences to API and localStorage
  const savePreferences = useCallback(
    async (updates: PreferencesUpdateRequest) => {
      // Always update localStorage immediately for instant UI feedback
      updateLocalStorage(updates)

      // If already syncing, merge updates into pending queue
      if (syncing) {
        pendingUpdatesRef.current = {
          ...(pendingUpdatesRef.current ?? {}),
          ...updates,
        }
        return
      }

      setSyncing(true)
      try {
        // Save to API
        const response = await api.updatePreferences(updates)

        if (response.error) {
          console.warn('Failed to save preferences to API:', response.error.message)
        }
      } catch (error) {
        console.error('Error saving preferences:', error)
      } finally {
        setSyncing(false)

        // If updates arrived while we were syncing, flush them now
        const pending = pendingUpdatesRef.current
        if (pending && Object.keys(pending).length > 0) {
          pendingUpdatesRef.current = null
          savePreferences(pending)
        }
      }
    },
    [syncing]
  )

  const setTheme = useCallback(
    (newTheme: 'light' | 'dark') => {
      setThemeState(newTheme)
      savePreferences({ theme_mode: newTheme })
    },
    [savePreferences]
  )

  const setAccentColor = useCallback(
    (color: string | undefined) => {
      setAccentColorState(color)
      savePreferences({ theme_accent_color: color ? color : null })
    },
    [savePreferences]
  )

  const setBackground = useCallback(
    (bg: string | undefined) => {
      setBackgroundState(bg)
      savePreferences({ theme_background: bg ? bg : null })
    },
    [savePreferences]
  )

  const setOpenInNewTab = useCallback(
    (value: boolean) => {
      setOpenInNewTabState(value)
      savePreferences({ open_in_new_tab: value })
    },
    [savePreferences]
  )

  const value = useMemo(
    () => ({
      theme,
      accentColor,
      background,
      openInNewTab,
      setTheme,
      setAccentColor,
      setBackground,
      setOpenInNewTab,
      loading,
    }),
    [
      theme,
      accentColor,
      background,
      openInNewTab,
      setTheme,
      setAccentColor,
      setBackground,
      setOpenInNewTab,
      loading,
    ]
  )

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
}

export function useTheme() {
  const context = useContext(ThemeContext)
  if (context === undefined) {
    throw new Error('useTheme must be used within a ThemeProvider')
  }
  return context
}
