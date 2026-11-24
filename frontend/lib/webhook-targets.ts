import { isValidUrl } from '@/lib/utils/url'
import type { WebhookCreateRequest, WebhookFormat } from '@/types'

// What the webhook form shows and checks per notification type.
// The backend validates again; this only catches mistakes before the request.
// The form's texts are in the notifications.targets messages; the English
// labels here word validateWebhookForm's default messages.

interface TextField {
  label: string
  placeholder?: string
}

export interface WebhookTarget {
  value: WebhookFormat
  label: string // product name; "Generic" is translated by the form
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
    url: { label: 'Webhook URL', placeholder: 'https://example.com/webhook' },
    selfHosted: true,
  },
  {
    value: 'discord',
    label: 'Discord',
    url: { label: 'Webhook URL', placeholder: 'https://discord.com/api/webhooks/...' },
  },
  {
    value: 'slack',
    label: 'Slack',
    url: { label: 'Webhook URL', placeholder: 'https://hooks.slack.com/services/...' },
  },
  {
    value: 'ntfy',
    label: 'ntfy',
    url: { label: 'Topic URL', placeholder: 'https://ntfy.sh/my-topic' },
    token: { label: 'Access token', placeholder: 'tk_...', optional: true },
    priority: true,
    selfHosted: true,
  },
  {
    value: 'gotify',
    label: 'Gotify',
    url: { label: 'Server URL', placeholder: 'http://192.168.1.10:8080' },
    token: { label: 'App token' },
    selfHosted: true,
  },
  {
    value: 'telegram',
    label: 'Telegram',
    token: { label: 'Bot token', placeholder: '123456789:AA...' },
    chatId: { label: 'Chat ID', placeholder: '-1001234567890' },
  },
  {
    value: 'pushover',
    label: 'Pushover',
    token: { label: 'App token' },
    userKey: { label: 'User key' },
  },
]

// ntfy's priorities; key names the form's message
export const NTFY_PRIORITIES = [
  { value: 1, key: 'min' },
  { value: 2, key: 'low' },
  { value: 3, key: 'default' },
  { value: 4, key: 'high' },
  { value: 5, key: 'urgent' },
] as const

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
export type WebhookErrorField = 'name' | 'url' | 'token' | 'user_key' | 'chat_id' | 'triggers'
export type WebhookProblem =
  'required' | 'nameTooLong' | 'invalidUrl' | 'ntfyTopic' | 'invalidChatId' | 'noTrigger'
export type DescribeWebhookProblem = (field: WebhookErrorField, problem: WebhookProblem) => string

const ENGLISH_PROBLEMS: Record<Exclude<WebhookProblem, 'required'>, string> = {
  nameTooLong: 'Name must be 100 characters or less',
  invalidUrl: 'Enter a full URL, starting with http:// or https://',
  ntfyTopic: 'The URL must end with the topic, like https://ntfy.sh/my-topic',
  invalidChatId: 'Chat ID must be a number or a public @channelname',
  noTrigger: 'Select at least one trigger event',
}

// English messages, "<label> is required" with the type's field labels
function inEnglish(target: WebhookTarget): DescribeWebhookProblem {
  const labels: Partial<Record<WebhookErrorField, string>> = {
    name: 'Name',
    url: target.url?.label,
    token: target.token?.label,
    user_key: target.userKey?.label,
    chat_id: target.chatId?.label,
  }
  return (field, problem) =>
    problem === 'required' ? `${labels[field]} is required` : ENGLISH_PROBLEMS[problem]
}

const TELEGRAM_CHAT = /^(-?\d{1,20}|@[A-Za-z0-9_]{1,64})$/
const NTFY_TOPIC = /^[-_A-Za-z0-9]{1,64}$/

// keepsCredentials: editing a webhook that has saved secrets for this type,
// so blank secret inputs mean "leave them as they are". describe words each
// problem; the form passes its translations, the default is English.
export function validateWebhookForm(
  values: WebhookFormValues,
  keepsCredentials: boolean,
  describe: DescribeWebhookProblem = inEnglish(getWebhookTarget(values.format))
): WebhookFieldErrors {
  const target = getWebhookTarget(values.format)
  const errors: WebhookFieldErrors = {}
  const name = values.name.trim()
  const url = values.url.trim()
  const token = values.token.trim()
  const userKey = values.userKey.trim()
  const chatId = values.chatId.trim()

  if (!name) errors.name = describe('name', 'required')
  else if (name.length > 100) errors.name = describe('name', 'nameTooLong')

  if (target.url) {
    if (!url) errors.url = describe('url', 'required')
    else if (!isValidUrl(url)) errors.url = describe('url', 'invalidUrl')
    else if (values.format === 'ntfy') {
      const topic = new URL(url).pathname.replace(/\/$/, '').split('/').pop() ?? ''
      if (!NTFY_TOPIC.test(topic)) errors.url = describe('url', 'ntfyTopic')
    }
  }

  // Saved secrets are replaced as a set, so typing one means entering all
  const keeping = keepsCredentials && !token && !userKey
  if (!keeping) {
    if (target.token && !target.token.optional && !token) {
      errors.token = describe('token', 'required')
    }
    if (target.userKey && !userKey) errors.user_key = describe('user_key', 'required')
  }

  if (target.chatId) {
    if (!chatId) errors.chat_id = describe('chat_id', 'required')
    else if (!TELEGRAM_CHAT.test(chatId)) errors.chat_id = describe('chat_id', 'invalidChatId')
  }

  if (!values.onOffline && !values.onOnline) errors.triggers = describe('triggers', 'noTrigger')

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
