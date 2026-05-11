import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import '@testing-library/jest-dom'
import IconSelector from '@/components/IconSelector'

const fetchServiceFaviconMock = vi.fn()

vi.mock('@/lib/api', () => ({
  api: {
    fetchServiceFavicon: (...args: unknown[]) => fetchServiceFaviconMock(...args),
  },
}))

// EmojiPicker pulls in a huge dataset; stub it out — these tests don't exercise it.
vi.mock('@/components/EmojiPicker', () => ({
  default: () => null,
}))

type Props = React.ComponentProps<typeof IconSelector>

function makeProps(overrides: Partial<Props> = {}): Props {
  return {
    icon: '🔗',
    iconType: 'emoji',
    iconImagePath: '',
    serviceUrl: '',
    onIconChange: vi.fn(),
    onIconTypeChange: vi.fn(),
    onIconImagePathChange: vi.fn(),
    onFileSelect: vi.fn(),
    ...overrides,
  }
}

describe('IconSelector — Fetch favicon button', () => {
  beforeEach(() => {
    fetchServiceFaviconMock.mockReset()
  })

  it('is disabled with an empty URL and exposes the hint via title', () => {
    render(<IconSelector {...makeProps({ serviceUrl: '' })} />)

    const button = screen.getByRole('button', { name: /fetch favicon from url/i })
    expect(button).toBeDisabled()
    expect(button).toHaveAttribute('title', 'Enter a URL first')
  })

  it('is disabled when the URL field contains junk', () => {
    render(<IconSelector {...makeProps({ serviceUrl: 'not a url' })} />)

    expect(screen.getByRole('button', { name: /fetch favicon from url/i })).toBeDisabled()
  })

  it('is enabled with a valid https URL', () => {
    render(<IconSelector {...makeProps({ serviceUrl: 'https://github.com' })} />)

    expect(screen.getByRole('button', { name: /fetch favicon from url/i })).toBeEnabled()
  })

  it('on success: calls API with the URL, switches to image_upload, sets path, clears file', async () => {
    fetchServiceFaviconMock.mockResolvedValueOnce({
      data: { icon_image_path: 'abc123.png', message: 'ok' },
    })

    const props = makeProps({ serviceUrl: 'https://github.com', iconType: 'emoji' })
    render(<IconSelector {...props} />)

    fireEvent.click(screen.getByRole('button', { name: /fetch favicon from url/i }))

    await waitFor(() => {
      expect(fetchServiceFaviconMock).toHaveBeenCalledWith('https://github.com')
    })
    expect(props.onIconTypeChange).toHaveBeenCalledWith('image_upload')
    expect(props.onIconImagePathChange).toHaveBeenCalledWith('abc123.png')
    expect(props.onFileSelect).toHaveBeenCalledWith(null)
  })

  it('on backend error: surfaces the message inline without changing icon state', async () => {
    fetchServiceFaviconMock.mockResolvedValueOnce({
      error: { message: 'Could not fetch favicon: 404' },
    })

    const props = makeProps({ serviceUrl: 'https://nope.invalid' })
    render(<IconSelector {...props} />)

    fireEvent.click(screen.getByRole('button', { name: /fetch favicon from url/i }))

    expect(await screen.findByRole('alert')).toHaveTextContent('Could not fetch favicon: 404')
    expect(props.onIconTypeChange).not.toHaveBeenCalled()
    expect(props.onIconImagePathChange).not.toHaveBeenCalled()
  })

  it('builds the preview src against the API base URL, not a relative path', () => {
    // Regression guard: the preview <img> used to point at "/api/v1/uploads/..." which the
    // Next.js dev server returns 404 for. It must use getApiUrl() so the browser hits the backend.
    render(
      <IconSelector
        {...makeProps({
          iconType: 'image_upload',
          iconImagePath: 'abc123.png',
        })}
      />
    )

    const img = screen.getByAltText('Icon preview') as HTMLImageElement
    // vitest.setup.ts sets NEXT_PUBLIC_API_URL=http://localhost:8080/api/v1
    expect(img.src).toBe('http://localhost:8080/api/v1/uploads/service-icons/abc123.png')
  })
})
