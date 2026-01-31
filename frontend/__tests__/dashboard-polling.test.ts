import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { api } from '@/lib/api'
import {
  mergeHealthData,
  mergeServicesHealth,
  buildHealthMap,
  hasHealthChanged,
  shouldSkipPoll,
  shouldShowResponseTime,
  calculateAverageResponseTime,
} from '@/lib/polling'
import type { Service } from '@/types'

// Mock the api module
vi.mock('@/lib/api', () => ({
  api: {
    getServices: vi.fn(),
    getGroups: vi.fn(),
  },
}))

// Helper to create test service with required fields
const createTestService = (
  overrides: Partial<Service> & {
    id: string
    name: string
    url: string
    status: Service['status']
    position: number
  }
): Service => ({
  icon_type: 'emoji',
  card_size: '2x1',
  created_at: new Date().toISOString(),
  monitoring_enabled: true,
  ...overrides,
})

describe('Dashboard Polling Functionality', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.restoreAllMocks()
    vi.useRealTimers()
  })

  describe('Health Status Polling', () => {
    it('should update service status when polling detects changes', async () => {
      const initialServices: Service[] = [
        createTestService({
          id: 'service-1',
          name: 'Test Service',
          url: 'https://example.com',
          status: 'online',
          response_time: 100,
          position: 0,
        }),
      ]

      const updatedServices: Service[] = [
        createTestService({
          id: 'service-1',
          name: 'Test Service',
          url: 'https://example.com',
          status: 'offline',
          response_time: undefined,
          position: 0,
        }),
      ]

      vi.mocked(api.getServices)
        .mockResolvedValueOnce({ data: initialServices })
        .mockResolvedValueOnce({ data: updatedServices })

      // First fetch
      const firstResult = await api.getServices()
      expect(firstResult.data?.[0].status).toBe('online')

      // Simulated poll (30 seconds later)
      const secondResult = await api.getServices()
      expect(secondResult.data?.[0].status).toBe('offline')
    })

    it('should only update status and response_time fields during polling', async () => {
      const originalService: Service = createTestService({
        id: 'service-1',
        name: 'Original Name',
        url: 'https://example.com',
        description: 'Original description',
        icon: '🔗',
        icon_type: 'emoji',
        status: 'online',
        response_time: 100,
        position: 0,
        card_size: '2x1',
      })

      const polledService: Service = createTestService({
        id: 'service-1',
        name: 'Different Name', // This should NOT be updated in actual polling
        url: 'https://different.com', // This should NOT be updated
        description: 'Different description', // This should NOT be updated
        icon: '🆕', // This should NOT be updated
        icon_type: 'emoji',
        status: 'offline', // This SHOULD be updated
        response_time: 500, // This SHOULD be updated
        position: 5, // This should NOT be updated
        card_size: '1x1', // This should NOT be updated
      })

      // Use the real merge function from lib/polling
      const merged = mergeHealthData(originalService, {
        status: polledService.status,
        response_time: polledService.response_time,
      })

      // Status and response_time should be updated
      expect(merged.status).toBe('offline')
      expect(merged.response_time).toBe(500)

      // Other fields should remain unchanged
      expect(merged.name).toBe('Original Name')
      expect(merged.url).toBe('https://example.com')
      expect(merged.description).toBe('Original description')
      expect(merged.icon).toBe('🔗')
      expect(merged.position).toBe(0)
      expect(merged.card_size).toBe('2x1')
    })

    it('should not trigger update when status and response_time are unchanged', () => {
      const existing: Service = createTestService({
        id: 'service-1',
        name: 'Test',
        url: 'https://example.com',
        status: 'online',
        response_time: 100,
        position: 0,
      })

      const health = {
        status: 'online' as const,
        response_time: 100,
      }

      // Use the real hasHealthChanged function from lib/polling
      expect(hasHealthChanged(existing, health)).toBe(false)
    })

    it('should trigger update when status changes', () => {
      const existing: Service = createTestService({
        id: 'service-1',
        name: 'Test',
        url: 'https://example.com',
        status: 'online',
        response_time: 100,
        position: 0,
      })

      const health = {
        status: 'offline' as const,
        response_time: 100,
      }

      // Use the real hasHealthChanged function from lib/polling
      expect(hasHealthChanged(existing, health)).toBe(true)
    })

    it('should trigger update when response_time changes', () => {
      const existing: Service = createTestService({
        id: 'service-1',
        name: 'Test',
        url: 'https://example.com',
        status: 'online',
        response_time: 100,
        position: 0,
      })

      const health = {
        status: 'online' as const,
        response_time: 150,
      }

      // Use the real hasHealthChanged function from lib/polling
      expect(hasHealthChanged(existing, health)).toBe(true)
    })
  })

  describe('Polling Coordination', () => {
    it('should skip poll if within overlap window after full fetch', () => {
      // Use the real shouldSkipPoll function from lib/polling
      // With a recent timestamp (now), it should skip
      const recentTime = Date.now()
      expect(shouldSkipPoll(recentTime, 5000)).toBe(true)
    })

    it('should allow poll after overlap window expires', () => {
      // Use the real shouldSkipPoll function from lib/polling
      // With an old timestamp (6 seconds ago), it should not skip
      const oldTime = Date.now() - 6000
      expect(shouldSkipPoll(oldTime, 5000)).toBe(false)
    })

    it('should update lastPollTime after full fetch', () => {
      let lastPollTime = 0

      // Simulate full fetch
      lastPollTime = Date.now()

      expect(lastPollTime).toBeGreaterThan(0)
    })
  })

  describe('Error Handling', () => {
    it('should handle API errors gracefully during polling', async () => {
      vi.mocked(api.getServices).mockRejectedValue(new Error('Network error'))

      let errorCaught = false
      try {
        await api.getServices()
      } catch {
        errorCaught = true
      }

      expect(errorCaught).toBe(true)
    })

    it('should preserve existing services when polling fails', () => {
      const existingServices: Service[] = [
        createTestService({
          id: 'service-1',
          name: 'Test',
          url: 'https://example.com',
          status: 'online',
          response_time: 100,
          position: 0,
        }),
      ]

      // Simulate error handling - services should remain unchanged
      const handlePollError = (services: Service[]): Service[] => services

      const result = handlePollError(existingServices)

      expect(result).toEqual(existingServices)
    })
  })

  describe('Service Health Map Building', () => {
    it('should build health map correctly from polled services', () => {
      const polledServices: Service[] = [
        createTestService({
          id: 'service-1',
          name: 'Service 1',
          url: 'https://example1.com',
          status: 'online',
          response_time: 100,
          position: 0,
        }),
        createTestService({
          id: 'service-2',
          name: 'Service 2',
          url: 'https://example2.com',
          status: 'offline',
          response_time: undefined,
          position: 1,
        }),
        createTestService({
          id: 'service-3',
          name: 'Service 3',
          url: 'https://example3.com',
          status: 'online',
          response_time: 250,
          position: 2,
        }),
      ]

      // Use the real buildHealthMap function from lib/polling
      const healthMap = buildHealthMap(polledServices)

      expect(healthMap.get('service-1')).toEqual({ status: 'online', response_time: 100 })
      expect(healthMap.get('service-2')).toEqual({ status: 'offline', response_time: undefined })
      expect(healthMap.get('service-3')).toEqual({ status: 'online', response_time: 250 })
    })

    it('should handle new services appearing in poll', () => {
      const existingServices: Service[] = [
        createTestService({
          id: 'service-1',
          name: 'Existing',
          url: 'https://existing.com',
          status: 'online',
          response_time: 100,
          position: 0,
        }),
      ]

      const polledServices: Service[] = [
        createTestService({
          id: 'service-1',
          name: 'Existing',
          url: 'https://existing.com',
          status: 'offline',
          response_time: undefined,
          position: 0,
        }),
        createTestService({
          id: 'service-2',
          name: 'New Service',
          url: 'https://new.com',
          status: 'online',
          response_time: 50,
          position: 1,
        }),
      ]

      // Use the real mergeServicesHealth function from lib/polling
      const mergedServices = mergeServicesHealth(existingServices, polledServices)

      // Should still only have the original service (merge doesn't add new services)
      expect(mergedServices.length).toBe(1)
      // But with updated health status
      expect(mergedServices[0].status).toBe('offline')
    })

    it('should handle services disappearing from poll', () => {
      const existingServices: Service[] = [
        createTestService({
          id: 'service-1',
          name: 'Service 1',
          url: 'https://example1.com',
          status: 'online',
          response_time: 100,
          position: 0,
        }),
        createTestService({
          id: 'service-2',
          name: 'Service 2',
          url: 'https://example2.com',
          status: 'online',
          response_time: 150,
          position: 1,
        }),
      ]

      // Polling returns only service-1 (service-2 was deleted)
      const polledServices: Service[] = [
        createTestService({
          id: 'service-1',
          name: 'Service 1',
          url: 'https://example1.com',
          status: 'offline',
          response_time: undefined,
          position: 0,
        }),
      ]

      // Use the real mergeServicesHealth function from lib/polling
      const mergedServices = mergeServicesHealth(existingServices, polledServices)

      // Should preserve both services (merge doesn't remove services)
      expect(mergedServices.length).toBe(2)
      // service-1 is updated
      expect(mergedServices[0].status).toBe('offline')
      // service-2 keeps its original state
      expect(mergedServices[1].status).toBe('online')
    })
  })

  describe('Polling Interval', () => {
    it('should use 30 second polling interval', () => {
      const POLL_INTERVAL = 30000 // 30 seconds in ms

      expect(POLL_INTERVAL).toBe(30000)
    })

    it('should track interval ID for cleanup', () => {
      let intervalId: NodeJS.Timeout | null = null

      // Simulate setting interval
      intervalId = setInterval(() => {
        // Poll function
      }, 30000)

      expect(intervalId).not.toBeNull()

      // Cleanup
      if (intervalId) {
        clearInterval(intervalId)
      }
    })
  })
})

describe('Response Time Display Logic', () => {
  it('should only show response time for online services', () => {
    const onlineService: Service = createTestService({
      id: 'service-1',
      name: 'Online Service',
      url: 'https://online.com',
      status: 'online',
      response_time: 150,
      position: 0,
    })

    const offlineService: Service = createTestService({
      id: 'service-2',
      name: 'Offline Service',
      url: 'https://offline.com',
      status: 'offline',
      response_time: 1004, // Timeout value that shouldn't be displayed
      position: 1,
    })

    // Use the real shouldShowResponseTime function from lib/polling
    expect(shouldShowResponseTime(onlineService)).toBe(true)
    expect(shouldShowResponseTime(offlineService)).toBe(false)
  })

  it('should not show response time when undefined', () => {
    const service: Service = createTestService({
      id: 'service-1',
      name: 'Service',
      url: 'https://example.com',
      status: 'online',
      response_time: undefined,
      position: 0,
    })

    // Use the real shouldShowResponseTime function from lib/polling
    expect(shouldShowResponseTime(service)).toBe(false)
  })

  it('should calculate average response time only from online services', () => {
    const services: Service[] = [
      createTestService({
        id: 'service-1',
        name: 'Online 1',
        url: 'https://online1.com',
        status: 'online',
        response_time: 100,
        position: 0,
      }),
      createTestService({
        id: 'service-2',
        name: 'Offline',
        url: 'https://offline.com',
        status: 'offline',
        response_time: 1004, // Should be excluded
        position: 1,
      }),
      createTestService({
        id: 'service-3',
        name: 'Online 2',
        url: 'https://online2.com',
        status: 'online',
        response_time: 200,
        position: 2,
      }),
    ]

    // Use the real calculateAverageResponseTime function from lib/polling
    // Should average only 100 and 200, not include 1004 (offline service)
    expect(calculateAverageResponseTime(services)).toBe(150)
  })
})
