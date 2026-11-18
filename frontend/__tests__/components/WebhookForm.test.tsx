import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import '@testing-library/jest-dom'
import { WebhookForm } from '@/components/WebhookForm'
import type { Webhook } from '@/types'

const saved = (overrides: Partial<Webhook>): Webhook => ({
  id: 'webhook-1',
  name: 'Phone',
  url: 'https://api.telegram.org',
  enabled: true,
  triggers: { on_offline: true, on_online: false },
  format: 'telegram',
  options: { chat_id: '42' },
  has_credentials: true,
  retry_count: 0,
  retry_delay_seconds: 30,
  consecutive_failures: 0,
  total_sent: 0,
  total_failed: 0,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
  ...overrides,
})

const type = (label: string | RegExp, value: string) =>
  fireEvent.change(screen.getByLabelText(label), { target: { value } })

describe('WebhookForm', () => {
  it('shows the fields of the chosen type', () => {
    render(<WebhookForm onSubmit={vi.fn()} onClose={() => {}} />)
    expect(screen.getByLabelText('Webhook URL')).toBeInTheDocument()
    expect(screen.queryByLabelText('Bot token')).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Telegram' }))
    expect(screen.queryByLabelText('Webhook URL')).not.toBeInTheDocument()
    expect(screen.getByLabelText('Bot token')).toHaveAttribute('type', 'password')
    expect(screen.getByLabelText('Chat ID')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Pushover' }))
    expect(screen.getByLabelText('App token')).toBeInTheDocument()
    expect(screen.getByLabelText('User key')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'ntfy' }))
    expect(screen.getByLabelText('Topic URL')).toBeInTheDocument()
    expect(screen.getByLabelText('Access token (optional)')).toBeInTheDocument()
    expect(screen.getByLabelText('Priority')).toHaveValue('3')
  })

  it('marks each invalid field and does not submit', () => {
    const onSubmit = vi.fn()
    render(<WebhookForm onSubmit={onSubmit} onClose={() => {}} />)
    fireEvent.click(screen.getByRole('button', { name: 'Telegram' }))
    fireEvent.click(screen.getByRole('button', { name: 'Create Webhook' }))

    expect(onSubmit).not.toHaveBeenCalled()
    for (const [label, message] of [
      ['Name', 'Name is required'],
      ['Bot token', 'Bot token is required'],
      ['Chat ID', 'Chat ID is required'],
    ]) {
      const input = screen.getByLabelText(label)
      expect(input).toHaveAttribute('aria-invalid', 'true')
      expect(input).toHaveAccessibleDescription(message)
    }
  })

  it('submits a Gotify target', async () => {
    const onSubmit = vi.fn().mockResolvedValue(null)
    render(<WebhookForm onSubmit={onSubmit} onClose={() => {}} />)
    fireEvent.click(screen.getByRole('button', { name: 'Gotify' }))
    type('Name', 'Gotify')
    type('Server URL', 'http://192.168.1.10:8080')
    type('App token', 'AppToken.123')
    fireEvent.click(screen.getByRole('button', { name: 'Create Webhook' }))

    await waitFor(() =>
      expect(onSubmit).toHaveBeenCalledWith({
        name: 'Gotify',
        format: 'gotify',
        url: 'http://192.168.1.10:8080',
        credentials: { token: 'AppToken.123' },
        triggers: { on_offline: true, on_online: false },
        retry_count: 0,
        retry_delay_seconds: 30,
      })
    )
  })

  it('shows server validation errors at their fields', async () => {
    const onSubmit = vi.fn().mockResolvedValue({
      message: 'Bot token is not valid',
      fields: { token: 'Bot token is not valid' },
    })
    render(<WebhookForm onSubmit={onSubmit} onClose={() => {}} />)
    fireEvent.click(screen.getByRole('button', { name: 'Telegram' }))
    type('Name', 'Phone')
    type('Bot token', 'nope')
    type('Chat ID', '42')
    fireEvent.click(screen.getByRole('button', { name: 'Create Webhook' }))

    await waitFor(() =>
      expect(screen.getByLabelText('Bot token')).toHaveAccessibleDescription(
        'Bot token is not valid'
      )
    )
    expect(screen.getByRole('button', { name: 'Create Webhook' })).toBeEnabled()
  })

  it('keeps saved secrets when editing and never shows them', async () => {
    const onSubmit = vi.fn().mockResolvedValue(null)
    render(<WebhookForm webhook={saved({})} onSubmit={onSubmit} onClose={() => {}} />)

    expect(screen.getByLabelText('Bot token')).toHaveValue('')
    expect(screen.getByLabelText('Chat ID')).toHaveValue('42')
    type('Chat ID', '43')
    fireEvent.click(screen.getByRole('button', { name: 'Save Changes' }))

    await waitFor(() => expect(onSubmit).toHaveBeenCalled())
    const request = onSubmit.mock.calls[0][0]
    expect(request.options).toEqual({ chat_id: '43' })
    expect('credentials' in request).toBe(false)
  })

  it('asks for new secrets when the type changes', () => {
    const onSubmit = vi.fn()
    render(<WebhookForm webhook={saved({})} onSubmit={onSubmit} onClose={() => {}} />)
    fireEvent.click(screen.getByRole('button', { name: 'Pushover' }))
    fireEvent.click(screen.getByRole('button', { name: 'Save Changes' }))

    expect(onSubmit).not.toHaveBeenCalled()
    expect(screen.getByLabelText('App token')).toHaveAccessibleDescription('App token is required')
  })

  it('can remove a saved ntfy token', async () => {
    const onSubmit = vi.fn().mockResolvedValue(null)
    const ntfy = saved({ format: 'ntfy', url: 'https://ntfy.sh/alerts', options: { priority: 4 } })
    render(<WebhookForm webhook={ntfy} onSubmit={onSubmit} onClose={() => {}} />)

    expect(screen.getByLabelText('Priority')).toHaveValue('4')
    fireEvent.click(screen.getByLabelText('Remove the saved access token'))
    fireEvent.click(screen.getByRole('button', { name: 'Save Changes' }))

    await waitFor(() => expect(onSubmit).toHaveBeenCalled())
    expect(onSubmit.mock.calls[0][0].credentials).toEqual({})
  })

  it('closes on Escape', () => {
    const onClose = vi.fn()
    render(<WebhookForm onSubmit={vi.fn()} onClose={onClose} />)
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledTimes(1)
  })
})
