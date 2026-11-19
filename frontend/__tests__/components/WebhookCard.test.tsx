import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import '@testing-library/jest-dom'
import { WebhookCard } from '@/components/WebhookCard'
import type { Webhook } from '@/types'

describe('WebhookCard', () => {
  it('has a named switch for turning the webhook on and off', () => {
    const onToggle = vi.fn()
    const webhook = {
      id: 'w1',
      name: 'Phone',
      url: 'https://ntfy.sh/x',
      enabled: true,
      triggers: { on_offline: true, on_online: false },
      format: 'ntfy',
      retry_count: 0,
      retry_delay_seconds: 0,
      consecutive_failures: 0,
      total_sent: 0,
      total_failed: 0,
      created_at: '',
      updated_at: '',
    } as unknown as Webhook
    render(
      <WebhookCard
        webhook={webhook}
        onEdit={vi.fn()}
        onDelete={vi.fn()}
        onTest={vi.fn()}
        onToggle={onToggle}
      />
    )
    const toggle = screen.getByRole('switch', { name: 'Phone enabled' })
    expect(toggle).toHaveAttribute('aria-checked', 'true')
    fireEvent.click(toggle)
    expect(onToggle).toHaveBeenCalled()
  })
})
