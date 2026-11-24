import type { IconSource, PreferencesUpdateRequest, ThemeFont, ThemePreset } from '@/types'

type Mode = 'light' | 'dark' | 'auto'

// The colours of each preset are in styles/themes.css; a preset belongs to
// one mode. The backend validates the same ids (models/preferences.go).
export const themePresets: { id: ThemePreset; name: string; mode?: 'light' | 'dark' }[] = [
  { id: 'default', name: 'Default' },
  { id: 'catppuccin-mocha', name: 'Catppuccin Mocha', mode: 'dark' },
  { id: 'catppuccin-latte', name: 'Catppuccin Latte', mode: 'light' },
  { id: 'nord', name: 'Nord', mode: 'dark' },
  { id: 'dracula', name: 'Dracula', mode: 'dark' },
  { id: 'gruvbox-dark', name: 'Gruvbox Dark', mode: 'dark' },
  { id: 'gruvbox-light', name: 'Gruvbox Light', mode: 'light' },
  { id: 'tokyo-night', name: 'Tokyo Night', mode: 'dark' },
]

export const themeFonts: { id: ThemeFont; name: string }[] = [
  { id: 'default', name: 'Default' },
  { id: 'system', name: 'System' },
  { id: 'inter', name: 'Inter' },
  { id: 'jetbrains-mono', name: 'JetBrains Mono' },
]

export const iconSources: { id: IconSource; name: string }[] = [
  { id: 'dashboard-icons-svg', name: 'Dashboard Icons (SVG)' },
  { id: 'dashboard-icons-png', name: 'Dashboard Icons (PNG)' },
  { id: 'dashboard-icons-webp', name: 'Dashboard Icons (WebP)' },
  { id: 'selfhst', name: 'selfh.st icons (SVG)' },
]

const oneOf = <T extends string>(list: { id: T }[], raw: unknown, fallback: T): T =>
  list.find((item) => item.id === raw)?.id ?? fallback

// A stored or imported value, or the default when it isn't one we know
export const parsePreset = (raw: unknown) => oneOf(themePresets, raw, 'default')
export const parseFont = (raw: unknown) => oneOf(themeFonts, raw, 'default')
export const parseIconSource = (raw: unknown) => oneOf(iconSources, raw, 'dashboard-icons-svg')

// presetChange is what picking a preset saves. A preset brings its own mode
// and accent, so the mode follows it and a custom accent is dropped; the
// user can pick a new accent on top. Default keeps the mode and only drops
// the preset, which is the theme as it was before presets.
export function presetChange(preset: ThemePreset): PreferencesUpdateRequest {
  const mode = themePresets.find((p) => p.id === preset)?.mode
  if (!mode) return { theme_preset: 'default' }
  return { theme_preset: preset, theme_mode: mode, theme_accent_color: null }
}

// presetForMode is the preset left after the user picks a mode: a preset
// only exists in its own mode, so any other mode goes back to default
export function presetForMode(preset: ThemePreset, mode: Mode): ThemePreset {
  return themePresets.find((p) => p.id === preset)?.mode === mode ? preset : 'default'
}

// applyThemeAttributes sets what styles/themes.css reads on <html>.
// ThemeScript does the same before React, from the cached values.
export function applyThemeAttributes(
  root: HTMLElement,
  theme: { preset: ThemePreset; font: ThemeFont; kpiMono: boolean }
) {
  if (theme.preset === 'default') root.removeAttribute('data-preset')
  else root.setAttribute('data-preset', theme.preset)
  if (theme.font === 'default') root.removeAttribute('data-font')
  else root.setAttribute('data-font', theme.font)
  root.toggleAttribute('data-kpi-mono', theme.kpiMono)
}
