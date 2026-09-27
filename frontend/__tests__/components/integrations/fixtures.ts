import type { Integration, IntegrationKindMeta } from '@/types'

export const kinds: IntegrationKindMeta[] = [
  {
    kind: 'radarr',
    name: 'Radarr',
    icon: 'radarr',
    default_port: 7878,
    auth_types: ['api_key'],
    kpis: [
      { key: 'wanted', label: 'Wanted' },
      { key: 'queued', label: 'Queued' },
      { key: 'movies', label: 'Movies' },
    ],
  },
  {
    kind: 'sonarr',
    name: 'Sonarr',
    icon: 'sonarr',
    default_port: 8989,
    auth_types: ['api_key'],
    kpis: [
      { key: 'wanted', label: 'Wanted' },
      { key: 'queued', label: 'Queued' },
      { key: 'series', label: 'Series' },
    ],
  },
  { kind: 'multi', name: 'Multi', auth_types: ['basic', 'token', 'none'] },
]

export const makeIntegration = (overrides: Partial<Integration> = {}): Integration => ({
  id: 'i1',
  kind: 'sonarr',
  name: 'Sonarr',
  base_url: 'http://192.168.1.10:8989',
  auth_type: 'api_key',
  has_credentials: true,
  verify_tls: true,
  options: {},
  refresh_seconds: 60,
  last_test_at: null,
  last_test_ok: null,
  last_error: null,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
  ...overrides,
})
