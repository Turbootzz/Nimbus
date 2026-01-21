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
import type { PreferencesUpdateRequest, CardScale, ViewMode } from '@/types'

interface ThemeContextType {
  theme: 'light' | 'dark' | 'auto'
  effectiveTheme: 'light' | 'dark' // The actual theme being displayed (resolved from auto)
  accentColor?: string
  background?: string
  openInNewTab: boolean
  enableCardResizing: boolean
  enableServiceGrouping: boolean
  cardScale: CardScale
  viewMode: ViewMode
  setTheme: (theme: 'light' | 'dark' | 'auto') => void
  setAccentColor: (color: string | undefined) => void
  setBackground: (background: string | undefined) => void
  setOpenInNewTab: (openInNewTab: boolean) => void
  setEnableCardResizing: (enableCardResizing: boolean) => void
  setEnableServiceGrouping: (enableServiceGrouping: boolean) => void
  setCardScale: (cardScale: CardScale) => void
  setViewMode: (viewMode: ViewMode) => void
  loading: boolean
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined)

export function ThemeProvider({ children }: { children: ReactNode }) {
  // Initialize with defaults (same for server and client to prevent hydration mismatch)
  const [theme, setThemeState] = useState<'light' | 'dark' | 'auto'>('auto')
  const [systemTheme, setSystemTheme] = useState<'light' | 'dark'>('light')
  const [accentColor, setAccentColorState] = useState<string | undefined>()
  const [background, setBackgroundState] = useState<string | undefined>()
  const [openInNewTab, setOpenInNewTabState] = useState<boolean>(true)
  const [enableCardResizing, setEnableCardResizingState] = useState<boolean>(true)
  const [enableServiceGrouping, setEnableServiceGroupingState] = useState<boolean>(true)
  const [cardScale, setCardScaleState] = useState<CardScale>('large')
  const [viewMode, setViewModeState] = useState<ViewMode>('grid')
  const [loading, setLoading] = useState(true)
  const [syncing, setSyncing] = useState(false)

  // Ref to store pending updates that arrive while syncing
  const pendingUpdatesRef = useRef<PreferencesUpdateRequest | null>(null)

  // Compute effective theme (resolve 'auto' to actual theme)
  const effectiveTheme = theme === 'auto' ? systemTheme : theme

  // Detect system theme preference and listen for changes
  useEffect(() => {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')

    // Set initial system theme
    setSystemTheme(mediaQuery.matches ? 'dark' : 'light')

    // Listen for system theme changes
    const handleChange = (e: MediaQueryListEvent) => {
      setSystemTheme(e.matches ? 'dark' : 'light')
    }

    mediaQuery.addEventListener('change', handleChange)
    return () => mediaQuery.removeEventListener('change', handleChange)
  }, [])

  // Sync theme across multiple tabs using storage events
  useEffect(() => {
    const handleStorageChange = (e: StorageEvent) => {
      // Only respond to changes from other tabs (e.storageArea will be set)
      if (!e.storageArea) return

      if (e.key === 'theme' && e.newValue) {
        setThemeState(e.newValue as 'light' | 'dark' | 'auto')
      } else if (e.key === 'accentColor') {
        setAccentColorState(e.newValue || undefined)
      } else if (e.key === 'background') {
        setBackgroundState(e.newValue || undefined)
      } else if (e.key === 'openInNewTab' && e.newValue !== null) {
        setOpenInNewTabState(e.newValue === 'true')
      } else if (e.key === 'enableCardResizing' && e.newValue !== null) {
        setEnableCardResizingState(e.newValue === 'true')
      } else if (e.key === 'enableServiceGrouping' && e.newValue !== null) {
        setEnableServiceGroupingState(e.newValue === 'true')
      } else if (e.key === 'cardScale' && e.newValue) {
        setCardScaleState(e.newValue as CardScale)
      } else if (e.key === 'viewMode' && e.newValue) {
        setViewModeState(e.newValue as ViewMode)
      }
    }

    window.addEventListener('storage', handleStorageChange)
    return () => window.removeEventListener('storage', handleStorageChange)
  }, [])

  // Load preferences on mount (from localStorage first, then API)
  useEffect(() => {
    const loadPreferences = async () => {
      // Step 1: Load from localStorage immediately (fast, prevents flash)
      const savedTheme = localStorage.getItem('theme') as 'light' | 'dark' | 'auto' | null
      const savedAccent = localStorage.getItem('accentColor')
      const savedBackground = localStorage.getItem('background')
      const savedOpenInNewTab = localStorage.getItem('openInNewTab')
      const savedEnableCardResizing = localStorage.getItem('enableCardResizing')
      const savedEnableServiceGrouping = localStorage.getItem('enableServiceGrouping')
      const savedCardScale = localStorage.getItem('cardScale') as CardScale | null
      const savedViewMode = localStorage.getItem('viewMode') as ViewMode | null

      if (savedTheme) setThemeState(savedTheme)
      if (savedAccent) setAccentColorState(savedAccent)
      if (savedBackground) setBackgroundState(savedBackground)
      if (savedOpenInNewTab !== null) setOpenInNewTabState(savedOpenInNewTab === 'true')
      if (savedEnableCardResizing !== null)
        setEnableCardResizingState(savedEnableCardResizing === 'true')
      if (savedEnableServiceGrouping !== null)
        setEnableServiceGroupingState(savedEnableServiceGrouping === 'true')
      if (savedCardScale) setCardScaleState(savedCardScale)
      if (savedViewMode) setViewModeState(savedViewMode)

      // Step 2: Try to load from API and sync
      try {
        const response = await api.getPreferences()

        if (response.data) {
          // API data is the source of truth - update state and localStorage
          const apiTheme = response.data.theme_mode || 'light'
          const apiAccent = response.data.theme_accent_color
          const apiBackground = response.data.theme_background
          const apiOpenInNewTab = response.data.open_in_new_tab ?? true
          const apiEnableCardResizing = response.data.enable_card_resizing ?? true
          const apiEnableServiceGrouping = response.data.enable_service_grouping ?? true
          const apiCardScale = response.data.card_scale ?? 'large'
          const apiViewMode = response.data.view_mode ?? 'grid'

          setThemeState(apiTheme)
          setAccentColorState(apiAccent)
          setBackgroundState(apiBackground)
          setOpenInNewTabState(apiOpenInNewTab)
          setEnableCardResizingState(apiEnableCardResizing)
          setEnableServiceGroupingState(apiEnableServiceGrouping)
          setCardScaleState(apiCardScale)
          setViewModeState(apiViewMode)

          // Update localStorage cache with API data
          localStorage.setItem('theme', apiTheme)
          localStorage.setItem('openInNewTab', String(apiOpenInNewTab))
          localStorage.setItem('enableCardResizing', String(apiEnableCardResizing))
          localStorage.setItem('enableServiceGrouping', String(apiEnableServiceGrouping))
          localStorage.setItem('cardScale', apiCardScale)
          localStorage.setItem('viewMode', apiViewMode)
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

    // Set theme mode based on theme setting
    if (theme === 'auto') {
      // Auto mode: remove manual classes and add auto class
      root.classList.remove('dark', 'light')
      root.classList.add('auto')
    } else if (theme === 'dark') {
      // Manual dark mode
      root.classList.remove('light', 'auto')
      root.classList.add('dark')
    } else {
      // Manual light mode
      root.classList.remove('dark', 'auto')
      root.classList.add('light')
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
    if (updates.enable_card_resizing !== undefined) {
      localStorage.setItem('enableCardResizing', String(updates.enable_card_resizing))
    }
    if (updates.enable_service_grouping !== undefined) {
      localStorage.setItem('enableServiceGrouping', String(updates.enable_service_grouping))
    }
    if (updates.card_scale) localStorage.setItem('cardScale', updates.card_scale)
    if (updates.view_mode) localStorage.setItem('viewMode', updates.view_mode)
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

  // Setters that update local state and sync to API
  const setTheme = useCallback(
    (v: 'light' | 'dark' | 'auto') => {
      setThemeState(v)
      savePreferences({ theme_mode: v })
    },
    [savePreferences]
  )

  const setAccentColor = useCallback(
    (v: string | undefined) => {
      setAccentColorState(v)
      savePreferences({ theme_accent_color: v ?? null })
    },
    [savePreferences]
  )

  const setBackground = useCallback(
    (v: string | undefined) => {
      setBackgroundState(v)
      savePreferences({ theme_background: v ?? null })
    },
    [savePreferences]
  )

  const setOpenInNewTab = useCallback(
    (v: boolean) => {
      setOpenInNewTabState(v)
      savePreferences({ open_in_new_tab: v })
    },
    [savePreferences]
  )

  const setEnableCardResizing = useCallback(
    (v: boolean) => {
      setEnableCardResizingState(v)
      savePreferences({ enable_card_resizing: v })
    },
    [savePreferences]
  )

  const setEnableServiceGrouping = useCallback(
    (v: boolean) => {
      setEnableServiceGroupingState(v)
      savePreferences({ enable_service_grouping: v })
    },
    [savePreferences]
  )

  const setCardScale = useCallback(
    (v: CardScale) => {
      setCardScaleState(v)
      savePreferences({ card_scale: v })
    },
    [savePreferences]
  )

  const setViewMode = useCallback(
    (v: ViewMode) => {
      setViewModeState(v)
      savePreferences({ view_mode: v })
    },
    [savePreferences]
  )

  const value = useMemo(
    () => ({
      theme,
      effectiveTheme,
      accentColor,
      background,
      openInNewTab,
      enableCardResizing,
      enableServiceGrouping,
      cardScale,
      viewMode,
      setTheme,
      setAccentColor,
      setBackground,
      setOpenInNewTab,
      setEnableCardResizing,
      setEnableServiceGrouping,
      setCardScale,
      setViewMode,
      loading,
    }),
    [
      theme,
      effectiveTheme,
      accentColor,
      background,
      openInNewTab,
      enableCardResizing,
      enableServiceGrouping,
      cardScale,
      viewMode,
      setTheme,
      setAccentColor,
      setBackground,
      setOpenInNewTab,
      setEnableCardResizing,
      setEnableServiceGrouping,
      setCardScale,
      setViewMode,
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
