import type { ApiToken, ApiTokenScope } from '@/types'

export interface ScopeOption {
  scope: ApiTokenScope
  label: string
  description: string
}

// In the order the API returns them
export const SCOPE_OPTIONS: ScopeOption[] = [
  {
    scope: 'read:services',
    label: 'Read services',
    description: 'Services and groups',
  },
  {
    scope: 'read:metrics',
    label: 'Read metrics',
    description: 'Uptime, status history and the Prometheus export',
  },
  {
    scope: 'read:widgets',
    label: 'Read widgets',
    description: 'Widgets and dashboard data',
  },
  {
    scope: 'write:services',
    label: 'Write services',
    description: 'Create, change, delete and recheck services and groups',
  },
  {
    scope: 'mcp',
    label: 'MCP',
    description: 'Let an AI assistant read your dashboard through /mcp',
  },
]

export const DEFAULT_SCOPES: ApiTokenScope[] = ['read:services', 'read:metrics', 'read:widgets']

// Days until a new token expires; 0 means never
export const EXPIRY_OPTIONS = [
  { days: 0, label: 'Never' },
  { days: 7, label: '7 days' },
  { days: 30, label: '30 days' },
  { days: 90, label: '90 days' },
  { days: 365, label: '1 year' },
]

// Writing services always includes reading them, so the two move together
export function toggleScope(
  scopes: ApiTokenScope[],
  scope: ApiTokenScope,
  checked: boolean
): ApiTokenScope[] {
  const next = new Set(scopes)
  if (checked) {
    next.add(scope)
    if (scope === 'write:services') next.add('read:services')
  } else {
    next.delete(scope)
    if (scope === 'read:services') next.delete('write:services')
  }
  return SCOPE_OPTIONS.map((option) => option.scope).filter((s) => next.has(s))
}

export function expiryFromDays(days: number, now = new Date()): string | undefined {
  if (days <= 0) return undefined
  return new Date(now.getTime() + days * 24 * 60 * 60 * 1000).toISOString()
}

export function isExpired(token: Pick<ApiToken, 'expires_at'>, now = new Date()): boolean {
  return token.expires_at !== null && new Date(token.expires_at).getTime() <= now.getTime()
}
