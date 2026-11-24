import { describe, expect, it } from 'vitest'
import { exportTheme, parseThemeFile } from '@/lib/theme-file'

const theme = {
  preset: 'nord' as const,
  mode: 'dark' as const,
  accentColor: '#EC4899',
  font: 'inter' as const,
  kpiMono: true,
  iconSource: 'selfhst' as const,
  glass: { wallpaperBlur: 4, wallpaperDim: 30, cardOpacity: 75, cardBlur: 12 },
  customCss: '.glass-card { border-radius: 0 }',
}

const parse = (file: unknown) => parseThemeFile(JSON.stringify(file))

describe('theme file', () => {
  it('round-trips an exported theme into the change that restores it', () => {
    const file = exportTheme(theme)
    expect(file.nimbus_theme).toBe(1)
    expect(parse(file)).toEqual({
      update: {
        theme_preset: 'nord',
        theme_mode: 'dark',
        theme_accent_color: '#EC4899',
        theme_font: 'inter',
        kpi_mono: true,
        icon_source: 'selfhst',
        wallpaper_blur: 4,
        wallpaper_dim: 30,
        card_opacity: 75,
        card_blur: 12,
        custom_css: '.glass-card { border-radius: 0 }',
      },
    })
  })

  it('exports no accent as null, so importing clears one', () => {
    const file = exportTheme({ ...theme, accentColor: undefined })
    expect(file.theme_accent_color).toBeNull()
    expect(parse(file)).toMatchObject({ update: { theme_accent_color: null } })
  })

  it('never carries the wallpaper or other preferences', () => {
    expect(exportTheme(theme)).not.toHaveProperty('theme_background')
    const parsed = parse({
      nimbus_theme: 1,
      theme_font: 'system',
      theme_background: 'https://evil.example/track.png',
      layout_mode: 'classic',
      status_strip: { enabled: true, chips: [] },
    })
    expect(parsed).toEqual({ update: { theme_font: 'system' } })
  })

  it('puts a preset in its own mode whatever the file says', () => {
    expect(parse({ nimbus_theme: 1, theme_preset: 'nord', theme_mode: 'light' })).toEqual({
      update: { theme_preset: 'nord', theme_mode: 'dark' },
    })
  })

  it('refuses what is not a theme file of this version', () => {
    expect(parseThemeFile('{oops')).toEqual({ error: 'This is not a JSON file' })
    expect(parse(null)).toHaveProperty('error')
    expect(parse([1])).toHaveProperty('error')
    expect(parse({ theme_preset: 'nord' })).toHaveProperty('error')
    expect(parse({ nimbus_theme: 2 })).toEqual({
      error: 'This theme file was made by a newer version of Nimbus',
    })
  })

  it.each([
    ['theme_preset', 'solarized'],
    ['theme_mode', 'sepia'],
    ['theme_accent_color', 'red; background: url(x)'],
    ['theme_font', 'comic-sans'],
    ['kpi_mono', 'yes'],
    ['icon_source', 'https://evil.example'],
    ['card_blur', '12px'],
    ['custom_css', { toString: 'x' }],
  ])('refuses an invalid %s', (field, value) => {
    expect(parse({ nimbus_theme: 1, [field]: value })).toEqual({
      error: `The theme file has an invalid ${field}`,
    })
  })
})
