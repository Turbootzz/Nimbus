/**
 * ThemeScript - Blocking script that prevents FOUC (Flash of Unstyled Content)
 *
 * This component renders a script that executes BEFORE React hydration.
 * It reads theme preferences from localStorage and applies them immediately.
 *
 */

// This function will be stringified and injected into the HTML
function applyThemeBeforeHydration() {
  try {
    // Read from localStorage
    const theme = localStorage.getItem('theme') || 'auto'
    const accentColor = localStorage.getItem('accentColor')
    const background = localStorage.getItem('background')

    const root = document.documentElement

    // Apply theme class
    if (theme === 'auto') {
      root.classList.add('auto')
    } else if (theme === 'dark') {
      root.classList.add('dark')
    } else {
      root.classList.add('light')
    }

    // Apply accent color
    if (accentColor) {
      root.style.setProperty('--color-primary', accentColor)
      root.style.setProperty('--color-primary-hover', accentColor)
      root.style.setProperty('--dark-primary', accentColor)
      root.style.setProperty('--dark-primary-hover', accentColor)
    }

    // Apply background
    if (background) {
      try {
        const url = new URL(background, window.location.href)
        if (url.protocol === 'http:' || url.protocol === 'https:') {
          document.body.style.backgroundImage = `url("${url.href}")`
        }
      } catch {
        // Invalid URL, skip
      }
    }

    // Apply sidebar collapsed state
    const sidebarCollapsed = localStorage.getItem('nimbus-sidebar-collapsed')
    if (sidebarCollapsed === 'true') {
      root.setAttribute('data-sidebar-collapsed', 'true')
    }
  } catch {
    // localStorage might be blocked, fail silently
  }
}

export function ThemeScript() {
  const scriptContent = `(${applyThemeBeforeHydration.toString()})()`

  return <script dangerouslySetInnerHTML={{ __html: scriptContent }} />
}
