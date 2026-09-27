import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import '@testing-library/jest-dom'
import ServicesGrid from '@/components/ServicesGrid'
import { mergeTiles } from '@/lib/tiles'
import type { Service } from '@/types'
import { makeWidget } from './widgets/fixtures'

const service: Service = {
  id: 's1',
  name: 'Plex',
  url: 'https://plex.test',
  icon: '🎬',
  icon_type: 'emoji',
  status: 'online',
  position: 0,
  card_size: '2x1',
  monitoring_enabled: true,
  created_at: '2026-01-01T00:00:00Z',
}
const note = makeWidget({ position: 1, config: { content: 'Remember the milk' } })
const tiles = mergeTiles([service], [note])

const props = {
  openInNewTab: false,
  enableCardResizing: true,
  cardScale: 'medium' as const,
}

describe('ServicesGrid', () => {
  it('renders services and widgets in one grid', () => {
    render(<ServicesGrid {...props} tiles={tiles} viewMode="grid" />)
    expect(screen.getByText('Plex')).toBeInTheDocument()
    expect(screen.getByText('Remember the milk')).toBeInTheDocument()
  })

  it('shows only services in list view, with a hint about widgets', () => {
    render(<ServicesGrid {...props} tiles={tiles} viewMode="list" />)
    expect(screen.getByText('Plex')).toBeInTheDocument()
    expect(screen.queryByText('Remember the milk')).not.toBeInTheDocument()
    expect(screen.getByText(/1 widget is only shown in grid view/)).toBeInTheDocument()
  })
})
