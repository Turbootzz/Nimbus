import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import '@testing-library/jest-dom'
import { Toggle } from '@/components/ui/Toggle'

describe('Toggle', () => {
  it('is a named switch also without an id', () => {
    render(<Toggle enabled onChange={vi.fn()} label="List view" description="Compact rows" />)
    const toggle = screen.getByRole('switch', { name: 'List view' })
    expect(toggle).toHaveAttribute('aria-checked', 'true')
    expect(toggle).toHaveAccessibleDescription('Compact rows')
  })
})
