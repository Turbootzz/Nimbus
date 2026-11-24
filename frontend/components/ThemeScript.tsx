/**
 * ThemeScript - Blocking script that prevents FOUC (Flash of Unstyled Content)
 *
 * This component renders a script that executes BEFORE React hydration.
 * It reads theme preferences from localStorage and applies them immediately.
 *
 */

import { isDashboardPath } from '@/lib/boards'
import { isKioskSearch } from '@/lib/kiosk'

// This function will be stringified and injected into the HTML. It gets the
// path rules of lib/boards and lib/kiosk the same way, as arguments.
function applyThemeBeforeHydration(
  isBoardPath: (pathname: string) => boolean,
  isKiosk: (search: string) => boolean
) {
  try {
    // Read from localStorage
    const theme = localStorage.getItem('theme') || 'auto'
    const accentColor = localStorage.getItem('accentColor')
    const glassVars = localStorage.getItem('glassVars')

    const root = document.documentElement

    // The public status page follows the visitor's system theme and takes
    // nothing a signed-in user left in this browser (see isStatusPagePath)
    if (window.location.pathname === '/status' || window.location.pathname.startsWith('/status/')) {
      root.classList.add('auto')
      return
    }

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
      // White text on a custom accent, also on a preset (styles/themes.css)
      root.style.setProperty('--color-on-primary', '#fff')
    }

    // Preset, font and monospace KPI labels: attributes that
    // styles/themes.css reads. A value it doesn't know matches no rule.
    const preset = localStorage.getItem('themePreset')
    if (preset && preset !== 'default') root.setAttribute('data-preset', preset)
    const font = localStorage.getItem('themeFont')
    if (font && font !== 'default') root.setAttribute('data-font', font)
    if (localStorage.getItem('kpiMono') === 'true') root.setAttribute('data-kpi-mono', '')

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
      // Lets CSS give text on the wallpaper a surface (globals.css)
      const image = vars['--wallpaper-image']
      if (typeof image === 'string' && image !== 'none') {
        root.setAttribute('data-wallpaper', '')
      }
    }

    // Canvas layout replaces the sidebar on a board; CSS hides the classic
    // chrome until React renders the canvas. Without a stored layout (new
    // user or browser) CSS hides the page until the preferences say which
    // one it is. Kiosk mode hides all chrome.
    if (isBoardPath(window.location.pathname)) {
      if (isKiosk(window.location.search)) root.setAttribute('data-kiosk', '')
      const layout = localStorage.getItem('nimbus-layout')
      if (layout === 'canvas') {
        root.setAttribute('data-layout', 'canvas')
      } else if (layout === null) {
        root.setAttribute('data-layout', 'pending')
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
  const scriptContent = `(${applyThemeBeforeHydration.toString()})(${isDashboardPath.toString()}, ${isKioskSearch.toString()})`

  return <script dangerouslySetInnerHTML={{ __html: scriptContent }} />
}
