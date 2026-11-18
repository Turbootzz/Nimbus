-- Notification targets (ntfy, Gotify, Telegram, Pushover) need settings and secrets.
-- options: non-secret settings (ntfy priority, Telegram chat id)
-- credentials_enc: AES-256-GCM encrypted JSON (see utils/crypto.go), like integrations
ALTER TABLE webhooks ADD COLUMN options JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE webhooks ADD COLUMN credentials_enc BYTEA;
