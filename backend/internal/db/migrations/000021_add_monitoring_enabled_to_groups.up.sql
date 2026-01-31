-- Add monitoring_enabled column to groups table
-- Default TRUE maintains backward compatibility (existing groups continue to be monitored)
-- When a group has monitoring disabled, all services in that group are excluded from health checks and metrics
ALTER TABLE groups ADD COLUMN monitoring_enabled BOOLEAN NOT NULL DEFAULT TRUE;

-- Partial index for efficient filtering when joining with services
CREATE INDEX IF NOT EXISTS idx_groups_monitoring_enabled ON groups(monitoring_enabled) WHERE monitoring_enabled = TRUE;
