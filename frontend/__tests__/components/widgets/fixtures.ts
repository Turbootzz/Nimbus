import type { Snapshot, WeatherPayload, Widget, WidgetTypeMeta } from '@/types'

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
    allowed_sizes: ['1x1', '2x1', '1x2', '2x2'],
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
    allowed_sizes: ['1x2', '2x2'],
    static: true,
  },
  {
    type: 'markdown',
    name: 'Note',
    category: 'general',
    default_size: '2x2',
    allowed_sizes: ['1x1', '2x1', '1x2', '2x2'],
    static: true,
  },
  {
    type: 'weather',
    name: 'Weather',
    category: 'info',
    default_size: '2x1',
    allowed_sizes: ['1x1', '2x1', '2x2'],
    static: false,
    min_refresh_seconds: 600,
  },
]

export const weatherPayload: WeatherPayload = {
  temperature: 16.6,
  feels_like: 15.2,
  weather_code: 61,
  wind_speed: 7.4,
  is_day: true,
  temperature_unit: '°C',
  wind_unit: 'km/h',
  daily: [
    { date: '2026-09-27', weather_code: 3, max: 20.8, min: 13.7 },
    { date: '2026-09-28', weather_code: 53, max: 20.1, min: 16.2 },
    { date: '2026-09-29', weather_code: 0, max: 24.7, min: 15.6 },
    { date: '2026-09-30', weather_code: 95, max: 23.7, min: 18 },
  ],
}

export const makeSnapshot = (overrides: Partial<Snapshot> = {}): Snapshot => ({
  source_kind: 'widget',
  source_id: 'w1',
  payload: weatherPayload,
  fetched_at: '2026-09-27T12:00:00Z',
  stale: false,
  ...overrides,
})
