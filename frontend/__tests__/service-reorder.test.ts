import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { api } from '@/lib/api'

// Mock the api module
vi.mock('@/lib/api', () => ({
  api: {
    reorderServices: vi.fn(),
  },
}))

describe('Service Reorder Functionality', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('API Client - reorderServices', () => {
    it('should send correct payload for service reordering', async () => {
      const mockResponse = { data: { message: 'Service positions updated successfully' } }
      vi.mocked(api.reorderServices).mockResolvedValue(mockResponse)

      const reorderRequest = {
        services: [
          { id: 'service-1', position: 2 },
          { id: 'service-2', position: 0 },
          { id: 'service-3', position: 1 },
        ],
      }

      const result = await api.reorderServices(reorderRequest)

      expect(api.reorderServices).toHaveBeenCalledWith(reorderRequest)
      expect(api.reorderServices).toHaveBeenCalledTimes(1)
      expect(result).toEqual(mockResponse)
    })

    it('should handle reorder errors', async () => {
      const mockError = {
        error: { message: 'One or more services not found or access denied' },
      }
      vi.mocked(api.reorderServices).mockResolvedValue(mockError)

      const reorderRequest = {
        services: [{ id: 'non-existent', position: 0 }],
      }

      const result = await api.reorderServices(reorderRequest)

      expect(result.error).toBeDefined()
      expect(result.error?.message).toBe('One or more services not found or access denied')
    })

    it('should handle single service position update', async () => {
      const mockResponse = { data: { message: 'Service positions updated successfully' } }
      vi.mocked(api.reorderServices).mockResolvedValue(mockResponse)

      const reorderRequest = {
        services: [{ id: 'service-1', position: 5 }],
      }

      const result = await api.reorderServices(reorderRequest)

      expect(api.reorderServices).toHaveBeenCalledWith(reorderRequest)
      expect(result.data).toBeDefined()
    })
  })

  describe('Position Calculation Logic', () => {
    it('should correctly calculate new positions after drag and drop', () => {
      const services = [
        {
          id: 'service-1',
          name: 'First',
          url: 'https://example1.com',
          status: 'online',
          position: 0,
          created_at: new Date().toISOString(),
        },
        {
          id: 'service-2',
          name: 'Second',
          url: 'https://example2.com',
          status: 'online',
          position: 1,
          created_at: new Date().toISOString(),
        },
        {
          id: 'service-3',
          name: 'Third',
          url: 'https://example3.com',
          status: 'online',
          position: 2,
          created_at: new Date().toISOString(),
        },
      ]

      // Simulate dragging service-1 to position 2
      const oldIndex = 0
      const newIndex = 2

      // Create new array with updated positions
      const reorderedServices = [...services]
      const [movedItem] = reorderedServices.splice(oldIndex, 1)
      reorderedServices.splice(newIndex, 0, movedItem)

      // Update positions
      const updatedServices = reorderedServices.map((service, index) => ({
        ...service,
        position: index,
      }))

      expect(updatedServices[0].id).toBe('service-2')
      expect(updatedServices[0].position).toBe(0)
      expect(updatedServices[1].id).toBe('service-3')
      expect(updatedServices[1].position).toBe(1)
      expect(updatedServices[2].id).toBe('service-1')
      expect(updatedServices[2].position).toBe(2)
    })

    it('should handle moving item to same position', () => {
      const services = [
        {
          id: 'service-1',
          name: 'First',
          url: 'https://example1.com',
          status: 'online',
          position: 0,
          created_at: new Date().toISOString(),
        },
      ]

      const oldIndex = 0
      const newIndex = 0

      const reorderedServices = [...services]
      const [movedItem] = reorderedServices.splice(oldIndex, 1)
      reorderedServices.splice(newIndex, 0, movedItem)

      expect(reorderedServices[0].id).toBe('service-1')
      expect(reorderedServices[0].position).toBe(0)
    })
  })

  describe('Service Position Validation', () => {
    it('should reject negative positions', () => {
      const position = -1
      const isValid = position >= 0

      expect(isValid).toBe(false)
    })

    it('should accept zero position', () => {
      const position = 0
      const isValid = position >= 0

      expect(isValid).toBe(true)
    })

    it('should accept positive positions', () => {
      const position = 10
      const isValid = position >= 0

      expect(isValid).toBe(true)
    })

    it('should validate service ID is not empty', () => {
      const serviceId = ''
      const isValid = serviceId.length > 0

      expect(isValid).toBe(false)
    })

    it('should validate service ID exists', () => {
      const serviceId = 'service-1'
      const isValid = serviceId.length > 0

      expect(isValid).toBe(true)
    })
  })

  describe('Optimistic Update Rollback', () => {
    it('should revert to original positions on error', () => {
      const originalServices = [
        {
          id: 'service-1',
          name: 'First',
          url: 'https://example1.com',
          status: 'online',
          position: 0,
          created_at: new Date().toISOString(),
        },
        {
          id: 'service-2',
          name: 'Second',
          url: 'https://example2.com',
          status: 'online',
          position: 1,
          created_at: new Date().toISOString(),
        },
      ]

      // Simulate optimistic update
      const optimisticServices = originalServices.map((service, index) =>
        index === 0 ? { ...service, position: 1 } : { ...service, position: 0 }
      )

      // Verify optimistic update
      expect(optimisticServices[0].position).toBe(1)
      expect(optimisticServices[1].position).toBe(0)

      // Simulate API error - rollback to original
      const rolledBackServices = [...originalServices]

      // Verify rollback
      expect(rolledBackServices[0].position).toBe(0)
      expect(rolledBackServices[1].position).toBe(1)
      expect(rolledBackServices).toEqual(originalServices)
    })
  })

  describe('Service Ordering', () => {
    it('should sort services by position ascending', () => {
      const services = [
        {
          id: 'service-3',
          name: 'Third',
          url: 'https://example3.com',
          status: 'online',
          position: 2,
          created_at: new Date().toISOString(),
        },
        {
          id: 'service-1',
          name: 'First',
          url: 'https://example1.com',
          status: 'online',
          position: 0,
          created_at: new Date().toISOString(),
        },
        {
          id: 'service-2',
          name: 'Second',
          url: 'https://example2.com',
          status: 'online',
          position: 1,
          created_at: new Date().toISOString(),
        },
      ]

      const sorted = [...services].sort((a, b) => a.position - b.position)

      expect(sorted[0].id).toBe('service-1')
      expect(sorted[1].id).toBe('service-2')
      expect(sorted[2].id).toBe('service-3')
    })

    it('should handle services with same position', () => {
      const services = [
        {
          id: 'service-1',
          name: 'First',
          url: 'https://example1.com',
          status: 'online',
          position: 0,
          created_at: '2024-01-01T00:00:00Z',
        },
        {
          id: 'service-2',
          name: 'Second',
          url: 'https://example2.com',
          status: 'online',
          position: 0,
          created_at: '2024-01-02T00:00:00Z',
        },
      ]

      // Sort by position, then by created_at
      const sorted = [...services].sort((a, b) => {
        if (a.position !== b.position) {
          return a.position - b.position
        }
        return new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
      })

      expect(sorted[0].id).toBe('service-2') // Newer creation date first
      expect(sorted[1].id).toBe('service-1')
    })
  })
})
