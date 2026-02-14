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

import {
  subscribeSidebar,
  getSidebarSnapshot,
  getSidebarServerSnapshot,
  setSidebarCollapsed,
  clearSidebarListeners,
} from '@/lib/sidebar-store'

describe('Sidebar collapsed store', () => {
  beforeEach(() => {
    localStorageMock.clear()
    clearSidebarListeners()
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

  describe('setSidebarCollapsed', () => {
    it('should persist true to localStorage', () => {
      setSidebarCollapsed(true)
      expect(localStorage.getItem('nimbus-sidebar-collapsed')).toBe('true')
    })

    it('should persist false to localStorage', () => {
      setSidebarCollapsed(false)
      expect(localStorage.getItem('nimbus-sidebar-collapsed')).toBe('false')
    })

    it('should notify all listeners on change', () => {
      let callCount = 0
      subscribeSidebar(() => callCount++)
      subscribeSidebar(() => callCount++)

      setSidebarCollapsed(true)
      expect(callCount).toBe(2)
    })
  })

  describe('subscribeSidebar', () => {
    it('should add listener and return unsubscribe function', () => {
      let called = false
      const unsubscribe = subscribeSidebar(() => {
        called = true
      })

      setSidebarCollapsed(true)
      expect(called).toBe(true)

      called = false
      unsubscribe()
      setSidebarCollapsed(false)
      expect(called).toBe(false)
    })

    it('should handle multiple subscriptions independently', () => {
      const calls: string[] = []
      const unsub1 = subscribeSidebar(() => calls.push('a'))
      subscribeSidebar(() => calls.push('b'))

      setSidebarCollapsed(true)
      expect(calls).toEqual(['a', 'b'])

      unsub1()
      calls.length = 0
      setSidebarCollapsed(false)
      expect(calls).toEqual(['b'])
    })
  })

  describe('round-trip', () => {
    it('should round-trip collapsed state through localStorage', () => {
      expect(getSidebarSnapshot()).toBe(false)

      setSidebarCollapsed(true)
      expect(getSidebarSnapshot()).toBe(true)

      setSidebarCollapsed(false)
      expect(getSidebarSnapshot()).toBe(false)
    })
  })
})
