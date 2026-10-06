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
    const glassVars = localStorage.getItem('glassVars')

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

    // Apply the wallpaper and glass card variables ThemeContext cached; a
    // broken cache must not stop the rest of this script
    if (glassVars) {
      let vars: Record<string, unknown> = {}
      try {
        vars = JSON.parse(glassVars)
      } catch {
        // Rewritten by ThemeContext on the next load
      }
      const names = [
        '--wallpaper-image',
        '--wallpaper-blur',
        '--wallpaper-dim',
        '--card-opacity',
        '--card-backdrop',
      ]
      for (const name of names) {
        if (typeof vars[name] === 'string') root.style.setProperty(name, vars[name])
      }
    }

    // Canvas layout replaces the sidebar on the dashboard; CSS hides the
    // classic chrome until React renders the canvas
    if (
      window.location.pathname === '/dashboard' &&
      localStorage.getItem('nimbus-layout') === 'canvas'
    ) {
      root.setAttribute('data-layout', 'canvas')
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
