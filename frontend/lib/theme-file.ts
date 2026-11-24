import type { PreferencesUpdateRequest } from '@/types'
import { type Glass, glassToPreferences } from '@/lib/wallpaper'
import { iconSources, themeFonts, themePresets } from '@/lib/theme-presets'

// A theme as a file to share: everything that makes the look, without the
// wallpaper image. Importing saves it through the preferences API, so the
// server checks every field, and the custom CSS, like any other change.
export const THEME_FILE_VERSION = 1

export interface ThemeFile extends PreferencesUpdateRequest {
  nimbus_theme: typeof THEME_FILE_VERSION
}

export function exportTheme(theme: {
  preset: PreferencesUpdateRequest['theme_preset']
  mode: PreferencesUpdateRequest['theme_mode']
  accentColor: string | undefined
  font: PreferencesUpdateRequest['theme_font']
  kpiMono: boolean
  iconSource: PreferencesUpdateRequest['icon_source']
  glass: Glass
  customCss: string
}): ThemeFile {
  return {
    nimbus_theme: THEME_FILE_VERSION,
    theme_preset: theme.preset,
    theme_mode: theme.mode,
    theme_accent_color: theme.accentColor ?? null,
    theme_font: theme.font,
    kpi_mono: theme.kpiMono,
    icon_source: theme.iconSource,
    ...glassToPreferences(theme.glass),
    custom_css: theme.customCss,
  }
}

const ids = (list: { id: string }[]) => list.map((item) => item.id)
const oneOf = (allowed: string[]) => (value: unknown) =>
  typeof value === 'string' && allowed.includes(value)
const isNumber = (value: unknown) => typeof value === 'number' && Number.isInteger(value)

// The fields a theme file may hold and what each must look like. Anything
// else in the file is ignored, so a file can't change other preferences.
const fields: Record<string, (value: unknown) => boolean> = {
  theme_preset: oneOf(ids(themePresets)),
  theme_mode: oneOf(['light', 'dark', 'auto']),
  theme_accent_color: (value) =>
    value === null || (typeof value === 'string' && /^#[0-9a-f]{6}$/i.test(value)),
  theme_font: oneOf(ids(themeFonts)),
  kpi_mono: (value) => typeof value === 'boolean',
  icon_source: oneOf(ids(iconSources)),
  wallpaper_blur: isNumber,
  wallpaper_dim: isNumber,
  card_opacity: isNumber,
  card_blur: isNumber,
  custom_css: (value) => typeof value === 'string',
}

// Why a file is refused; the settings page passes them translated
export interface ThemeFileErrors {
  notJson: string
  notTheme: string
  newerVersion: string
  invalidField: (field: string) => string
}

const englishErrors: ThemeFileErrors = {
  notJson: 'This is not a JSON file',
  notTheme: 'This is not a Nimbus theme file',
  newerVersion: 'This theme file was made by a newer version of Nimbus',
  invalidField: (field) => `The theme file has an invalid ${field}`,
}

// parseThemeFile reads a theme file into the change it asks for, or says
// why it can't. A field the file leaves out stays as it is.
export function parseThemeFile(
  text: string,
  errors: ThemeFileErrors = englishErrors
): { update: PreferencesUpdateRequest } | { error: string } {
  let file: Record<string, unknown>
  try {
    file = JSON.parse(text)
  } catch {
    return { error: errors.notJson }
  }
  if (typeof file !== 'object' || file === null || typeof file.nimbus_theme !== 'number') {
    return { error: errors.notTheme }
  }
  if (file.nimbus_theme !== THEME_FILE_VERSION) {
    return { error: errors.newerVersion }
  }

  const update: Record<string, unknown> = {}
  for (const [name, valid] of Object.entries(fields)) {
    if (!(name in file)) continue
    if (!valid(file[name])) return { error: errors.invalidField(name) }
    update[name] = file[name]
  }
  // A preset only exists in its own mode
  const mode = themePresets.find((preset) => preset.id === update.theme_preset)?.mode
  if (mode) update.theme_mode = mode
  return { update: update as PreferencesUpdateRequest }
}
