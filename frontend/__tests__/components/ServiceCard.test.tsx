import { fireEvent, render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import '@testing-library/jest-dom'
import ServiceCard from '@/components/ServiceCard'
import type { CardSize, Service } from '@/types'

const makeService = (card_size: CardSize): Service => ({
  id: 's1',
  name: 'Plex',
  url: 'https://plex.test',
  icon: '🎬',
  icon_type: 'emoji',
  status: 'online',
  position: 0,
  card_size,
  monitoring_enabled: true,
  created_at: '2026-01-01T00:00:00Z',
})

describe('ServiceCard edit button', () => {
  it.each<CardSize>(['1x1', '2x1', '2x2'])(
    'opens the editor from a %s card without resizing it',
    (size) => {
      const service = makeService(size)
      const onEdit = vi.fn()
      const onSizeChange = vi.fn()
      render(
        <ServiceCard
          service={service}
          openInNewTab={false}
          isEditMode
          onEdit={onEdit}
          onSizeChange={onSizeChange}
        />
      )

      fireEvent.click(screen.getByLabelText('Edit Plex'))
      expect(onEdit).toHaveBeenCalledWith(service)
      expect(onSizeChange).not.toHaveBeenCalled()
    }
  )

  it('has no edit button outside edit mode', () => {
    render(<ServiceCard service={makeService('2x1')} openInNewTab={false} onEdit={() => {}} />)
    expect(screen.queryByLabelText('Edit Plex')).not.toBeInTheDocument()
  })
})
