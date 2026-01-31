import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import '@testing-library/jest-dom'
import ServiceListItem from '@/components/ServiceListItem'
import type { Service } from '@/types'

// Mock ServiceIcon component
vi.mock('@/components/ServiceIcon', () => ({
  default: ({ service, size }: { service: Service; size: string }) => (
    <div data-testid="service-icon" data-size={size}>
      {service.icon || '🔗'}
    </div>
  ),
}))

// Mock status-utils
vi.mock('@/lib/status-utils', () => ({
  getStatusBgColor: (status: string) => {
    switch (status) {
      case 'online':
        return 'bg-success'
      case 'offline':
        return 'bg-error'
      default:
        return 'bg-warning'
    }
  },
  getResponseTimeColor: (responseTime: number) => {
    if (responseTime < 100) return 'text-success'
    if (responseTime < 500) return 'text-warning'
    return 'text-error'
  },
}))

describe('ServiceListItem', () => {
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
    description: 'A test service description',
    response_time: 45,
    monitoring_enabled: true,
    ...overrides,
  })

  describe('Basic rendering', () => {
    it('should render service name', () => {
      const service = createMockService({ name: 'My Service' })
      render(<ServiceListItem service={service} openInNewTab={false} />)

      expect(screen.getByText('My Service')).toBeInTheDocument()
    })

    it('should render service icon with sm size', () => {
      const service = createMockService()
      render(<ServiceListItem service={service} openInNewTab={false} />)

      const icon = screen.getByTestId('service-icon')
      expect(icon).toHaveAttribute('data-size', 'sm')
    })

    it('should render service description', () => {
      const service = createMockService({ description: 'Test description' })
      render(<ServiceListItem service={service} openInNewTab={false} />)

      expect(screen.getByText('Test description')).toBeInTheDocument()
    })

    it('should not render description if not provided', () => {
      const service = createMockService({ description: undefined })
      render(<ServiceListItem service={service} openInNewTab={false} />)

      expect(screen.queryByText('Test description')).not.toBeInTheDocument()
    })

    it('should render response time when available', () => {
      const service = createMockService({ response_time: 123 })
      render(<ServiceListItem service={service} openInNewTab={false} />)

      expect(screen.getByText('123ms')).toBeInTheDocument()
    })

    it('should not render response time when undefined', () => {
      const service = createMockService({ response_time: undefined })
      render(<ServiceListItem service={service} openInNewTab={false} />)

      expect(screen.queryByText(/ms$/)).not.toBeInTheDocument()
    })

    it('should not render response time when service has no response_time', () => {
      const serviceWithoutResponseTime = createMockService()
      // Remove response_time from the mock
      delete (serviceWithoutResponseTime as { response_time?: number }).response_time
      render(<ServiceListItem service={serviceWithoutResponseTime} openInNewTab={false} />)

      expect(screen.queryByText(/ms$/)).not.toBeInTheDocument()
    })
  })

  describe('Link behavior', () => {
    it('should render as link when not in edit mode', () => {
      const service = createMockService({ url: 'https://example.com' })
      render(<ServiceListItem service={service} openInNewTab={false} />)

      const link = screen.getByRole('link')
      expect(link).toHaveAttribute('href', 'https://example.com')
    })

    it('should open in new tab when openInNewTab is true', () => {
      const service = createMockService()
      render(<ServiceListItem service={service} openInNewTab={true} />)

      const link = screen.getByRole('link')
      expect(link).toHaveAttribute('target', '_blank')
      expect(link).toHaveAttribute('rel', 'noopener noreferrer')
    })

    it('should open in same tab when openInNewTab is false', () => {
      const service = createMockService()
      render(<ServiceListItem service={service} openInNewTab={false} />)

      const link = screen.getByRole('link')
      expect(link).toHaveAttribute('target', '_self')
      expect(link).not.toHaveAttribute('rel')
    })

    it('should render as div when in edit mode', () => {
      const service = createMockService()
      render(<ServiceListItem service={service} openInNewTab={false} isEditMode={true} />)

      expect(screen.queryByRole('link')).not.toBeInTheDocument()
    })
  })

  describe('Edit mode', () => {
    it('should show drag handle in edit mode', () => {
      const service = createMockService()
      render(<ServiceListItem service={service} openInNewTab={false} isEditMode={true} />)

      // Drag handle has cursor-grab class
      const dragHandle = document.querySelector('.cursor-grab')
      expect(dragHandle).toBeInTheDocument()
    })

    it('should not show drag handle when not in edit mode', () => {
      const service = createMockService()
      render(<ServiceListItem service={service} openInNewTab={false} isEditMode={false} />)

      const dragHandle = document.querySelector('.cursor-grab')
      expect(dragHandle).not.toBeInTheDocument()
    })

    it('should apply dashed border in edit mode', () => {
      const service = createMockService()
      const { container } = render(
        <ServiceListItem service={service} openInNewTab={false} isEditMode={true} />
      )

      const element = container.firstChild
      expect(element).toHaveClass('border-dashed')
    })

    it('should not apply dashed border when not in edit mode', () => {
      const service = createMockService()
      const { container } = render(
        <ServiceListItem service={service} openInNewTab={false} isEditMode={false} />
      )

      const element = container.firstChild
      expect(element).not.toHaveClass('border-dashed')
    })

    it('should apply ring styling when dragging', () => {
      const service = createMockService()
      const { container } = render(
        <ServiceListItem
          service={service}
          openInNewTab={false}
          isEditMode={true}
          isDragging={true}
        />
      )

      const element = container.firstChild
      expect(element).toHaveClass('ring-2')
    })

    it('should spread drag handle props', () => {
      const service = createMockService()
      const dragHandleProps = {
        'data-testid': 'drag-handle',
        tabIndex: 0,
      }
      render(
        <ServiceListItem
          service={service}
          openInNewTab={false}
          isEditMode={true}
          dragHandleProps={dragHandleProps}
        />
      )

      expect(screen.getByTestId('drag-handle')).toBeInTheDocument()
    })
  })

  describe('Status indicator', () => {
    it('should show online status with success color', () => {
      const service = createMockService({ status: 'online' })
      const { container } = render(<ServiceListItem service={service} openInNewTab={false} />)

      const statusDot = container.querySelector('.bg-success')
      expect(statusDot).toBeInTheDocument()
    })

    it('should show offline status with error color', () => {
      const service = createMockService({ status: 'offline' })
      const { container } = render(<ServiceListItem service={service} openInNewTab={false} />)

      const statusDot = container.querySelector('.bg-error')
      expect(statusDot).toBeInTheDocument()
    })

    it('should show unknown status with warning color', () => {
      const service = createMockService({ status: 'unknown' })
      const { container } = render(<ServiceListItem service={service} openInNewTab={false} />)

      const statusDot = container.querySelector('.bg-warning')
      expect(statusDot).toBeInTheDocument()
    })

    it('should display status text', () => {
      const service = createMockService({ status: 'online' })
      render(<ServiceListItem service={service} openInNewTab={false} />)

      expect(screen.getByText('online')).toBeInTheDocument()
    })
  })

  describe('Styling', () => {
    it('should apply base card classes', () => {
      const service = createMockService()
      const { container } = render(<ServiceListItem service={service} openInNewTab={false} />)

      const element = container.firstChild
      expect(element).toHaveClass('bg-card')
      expect(element).toHaveClass('border-card-border')
      expect(element).toHaveClass('rounded-lg')
    })

    it('should apply hover classes when not in edit mode', () => {
      const service = createMockService()
      const { container } = render(
        <ServiceListItem service={service} openInNewTab={false} isEditMode={false} />
      )

      const element = container.firstChild
      expect(element).toHaveClass('hover:border-primary')
      expect(element).toHaveClass('hover:shadow-md')
    })

    it('should not have redundant hover class in edit mode', () => {
      const service = createMockService()
      const { container } = render(
        <ServiceListItem service={service} openInNewTab={false} isEditMode={true} />
      )

      const element = container.firstChild
      // In edit mode, only border-dashed should be applied, no hover classes
      expect(element).toHaveClass('border-dashed')
      // Should not have shadow-md hover (which is only for non-edit mode)
      expect(element).not.toHaveClass('hover:shadow-md')
    })
  })
})
