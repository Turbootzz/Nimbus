import { describe, it, expect } from 'vitest'
import { expiryFromDays, isExpired, toggleScope } from '@/lib/api-tokens'

describe('toggleScope', () => {
  it('adds and removes a scope, keeping the display order', () => {
    expect(toggleScope(['read:widgets'], 'read:services', true)).toEqual([
      'read:services',
      'read:widgets',
    ])
    expect(toggleScope(['read:services', 'mcp'], 'mcp', false)).toEqual(['read:services'])
  })

  it('adds read:services with write:services', () => {
    expect(toggleScope(['mcp'], 'write:services', true)).toEqual([
      'read:services',
      'write:services',
      'mcp',
    ])
  })

  it('drops write:services with read:services', () => {
    expect(toggleScope(['read:services', 'write:services', 'mcp'], 'read:services', false)).toEqual(
      ['mcp']
    )
  })

  it('does not duplicate a scope', () => {
    expect(toggleScope(['mcp'], 'mcp', true)).toEqual(['mcp'])
  })
})

describe('expiryFromDays', () => {
  const now = new Date('2026-11-18T12:00:00Z')

  it('returns nothing for a token that never expires', () => {
    expect(expiryFromDays(0, now)).toBeUndefined()
  })

  it('adds whole days', () => {
    expect(expiryFromDays(7, now)).toBe('2026-11-25T12:00:00.000Z')
  })
})

describe('isExpired', () => {
  const now = new Date('2026-11-18T12:00:00Z')

  it('is false without an expiry', () => {
    expect(isExpired({ expires_at: null }, now)).toBe(false)
  })

  it('compares the expiry with now', () => {
    expect(isExpired({ expires_at: '2026-11-18T11:59:59Z' }, now)).toBe(true)
    expect(isExpired({ expires_at: '2026-11-18T12:00:01Z' }, now)).toBe(false)
  })
})
