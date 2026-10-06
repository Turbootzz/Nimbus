import { describe, expect, it } from 'vitest'
import {
  defaultGlass,
  glassVariables,
  isUploadedWallpaper,
  parseGlass,
  withContrast,
} from '@/lib/wallpaper'

describe('wallpaper helpers', () => {
  it('reads a cached copy and falls back on broken values', () => {
    expect(parseGlass(null)).toBeNull()
    expect(parseGlass('{oops')).toBeNull()
    expect(parseGlass('{"wallpaperBlur":4,"cardOpacity":"1px; x"}')).toEqual({
      ...defaultGlass,
      wallpaperBlur: 4,
    })
  })

  it('only dims and blurs when there is something to dim or blur', () => {
    expect(glassVariables(defaultGlass, null)).toEqual({
      '--wallpaper-image': 'none',
      '--wallpaper-blur': '0px',
      '--wallpaper-dim': '0',
      '--card-opacity': '100%',
      '--card-backdrop': 'none',
    })
    const vars = glassVariables({ ...defaultGlass, wallpaperDim: 45 }, 'https://a.test/bg.jpg')
    expect(vars['--wallpaper-dim']).toBe('0.45')
  })

  it('raises the dim only for see-through cards', () => {
    expect(withContrast({ ...defaultGlass, cardOpacity: 59 }).wallpaperDim).toBe(30)
    expect(withContrast({ ...defaultGlass, cardOpacity: 59, wallpaperDim: 50 }).wallpaperDim).toBe(
      50
    )
    expect(withContrast({ ...defaultGlass, cardOpacity: 60 }).wallpaperDim).toBe(0)
  })

  it('tells uploads from URLs', () => {
    expect(isUploadedWallpaper('/uploads/wallpapers/a.png')).toBe(true)
    expect(isUploadedWallpaper('https://a.test/uploads/wallpapers/a.png')).toBe(false)
    expect(isUploadedWallpaper(undefined)).toBe(false)
  })
})
