import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import '@testing-library/jest-dom'
import GroupForm from '@/components/GroupForm'
import ServiceChip from '@/components/ServiceChip'
import ServicesGrid from '@/components/ServicesGrid'
import { mergeTiles, tileStyleOf } from '@/lib/tiles'
import type { Group, Service } from '@/types'
import { makeWidget } from './widgets/fixtures'

const service = (overrides: Partial<Service> = {}): Service => ({
  id: 's1',
  name: 'Plex',
  url: 'https://plex.test',
  icon: '🎬',
  icon_type: 'emoji',
  status: 'online',
  response_time: 42,
  position: 0,
  card_size: '2x1',
  monitoring_enabled: true,
  created_at: '2026-01-01T00:00:00Z',
  ...overrides,
})

const group = (overrides: Partial<Group> = {}): Group => ({
  id: 'g1',
  name: 'Links',
  color: '#ff0000',
  position: 0,
  is_default: false,
  monitoring_enabled: true,
  created_at: '2026-01-01T00:00:00Z',
  ...overrides,
})

describe('ServiceChip', () => {
  it('is a link with icon, name and a status dot from the status utils', () => {
    const { container } = render(<ServiceChip service={service()} openInNewTab />)
    const link = screen.getByRole('link', { name: /Plex/ })
    expect(link).toHaveAttribute('href', 'https://plex.test')
    expect(link).toHaveAttribute('target', '_blank')
    expect(link).toHaveAttribute('rel', 'noopener noreferrer')
    expect(screen.getByText('🎬')).toBeInTheDocument()
    expect(container.querySelector('.bg-success')).toBeInTheDocument()
    expect(screen.getByText('online')).toHaveClass('sr-only')
  })

  it('opens in the same tab when asked to', () => {
    render(<ServiceChip service={service()} openInNewTab={false} />)
    const link = screen.getByRole('link', { name: /Plex/ })
    expect(link).toHaveAttribute('target', '_self')
    expect(link).not.toHaveAttribute('rel')
  })

  it('shows an offline service in the error colour', () => {
    const { container } = render(
      <ServiceChip service={service({ status: 'offline' })} openInNewTab />
    )
    expect(container.querySelector('.bg-error')).toBeInTheDocument()
    expect(screen.getByText('offline')).toBeInTheDocument()
  })

  it('has a muted dot when the service is not monitored', () => {
    const { container } = render(
      <ServiceChip service={service()} openInNewTab isMonitored={false} />
    )
    expect(container.querySelector('.bg-success')).not.toBeInTheDocument()
    expect(container.querySelector('.bg-gray-400')).toBeInTheDocument()
    expect(screen.getByText('not monitored')).toBeInTheDocument()
  })

  it('follows the card scale for density', () => {
    const { rerender } = render(<ServiceChip service={service()} openInNewTab cardScale="small" />)
    expect(screen.getByRole('link')).toHaveClass('text-xs')
    rerender(<ServiceChip service={service()} openInNewTab cardScale="large" />)
    expect(screen.getByRole('link')).toHaveClass('text-base')
  })

  it('is not a link in edit mode and can be edited', () => {
    const onEdit = vi.fn()
    render(<ServiceChip service={service()} openInNewTab isEditMode onEdit={onEdit} />)
    expect(screen.queryByRole('link')).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Edit Plex' }))
    expect(onEdit).toHaveBeenCalledWith(service())
  })
})

describe('chips in ServicesGrid', () => {
  const tiles = mergeTiles(
    [service(), service({ id: 's2', name: 'Sonarr', position: 1 })],
    [makeWidget({ position: 2, config: { content: 'Remember the milk' } })]
  )
  const props = { openInNewTab: false, enableCardResizing: true, cardScale: 'medium' as const }

  it('renders services as chips and keeps widgets as cards', () => {
    render(<ServicesGrid {...props} tiles={tiles} tileStyle="chips" viewMode="grid" />)
    const chip = screen.getByRole('link', { name: /Sonarr/ })
    expect(chip).toHaveClass('rounded-full')
    expect(chip.parentElement).toHaveClass('flex-wrap')
    expect(screen.getByText('Remember the milk')).toBeInTheDocument()
  })

  it('keeps chips in list view, where widgets are left out', () => {
    render(<ServicesGrid {...props} tiles={tiles} tileStyle="chips" viewMode="list" />)
    expect(screen.getByRole('link', { name: /Plex/ })).toHaveClass('rounded-full')
    expect(screen.queryByText('Remember the milk')).not.toBeInTheDocument()
    expect(screen.getByText(/1 widget is only shown in grid view/)).toBeInTheDocument()
  })

  it('renders cards by default', () => {
    render(<ServicesGrid {...props} tiles={tiles} viewMode="grid" />)
    expect(screen.getByRole('link', { name: /Plex/ })).not.toHaveClass('rounded-full')
  })
})

describe('tileStyleOf', () => {
  const groups = [
    group({ id: 'home', is_default: true }),
    group({ id: 'links', tile_style: 'chips' }),
  ]
  const [ungrouped, linked] = mergeTiles(
    [service(), service({ id: 's2', position: 1, group_id: 'links' })],
    []
  )

  it('is the style of the group of the tile', () => {
    expect(tileStyleOf(linked, groups)).toBe('chips')
  })

  it('uses the default group for ungrouped tiles, and cards when unset', () => {
    expect(tileStyleOf(ungrouped, groups)).toBe('cards')
    expect(tileStyleOf(ungrouped, [{ ...groups[0], tile_style: 'chips' }])).toBe('chips')
    expect(tileStyleOf(ungrouped, [])).toBe('cards')
  })
})

describe('GroupForm tile style', () => {
  it('starts on cards and submits the chosen style', async () => {
    const onSubmit = vi.fn(() => Promise.resolve())
    render(<GroupForm onSubmit={onSubmit} onClose={vi.fn()} />)
    expect(screen.getByRole('radio', { name: /Cards/ })).toBeChecked()

    fireEvent.change(screen.getByLabelText('Name'), { target: { value: 'Quick launch' } })
    fireEvent.click(screen.getByRole('radio', { name: /Chips/ }))
    fireEvent.click(screen.getByRole('button', { name: 'Create Group' }))
    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({ name: 'Quick launch', tile_style: 'chips' })
    )
  })

  it('shows the style of the group being edited', () => {
    render(
      <GroupForm group={group({ tile_style: 'chips' })} onSubmit={vi.fn()} onClose={vi.fn()} />
    )
    expect(screen.getByRole('radio', { name: /Chips/ })).toBeChecked()
  })
})
