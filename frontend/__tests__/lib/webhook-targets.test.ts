import { describe, it, expect } from 'vitest'
import {
  WEBHOOK_TARGETS,
  buildWebhookRequest,
  getWebhookTarget,
  validateWebhookForm,
  type WebhookFormValues,
} from '@/lib/webhook-targets'

const base: WebhookFormValues = {
  name: 'Alerts',
  format: 'generic',
  url: 'https://example.com/hook',
  token: '',
  userKey: '',
  chatId: '',
  priority: 3,
  removeToken: false,
  verifyTls: true,
  onOffline: true,
  onOnline: false,
  retryCount: 0,
  retryDelaySeconds: 30,
}

const form = (overrides: Partial<WebhookFormValues>): WebhookFormValues => ({
  ...base,
  ...overrides,
})

describe('getWebhookTarget', () => {
  it('knows every format and falls back to generic', () => {
    expect(WEBHOOK_TARGETS.map((target) => target.value)).toEqual([
      'generic',
      'discord',
      'slack',
      'ntfy',
      'gotify',
      'telegram',
      'pushover',
    ])
    expect(getWebhookTarget('gotify').label).toBe('Gotify')
    expect(getWebhookTarget('unknown').value).toBe('generic')
  })
})

describe('validateWebhookForm', () => {
  it('accepts a plain webhook', () => {
    expect(validateWebhookForm(base, false)).toEqual({})
  })

  it('reports name, URL and trigger errors per field', () => {
    const errors = validateWebhookForm(
      form({ name: ' ', url: 'example.com', onOffline: false }),
      false
    )
    expect(Object.keys(errors).sort()).toEqual(['name', 'triggers', 'url'])
  })

  it.each([
    ['https://ntfy.sh/alerts', true],
    ['http://192.168.1.5:8080/ntfy/home_lab-1/', true],
    ['https://ntfy.sh', false],
    ['https://ntfy.sh/', false],
    ['https://ntfy.sh/bad%20topic', false],
  ])('checks the ntfy topic in %s', (url, valid) => {
    const errors = validateWebhookForm(form({ format: 'ntfy', url }), false)
    expect('url' in errors).toBe(!valid)
  })

  it('treats the ntfy token as optional and the Gotify token as required', () => {
    expect(validateWebhookForm(form({ format: 'ntfy', url: 'https://ntfy.sh/a' }), false)).toEqual(
      {}
    )
    expect(validateWebhookForm(form({ format: 'gotify' }), false)).toEqual({
      token: 'App token is required',
    })
  })

  it('needs a bot token and a valid chat ID for Telegram, but no URL', () => {
    expect(validateWebhookForm(form({ format: 'telegram', url: '' }), false)).toEqual({
      token: 'Bot token is required',
      chat_id: 'Chat ID is required',
    })
    const errors = validateWebhookForm(
      form({ format: 'telegram', url: '', token: '1:a', chatId: 'my chat' }),
      false
    )
    expect(errors).toEqual({ chat_id: 'Chat ID must be a number or a public @channelname' })
    for (const chatId of ['12345', '-1001234567890', '@homelab']) {
      expect(
        validateWebhookForm(form({ format: 'telegram', url: '', token: '1:a', chatId }), false)
      ).toEqual({})
    }
  })

  it('needs both Pushover keys', () => {
    expect(validateWebhookForm(form({ format: 'pushover', url: '', token: 'app' }), false)).toEqual(
      { user_key: 'User key is required' }
    )
  })

  it('lets saved secrets stay blank, but replaces them as a set', () => {
    const pushover = form({ format: 'pushover', url: '' })
    expect(validateWebhookForm(pushover, true)).toEqual({})
    expect(validateWebhookForm({ ...pushover, userKey: 'new' }, true)).toEqual({
      token: 'App token is required',
    })
  })
})

describe('buildWebhookRequest', () => {
  it('sends a plain webhook without options or credentials', () => {
    expect(
      buildWebhookRequest(form({ name: ' Alerts ', url: ' https://example.com/hook ' }), false)
    ).toEqual({
      name: 'Alerts',
      format: 'generic',
      url: 'https://example.com/hook',
      verify_tls: true,
      triggers: { on_offline: true, on_online: false },
      retry_count: 0,
      retry_delay_seconds: 30,
    })
  })

  it('sends the ntfy priority and token', () => {
    const request = buildWebhookRequest(
      form({ format: 'ntfy', url: 'https://ntfy.sh/a', token: ' tk_1 ', priority: 5 }),
      false
    )
    expect(request.options).toEqual({ priority: 5 })
    expect(request.credentials).toEqual({ token: 'tk_1' })
  })

  it('sends verify_tls only for types that can be self-hosted', () => {
    const gotify = form({ format: 'gotify', token: 'app', verifyTls: false })
    expect(buildWebhookRequest(gotify, false).verify_tls).toBe(false)
    const discord = form({ format: 'discord', verifyTls: false })
    expect('verify_tls' in buildWebhookRequest(discord, false)).toBe(false)
  })

  it('sends no URL for Telegram and Pushover', () => {
    const telegram = buildWebhookRequest(
      form({ format: 'telegram', url: 'https://ignored.example', token: '1:a', chatId: ' 42 ' }),
      false
    )
    expect(telegram.url).toBeUndefined()
    expect(telegram.options).toEqual({ chat_id: '42' })
    expect(telegram.credentials).toEqual({ token: '1:a' })

    const pushover = buildWebhookRequest(
      form({ format: 'pushover', token: 'app', userKey: 'user' }),
      false
    )
    expect(pushover.url).toBeUndefined()
    expect(pushover.credentials).toEqual({ token: 'app', user_key: 'user' })
  })

  it('leaves credentials out when the saved ones are kept', () => {
    const request = buildWebhookRequest(form({ format: 'gotify' }), true)
    expect('credentials' in request).toBe(false)
  })

  it('replaces saved credentials once something is typed', () => {
    const request = buildWebhookRequest(form({ format: 'gotify', token: 'new' }), true)
    expect(request.credentials).toEqual({ token: 'new' })
  })

  it('removes a saved ntfy token with empty credentials', () => {
    const request = buildWebhookRequest(
      form({ format: 'ntfy', url: 'https://ntfy.sh/a', removeToken: true }),
      true
    )
    expect(request.credentials).toEqual({})
  })
})
