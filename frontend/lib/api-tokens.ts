import type { ApiToken, ApiTokenScope } from '@/types'

// In the order the API returns them; labels are in the apiTokens.scopes messages
export const SCOPES: ApiTokenScope[] = [
  'read:services',
  'read:metrics',
  'read:widgets',
  'write:services',
  'mcp',
]

export const DEFAULT_SCOPES: ApiTokenScope[] = ['read:services', 'read:metrics', 'read:widgets']

// Days until a new token expires; 0 means never
export const EXPIRY_DAYS = [0, 7, 30, 90, 365]

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
  return SCOPES.filter((s) => next.has(s))
}

export function expiryFromDays(days: number, now = new Date()): string | undefined {
  if (days <= 0) return undefined
  return new Date(now.getTime() + days * 24 * 60 * 60 * 1000).toISOString()
}

export function isExpired(token: Pick<ApiToken, 'expires_at'>, now = new Date()): boolean {
  return token.expires_at !== null && new Date(token.expires_at).getTime() <= now.getTime()
}
