import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import '@testing-library/jest-dom'
import StatusPageIcon, { StatusPageImage } from '@/components/status/StatusPageIcon'

describe('StatusPageIcon', () => {
  it('falls back to the emoji for a broken URL, and tries a new URL again', () => {
    const icon = (url: string) => (
      <StatusPageIcon service={{ icon: '🎬', icon_type: 'image_url', icon_image_path: url }} />
    )
    const { container, rerender } = render(icon('https://a.example/icon.png'))
    fireEvent.error(container.querySelector('img')!)
    expect(container.querySelector('img')).toBeNull()
    expect(screen.getByText('🎬')).toBeInTheDocument()

    rerender(icon('https://b.example/icon.png'))
    expect(container.querySelector('img')).toHaveAttribute('src', 'https://b.example/icon.png')
  })
})

describe('StatusPageImage', () => {
  it('shows the fallback for a broken file, and tries a new file again', () => {
    const image = (file: string) => (
      <StatusPageImage file={file} alt="Logo" className="" fallback={<span>fallback</span>} />
    )
    const { rerender } = render(image('a.png'))
    fireEvent.error(screen.getByAltText('Logo'))
    expect(screen.getByText('fallback')).toBeInTheDocument()

    rerender(image('b.png'))
    expect(screen.getByAltText('Logo').getAttribute('src')).toMatch(
      /\/uploads\/service-icons\/b\.png$/
    )
  })
})
