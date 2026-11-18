# Notifications

Nimbus can tell you when a service goes offline or comes back. Add a target under **Settings, Notifications**, pick what triggers it, and use **Test** on its card to send a sample alert.

Each user manages their own targets (up to 10).

## Targets

| Type | What you enter | Notes |
|---|---|---|
| Generic | Webhook URL | Nimbus posts JSON, see [the payload](#generic-payload). |
| Discord | Webhook URL | Channel settings, Integrations, Webhooks. |
| Slack | Webhook URL | An incoming webhook URL. |
| ntfy | Topic URL, optional access token, priority | The topic URL is the server plus the topic, like `https://ntfy.sh/my-topic` or `http://192.168.1.5/alerts`. The token (`tk_...`) is only needed for protected topics. Priority runs from Min (1) to Urgent (5), Default is 3. |
| Gotify | Server URL, app token | Create an application in Gotify and copy its token. Include the sub path if Gotify runs behind a reverse proxy. Offline alerts use priority 8, recoveries priority 4. |
| Telegram | Bot token, chat ID | Create a bot with [@BotFather](https://t.me/BotFather) and send it a message first, bots can't start a chat. The chat ID is a user, group or channel ID (groups are negative), or a public `@channelname`. |
| Pushover | App token, user key | Create an application on pushover.net for the app token. The user key is on your Pushover dashboard; a group key works too. |

The Nimbus server sends the notifications, so ntfy and Gotify URLs must be reachable from the server, not from your browser. Addresses on your LAN are fine. A server with a self-signed certificate needs plain `http://` or a certificate the Nimbus host trusts.

## Secrets

Tokens and keys are encrypted with `ENCRYPTION_KEY` (AES-256-GCM) before they are stored, the same way integration credentials are. See [CONFIGURATION.md](CONFIGURATION.md).

- The API and the settings page never show a saved token again. When you edit a target, leave the token fields blank to keep what is saved, or enter new values to replace them.
- Tokens are not written to the server log or the delivery log.
- If `ENCRYPTION_KEY` changes, saved tokens can't be read anymore. The target's test then says so; edit it and enter the tokens again.

Discord, Slack and generic webhook URLs are secrets too, but they are stored as entered because the URL is the whole configuration.

## Triggers and retries

A target fires when a service goes offline, comes back online, or both.

**Notification retries** guard against false alarms. With retries set to 1 or more, Nimbus waits for the delay and checks the service again, that many times, before sending an offline alert. If the service recovered in the meantime nothing is sent, and the matching "back online" alert is skipped too. Retries don't resend a notification that a target refused; a failed delivery is counted on the card as **Failed**, and the reason the service gave is in the delivery log (`GET /api/v1/webhooks/:id/logs`) and in the result of **Test**.

The same alert for the same service is sent at most once per 5 minutes per target.

## Generic payload

```json
{
  "event": "service_status_changed",
  "service_id": "0b0e7b1e-...",
  "service_name": "Jellyfin",
  "service_url": "http://jellyfin.lan:8096",
  "old_status": "online",
  "new_status": "offline",
  "timestamp": "2026-01-15T12:00:00Z",
  "error": "connection refused"
}
```

`error` is only present when the health check reported one.

## API

Targets are managed with `/api/v1/webhooks` (`GET`, `POST`, `PUT /:id`, `DELETE /:id`, `POST /:id/test`, `GET /:id/logs`).

```json
{
  "name": "Phone",
  "format": "telegram",
  "options": { "chat_id": "-1001234567890" },
  "credentials": { "token": "123456789:AA..." },
  "triggers": { "on_offline": true, "on_online": true }
}
```

- `format`: `generic`, `discord`, `slack`, `ntfy`, `gotify`, `telegram` or `pushover`.
- `url`: required except for `telegram` and `pushover`.
- `options`: `priority` (ntfy, 1-5) or `chat_id` (Telegram).
- `credentials`: `token`, plus `user_key` for Pushover. Responses only carry `has_credentials`. On update, leave `credentials` out to keep the saved secrets; sending it replaces them, and `{}` removes an ntfy token.
- Validation errors come back as `{"error": "...", "fields": {"token": "..."}}`.
