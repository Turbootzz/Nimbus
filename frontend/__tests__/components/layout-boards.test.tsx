import { act, render, screen } from '@testing-library/react'
import { renderToStaticMarkup } from 'react-dom/server'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import '@testing-library/jest-dom'
import DashboardLayout from '@/app/(dashboard)/layout'
import { ThemeScript } from '@/components/ThemeScript'
import { setLayoutMode, settleLayout } from '@/lib/layout-store'
import { isDashboardPath } from '@/lib/boards'
import { isKioskSearch } from '@/lib/kiosk'

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

let pathname = '/board/family'
vi.mock('next/navigation', () => ({
  usePathname: () => pathname,
  useRouter: () => ({ push: vi.fn() }),
}))
vi.mock('@/components/Sidebar', () => ({ default: () => <nav data-testid="classic" /> }))
vi.mock('@/components/Header', () => ({ default: () => <header data-testid="header" /> }))
vi.mock('@/components/layout/CanvasTopBar', () => ({
  default: () => <header data-testid="topbar" />,
}))
vi.mock('@/components/CommandPalette', () => ({ default: () => <div data-testid="palette" /> }))
vi.mock('@/components/ConfigFileBanner', () => ({ default: () => <div data-testid="banner" /> }))
vi.mock('@/components/CustomCss', () => ({ default: () => null }))

const root = document.documentElement

function runThemeScript(url: string) {
  window.history.replaceState(null, '', url)
  const script = renderToStaticMarkup(<ThemeScript />).replace(/^<script>|<\/script>$/g, '')
  new Function(script)()
}

const renderLayout = () =>
  render(
    <DashboardLayout>
      <p>page</p>
    </DashboardLayout>
  )

beforeEach(() => {
  Object.keys(store).forEach((key) => delete store[key])
  root.removeAttribute('data-layout')
  root.removeAttribute('data-kiosk')
  pathname = '/board/family'
  window.history.replaceState(null, '', '/board/family')
  settleLayout()
})

afterEach(() => window.history.replaceState(null, '', '/'))

describe('board paths', () => {
  it('are /dashboard and /board/<slug> only', () => {
    expect(isDashboardPath('/dashboard')).toBe(true)
    expect(isDashboardPath('/board/family')).toBe(true)
    expect(isDashboardPath('/dashboard/x')).toBe(false)
    expect(isDashboardPath('/boards')).toBe(false)
    expect(isDashboardPath('/status')).toBe(false)
    expect(isDashboardPath('/services')).toBe(false)
  })

  it('kiosk is ?kiosk=1 only', () => {
    expect(isKioskSearch('?kiosk=1')).toBe(true)
    expect(isKioskSearch('?a=b&kiosk=1')).toBe(true)
    expect(isKioskSearch('?kiosk=0')).toBe(false)
    expect(isKioskSearch('')).toBe(false)
  })

  it('ThemeScript paints a board like the dashboard', () => {
    store['nimbus-layout'] = 'canvas'
    runThemeScript('/board/family')
    expect(root).toHaveAttribute('data-layout', 'canvas')
    expect(root).not.toHaveAttribute('data-kiosk')
  })

  it('ThemeScript hides the chrome of a kiosk board before React', () => {
    runThemeScript('/board/family?kiosk=1')
    expect(root).toHaveAttribute('data-kiosk')
    root.removeAttribute('data-kiosk')
    runThemeScript('/dashboard?kiosk=1')
    expect(root).toHaveAttribute('data-kiosk')
  })

  it('ThemeScript leaves kiosk alone on other pages and the status page', () => {
    for (const url of ['/services?kiosk=1', '/status?kiosk=1']) {
      runThemeScript(url)
      expect(root, url).not.toHaveAttribute('data-kiosk')
      expect(root, url).not.toHaveAttribute('data-layout')
    }
  })

  it('a board gets the canvas like the dashboard', () => {
    renderLayout()
    act(() => setLayoutMode('canvas'))
    expect(screen.getByTestId('topbar')).toBeInTheDocument()
    expect(root).toHaveAttribute('data-layout', 'canvas')
  })
})

describe('kiosk mode', () => {
  it('shows the page and nothing around it, in either layout', () => {
    window.history.replaceState(null, '', '/board/family?kiosk=1')
    for (const layout of ['classic', 'canvas'] as const) {
      setLayoutMode(layout)
      const { unmount } = renderLayout()
      expect(screen.getByText('page')).toBeInTheDocument()
      for (const id of ['classic', 'header', 'topbar', 'palette', 'banner']) {
        expect(screen.queryByTestId(id), `${layout} ${id}`).not.toBeInTheDocument()
      }
      expect(root).toHaveAttribute('data-kiosk')
      unmount()
    }
  })

  it('is only for boards', () => {
    pathname = '/settings'
    window.history.replaceState(null, '', '/settings?kiosk=1')
    renderLayout()
    expect(screen.getByTestId('classic')).toBeInTheDocument()
    expect(screen.getByTestId('palette')).toBeInTheDocument()
    expect(root).not.toHaveAttribute('data-kiosk')
  })
})
