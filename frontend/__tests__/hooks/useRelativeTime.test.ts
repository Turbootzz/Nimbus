import { renderHook } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { useRelativeTime } from '@/hooks/useRelativeTime'

describe('useRelativeTime', () => {
  it('words a time in its largest unit, without digit grouping', () => {
    const { result } = renderHook(() => useRelativeTime())
    const ago = (seconds: number) => new Date(Date.now() - seconds * 1000).toISOString()
    expect(result.current(ago(5))).toBe('just now')
    expect(result.current(ago(3 * 3600))).toBe('3h ago')
    expect(result.current(ago(1234 * 86400))).toBe('1234d ago')
  })
})
