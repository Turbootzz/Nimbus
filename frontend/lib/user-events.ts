import type { User } from '@/types'

// Tells components that show the current user (the user menu) that it changed,
// e.g. after a new avatar, so they update without a reload
const USER_UPDATED = 'nimbus:user-updated'

export function announceUser(user: User) {
  window.dispatchEvent(new CustomEvent<User>(USER_UPDATED, { detail: user }))
}

export function onUserUpdated(listener: (user: User) => void): () => void {
  const handler = (event: Event) => listener((event as CustomEvent<User>).detail)
  window.addEventListener(USER_UPDATED, handler)
  return () => window.removeEventListener(USER_UPDATED, handler)
}
