-- Add index on service_id for webhook_logs to optimize queries by service
CREATE INDEX IF NOT EXISTS idx_webhook_logs_service_id ON webhook_logs(service_id, created_at DESC);
