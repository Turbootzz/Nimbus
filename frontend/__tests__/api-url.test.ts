/**
 * API URL Utility Tests
 *
 * Tests the getApiUrl function which determines the API base URL
 * based on environment variables and runtime detection.
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'

// Store original env
const originalEnv = process.env

describe('getApiUrl', () => {
  beforeEach(() => {
    vi.resetModules()
    process.env = { ...originalEnv }
  })

  afterEach(() => {
    process.env = originalEnv
  })

  describe('NEXT_PUBLIC_API_URL handling', () => {
    it('should return /api/v1 for "same-origin" value', async () => {
      process.env.NEXT_PUBLIC_API_URL = 'same-origin'
      const { getApiUrl } = await import('@/lib/utils/api-url')
      expect(getApiUrl()).toBe('/api/v1')
    })

    it('should handle "same-origin" case-insensitively', async () => {
      process.env.NEXT_PUBLIC_API_URL = 'SAME-ORIGIN'
      const { getApiUrl } = await import('@/lib/utils/api-url')
      expect(getApiUrl()).toBe('/api/v1')
    })

    it('should handle "Same-Origin" mixed case', async () => {
      process.env.NEXT_PUBLIC_API_URL = 'Same-Origin'
      const { getApiUrl } = await import('@/lib/utils/api-url')
      expect(getApiUrl()).toBe('/api/v1')
    })

    it('should use full URL when provided', async () => {
      process.env.NEXT_PUBLIC_API_URL = 'http://backend:8080'
      const { getApiUrl } = await import('@/lib/utils/api-url')
      expect(getApiUrl()).toBe('http://backend:8080/api/v1')
    })

    it('should not duplicate /api/v1 if already present', async () => {
      process.env.NEXT_PUBLIC_API_URL = 'http://backend:8080/api/v1'
      const { getApiUrl } = await import('@/lib/utils/api-url')
      expect(getApiUrl()).toBe('http://backend:8080/api/v1')
    })

    it('should strip trailing slash from URL', async () => {
      process.env.NEXT_PUBLIC_API_URL = 'http://backend:8080/'
      const { getApiUrl } = await import('@/lib/utils/api-url')
      expect(getApiUrl()).toBe('http://backend:8080/api/v1')
    })

    it('should fall through to runtime detection for empty string', async () => {
      process.env.NEXT_PUBLIC_API_URL = ''
      const { getApiUrl } = await import('@/lib/utils/api-url')
      // Without window, falls back to localhost
      expect(getApiUrl()).toBe('http://localhost:8080/api/v1')
    })

    it('should fall through to runtime detection for whitespace-only string', async () => {
      process.env.NEXT_PUBLIC_API_URL = '   '
      const { getApiUrl } = await import('@/lib/utils/api-url')
      expect(getApiUrl()).toBe('http://localhost:8080/api/v1')
    })

    it('should fall through to runtime detection when undefined', async () => {
      delete process.env.NEXT_PUBLIC_API_URL
      const { getApiUrl } = await import('@/lib/utils/api-url')
      expect(getApiUrl()).toBe('http://localhost:8080/api/v1')
    })
  })

  describe('URL normalization', () => {
    it('should trim whitespace from URL', async () => {
      process.env.NEXT_PUBLIC_API_URL = '  http://backend:8080  '
      const { getApiUrl } = await import('@/lib/utils/api-url')
      expect(getApiUrl()).toBe('http://backend:8080/api/v1')
    })

    it('should trim whitespace from same-origin', async () => {
      process.env.NEXT_PUBLIC_API_URL = '  same-origin  '
      const { getApiUrl } = await import('@/lib/utils/api-url')
      expect(getApiUrl()).toBe('/api/v1')
    })
  })
})
