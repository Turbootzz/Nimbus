import { describe, it, expect, vi } from 'vitest'
import { announceUser, onUserUpdated } from '@/lib/user-events'
import type { User } from '@/types'

describe('user events', () => {
  it('hands an announced user to listeners until they stop listening', () => {
    const listener = vi.fn()
    const stop = onUserUpdated(listener)
    const user = { id: 'u1', avatar_url: '/uploads/avatars/a.png' } as User

    announceUser(user)
    expect(listener).toHaveBeenCalledWith(user)

    stop()
    announceUser(user)
    expect(listener).toHaveBeenCalledTimes(1)
  })
})
