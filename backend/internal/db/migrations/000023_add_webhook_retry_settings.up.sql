-- Add per-webhook retry settings for notification verification
ALTER TABLE webhooks ADD COLUMN retry_count INT NOT NULL DEFAULT 0;
ALTER TABLE webhooks ADD COLUMN retry_delay_seconds INT NOT NULL DEFAULT 30;
