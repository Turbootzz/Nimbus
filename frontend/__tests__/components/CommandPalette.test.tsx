import { act, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import '@testing-library/jest-dom'
import CommandPalette, { openCommandPalette } from '@/components/CommandPalette'
import { api } from '@/lib/api'
import type { Service } from '@/types'

const service = (id: string, name: string, url: string, group_id?: string) =>
  ({
    id,
    name,
    url,
    group_id,
    icon: '🔗',
    icon_type: 'emoji',
    status: 'online',
    position: 0,
    card_size: '1x1',
    monitoring_enabled: true,
    created_at: '',
  }) as Service

const services = [
  service('1', 'Sonarr', 'http://sonarr.lan', 'g1'),
  service('2', 'Plex', 'http://plex.lan', 'g1'),
  service('3', 'Router', 'http://192.168.1.1'),
  service('4', 'Sneaky', 'javascript://x/%0Aalert(1)'),
]
vi.mock('@/lib/api', () => ({
  api: { getServices: vi.fn(), getGroups: vi.fn() },
}))
const theme = { openInNewTab: true }
vi.mock('@/contexts/ThemeContext', () => ({ useTheme: () => theme }))

const press = (key: string, init: KeyboardEventInit = {}) =>
  act(async () => {
    window.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true, ...init }))
  })

describe('CommandPalette', () => {
  let clicked: string[]
  beforeEach(() => {
    theme.openInNewTab = true
    clicked = []
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (
      this: HTMLAnchorElement
    ) {
      clicked.push(`${this.getAttribute('href')} ${this.getAttribute('target') ?? 'same tab'}`)
    })
    vi.mocked(api.getServices).mockResolvedValue({ data: services })
    vi.mocked(api.getGroups).mockResolvedValue({ data: [{ id: 'g1', name: 'Media' }] as never })
  })
  afterEach(() => vi.restoreAllMocks())

  it('opens with /, filters, picks with the arrows and opens the link', async () => {
    render(<CommandPalette />)
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()

    await press('/')
    const input = await screen.findByRole('combobox')
    expect(input).toHaveFocus()
    expect(await screen.findAllByRole('option')).toHaveLength(3) // never the javascript: one

    fireEvent.change(input, { target: { value: 'media' } })
    expect(screen.getAllByRole('option').map((o) => o.textContent)).toEqual([
      expect.stringContaining('Plex'),
      expect.stringContaining('Sonarr'),
    ])

    fireEvent.keyDown(input, { key: 'ArrowDown' })
    expect(screen.getAllByRole('option')[1]).toHaveAttribute('aria-selected', 'true')
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(clicked).toEqual(['http://sonarr.lan _blank'])
  })

  it('opens in the same tab when that is the preference', async () => {
    theme.openInNewTab = false
    render(<CommandPalette />)
    await press('/')
    const input = await screen.findByRole('combobox')
    fireEvent.change(input, { target: { value: 'router' } })
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(clicked).toEqual(['http://192.168.1.1 same tab'])
  })

  it('opens with Ctrl+K and from a button, not with / while typing, closes with Esc', async () => {
    render(
      <>
        <input aria-label="other field" />
        <CommandPalette />
      </>
    )
    const other = screen.getByLabelText('other field')
    other.focus()
    await act(async () => {
      fireEvent.keyDown(other, { key: '/' })
    })
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()

    await press('k', { code: 'KeyK', ctrlKey: true })
    expect(await screen.findByRole('dialog')).toBeInTheDocument()
    fireEvent.keyDown(screen.getByRole('combobox'), { key: 'Escape' })
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(other).toHaveFocus() // back where it was

    await act(async () => openCommandPalette())
    expect(await screen.findByRole('dialog')).toBeInTheDocument()
  })

  it('shows loading, errors and no matches', async () => {
    let finish: (value: unknown) => void = () => {}
    vi.mocked(api.getServices).mockReturnValue(
      new Promise((resolve) => (finish = resolve)) as never
    )
    render(<CommandPalette />)
    await press('/')
    expect(await screen.findByText('Loading services...')).toBeInTheDocument()
    await act(async () => finish({ error: { message: 'offline' } }))
    expect(screen.getByText('Could not load your services: offline')).toBeInTheDocument()

    fireEvent.keyDown(screen.getByRole('combobox'), { key: 'Escape' })
    vi.mocked(api.getServices).mockResolvedValue({ data: services })
    await press('/')
    fireEvent.change(await screen.findByRole('combobox'), { target: { value: 'zzzz' } })
    expect(screen.getByText('No services match')).toBeInTheDocument()
  })
})
