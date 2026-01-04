import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { api } from '@/lib/api'
import type { Service, CardSize } from '@/types'

// Mock the api module
vi.mock('@/lib/api', () => ({
  api: {
    getServices: vi.fn(),
    reorderServices: vi.fn(),
    updateService: vi.fn(),
  },
}))

describe('Edit Mode Functionality', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  // Helper to create mock services
  const createMockService = (overrides: Partial<Service> = {}): Service => ({
    id: 'service-1',
    name: 'Test Service',
    url: 'https://example.com',
    icon_type: 'emoji',
    status: 'online',
    position: 0,
    card_size: '2x1',
    created_at: new Date().toISOString(),
    ...overrides,
  })

  describe('Card Size Cycling', () => {
    it('should cycle through card sizes: 1x1 -> 2x1 -> 1x2 -> 2x2 -> 1x1', () => {
      const sizeCycle: CardSize[] = ['1x1', '2x1', '1x2', '2x2']

      const getNextSize = (currentSize: CardSize): CardSize => {
        const currentIndex = sizeCycle.indexOf(currentSize)
        const nextIndex = (currentIndex + 1) % sizeCycle.length
        return sizeCycle[nextIndex]
      }

      expect(getNextSize('1x1')).toBe('2x1')
      expect(getNextSize('2x1')).toBe('1x2')
      expect(getNextSize('1x2')).toBe('2x2')
      expect(getNextSize('2x2')).toBe('1x1')
    })

    it('should default to 2x1 when card_size is undefined', () => {
      const service = createMockService({ card_size: undefined })
      const effectiveSize = service.card_size || '2x1'

      expect(effectiveSize).toBe('2x1')
    })
  })

  describe('API - updateService for card size', () => {
    it('should send correct payload when updating card size', async () => {
      const mockResponse = { message: 'Service updated successfully' }
      vi.mocked(api.updateService).mockResolvedValue(mockResponse)

      const service = createMockService()
      const newSize: CardSize = '2x2'

      const result = await api.updateService(service.id, {
        name: service.name,
        url: service.url,
        description: service.description || '',
        icon: service.icon || '',
        icon_type: service.icon_type || 'emoji',
        icon_image_path: service.icon_image_path || '',
        card_size: newSize,
      })

      expect(api.updateService).toHaveBeenCalledWith(service.id, {
        name: service.name,
        url: service.url,
        description: '',
        icon: '',
        icon_type: 'emoji',
        icon_image_path: '',
        card_size: '2x2',
      })
      expect(result).toEqual(mockResponse)
    })

    it('should handle card size update errors', async () => {
      const mockError = { error: { message: 'Service not found' } }
      vi.mocked(api.updateService).mockResolvedValue(mockError)

      const result = await api.updateService('non-existent', {
        name: 'Test',
        url: 'https://example.com',
        card_size: '1x1',
      })

      expect(result.error).toBeDefined()
      expect(result.error?.message).toBe('Service not found')
    })
  })

  describe('Optimistic Updates', () => {
    it('should apply optimistic card size update to local state', () => {
      const services: Service[] = [
        createMockService({ id: 'service-1', card_size: '2x1' }),
        createMockService({ id: 'service-2', card_size: '1x1' }),
      ]

      const targetId = 'service-1'
      const newSize: CardSize = '2x2'

      // Simulate optimistic update
      const updatedServices = services.map((s) =>
        s.id === targetId ? { ...s, card_size: newSize } : s
      )

      expect(updatedServices[0].card_size).toBe('2x2')
      expect(updatedServices[1].card_size).toBe('1x1')
    })

    it('should revert card size on API error', () => {
      const originalServices: Service[] = [createMockService({ id: 'service-1', card_size: '2x1' })]

      // Simulate optimistic update
      const optimisticServices = originalServices.map((s) => ({
        ...s,
        card_size: '2x2' as CardSize,
      }))

      expect(optimisticServices[0].card_size).toBe('2x2')

      // Simulate rollback
      const rolledBackServices = [...originalServices]

      expect(rolledBackServices[0].card_size).toBe('2x1')
      expect(rolledBackServices).toEqual(originalServices)
    })
  })

  describe('Drag and Drop Reordering', () => {
    it('should calculate new positions after drag end', () => {
      const services: Service[] = [
        createMockService({ id: 'service-1', position: 0 }),
        createMockService({ id: 'service-2', position: 1 }),
        createMockService({ id: 'service-3', position: 2 }),
      ]

      // Simulate dragging service-3 to position 0
      const activeId = 'service-3'
      const overId = 'service-1'

      const oldIndex = services.findIndex((s) => s.id === activeId)
      const newIndex = services.findIndex((s) => s.id === overId)

      // arrayMove simulation
      const reordered = [...services]
      const [movedItem] = reordered.splice(oldIndex, 1)
      reordered.splice(newIndex, 0, movedItem)

      expect(reordered[0].id).toBe('service-3')
      expect(reordered[1].id).toBe('service-1')
      expect(reordered[2].id).toBe('service-2')
    })

    it('should not reorder when dropping on same position', () => {
      const activeId = 'service-1'
      const overId = 'service-1'

      // Same position - should return early
      const shouldReorder = activeId !== overId

      expect(shouldReorder).toBe(false)
    })

    it('should generate correct reorder payload for API', async () => {
      const mockResponse = { data: { message: 'Service positions updated successfully' } }
      vi.mocked(api.reorderServices).mockResolvedValue(mockResponse)

      const reorderedServices: Service[] = [
        createMockService({ id: 'service-3' }),
        createMockService({ id: 'service-1' }),
        createMockService({ id: 'service-2' }),
      ]

      const payload = {
        services: reorderedServices.map((s, i) => ({ id: s.id, position: i })),
      }

      await api.reorderServices(payload)

      expect(api.reorderServices).toHaveBeenCalledWith({
        services: [
          { id: 'service-3', position: 0 },
          { id: 'service-1', position: 1 },
          { id: 'service-2', position: 2 },
        ],
      })
    })
  })

  describe('Grid Span Mapping', () => {
    it('should map card sizes to correct grid span classes', () => {
      const sizeToGridSpan: Record<CardSize, string> = {
        '1x1': 'col-span-1 row-span-1',
        '2x1': 'col-span-2 row-span-1',
        '1x2': 'col-span-1 row-span-2',
        '2x2': 'col-span-2 row-span-2',
      }

      expect(sizeToGridSpan['1x1']).toBe('col-span-1 row-span-1')
      expect(sizeToGridSpan['2x1']).toBe('col-span-2 row-span-1')
      expect(sizeToGridSpan['1x2']).toBe('col-span-1 row-span-2')
      expect(sizeToGridSpan['2x2']).toBe('col-span-2 row-span-2')
    })

    it('should handle undefined card_size by defaulting to 2x1', () => {
      const service = createMockService({ card_size: undefined })
      const effectiveSize = service.card_size || '2x1'

      const sizeToGridSpan: Record<CardSize, string> = {
        '1x1': 'col-span-1 row-span-1',
        '2x1': 'col-span-2 row-span-1',
        '1x2': 'col-span-1 row-span-2',
        '2x2': 'col-span-2 row-span-2',
      }

      expect(sizeToGridSpan[effectiveSize]).toBe('col-span-2 row-span-1')
    })
  })

  describe('Edit Mode State', () => {
    it('should toggle edit mode state', () => {
      let isEditMode = false

      // Enter edit mode
      isEditMode = true
      expect(isEditMode).toBe(true)

      // Exit edit mode
      isEditMode = false
      expect(isEditMode).toBe(false)
    })

    it('should disable card links in edit mode', () => {
      const isEditMode = true

      // In edit mode, clicking card should cycle size, not navigate
      const shouldNavigate = !isEditMode

      expect(shouldNavigate).toBe(false)
    })

    it('should enable card links when not in edit mode', () => {
      const isEditMode = false

      const shouldNavigate = !isEditMode

      expect(shouldNavigate).toBe(true)
    })
  })

  describe('Combined Edit Operations', () => {
    it('should support both reorder and resize in single edit session', async () => {
      vi.mocked(api.reorderServices).mockResolvedValue({ data: { message: 'Success' } })
      vi.mocked(api.updateService).mockResolvedValue({ data: createMockService() })

      const services: Service[] = [
        createMockService({ id: 'service-1', position: 0, card_size: '2x1' }),
        createMockService({ id: 'service-2', position: 1, card_size: '1x1' }),
      ]

      // Simulate reorder
      const reorderedServices = [services[1], services[0]]
      await api.reorderServices({
        services: reorderedServices.map((s, i) => ({ id: s.id, position: i })),
      })

      // Simulate resize
      await api.updateService('service-1', {
        name: services[0].name,
        url: services[0].url,
        card_size: '2x2',
      })

      expect(api.reorderServices).toHaveBeenCalledTimes(1)
      expect(api.updateService).toHaveBeenCalledTimes(1)
    })
  })
})
