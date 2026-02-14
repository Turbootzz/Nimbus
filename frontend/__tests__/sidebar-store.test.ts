import { describe, it, expect, beforeEach } from 'vitest'

// Mock localStorage (happy-dom's doesn't support .clear())
const localStorageMock = (() => {
  let store: Record<string, string> = {}

  return {
    getItem: (key: string) => store[key] || null,
    setItem: (key: string, value: string) => {
      store[key] = value.toString()
    },
    removeItem: (key: string) => {
      delete store[key]
    },
    clear: () => {
      store = {}
    },
  }
})()

Object.defineProperty(window, 'localStorage', {
  value: localStorageMock,
})

// Re-implement the store logic to test in isolation
// (mirrors layout.tsx module-level functions)
const sidebarListeners = new Set<() => void>()

function subscribeSidebar(onStoreChange: () => void) {
  sidebarListeners.add(onStoreChange)
  return () => {
    sidebarListeners.delete(onStoreChange)
  }
}

function getSidebarSnapshot() {
  return localStorage.getItem('nimbus-sidebar-collapsed') === 'true'
}

function getSidebarServerSnapshot() {
  return false
}

function setIsDesktopCollapsed(value: boolean) {
  localStorage.setItem('nimbus-sidebar-collapsed', String(value))
  sidebarListeners.forEach((l) => l())
}

describe('Sidebar collapsed store', () => {
  beforeEach(() => {
    localStorageMock.clear()
    sidebarListeners.clear()
  })

  describe('getSidebarSnapshot', () => {
    it('should return false when localStorage is empty', () => {
      expect(getSidebarSnapshot()).toBe(false)
    })

    it('should return true when localStorage has "true"', () => {
      localStorage.setItem('nimbus-sidebar-collapsed', 'true')
      expect(getSidebarSnapshot()).toBe(true)
    })

    it('should return false when localStorage has "false"', () => {
      localStorage.setItem('nimbus-sidebar-collapsed', 'false')
      expect(getSidebarSnapshot()).toBe(false)
    })
  })

  describe('getSidebarServerSnapshot', () => {
    it('should always return false', () => {
      expect(getSidebarServerSnapshot()).toBe(false)
    })
  })

  describe('setIsDesktopCollapsed', () => {
    it('should persist true to localStorage', () => {
      setIsDesktopCollapsed(true)
      expect(localStorage.getItem('nimbus-sidebar-collapsed')).toBe('true')
    })

    it('should persist false to localStorage', () => {
      setIsDesktopCollapsed(false)
      expect(localStorage.getItem('nimbus-sidebar-collapsed')).toBe('false')
    })

    it('should notify all listeners on change', () => {
      let callCount = 0
      subscribeSidebar(() => callCount++)
      subscribeSidebar(() => callCount++)

      setIsDesktopCollapsed(true)
      expect(callCount).toBe(2)
    })
  })

  describe('subscribeSidebar', () => {
    it('should add listener and return unsubscribe function', () => {
      let called = false
      const unsubscribe = subscribeSidebar(() => {
        called = true
      })

      setIsDesktopCollapsed(true)
      expect(called).toBe(true)

      called = false
      unsubscribe()
      setIsDesktopCollapsed(false)
      expect(called).toBe(false)
    })

    it('should handle multiple subscriptions independently', () => {
      const calls: string[] = []
      const unsub1 = subscribeSidebar(() => calls.push('a'))
      subscribeSidebar(() => calls.push('b'))

      setIsDesktopCollapsed(true)
      expect(calls).toEqual(['a', 'b'])

      unsub1()
      calls.length = 0
      setIsDesktopCollapsed(false)
      expect(calls).toEqual(['b'])
    })
  })

  describe('round-trip', () => {
    it('should round-trip collapsed state through localStorage', () => {
      expect(getSidebarSnapshot()).toBe(false)

      setIsDesktopCollapsed(true)
      expect(getSidebarSnapshot()).toBe(true)

      setIsDesktopCollapsed(false)
      expect(getSidebarSnapshot()).toBe(false)
    })
  })
})
