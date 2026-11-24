import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'
import {
  applyThemeAttributes,
  parseFont,
  parsePreset,
  presetChange,
  presetForMode,
  themePresets,
} from '@/lib/theme-presets'

const css = readFileSync(join(process.cwd(), 'styles/themes.css'), 'utf8')
const globals = readFileSync(join(process.cwd(), 'styles/globals.css'), 'utf8')

const presetBlock = (id: string) => {
  const start = css.indexOf(`:root[data-preset='${id}'],\n[data-preset='${id}'] {`)
  return start === -1 ? '' : css.slice(start, css.indexOf('}', start))
}

// WCAG contrast ratio of two #rrggbb colours
function contrast(a: string, b: string) {
  const luminance = (hex: string) => {
    const [r, g, b] = [1, 3, 5].map((i) => {
      const c = parseInt(hex.slice(i, i + 2), 16) / 255
      return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4
    })
    return 0.2126 * r + 0.7152 * g + 0.0722 * b
  }
  const [light, dark] = [luminance(a), luminance(b)].sort((x, y) => y - x)
  return (light + 0.05) / (dark + 0.05)
}

describe('theme presets', () => {
  it('defines every colour token of globals.css in every preset', () => {
    const root = globals.slice(globals.indexOf(':root {'), globals.indexOf('/* Dark mode color'))
    const tokens = [...root.matchAll(/(--color-[a-z-]+):/g)].map((match) => match[1])
    expect(tokens).toHaveLength(17)

    for (const preset of themePresets.filter((p) => p.mode)) {
      const block = presetBlock(preset.id)
      expect(block, preset.id).not.toBe('')
      for (const token of [...tokens, '--color-on-primary']) {
        expect(block, `${preset.id} ${token}`).toContain(`${token}:`)
      }
    }
    // No block without an entry in the list, and none for default
    const blocks = [...css.matchAll(/^:root\[data-preset='([a-z-]+)'\]/gm)].map((m) => m[1])
    expect(blocks).toEqual(themePresets.filter((p) => p.mode).map((p) => p.id))
  })

  it('text on a status colour reads at 4.5:1 or more in every preset', () => {
    for (const status of ['success', 'warning', 'error', 'info']) {
      for (const preset of themePresets.filter((p) => p.mode)) {
        const block = presetBlock(preset.id)
        const token = (name: string) => block.match(new RegExp(`${name}: (#[0-9a-f]{6});`))?.[1]
        // The text colour the rule below resolves to
        const text = token(`--color-on-${status}`) ?? token('--color-background')
        const ratio = contrast(text!, token(`--color-${status}`)!)
        expect.soft(ratio, `${preset.id} ${status}`).toBeGreaterThanOrEqual(4.5)
      }
      expect(css).toContain(
        `[data-preset] .bg-${status}.text-white {\n  color: var(--color-on-${status}, var(--color-background));`
      )
    }
  })

  it('a preset brings its mode and accent; default only drops the preset', () => {
    expect(presetChange('nord')).toEqual({
      theme_preset: 'nord',
      theme_mode: 'dark',
      theme_accent_color: null,
    })
    expect(presetChange('catppuccin-latte').theme_mode).toBe('light')
    expect(presetChange('default')).toEqual({ theme_preset: 'default' })
  })

  it('a preset only survives its own mode', () => {
    expect(presetForMode('nord', 'dark')).toBe('nord')
    expect(presetForMode('nord', 'light')).toBe('default')
    expect(presetForMode('nord', 'auto')).toBe('default')
    expect(presetForMode('default', 'light')).toBe('default')
  })

  it('reads stored values and falls back on unknown ones', () => {
    expect(parsePreset('dracula')).toBe('dracula')
    expect(parsePreset('"><script>')).toBe('default')
    expect(parsePreset(null)).toBe('default')
    expect(parseFont('inter')).toBe('inter')
    expect(parseFont('comic-sans')).toBe('default')
  })

  it('sets the attributes the CSS reads, and none for the defaults', () => {
    const root = document.createElement('html')
    applyThemeAttributes(root, { preset: 'nord', font: 'inter', kpiMono: true })
    expect(root.getAttribute('data-preset')).toBe('nord')
    expect(root.getAttribute('data-font')).toBe('inter')
    expect(root.hasAttribute('data-kpi-mono')).toBe(true)

    applyThemeAttributes(root, { preset: 'default', font: 'default', kpiMono: false })
    expect(root.attributes).toHaveLength(0)
  })
})
