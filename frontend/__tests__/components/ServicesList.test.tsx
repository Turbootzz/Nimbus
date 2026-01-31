import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import '@testing-library/jest-dom'
import ServicesList from '@/components/ServicesList'
import type { Service } from '@/types'

// Mock ServiceListItem
vi.mock('@/components/ServiceListItem', () => ({
  default: ({
    service,
    isEditMode,
    openInNewTab,
  }: {
    service: Service
    isEditMode?: boolean
    openInNewTab: boolean
  }) => (
    <div
      data-testid={`service-item-${service.id}`}
      data-edit-mode={isEditMode}
      data-open-in-new-tab={String(openInNewTab)}
    >
      {service.name}
    </div>
  ),
}))

// Mock SortableServiceListItem
vi.mock('@/components/SortableServiceListItem', () => ({
  default: ({ service }: { service: Service }) => (
    <div data-testid={`sortable-service-item-${service.id}`}>{service.name} (sortable)</div>
  ),
}))

describe('ServicesList', () => {
  const createMockService = (overrides: Partial<Service> = {}): Service => ({
    id: 'service-1',
    name: 'Test Service',
    url: 'https://example.com',
    icon_type: 'emoji',
    icon: '🚀',
    status: 'online',
    position: 0,
    card_size: '2x1',
    created_at: new Date().toISOString(),
    monitoring_enabled: true,
    ...overrides,
  })

  const createMockServices = (count: number): Service[] => {
    return Array.from({ length: count }, (_, i) =>
      createMockService({
        id: `service-${i + 1}`,
        name: `Service ${i + 1}`,
        position: i,
      })
    )
  }

  describe('Basic rendering', () => {
    it('should render all services', () => {
      const services = createMockServices(3)
      render(<ServicesList services={services} openInNewTab={false} />)

      expect(screen.getByText('Service 1')).toBeInTheDocument()
      expect(screen.getByText('Service 2')).toBeInTheDocument()
      expect(screen.getByText('Service 3')).toBeInTheDocument()
    })

    it('should render empty state when no services', () => {
      const { container } = render(<ServicesList services={[]} openInNewTab={false} />)

      // Container should be rendered but empty
      const grid = container.firstChild
      expect(grid).toBeInTheDocument()
      expect(grid?.childNodes.length).toBe(0)
    })

    it('should use ServiceListItem when not in edit mode', () => {
      const services = createMockServices(2)
      render(<ServicesList services={services} openInNewTab={false} isEditMode={false} />)

      expect(screen.getByTestId('service-item-service-1')).toBeInTheDocument()
      expect(screen.getByTestId('service-item-service-2')).toBeInTheDocument()
      expect(screen.queryByTestId('sortable-service-item-service-1')).not.toBeInTheDocument()
    })

    it('should use SortableServiceListItem when in edit mode', () => {
      const services = createMockServices(2)
      render(<ServicesList services={services} openInNewTab={false} isEditMode={true} />)

      expect(screen.getByTestId('sortable-service-item-service-1')).toBeInTheDocument()
      expect(screen.getByTestId('sortable-service-item-service-2')).toBeInTheDocument()
      expect(screen.queryByTestId('service-item-service-1')).not.toBeInTheDocument()
    })
  })

  describe('Grid layout', () => {
    it('should render with grid layout classes', () => {
      const services = createMockServices(1)
      const { container } = render(<ServicesList services={services} openInNewTab={false} />)

      const grid = container.firstChild
      expect(grid).toHaveClass('grid')
      expect(grid).toHaveClass('grid-cols-1')
    })

    it('should have responsive multi-column layout', () => {
      const services = createMockServices(1)
      const { container } = render(<ServicesList services={services} openInNewTab={false} />)

      const grid = container.firstChild
      expect(grid).toHaveClass('lg:grid-cols-2')
      expect(grid).toHaveClass('2xl:grid-cols-3')
    })

    it('should have compact gap styling', () => {
      const services = createMockServices(1)
      const { container } = render(<ServicesList services={services} openInNewTab={false} />)

      const grid = container.firstChild
      expect(grid).toHaveClass('gap-x-3')
      expect(grid).toHaveClass('gap-y-1')
    })
  })

  describe('Props forwarding', () => {
    it('should forward openInNewTab to ServiceListItem', () => {
      const services = createMockServices(1)
      render(<ServicesList services={services} openInNewTab={true} />)

      const item = screen.getByTestId('service-item-service-1')
      expect(item).toHaveAttribute('data-open-in-new-tab', 'true')
    })

    it('should forward openInNewTab=false to ServiceListItem', () => {
      const services = createMockServices(1)
      render(<ServicesList services={services} openInNewTab={false} />)

      const item = screen.getByTestId('service-item-service-1')
      expect(item).toHaveAttribute('data-open-in-new-tab', 'false')
    })

    it('should default isEditMode to false', () => {
      const services = createMockServices(1)
      render(<ServicesList services={services} openInNewTab={false} />)

      // Without explicit isEditMode, should use regular ServiceListItem
      expect(screen.getByTestId('service-item-service-1')).toBeInTheDocument()
    })
  })

  describe('Service ordering', () => {
    it('should render services in array order', () => {
      const services = [
        createMockService({ id: 'svc-a', name: 'Alpha' }),
        createMockService({ id: 'svc-b', name: 'Beta' }),
        createMockService({ id: 'svc-c', name: 'Gamma' }),
      ]
      const { container } = render(<ServicesList services={services} openInNewTab={false} />)

      const items = container.querySelectorAll('[data-testid^="service-item-"]')
      expect(items[0]).toHaveAttribute('data-testid', 'service-item-svc-a')
      expect(items[1]).toHaveAttribute('data-testid', 'service-item-svc-b')
      expect(items[2]).toHaveAttribute('data-testid', 'service-item-svc-c')
    })
  })
})
