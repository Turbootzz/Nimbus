import { act, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, it, expect, vi } from 'vitest'
import '@testing-library/jest-dom'
import ClockWidget from '@/components/widgets/renderers/ClockWidget'
import NoteWidget from '@/components/widgets/renderers/NoteWidget'
import BookmarksWidget from '@/components/widgets/renderers/BookmarksWidget'
import EmbedWidget from '@/components/widgets/renderers/EmbedWidget'
import { makeWidget } from './fixtures'

const base = { widget: makeWidget(), cardSize: '2x1' as const, openInNewTab: false }

describe('ClockWidget', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-03-01T12:34:56Z'))
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  it('shows the time in the configured timezone and ticks', () => {
    render(
      <ClockWidget
        {...base}
        config={{ timezone: 'UTC', hour12: false, show_seconds: true, date_format: 'none' }}
      />
    )
    expect(screen.getByText(/12:34:56/)).toBeInTheDocument()
    expect(screen.getByText('UTC')).toBeInTheDocument()

    act(() => {
      vi.advanceTimersByTime(1000)
    })
    expect(screen.getByText(/12:34:57/)).toBeInTheDocument()
  })

  it('shows the place name of the zone', () => {
    render(
      <ClockWidget
        {...base}
        config={{
          timezone: 'America/New_York',
          hour12: true,
          show_seconds: false,
          date_format: 'long',
        }}
      />
    )
    expect(screen.getByText('New York')).toBeInTheDocument()
    expect(screen.getByText(/2026/)).toBeInTheDocument()
  })

  it('reports a timezone the browser does not know', () => {
    render(
      <ClockWidget
        {...base}
        config={{
          timezone: 'Mars/Olympus',
          hour12: false,
          show_seconds: false,
          date_format: 'short',
        }}
      />
    )
    expect(screen.getByText(/Unknown timezone/)).toBeInTheDocument()
  })
})

describe('NoteWidget', () => {
  it('renders markdown', () => {
    render(<NoteWidget {...base} config={{ content: '# Title\n\n- one\n- two' }} />)
    expect(screen.getByRole('heading', { name: 'Title' })).toBeInTheDocument()
    expect(screen.getAllByRole('listitem')).toHaveLength(2)
  })

  it('never renders raw HTML or script links', () => {
    const { container } = render(
      <NoteWidget
        {...base}
        config={{
          content:
            '<img src=x onerror="alert(1)"><script>alert(2)</script>\n\n[click](javascript:alert(3))',
        }}
      />
    )
    expect(container.querySelector('img')).toBeNull()
    expect(container.querySelector('script')).toBeNull()
    const link = screen.getByText('click').closest('a')
    expect(link?.getAttribute('href') ?? '').not.toContain('javascript')
  })

  it('shows a placeholder when empty', () => {
    render(<NoteWidget {...base} config={{ content: '  ' }} />)
    expect(screen.getByText('Empty note')).toBeInTheDocument()
  })
})

describe('BookmarksWidget', () => {
  const items = [
    { name: 'Router', url: 'http://192.168.1.1', icon: '📡' },
    { name: 'Docs', url: 'https://docs.test', icon: '' },
  ]

  it('renders links that follow the new tab preference', () => {
    render(<BookmarksWidget {...base} openInNewTab config={{ items }} />)
    const router = screen.getByText('Router').closest('a')
    expect(router).toHaveAttribute('href', 'http://192.168.1.1')
    expect(router).toHaveAttribute('target', '_blank')
    expect(router).toHaveAttribute('rel', 'noopener noreferrer')
    expect(screen.getByText('📡')).toBeInTheDocument()
    expect(screen.getByText('D')).toBeInTheDocument()
  })

  it('opens in the same tab by default', () => {
    render(<BookmarksWidget {...base} config={{ items }} />)
    expect(screen.getByText('Docs').closest('a')).not.toHaveAttribute('target')
  })

  it('shows a placeholder when empty', () => {
    render(<BookmarksWidget {...base} config={{ items: [] }} />)
    expect(screen.getByText('No bookmarks yet')).toBeInTheDocument()
  })
})

describe('EmbedWidget', () => {
  it('renders a sandboxed iframe', () => {
    const { container } = render(
      <EmbedWidget {...base} config={{ url: 'https://grafana.test/d/1', height: 400 }} />
    )
    const iframe = container.querySelector('iframe')
    expect(iframe).toHaveAttribute('src', 'https://grafana.test/d/1')
    expect(iframe?.getAttribute('sandbox')).toContain('allow-scripts')
    expect(iframe?.getAttribute('sandbox')).not.toContain('allow-top-navigation')
    expect(iframe).toHaveStyle({ height: '400px' })
  })

  it('shows a placeholder without a URL', () => {
    render(<EmbedWidget {...base} config={{ url: '', height: 300 }} />)
    expect(screen.getByText('No URL set')).toBeInTheDocument()
  })
})
