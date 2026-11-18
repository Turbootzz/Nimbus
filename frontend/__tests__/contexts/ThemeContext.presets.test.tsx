import { act, renderHook, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ThemeProvider, useTheme } from '@/contexts/ThemeContext'
import { api } from '@/lib/api'

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

const stored = {
  theme_mode: 'dark',
  theme_accent_color: '#EC4899',
  theme_preset: 'nord',
  theme_font: 'inter',
  kpi_mono: true,
  icon_source: 'selfhst',
  custom_css: 'body { color: red }',
}

vi.mock('@/lib/api', () => ({
  api: {
    getPreferences: vi.fn(),
    updatePreferences: vi.fn(),
    uploadWallpaper: vi.fn(),
  },
}))

const root = document.documentElement

async function renderTheme() {
  const hook = renderHook(() => useTheme(), { wrapper: ThemeProvider })
  await waitFor(() => expect(hook.result.current.loading).toBe(false))
  return hook.result
}

describe('ThemeContext presets, font and custom CSS', () => {
  beforeEach(() => {
    clearStore()
    root.removeAttribute('data-preset')
    root.removeAttribute('data-font')
    root.removeAttribute('data-kpi-mono')
    root.removeAttribute('style')
    window.matchMedia = vi.fn().mockReturnValue({
      matches: false,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    })
    vi.mocked(api.getPreferences).mockResolvedValue({ data: stored } as never)
    vi.mocked(api.updatePreferences)
      .mockReset()
      .mockResolvedValue({ data: stored } as never)
  })

  it('applies what the server has and caches it for ThemeScript, except the CSS', async () => {
    const theme = await renderTheme()

    expect(theme.current).toMatchObject({
      preset: 'nord',
      font: 'inter',
      kpiMono: true,
      iconSource: 'selfhst',
      customCss: 'body { color: red }',
    })
    expect(root.getAttribute('data-preset')).toBe('nord')
    expect(root.getAttribute('data-font')).toBe('inter')
    expect(root.hasAttribute('data-kpi-mono')).toBe(true)
    expect(localStorage.getItem('themePreset')).toBe('nord')
    expect(localStorage.getItem('themeFont')).toBe('inter')
    expect(localStorage.getItem('kpiMono')).toBe('true')
    expect(JSON.stringify(store)).not.toContain('color: red')
  })

  it('keeps today’s look when the server sends none of it', async () => {
    vi.mocked(api.getPreferences).mockResolvedValue({ data: { theme_mode: 'auto' } } as never)
    const theme = await renderTheme()

    expect(theme.current).toMatchObject({ preset: 'default', font: 'default', kpiMono: false })
    expect(root.hasAttribute('data-preset')).toBe(false)
    expect(root.hasAttribute('data-font')).toBe(false)
    expect(root.style.getPropertyValue('--color-on-primary')).toBe('')
  })

  it('picking a preset applies at once, with its mode and without the old accent', async () => {
    const theme = await renderTheme()
    act(() => theme.current.setPreset('catppuccin-latte'))

    expect(root.getAttribute('data-preset')).toBe('catppuccin-latte')
    expect(root.classList.contains('light')).toBe(true)
    expect(root.style.getPropertyValue('--color-primary')).toBe('')
    expect(theme.current.accentColor).toBeUndefined()
    expect(localStorage.getItem('themePreset')).toBe('catppuccin-latte')
    expect(localStorage.getItem('accentColor')).toBeNull()
    expect(api.updatePreferences).toHaveBeenCalledWith({
      theme_preset: 'catppuccin-latte',
      theme_mode: 'light',
      theme_accent_color: null,
    })
  })

  it('an accent picked on a preset wins and gets white text again', async () => {
    const theme = await renderTheme()
    act(() => theme.current.setPreset('nord'))
    act(() => theme.current.setAccentColor('#10B981'))

    expect(root.getAttribute('data-preset')).toBe('nord')
    expect(root.style.getPropertyValue('--color-primary')).toBe('#10B981')
    expect(root.style.getPropertyValue('--color-on-primary')).toBe('#fff')
  })

  it('a mode the preset does not have goes back to the default preset', async () => {
    const theme = await renderTheme()
    act(() => theme.current.setTheme('dark'))
    expect(theme.current.preset).toBe('nord')
    expect(api.updatePreferences).toHaveBeenLastCalledWith({ theme_mode: 'dark' })

    await waitFor(() => expect(api.updatePreferences).toHaveBeenCalledTimes(1))
    act(() => theme.current.setTheme('light'))
    expect(theme.current.preset).toBe('default')
    expect(root.hasAttribute('data-preset')).toBe(false)
    expect(localStorage.getItem('themePreset')).toBe('default')
  })

  it('saveTheme reports what the server refused and applies nothing', async () => {
    const theme = await renderTheme()
    vi.mocked(api.updatePreferences).mockResolvedValue({
      error: { message: 'Validation failed', fields: { custom_css: '@import is not allowed' } },
    })

    let problem: string | null = null
    await act(async () => {
      problem = await theme.current.saveTheme({ custom_css: '@import "x.css";' })
    })
    expect(problem).toBe('@import is not allowed')
    expect(theme.current.customCss).toBe('body { color: red }')
  })

  it('saveTheme applies the theme the server stored', async () => {
    const theme = await renderTheme()
    vi.mocked(api.updatePreferences).mockResolvedValue({
      data: {
        ...stored,
        theme_mode: 'light',
        theme_accent_color: undefined,
        theme_preset: 'gruvbox-light',
        theme_font: 'jetbrains-mono',
        kpi_mono: false,
        custom_css: '.a { color: blue }',
        card_opacity: 50,
      },
    } as never)

    let problem: string | null = 'unset'
    await act(async () => {
      problem = await theme.current.saveTheme({ theme_preset: 'gruvbox-light' })
    })
    expect(problem).toBeNull()
    expect(theme.current).toMatchObject({
      preset: 'gruvbox-light',
      theme: 'light',
      font: 'jetbrains-mono',
      customCss: '.a { color: blue }',
    })
    expect(theme.current.accentColor).toBeUndefined()
    expect(theme.current.glass.cardOpacity).toBe(50)
    expect(root.getAttribute('data-preset')).toBe('gruvbox-light')
    expect(root.getAttribute('data-font')).toBe('jetbrains-mono')
    expect(root.hasAttribute('data-kpi-mono')).toBe(false)
    expect(localStorage.getItem('theme')).toBe('light')
    expect(localStorage.getItem('accentColor')).toBeNull()
  })
})
