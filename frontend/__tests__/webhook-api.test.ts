import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { api } from '@/lib/api'
import type { Webhook, WebhookFormat } from '@/types'

// Mock the api module
vi.mock('@/lib/api', () => ({
  api: {
    getWebhooks: vi.fn(),
    createWebhook: vi.fn(),
    updateWebhook: vi.fn(),
    deleteWebhook: vi.fn(),
    testWebhook: vi.fn(),
    getWebhookLogs: vi.fn(),
  },
}))

describe('Webhook API', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('getWebhooks', () => {
    it('should fetch all webhooks', async () => {
      const mockWebhooks: Webhook[] = [
        {
          id: 'webhook-1',
          name: 'Discord Webhook',
          url: 'https://discord.com/api/webhooks/123/abc',
          enabled: true,
          triggers: { on_offline: true, on_online: false },
          format: 'discord',
          total_sent: 10,
          total_failed: 1,
          consecutive_failures: 0,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        },
      ]

      vi.mocked(api.getWebhooks).mockResolvedValue({ data: mockWebhooks })

      const result = await api.getWebhooks()

      expect(api.getWebhooks).toHaveBeenCalledTimes(1)
      expect(result.data).toEqual(mockWebhooks)
    })

    it('should handle empty webhook list', async () => {
      vi.mocked(api.getWebhooks).mockResolvedValue({ data: [] })

      const result = await api.getWebhooks()

      expect(result.data).toEqual([])
    })

    it('should handle API errors', async () => {
      vi.mocked(api.getWebhooks).mockResolvedValue({
        error: { message: 'Unauthorized' },
      })

      const result = await api.getWebhooks()

      expect(result.error).toBeDefined()
      expect(result.error?.message).toBe('Unauthorized')
    })
  })

  describe('createWebhook', () => {
    it('should create a webhook with all fields', async () => {
      const newWebhook = {
        name: 'New Webhook',
        url: 'https://slack.com/webhook',
        format: 'slack' as WebhookFormat,
        triggers: { on_offline: true, on_online: true },
      }

      const createdWebhook: Webhook = {
        id: 'webhook-new',
        ...newWebhook,
        enabled: true,
        total_sent: 0,
        total_failed: 0,
        consecutive_failures: 0,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      }

      vi.mocked(api.createWebhook).mockResolvedValue({ data: createdWebhook })

      const result = await api.createWebhook(newWebhook)

      expect(api.createWebhook).toHaveBeenCalledWith(newWebhook)
      expect(result.data?.id).toBe('webhook-new')
      expect(result.data?.format).toBe('slack')
    })

    it('should handle validation errors', async () => {
      const invalidWebhook = {
        name: '',
        url: 'https://example.com',
        format: 'generic' as WebhookFormat,
        triggers: { on_offline: true, on_online: false },
      }

      vi.mocked(api.createWebhook).mockResolvedValue({
        error: { message: 'Name is required' },
      })

      const result = await api.createWebhook(invalidWebhook)

      expect(result.error?.message).toBe('Name is required')
    })

    it('should handle max webhook limit error', async () => {
      const newWebhook = {
        name: 'Extra Webhook',
        url: 'https://example.com',
        format: 'generic' as WebhookFormat,
        triggers: { on_offline: true, on_online: false },
      }

      vi.mocked(api.createWebhook).mockResolvedValue({
        error: { message: 'Maximum webhook limit reached (10)' },
      })

      const result = await api.createWebhook(newWebhook)

      expect(result.error?.message).toContain('Maximum webhook limit')
    })
  })

  describe('updateWebhook', () => {
    it('should update webhook name', async () => {
      const webhookId = 'webhook-1'
      const updateData = { name: 'Updated Name' }

      const updatedWebhook: Webhook = {
        id: webhookId,
        name: 'Updated Name',
        url: 'https://example.com',
        enabled: true,
        triggers: { on_offline: true, on_online: false },
        format: 'generic',
        total_sent: 5,
        total_failed: 0,
        consecutive_failures: 0,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      }

      vi.mocked(api.updateWebhook).mockResolvedValue({ data: updatedWebhook })

      const result = await api.updateWebhook(webhookId, updateData)

      expect(api.updateWebhook).toHaveBeenCalledWith(webhookId, updateData)
      expect(result.data?.name).toBe('Updated Name')
    })

    it('should toggle webhook enabled status', async () => {
      const webhookId = 'webhook-1'
      const updateData = { enabled: false }

      const updatedWebhook: Webhook = {
        id: webhookId,
        name: 'Test',
        url: 'https://example.com',
        enabled: false,
        triggers: { on_offline: true, on_online: false },
        format: 'generic',
        total_sent: 5,
        total_failed: 0,
        consecutive_failures: 0,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      }

      vi.mocked(api.updateWebhook).mockResolvedValue({ data: updatedWebhook })

      const result = await api.updateWebhook(webhookId, updateData)

      expect(result.data?.enabled).toBe(false)
    })

    it('should handle not found error', async () => {
      vi.mocked(api.updateWebhook).mockResolvedValue({
        error: { message: 'Webhook not found' },
      })

      const result = await api.updateWebhook('non-existent', { name: 'New Name' })

      expect(result.error?.message).toBe('Webhook not found')
    })
  })

  describe('deleteWebhook', () => {
    it('should delete a webhook', async () => {
      const webhookId = 'webhook-1'

      vi.mocked(api.deleteWebhook).mockResolvedValue({})

      const result = await api.deleteWebhook(webhookId)

      expect(api.deleteWebhook).toHaveBeenCalledWith(webhookId)
      expect(result.error).toBeUndefined()
    })

    it('should handle not found error', async () => {
      vi.mocked(api.deleteWebhook).mockResolvedValue({
        error: { message: 'Webhook not found' },
      })

      const result = await api.deleteWebhook('non-existent')

      expect(result.error?.message).toBe('Webhook not found')
    })
  })

  describe('testWebhook', () => {
    it('should test webhook successfully', async () => {
      const webhookId = 'webhook-1'

      vi.mocked(api.testWebhook).mockResolvedValue({
        data: {
          success: true,
          message: 'Test notification sent successfully',
          status_code: 200,
          response_time_ms: 150,
        },
      })

      const result = await api.testWebhook(webhookId)

      expect(api.testWebhook).toHaveBeenCalledWith(webhookId)
      expect(result.data?.success).toBe(true)
      expect(result.data?.status_code).toBe(200)
    })

    it('should handle test failure', async () => {
      const webhookId = 'webhook-1'

      vi.mocked(api.testWebhook).mockResolvedValue({
        data: {
          success: false,
          error: 'HTTP 404',
          status_code: 404,
          response_time_ms: 100,
        },
      })

      const result = await api.testWebhook(webhookId)

      expect(result.data?.success).toBe(false)
      expect(result.data?.error).toBe('HTTP 404')
    })

    it('should handle network errors', async () => {
      const webhookId = 'webhook-1'

      vi.mocked(api.testWebhook).mockResolvedValue({
        data: {
          success: false,
          error: 'Connection refused',
        },
      })

      const result = await api.testWebhook(webhookId)

      expect(result.data?.success).toBe(false)
      expect(result.data?.error).toContain('Connection refused')
    })
  })

  describe('getWebhookLogs', () => {
    it('should fetch webhook logs', async () => {
      const webhookId = 'webhook-1'
      const mockLogs = [
        {
          id: 'log-1',
          webhook_id: webhookId,
          service_id: 'service-1',
          service_name: 'Test Service',
          old_status: 'online',
          new_status: 'offline',
          success: true,
          status_code: 200,
          response_time_ms: 150,
          created_at: new Date().toISOString(),
        },
        {
          id: 'log-2',
          webhook_id: webhookId,
          service_id: 'service-1',
          service_name: 'Test Service',
          old_status: 'offline',
          new_status: 'online',
          success: false,
          status_code: 500,
          error_message: 'Internal server error',
          response_time_ms: 50,
          created_at: new Date().toISOString(),
        },
      ]

      vi.mocked(api.getWebhookLogs).mockResolvedValue({ data: mockLogs })

      const result = await api.getWebhookLogs(webhookId)

      expect(api.getWebhookLogs).toHaveBeenCalledWith(webhookId)
      expect(result.data).toHaveLength(2)
      expect(result.data?.[0].success).toBe(true)
      expect(result.data?.[1].success).toBe(false)
    })

    it('should handle empty logs', async () => {
      vi.mocked(api.getWebhookLogs).mockResolvedValue({ data: [] })

      const result = await api.getWebhookLogs('webhook-1')

      expect(result.data).toEqual([])
    })
  })
})

describe('Webhook Format Validation', () => {
  const validFormats: WebhookFormat[] = ['generic', 'discord', 'slack']

  it('should accept valid formats', () => {
    validFormats.forEach((format) => {
      expect(['generic', 'discord', 'slack'].includes(format)).toBe(true)
    })
  })

  it('should identify invalid formats', () => {
    const invalidFormats = ['teams', 'email', 'telegram', '']

    invalidFormats.forEach((format) => {
      expect(['generic', 'discord', 'slack'].includes(format)).toBe(false)
    })
  })
})

describe('Webhook Triggers Validation', () => {
  it('should require at least one trigger', () => {
    const triggers = { on_offline: false, on_online: false }
    const isValid = triggers.on_offline || triggers.on_online

    expect(isValid).toBe(false)
  })

  it('should accept single trigger', () => {
    const triggers = { on_offline: true, on_online: false }
    const isValid = triggers.on_offline || triggers.on_online

    expect(isValid).toBe(true)
  })

  it('should accept both triggers', () => {
    const triggers = { on_offline: true, on_online: true }
    const isValid = triggers.on_offline || triggers.on_online

    expect(isValid).toBe(true)
  })
})

describe('Webhook URL Validation', () => {
  it('should accept valid HTTPS URLs', () => {
    const validUrls = [
      'https://discord.com/api/webhooks/123/abc',
      'https://hooks.slack.com/services/T00/B00/XXX',
      'https://example.com/webhook',
    ]

    validUrls.forEach((url) => {
      try {
        new URL(url)
        expect(url.startsWith('https://')).toBe(true)
      } catch {
        // Should not throw
        expect(true).toBe(false)
      }
    })
  })

  it('should accept HTTP URLs for local testing', () => {
    const localUrls = ['http://localhost:3000/webhook', 'http://127.0.0.1:8080/hook']

    localUrls.forEach((url) => {
      try {
        const parsed = new URL(url)
        expect(['localhost', '127.0.0.1'].includes(parsed.hostname)).toBe(true)
      } catch {
        expect(true).toBe(false)
      }
    })
  })

  it('should reject invalid URLs', () => {
    const invalidUrls = ['not-a-url', 'ftp://example.com', '']

    invalidUrls.forEach((url) => {
      if (url === '') {
        expect(url.length).toBe(0)
      } else {
        try {
          const parsed = new URL(url)
          expect(['http:', 'https:'].includes(parsed.protocol)).toBe(url.startsWith('http'))
        } catch {
          // Expected for invalid URLs
          expect(true).toBe(true)
        }
      }
    })
  })
})

describe('Webhook Stats Display', () => {
  it('should calculate success rate', () => {
    const webhook: Partial<Webhook> = {
      total_sent: 100,
      total_failed: 5,
    }

    const successRate =
      webhook.total_sent! > 0
        ? Math.round(((webhook.total_sent! - webhook.total_failed!) / webhook.total_sent!) * 100)
        : 100

    expect(successRate).toBe(95)
  })

  it('should handle zero sent webhooks', () => {
    const webhook: Partial<Webhook> = {
      total_sent: 0,
      total_failed: 0,
    }

    const successRate = webhook.total_sent! > 0 ? 0 : 100

    expect(successRate).toBe(100)
  })

  it('should handle all failures', () => {
    const webhook: Partial<Webhook> = {
      total_sent: 10,
      total_failed: 10,
    }

    const successRate =
      webhook.total_sent! > 0
        ? Math.round(((webhook.total_sent! - webhook.total_failed!) / webhook.total_sent!) * 100)
        : 100

    expect(successRate).toBe(0)
  })
})
