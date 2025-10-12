'use client'

import { createContext, useContext, useEffect, useState, ReactNode } from 'react'
import { api } from '@/lib/api'
import type { UserPreferences } from '@/types'

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

    // Set background image on body
    if (background) {
      document.body.style.backgroundImage = `url(${background})`
      document.body.style.backgroundSize = 'cover'
      document.body.style.backgroundPosition = 'center'
      document.body.style.backgroundAttachment = 'fixed'
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
  }, [theme, accentColor, background])

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
      if (response.data) {
        setThemeState(response.data.theme_mode)
        setAccentColorState(response.data.theme_accent_color)
        setBackgroundState(response.data.theme_background)
      }
    } catch (error) {
      console.error('Failed to load preferences:', error)
    } finally {
      setLoading(false)
    }
  }

  const savePreferences = async () => {
    try {
      await api.updatePreferences({
        theme_mode: theme,
        theme_accent_color: accentColor,
        theme_background: background,
      })
    } catch (error) {
      console.error('Failed to save preferences:', error)
      throw error
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
