'use client'

import { SunIcon, MoonIcon } from '@heroicons/react/24/outline'
import { useTheme } from '@/contexts/ThemeContext'

export default function ThemeToggle() {
  const { theme, effectiveTheme, setTheme } = useTheme()

  const toggleTheme = () => {
    // If auto mode is on, switch to manual mode with the opposite of current effective theme
    if (theme === 'auto') {
      setTheme(effectiveTheme === 'dark' ? 'light' : 'dark')
    } else {
      // If manual mode, just toggle between light and dark
      setTheme(theme === 'dark' ? 'light' : 'dark')
    }
  }

  return (
    <button
      onClick={toggleTheme}
      className="rounded-md p-2 transition-colors focus:outline-none"
      style={{ color: 'var(--color-text-secondary)' }}
      onMouseEnter={(e) => {
        e.currentTarget.style.backgroundColor = 'var(--color-card-border)'
        e.currentTarget.style.color = 'var(--color-text-primary)'
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.backgroundColor = 'transparent'
        e.currentTarget.style.color = 'var(--color-text-secondary)'
      }}
      aria-label={theme === 'auto' ? 'Toggle theme (currently auto)' : 'Toggle theme'}
      title={theme === 'auto' ? 'Auto mode - click to switch to manual' : undefined}
    >
      {effectiveTheme === 'dark' ? (
        <SunIcon className="h-5 w-5" />
      ) : (
        <MoonIcon className="h-5 w-5" />
      )}
    </button>
  )
}
