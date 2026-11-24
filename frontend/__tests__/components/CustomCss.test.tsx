import { render } from '@testing-library/react'
import { renderToStaticMarkup } from 'react-dom/server'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ThemeScript } from '@/components/ThemeScript'

// happy-dom's localStorage has no clear()
const store: Record<string, string> = {}
Object.defineProperty(window, 'localStorage', {
  value: {
    getItem: (key: string) => store[key] ?? null,
    setItem: (key: string, value: string) => {
      store[key] = String(value)
    },
    removeItem: (key: string) => {
      delete store[key]
    },
  },
})
const clearStore = () => Object.keys(store).forEach((key) => delete store[key])

const theme = { customCss: '' }
vi.mock('@/contexts/ThemeContext', () => ({ useTheme: () => theme }))

// safeMode is read once when the module loads, so each test loads it anew
async function renderCustomCss(path = '/dashboard') {
  window.history.replaceState(null, '', path)
  vi.resetModules()
  const { default: CustomCss } = await import('@/components/CustomCss')
  return render(<CustomCss />)
}

describe('CustomCss', () => {
  beforeEach(() => {
    theme.customCss = ''
  })

  it('renders nothing without CSS', async () => {
    const { container } = await renderCustomCss()
    expect(container.querySelector('style')).toBeNull()
  })

  it('writes the CSS as text, so it cannot leave the style element', async () => {
    theme.customCss =
      '</style><script>window.pwned = true</script><!-- <img src=x onerror=alert(1)>'
    const { container } = await renderCustomCss()

    const style = container.querySelector('style')
    expect(style?.textContent).toBe(theme.customCss)
    expect(style?.children).toHaveLength(0)
    expect(container.querySelector('script, img')).toBeNull()
    expect(container.children).toHaveLength(1)
  })

  it('leaves the CSS out in safe mode', async () => {
    theme.customCss = 'body { display: none }'
    const { container } = await renderCustomCss('/settings/theme?safe=1')
    expect(container.querySelector('style')).toBeNull()
  })

  it('is not turned off by other values of safe', async () => {
    theme.customCss = 'body { color: red }'
    const { container } = await renderCustomCss('/dashboard?safe=0')
    expect(container.querySelector('style')).not.toBeNull()
  })
})

describe('ThemeScript', () => {
  const root = document.documentElement
  const run = () => {
    const script = renderToStaticMarkup(<ThemeScript />).replace(/^<script>|<\/script>$/g, '')
    new Function(script)()
  }

  beforeEach(() => {
    clearStore()
    for (const name of ['data-preset', 'data-font', 'data-kpi-mono']) root.removeAttribute(name)
  })

  it('applies the cached preset and font before React', () => {
    localStorage.setItem('themePreset', 'nord')
    localStorage.setItem('themeFont', 'inter')
    localStorage.setItem('kpiMono', 'true')
    run()
    expect(root.getAttribute('data-preset')).toBe('nord')
    expect(root.getAttribute('data-font')).toBe('inter')
    expect(root.hasAttribute('data-kpi-mono')).toBe(true)
  })

  it('sets nothing for the defaults', () => {
    localStorage.setItem('themePreset', 'default')
    localStorage.setItem('themeFont', 'default')
    localStorage.setItem('kpiMono', 'false')
    run()
    expect(root.hasAttribute('data-preset')).toBe(false)
    expect(root.hasAttribute('data-font')).toBe(false)
    expect(root.hasAttribute('data-kpi-mono')).toBe(false)
  })

  it('never reads custom CSS', () => {
    expect(renderToStaticMarkup(<ThemeScript />)).not.toMatch(/custom_?css/i)
  })
})
