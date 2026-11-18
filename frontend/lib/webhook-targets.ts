import { isValidUrl } from '@/lib/utils/url'
import type { WebhookCreateRequest, WebhookFormat } from '@/types'

// What the webhook form shows and checks per notification type.
// The backend validates again; this only catches mistakes before the request.

interface TextField {
  label: string
  placeholder?: string
  help?: string
}

export interface WebhookTarget {
  value: WebhookFormat
  label: string
  description: string
  url?: TextField // absent: the service has one fixed endpoint
  token?: TextField & { optional?: boolean }
  userKey?: TextField
  chatId?: TextField
  priority?: boolean
  selfHosted?: boolean // may have a self-signed certificate: offer the TLS toggle
}

export const WEBHOOK_TARGETS: WebhookTarget[] = [
  {
    value: 'generic',
    label: 'Generic',
    description: 'Standard JSON payload',
    url: { label: 'Webhook URL', placeholder: 'https://example.com/webhook' },
    selfHosted: true,
  },
  {
    value: 'discord',
    label: 'Discord',
    description: 'Discord-compatible embed format',
    url: { label: 'Webhook URL', placeholder: 'https://discord.com/api/webhooks/...' },
  },
  {
    value: 'slack',
    label: 'Slack',
    description: 'Slack Block Kit format',
    url: { label: 'Webhook URL', placeholder: 'https://hooks.slack.com/services/...' },
  },
  {
    value: 'ntfy',
    label: 'ntfy',
    description: 'Push to an ntfy topic, on ntfy.sh or your own server',
    url: {
      label: 'Topic URL',
      placeholder: 'https://ntfy.sh/my-topic',
      help: 'The server address followed by the topic name.',
    },
    token: {
      label: 'Access token',
      placeholder: 'tk_...',
      help: 'Only needed for protected topics.',
      optional: true,
    },
    priority: true,
    selfHosted: true,
  },
  {
    value: 'gotify',
    label: 'Gotify',
    description: 'Push to a Gotify server',
    url: { label: 'Server URL', placeholder: 'http://192.168.1.10:8080' },
    token: { label: 'App token', help: 'Create an application in Gotify and copy its token.' },
    selfHosted: true,
  },
  {
    value: 'telegram',
    label: 'Telegram',
    description: 'Message from your Telegram bot',
    token: {
      label: 'Bot token',
      placeholder: '123456789:AA...',
      help: 'From @BotFather.',
    },
    chatId: {
      label: 'Chat ID',
      placeholder: '-1001234567890',
      help: 'A user, group or channel ID, or a public @channelname.',
    },
  },
  {
    value: 'pushover',
    label: 'Pushover',
    description: 'Push through Pushover',
    token: { label: 'App token', help: 'The API token of your Pushover application.' },
    userKey: { label: 'User key', help: 'Your user key or a group key.' },
  },
]

export const NTFY_PRIORITIES = [
  { value: 1, label: 'Min' },
  { value: 2, label: 'Low' },
  { value: 3, label: 'Default' },
  { value: 4, label: 'High' },
  { value: 5, label: 'Urgent' },
]

export function getWebhookTarget(format: string): WebhookTarget {
  return WEBHOOK_TARGETS.find((target) => target.value === format) ?? WEBHOOK_TARGETS[0]
}

export interface WebhookFormValues {
  name: string
  format: WebhookFormat
  url: string
  token: string
  userKey: string
  chatId: string
  priority: number
  removeToken: boolean // ntfy: drop the saved access token
  verifyTls: boolean
  onOffline: boolean
  onOnline: boolean
  retryCount: number
  retryDelaySeconds: number
}

// Field names match the backend's "fields" keys, so both error sources
// land under the same input.
export type WebhookFieldErrors = Record<string, string>

const TELEGRAM_CHAT = /^(-?\d{1,20}|@[A-Za-z0-9_]{1,64})$/
const NTFY_TOPIC = /^[-_A-Za-z0-9]{1,64}$/

// keepsCredentials: editing a webhook that has saved secrets for this type,
// so blank secret inputs mean "leave them as they are".
export function validateWebhookForm(
  values: WebhookFormValues,
  keepsCredentials: boolean
): WebhookFieldErrors {
  const target = getWebhookTarget(values.format)
  const errors: WebhookFieldErrors = {}
  const name = values.name.trim()
  const url = values.url.trim()
  const token = values.token.trim()
  const userKey = values.userKey.trim()
  const chatId = values.chatId.trim()

  if (!name) errors.name = 'Name is required'
  else if (name.length > 100) errors.name = 'Name must be 100 characters or less'

  if (target.url) {
    if (!url) errors.url = `${target.url.label} is required`
    else if (!isValidUrl(url)) errors.url = 'Enter a full URL, starting with http:// or https://'
    else if (values.format === 'ntfy') {
      const topic = new URL(url).pathname.replace(/\/$/, '').split('/').pop() ?? ''
      if (!NTFY_TOPIC.test(topic)) {
        errors.url = 'The URL must end with the topic, like https://ntfy.sh/my-topic'
      }
    }
  }

  // Saved secrets are replaced as a set, so typing one means entering all
  const keeping = keepsCredentials && !token && !userKey
  if (!keeping) {
    if (target.token && !target.token.optional && !token) {
      errors.token = `${target.token.label} is required`
    }
    if (target.userKey && !userKey) errors.user_key = `${target.userKey.label} is required`
  }

  if (target.chatId) {
    if (!chatId) errors.chat_id = `${target.chatId.label} is required`
    else if (!TELEGRAM_CHAT.test(chatId)) {
      errors.chat_id = 'Chat ID must be a number or a public @channelname'
    }
  }

  if (!values.onOffline && !values.onOnline) errors.triggers = 'Select at least one trigger event'

  return errors
}

// buildWebhookRequest turns valid form values into the API body. Secrets are
// left out when the saved ones are kept.
export function buildWebhookRequest(
  values: WebhookFormValues,
  keepsCredentials: boolean
): WebhookCreateRequest {
  const target = getWebhookTarget(values.format)
  const token = values.token.trim()
  const userKey = values.userKey.trim()

  const request: WebhookCreateRequest = {
    name: values.name.trim(),
    format: values.format,
    triggers: { on_offline: values.onOffline, on_online: values.onOnline },
    retry_count: values.retryCount,
    retry_delay_seconds: values.retryDelaySeconds,
  }
  if (target.url) request.url = values.url.trim()
  // Other types always verify; the backend enforces that
  if (target.selfHosted) request.verify_tls = values.verifyTls
  if (target.priority) request.options = { priority: values.priority }
  if (target.chatId) request.options = { chat_id: values.chatId.trim() }

  if (target.token) {
    if (values.removeToken && target.token.optional && !token) {
      request.credentials = {}
    } else if (!(keepsCredentials && !token && !userKey)) {
      request.credentials = { token, ...(target.userKey ? { user_key: userKey } : {}) }
    }
  }
  return request
}
