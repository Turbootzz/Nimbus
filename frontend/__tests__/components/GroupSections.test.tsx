import { fireEvent, render, screen, within } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import '@testing-library/jest-dom'
import GroupSections from '@/components/GroupSections'
import { groupTilesBySection, mergeTiles } from '@/lib/tiles'
import type { Group, Service } from '@/types'
import { makeWidget } from './widgets/fixtures'

// happy-dom's localStorage has no clear()
const store: Record<string, string> = {}
Object.defineProperty(window, 'localStorage', {
  value: {
    getItem: (key: string) => store[key] ?? null,
    setItem: (key: string, value: string) => {
      store[key] = String(value)
    },
    removeItem: (key: string) => {
      delete store[key]
    },
  },
})

const group = (id: string, overrides: Partial<Group> = {}): Group => ({
  id,
  name: id,
  color: '#ff0000',
  position: 0,
  is_default: false,
  monitoring_enabled: true,
  created_at: '2026-01-01T00:00:00Z',
  ...overrides,
})

const service = (id: string, group_id?: string): Service => ({
  id,
  name: id,
  url: `https://${id}.test`,
  icon_type: 'emoji',
  status: 'online',
  position: 0,
  card_size: '2x1',
  group_id,
  monitoring_enabled: true,
  created_at: '2026-01-01T00:00:00Z',
})

const groups = [group('Core', { is_default: true }), group('Media'), group('Later')]
const sections = groupTilesBySection(
  mergeTiles(
    [service('Proxmox'), service('Plex', 'Media')],
    [makeWidget({ id: 'w1', group_id: 'Media' })]
  ),
  groups
)

const handlers = {
  onCreateGroup: vi.fn(),
  onEditGroup: vi.fn(),
  onDeleteGroup: vi.fn(),
  onMoveGroup: vi.fn(),
}

const renderSections = (isEditMode = false) =>
  render(
    <GroupSections sections={sections} isEditMode={isEditMode} isDraggingTile={false} {...handlers}>
      {(section) => (
        <ul>
          {section.tiles.map((tile) => (
            <li key={tile.id}>{tile.kind === 'service' ? tile.service.name : 'widget'}</li>
          ))}
        </ul>
      )}
    </GroupSections>
  )

describe('GroupSections', () => {
  beforeEach(() => {
    Object.keys(store).forEach((key) => delete store[key])
    vi.clearAllMocks()
  })

  it('shows every group as a section with its name, count and tiles', () => {
    renderSections()
    const media = screen.getByRole('region', { name: 'Media' })
    expect(within(media).getByRole('button', { name: /Media\s*2/ })).toBeInTheDocument()
    expect(within(media).getByText('Plex')).toBeInTheDocument()
    expect(within(media).getByText('widget')).toBeInTheDocument()
    // Ungrouped tiles show in the default group
    expect(within(screen.getByRole('region', { name: 'Core' })).getByText('Proxmox')).toBeVisible()
    expect(
      within(screen.getByRole('region', { name: 'Later' })).getByText('Nothing in this group yet.')
    ).toBeInTheDocument()
  })

  it('collapses a section and remembers it', () => {
    const { unmount } = renderSections()
    const toggle = screen.getByRole('button', { name: /Media/ })
    expect(toggle).toHaveAttribute('aria-expanded', 'true')

    fireEvent.click(toggle)
    expect(toggle).toHaveAttribute('aria-expanded', 'false')
    expect(screen.queryByText('Plex')).not.toBeInTheDocument()
    expect(screen.getByText('Proxmox')).toBeInTheDocument()

    unmount()
    renderSections()
    expect(screen.queryByText('Plex')).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: /Media/ }))
    expect(screen.getByText('Plex')).toBeInTheDocument()
  })

  it('survives a broken collapse cache', () => {
    store['nimbus-collapsed-sections'] = '{not json'
    renderSections()
    expect(screen.getByText('Plex')).toBeInTheDocument()
  })

  it('has no group controls outside edit mode', () => {
    renderSections()
    expect(screen.queryByRole('button', { name: 'Edit Media' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Add group' })).not.toBeInTheDocument()
  })

  it('manages groups in edit mode', () => {
    renderSections(true)
    fireEvent.click(screen.getByRole('button', { name: 'Edit Media' }))
    expect(handlers.onEditGroup).toHaveBeenCalledWith(groups[1])
    fireEvent.click(screen.getByRole('button', { name: 'Delete Media' }))
    expect(handlers.onDeleteGroup).toHaveBeenCalledWith(groups[1])
    fireEvent.click(screen.getByRole('button', { name: 'Move Media down' }))
    expect(handlers.onMoveGroup).toHaveBeenCalledWith(groups[1], 1)
    fireEvent.click(screen.getByRole('button', { name: 'Add group' }))
    expect(handlers.onCreateGroup).toHaveBeenCalled()

    // The default group stays, and the ends can't move further
    expect(screen.queryByRole('button', { name: 'Delete Core' })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Move Core up' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Move Later down' })).toBeDisabled()
  })
})
