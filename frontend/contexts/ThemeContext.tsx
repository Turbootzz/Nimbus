'use client'

import { createContext, useContext, useEffect, useState, ReactNode } from 'react'

interface ThemeContextType {
  theme: 'light' | 'dark'
  accentColor?: string
  background?: string
  setTheme: (theme: 'light' | 'dark') => void
  setAccentColor: (color: string | undefined) => void
  setBackground: (background: string | undefined) => void
  loading: boolean
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined)

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [theme, setThemeState] = useState<'light' | 'dark'>('light')
  const [accentColor, setAccentColorState] = useState<string | undefined>()
  const [background, setBackgroundState] = useState<string | undefined>()
  const [loading, setLoading] = useState(true)

  // Load preferences on mount (from localStorage only)
  useEffect(() => {
    // Load from localStorage immediately
    const savedTheme = localStorage.getItem('theme') as 'light' | 'dark' | null
    const savedAccent = localStorage.getItem('accentColor')
    const savedBackground = localStorage.getItem('background')

    if (savedTheme) setThemeState(savedTheme)
    if (savedAccent) setAccentColorState(savedAccent)
    if (savedBackground) setBackgroundState(savedBackground)

    setLoading(false)
  }, [])

  // Apply theme to document and save to localStorage
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

    // Set background image on body with XSS protection
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

    // Save to localStorage for persistence
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
  }, [theme, accentColor, background])

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
