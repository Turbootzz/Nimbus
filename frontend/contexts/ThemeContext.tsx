'use client'

import { createContext, useContext, useEffect, useState, ReactNode, useCallback } from 'react'
import { api } from '@/lib/api'

interface ThemeContextType {
  theme: 'light' | 'dark'
  accentColor?: string
  background?: string
  setTheme: (theme: 'light' | 'dark') => void
  setAccentColor: (color: string | undefined) => void
  setBackground: (background: string | undefined) => void
  savePreferences: () => Promise<void>
  loading: boolean
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined)

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [theme, setThemeState] = useState<'light' | 'dark'>('light')
  const [accentColor, setAccentColorState] = useState<string | undefined>()
  const [background, setBackgroundState] = useState<string | undefined>()
  const [loading, setLoading] = useState(true)

  // Save preferences function (defined early so it can be used in useEffect)
  const savePreferences = useCallback(async () => {
    try {
      const response = await api.updatePreferences({
        theme_mode: theme,
        theme_accent_color: accentColor,
        theme_background: background,
      })

      // Check if the API returned an error
      if (response.error) {
        console.error('Failed to save preferences:', response.error.message)
        throw new Error(response.error.message || 'Failed to save preferences')
      }

      // If successful, the valid preferences are confirmed by server
    } catch (error) {
      console.error('Failed to save preferences (network error):', error)
      // On validation errors, we should revert to last known good state
      throw error
    }
  }, [theme, accentColor, background])

  // Load preferences on mount
  useEffect(() => {
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

    // Set accent color - override the primary color CSS variables
    if (accentColor) {
      root.style.setProperty('--color-primary', accentColor)
      root.style.setProperty('--color-primary-hover', accentColor)
      root.style.setProperty('--dark-primary', accentColor)
      root.style.setProperty('--dark-primary-hover', accentColor)
    } else {
      // Reset to defaults
      root.style.removeProperty('--color-primary')
      root.style.removeProperty('--color-primary-hover')
      root.style.removeProperty('--dark-primary')
      root.style.removeProperty('--dark-primary-hover')
    }

    // Set background image on body with XSS protection
    if (background) {
      // Validate URL to prevent XSS via javascript: or data: schemes
      try {
        const parsedURL = new URL(background)
        if (parsedURL.protocol === 'http:' || parsedURL.protocol === 'https:') {
          // Safe to apply - only http(s) URLs allowed
          document.body.style.backgroundImage = `url(${background})`
          document.body.style.backgroundSize = 'cover'
          document.body.style.backgroundPosition = 'center'
          document.body.style.backgroundAttachment = 'fixed'
        } else {
          // Invalid protocol - skip setting background
          console.warn(
            `Background URL rejected: only HTTP(S) URLs are allowed, got ${parsedURL.protocol}`
          )
        }
      } catch (error) {
        // Invalid URL - skip setting background
        console.warn('Invalid background URL:', background, error)
      }
    } else {
      document.body.style.backgroundImage = ''
      document.body.style.backgroundSize = ''
      document.body.style.backgroundPosition = ''
      document.body.style.backgroundAttachment = ''
    }

    // Save to localStorage for immediate persistence
    localStorage.setItem('theme', theme)
    if (accentColor) {
      localStorage.setItem('accentColor', accentColor)
    } else {
      localStorage.removeItem('accentColor')
    }
    if (background) {
      localStorage.setItem('background', background)
    } else {
      localStorage.removeItem('background')
    }

    // Auto-save to API after a short delay (debounce)
    const timeoutId = setTimeout(() => {
      savePreferences().catch(console.error)
    }, 1000) // Save 1 second after last change

    return () => clearTimeout(timeoutId)
  }, [theme, accentColor, background, savePreferences])

  const loadPreferences = async () => {
    setLoading(true)

    // First, load from localStorage for instant apply
    const savedTheme = localStorage.getItem('theme') as 'light' | 'dark' | null
    const savedAccent = localStorage.getItem('accentColor')
    const savedBackground = localStorage.getItem('background')

    if (savedTheme) setThemeState(savedTheme)
    if (savedAccent) setAccentColorState(savedAccent)
    if (savedBackground) setBackgroundState(savedBackground)

    // Then fetch from API to sync with server
    try {
      const response = await api.getPreferences()

      // Check if the API returned an error
      if (response.error) {
        console.error('Failed to load preferences from server:', response.error.message)
        // Keep localStorage values if API fails
      } else if (response.data) {
        // Successfully loaded from server - update state
        setThemeState(response.data.theme_mode)
        setAccentColorState(response.data.theme_accent_color)
        setBackgroundState(response.data.theme_background)
      }
    } catch (error) {
      console.error('Failed to load preferences (network error):', error)
      // Keep localStorage values if network fails
    } finally {
      setLoading(false)
    }
  }

  const setTheme = (newTheme: 'light' | 'dark') => {
    setThemeState(newTheme)
  }

  const setAccentColor = (color: string | undefined) => {
    setAccentColorState(color)
  }

  const setBackground = (bg: string | undefined) => {
    setBackgroundState(bg)
  }

  return (
    <ThemeContext.Provider
      value={{
        theme,
        accentColor,
        background,
        setTheme,
        setAccentColor,
        setBackground,
        savePreferences,
        loading,
      }}
    >
      {children}
    </ThemeContext.Provider>
  )
}

export function useTheme() {
  const context = useContext(ThemeContext)
  if (context === undefined) {
    throw new Error('useTheme must be used within a ThemeProvider')
  }
  return context
}
