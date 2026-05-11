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
    serviceName: '',
    serviceUrl: '',
    onIconChange: vi.fn(),
    onIconTypeChange: vi.fn(),
    onIconImagePathChange: vi.fn(),
    onFileSelect: vi.fn(),
    ...overrides,
  }
}

describe('IconSelector — Auto-fetch icon button', () => {
  beforeEach(() => {
    fetchServiceFaviconMock.mockReset()
  })

  it('is disabled with both name and URL empty', () => {
    render(<IconSelector {...makeProps()} />)

    const button = screen.getByRole('button', { name: /auto-fetch icon/i })
    expect(button).toBeDisabled()
    expect(button).toHaveAttribute('title', 'Enter a service name or URL first')
  })

  it('is enabled when only a name is provided (homelab case)', () => {
    render(<IconSelector {...makeProps({ serviceName: 'Plex' })} />)

    expect(screen.getByRole('button', { name: /auto-fetch icon/i })).toBeEnabled()
  })

  it('is enabled when only a valid URL is provided', () => {
    render(<IconSelector {...makeProps({ serviceUrl: 'https://github.com' })} />)

    expect(screen.getByRole('button', { name: /auto-fetch icon/i })).toBeEnabled()
  })

  it('is still disabled when URL is junk and name is empty', () => {
    render(<IconSelector {...makeProps({ serviceUrl: 'not a url' })} />)

    expect(screen.getByRole('button', { name: /auto-fetch icon/i })).toBeDisabled()
  })

  it('sends both name and url to the API when both are present', async () => {
    fetchServiceFaviconMock.mockResolvedValueOnce({
      data: { icon_image_path: 'plex.svg', message: 'ok' },
    })

    const props = makeProps({
      serviceName: 'Plex',
      serviceUrl: 'https://plex.example.com',
      iconType: 'emoji',
    })
    render(<IconSelector {...props} />)

    fireEvent.click(screen.getByRole('button', { name: /auto-fetch icon/i }))

    await waitFor(() => {
      expect(fetchServiceFaviconMock).toHaveBeenCalledWith({
        name: 'Plex',
        url: 'https://plex.example.com',
      })
    })
    expect(props.onIconTypeChange).toHaveBeenCalledWith('image_upload')
    expect(props.onIconImagePathChange).toHaveBeenCalledWith('plex.svg')
    expect(props.onFileSelect).toHaveBeenCalledWith(null)
  })

  it('omits url when the URL field is invalid (still sends name)', async () => {
    fetchServiceFaviconMock.mockResolvedValueOnce({
      data: { icon_image_path: 'plex.svg', message: 'ok' },
    })

    render(<IconSelector {...makeProps({ serviceName: 'Plex', serviceUrl: 'garbage' })} />)

    fireEvent.click(screen.getByRole('button', { name: /auto-fetch icon/i }))

    await waitFor(() => {
      expect(fetchServiceFaviconMock).toHaveBeenCalledWith({
        name: 'Plex',
        url: undefined,
      })
    })
  })

  it('shows the backend error inline without mutating icon state on failure', async () => {
    fetchServiceFaviconMock.mockResolvedValueOnce({
      error: { message: 'Could not find an icon for that service' },
    })

    const props = makeProps({ serviceName: 'Plex' })
    render(<IconSelector {...props} />)

    fireEvent.click(screen.getByRole('button', { name: /auto-fetch icon/i }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Could not find an icon for that service'
    )
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
