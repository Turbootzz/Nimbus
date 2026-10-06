import { uploadUrl } from '@/lib/utils/api-url'

// Wallpaper and glass card settings, in the units of the preferences API
export interface Glass {
  wallpaperBlur: number // px
  wallpaperDim: number // %
  cardOpacity: number // %
  cardBlur: number // px
}

export const defaultGlass: Glass = {
  wallpaperBlur: 0,
  wallpaperDim: 0,
  cardOpacity: 100,
  cardBlur: 0,
}

// Matches MaxWallpaperSize in the backend
export const MAX_WALLPAPER_BYTES = 8 * 1024 * 1024

// Translucent cards need a dimmed wallpaper to stay readable
export const MIN_DIM_FOR_GLASS = 30
const GLASS_OPACITY_THRESHOLD = 60

// withContrast raises the dim when cards are see-through enough to need it
export function withContrast(glass: Glass): Glass {
  if (glass.cardOpacity < GLASS_OPACITY_THRESHOLD && glass.wallpaperDim < MIN_DIM_FOR_GLASS) {
    return { ...glass, wallpaperDim: MIN_DIM_FOR_GLASS }
  }
  return glass
}

// wallpaperUrl turns a theme_background into an absolute http(s) URL, or
// null. Uploaded wallpapers are a path on the API server.
export function wallpaperUrl(background: string | undefined): string | null {
  if (!background) return null
  try {
    const url = new URL(uploadUrl(background), window.location.href)
    return url.protocol === 'http:' || url.protocol === 'https:' ? url.href : null
  } catch {
    return null
  }
}

// isUploadedWallpaper tells an uploaded image from a URL the user typed
export function isUploadedWallpaper(background: string | undefined): boolean {
  return !!background?.startsWith('/uploads/wallpapers/')
}

// glassVariables are the CSS custom properties globals.css reads. The dim
// only applies on top of a wallpaper; the card backdrop filter is left out
// at 0, since any backdrop-filter changes how fixed children are placed.
export function glassVariables(glass: Glass, imageUrl: string | null): Record<string, string> {
  return {
    '--wallpaper-image': imageUrl ? `url("${imageUrl}")` : 'none',
    '--wallpaper-blur': `${glass.wallpaperBlur}px`,
    '--wallpaper-dim': imageUrl ? String(glass.wallpaperDim / 100) : '0',
    '--card-opacity': `${glass.cardOpacity}%`,
    '--card-backdrop': glass.cardBlur > 0 ? `blur(${glass.cardBlur}px)` : 'none',
  }
}

interface GlassPreferences {
  wallpaper_blur?: number
  wallpaper_dim?: number
  card_opacity?: number
  card_blur?: number
}

export function glassFromPreferences(p: GlassPreferences): Glass {
  return {
    wallpaperBlur: p.wallpaper_blur ?? defaultGlass.wallpaperBlur,
    wallpaperDim: p.wallpaper_dim ?? defaultGlass.wallpaperDim,
    cardOpacity: p.card_opacity ?? defaultGlass.cardOpacity,
    cardBlur: p.card_blur ?? defaultGlass.cardBlur,
  }
}

export function glassToPreferences(glass: Glass): Required<GlassPreferences> {
  return {
    wallpaper_blur: glass.wallpaperBlur,
    wallpaper_dim: glass.wallpaperDim,
    card_opacity: glass.cardOpacity,
    card_blur: glass.cardBlur,
  }
}

// parseGlass reads the copy cached in localStorage; null when missing or broken
export function parseGlass(raw: string | null): Glass | null {
  if (!raw) return null
  try {
    const cached = JSON.parse(raw) as Record<keyof Glass, unknown>
    const number = (value: unknown, fallback: number) =>
      typeof value === 'number' && Number.isFinite(value) ? value : fallback
    return {
      wallpaperBlur: number(cached.wallpaperBlur, defaultGlass.wallpaperBlur),
      wallpaperDim: number(cached.wallpaperDim, defaultGlass.wallpaperDim),
      cardOpacity: number(cached.cardOpacity, defaultGlass.cardOpacity),
      cardBlur: number(cached.cardBlur, defaultGlass.cardBlur),
    }
  } catch {
    return null
  }
}
