import { act, render, screen } from '@testing-library/react'
import { renderToStaticMarkup } from 'react-dom/server'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import '@testing-library/jest-dom'
import DashboardLayout from '@/app/(dashboard)/layout'
import { ThemeScript } from '@/components/ThemeScript'
import { getLayoutPendingSnapshot, setLayoutMode, settleLayout } from '@/lib/layout-store'

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

let pathname = '/dashboard'
vi.mock('next/navigation', () => ({
  usePathname: () => pathname,
  useRouter: () => ({ push: vi.fn() }),
}))
vi.mock('@/components/Sidebar', () => ({ default: () => <nav data-testid="classic" /> }))
vi.mock('@/components/Header', () => ({ default: () => <header data-testid="header" /> }))
vi.mock('@/components/layout/CanvasTopBar', () => ({
  default: () => <header data-testid="topbar" />,
}))
vi.mock('@/components/CommandPalette', () => ({ default: () => null }))

const root = document.documentElement

// Runs the inline script the way the browser does before React
function runThemeScript(path: string) {
  window.history.replaceState(null, '', path)
  const script = renderToStaticMarkup(<ThemeScript />).replace(/^<script>|<\/script>$/g, '')
  new Function(script)()
}

const renderLayout = () =>
  render(
    <DashboardLayout>
      <p>page</p>
    </DashboardLayout>
  )

// The order matters: settleLayout ends the wait for the rest of the file
describe('first paint without a stored layout', () => {
  beforeEach(() => {
    Object.keys(store).forEach((key) => delete store[key])
    root.removeAttribute('data-layout')
    pathname = '/dashboard'
  })

  it('ThemeScript hides the dashboard until the layout is known', () => {
    runThemeScript('/dashboard')
    expect(root).toHaveAttribute('data-layout', 'pending')
  })

  it('ThemeScript paints a stored layout at once', () => {
    store['nimbus-layout'] = 'canvas'
    runThemeScript('/dashboard')
    expect(root).toHaveAttribute('data-layout', 'canvas')

    root.removeAttribute('data-layout')
    store['nimbus-layout'] = 'classic'
    runThemeScript('/dashboard')
    expect(root).not.toHaveAttribute('data-layout')
  })

  it('ThemeScript leaves other pages alone', () => {
    runThemeScript('/services')
    expect(root).not.toHaveAttribute('data-layout')
  })

  it('stays hidden through hydration and shows the canvas once it is known', () => {
    root.setAttribute('data-layout', 'pending')
    renderLayout()
    expect(getLayoutPendingSnapshot()).toBe(true)
    expect(root).toHaveAttribute('data-layout', 'pending')

    act(() => setLayoutMode('canvas'))
    expect(root).toHaveAttribute('data-layout', 'canvas')
    expect(screen.getByTestId('topbar')).toBeInTheDocument()
  })

  it('shows the classic layout once that is known', () => {
    renderLayout()
    expect(root).toHaveAttribute('data-layout', 'pending')
    act(() => setLayoutMode('classic'))
    expect(root).not.toHaveAttribute('data-layout')
    expect(screen.getByTestId('classic')).toBeInTheDocument()
  })

  it('does not hide other pages', () => {
    pathname = '/settings'
    renderLayout()
    expect(root).not.toHaveAttribute('data-layout')
  })

  it('stops waiting when the preferences cannot be loaded', () => {
    renderLayout()
    expect(root).toHaveAttribute('data-layout', 'pending')
    act(() => settleLayout())
    expect(getLayoutPendingSnapshot()).toBe(false)
    expect(root).not.toHaveAttribute('data-layout')
    expect(screen.getByTestId('classic')).toBeInTheDocument()
  })
})
