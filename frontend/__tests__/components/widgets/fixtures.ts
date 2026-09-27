import type { Widget, WidgetTypeMeta } from '@/types'

export const makeWidget = (overrides: Partial<Widget> = {}): Widget => ({
  id: 'w1',
  type: 'markdown',
  title: '',
  group_id: null,
  integration_id: null,
  config: {},
  card_size: '2x2',
  position: 0,
  refresh_seconds: 300,
  enabled: true,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
  ...overrides,
})

// Mirrors the static types the backend registers in internal/widgets
export const backendStaticTypes: WidgetTypeMeta[] = [
  {
    type: 'bookmarks',
    name: 'Bookmarks',
    category: 'general',
    default_size: '2x2',
    allowed_sizes: ['1x1', '2x1', '2x2'],
    static: true,
  },
  {
    type: 'clock',
    name: 'Clock',
    category: 'general',
    default_size: '2x1',
    allowed_sizes: ['1x1', '2x1', '2x2'],
    static: true,
  },
  {
    type: 'iframe',
    name: 'Embed',
    category: 'general',
    default_size: '2x2',
    allowed_sizes: ['1x1', '2x1', '2x2'],
    static: true,
  },
  {
    type: 'markdown',
    name: 'Note',
    category: 'general',
    default_size: '2x2',
    allowed_sizes: ['1x1', '2x1', '2x2'],
    static: true,
  },
]
