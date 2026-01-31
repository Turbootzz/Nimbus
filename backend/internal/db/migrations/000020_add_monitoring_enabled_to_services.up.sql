-- Add monitoring_enabled column to services table
-- Default TRUE maintains backward compatibility (existing services continue to be monitored)
ALTER TABLE services ADD COLUMN monitoring_enabled BOOLEAN NOT NULL DEFAULT TRUE;

-- Partial index for efficient filtering in health check queries
CREATE INDEX IF NOT EXISTS idx_services_monitoring_enabled ON services(monitoring_enabled) WHERE monitoring_enabled = TRUE;
