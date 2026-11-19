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

describe('theme presets', () => {
  it('defines every colour token of globals.css in every preset', () => {
    const root = globals.slice(globals.indexOf(':root {'), globals.indexOf('/* Dark mode color'))
    const tokens = [...root.matchAll(/(--color-[a-z-]+):/g)].map((match) => match[1])
    expect(tokens).toHaveLength(17)

    for (const preset of themePresets.filter((p) => p.mode)) {
      const start = css.indexOf(
        `:root[data-preset='${preset.id}'],\n[data-preset='${preset.id}'] {`
      )
      expect(start, preset.id).toBeGreaterThan(-1)
      const block = css.slice(start, css.indexOf('}', start))
      for (const token of [...tokens, '--color-on-primary']) {
        expect(block, `${preset.id} ${token}`).toContain(`${token}:`)
      }
    }
    // No block without an entry in the list, and none for default
    const blocks = [...css.matchAll(/^:root\[data-preset='([a-z-]+)'\]/gm)].map((m) => m[1])
    expect(blocks).toEqual(themePresets.filter((p) => p.mode).map((p) => p.id))
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
