import { useEffect } from 'react'
import { act, fireEvent, render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import '@testing-library/jest-dom'
import DashboardLayout from '@/app/(dashboard)/layout'
import LayoutSettings from '@/components/LayoutSettings'
import { getLayoutSnapshot, setLayoutMode } from '@/lib/layout-store'
import { defaultGlass } from '@/lib/wallpaper'

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
    clear: () => Object.keys(store).forEach((key) => delete store[key]),
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

// Counts mounts, like a page that would refetch everything when remounted
let mounts = 0
function Page({ label }: { label: string }) {
  useEffect(() => {
    mounts++
  }, [])
  return <p>{label}</p>
}

const theme = {
  effectiveTheme: 'light' as const,
  setTheme: vi.fn(),
  glass: defaultGlass,
  setGlass: vi.fn(),
  setLayoutMode: vi.fn((mode: 'classic' | 'canvas') => setLayoutMode(mode)),
}
vi.mock('@/contexts/ThemeContext', () => ({ useTheme: () => theme }))

describe('layout store and DashboardLayout', () => {
  beforeEach(() => {
    localStorage.clear()
    pathname = '/dashboard'
    document.documentElement.removeAttribute('data-layout')
  })

  it('defaults to classic and remembers the choice', () => {
    expect(getLayoutSnapshot()).toBe('classic')
    setLayoutMode('canvas')
    expect(getLayoutSnapshot()).toBe('canvas')
    expect(localStorage.getItem('nimbus-layout')).toBe('canvas')
  })

  it('uses the canvas only on the dashboard, without remounting the page', () => {
    mounts = 0
    const { rerender } = render(
      <DashboardLayout>
        <Page label="page" />
      </DashboardLayout>
    )
    expect(screen.getByTestId('classic')).toBeInTheDocument()
    expect(screen.getByTestId('header')).toBeInTheDocument()

    act(() => setLayoutMode('canvas'))
    expect(screen.getByTestId('topbar')).toBeInTheDocument()
    expect(screen.queryByTestId('classic')).not.toBeInTheDocument()
    expect(screen.queryByTestId('header')).not.toBeInTheDocument()
    expect(screen.getByText('page')).toBeInTheDocument()
    expect(mounts).toBe(1)
    expect(document.documentElement).toHaveAttribute('data-layout', 'canvas')

    pathname = '/settings'
    rerender(
      <DashboardLayout>
        <Page label="page" />
      </DashboardLayout>
    )
    expect(screen.getByTestId('classic')).toBeInTheDocument()
    expect(document.documentElement).not.toHaveAttribute('data-layout')
  })

  it('follows a layout change from another tab', () => {
    render(
      <DashboardLayout>
        <Page label="page" />
      </DashboardLayout>
    )
    act(() => {
      localStorage.setItem('nimbus-layout', 'canvas')
      window.dispatchEvent(new StorageEvent('storage', { key: 'nimbus-layout' }))
    })
    expect(screen.getByTestId('topbar')).toBeInTheDocument()
  })
})

describe('LayoutSettings', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.clearAllMocks()
    theme.glass = defaultGlass
  })

  it('turns on glass cards when switching to canvas with plain cards', () => {
    render(<LayoutSettings />)
    expect(screen.getByRole('button', { name: /Classic/ })).toHaveAttribute('aria-pressed', 'true')

    fireEvent.click(screen.getByRole('button', { name: /Canvas/ }))
    expect(theme.setLayoutMode).toHaveBeenCalledWith('canvas')
    expect(theme.setGlass).toHaveBeenCalledWith({ cardOpacity: 75, cardBlur: 12 })
    expect(screen.getByRole('button', { name: /Canvas/ })).toHaveAttribute('aria-pressed', 'true')

    // Suggests dark mode, which the user can take or leave
    fireEvent.click(screen.getByRole('button', { name: 'Use dark mode' }))
    expect(theme.setTheme).toHaveBeenCalledWith('dark')

    // Picking it again changes nothing
    theme.setGlass.mockClear()
    theme.setLayoutMode.mockClear()
    fireEvent.click(screen.getByRole('button', { name: /Canvas/ }))
    expect(theme.setLayoutMode).not.toHaveBeenCalled()
    expect(theme.setGlass).not.toHaveBeenCalled()
  })

  it('keeps glass settings the user already chose', () => {
    theme.glass = { ...defaultGlass, cardOpacity: 90 }
    render(<LayoutSettings />)
    fireEvent.click(screen.getByRole('button', { name: /Canvas/ }))
    expect(theme.setGlass).not.toHaveBeenCalled()
  })
})
