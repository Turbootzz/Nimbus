import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import '@testing-library/jest-dom'
import WallpaperSettings from '@/components/WallpaperSettings'
import { defaultGlass } from '@/lib/wallpaper'

const theme = {
  background: undefined as string | undefined,
  setBackground: vi.fn(),
  glass: defaultGlass,
  setGlass: vi.fn(),
  uploadWallpaper: vi.fn(),
}
vi.mock('@/contexts/ThemeContext', () => ({ useTheme: () => theme }))

describe('WallpaperSettings', () => {
  it('saves a typed URL when leaving the field, not on every key', () => {
    render(<WallpaperSettings />)
    const input = screen.getByLabelText('Wallpaper URL')
    fireEvent.change(input, { target: { value: 'https://a.test/bg.jpg' } })
    expect(theme.setBackground).not.toHaveBeenCalled()
    fireEvent.blur(input)
    expect(theme.setBackground).toHaveBeenCalledWith('https://a.test/bg.jpg')
  })

  it('shows upload errors and an uploaded image as such', async () => {
    theme.uploadWallpaper.mockResolvedValue('File size exceeds maximum allowed size of 8 MB')
    theme.background = '/uploads/wallpapers/abc.jpg'
    render(<WallpaperSettings />)
    expect(screen.getByLabelText('Wallpaper URL')).toHaveValue('')
    expect(screen.getByPlaceholderText('Uploaded image')).toBeInTheDocument()

    fireEvent.change(screen.getByLabelText('Wallpaper file'), {
      target: { files: [new File(['x'], 'big.jpg', { type: 'image/jpeg' })] },
    })
    expect(await screen.findByRole('alert')).toHaveTextContent('8 MB')
  })

  it('keeps an uploaded wallpaper when the URL field is emptied', () => {
    theme.background = '/uploads/wallpapers/abc.jpg'
    theme.setBackground.mockClear()
    render(<WallpaperSettings />)
    const input = screen.getByLabelText('Wallpaper URL')
    fireEvent.change(input, { target: { value: 'https://a' } })
    fireEvent.change(input, { target: { value: '' } })
    fireEvent.blur(input)
    expect(theme.setBackground).not.toHaveBeenCalled()
  })

  it('refuses images over 8 MB before uploading', async () => {
    theme.uploadWallpaper.mockClear()
    render(<WallpaperSettings />)
    const big = new File(['x'], 'big.jpg', { type: 'image/jpeg' })
    Object.defineProperty(big, 'size', { value: 9 * 1024 * 1024 })
    fireEvent.change(screen.getByLabelText('Wallpaper file'), { target: { files: [big] } })
    expect(await screen.findByRole('alert')).toHaveTextContent('larger than 8 MB')
    expect(theme.uploadWallpaper).not.toHaveBeenCalled()
  })

  it('changes the glass settings with the sliders', () => {
    render(<WallpaperSettings />)
    fireEvent.change(screen.getByLabelText(/Card opacity/), { target: { value: '70' } })
    expect(theme.setGlass).toHaveBeenCalledWith({ cardOpacity: 70 })
  })
})
