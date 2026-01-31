DROP INDEX IF EXISTS idx_groups_monitoring_enabled;
ALTER TABLE groups DROP COLUMN monitoring_enabled;
