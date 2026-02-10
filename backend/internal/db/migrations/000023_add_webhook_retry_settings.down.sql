ALTER TABLE webhooks DROP COLUMN IF EXISTS retry_count;
ALTER TABLE webhooks DROP COLUMN IF EXISTS retry_delay_seconds;
