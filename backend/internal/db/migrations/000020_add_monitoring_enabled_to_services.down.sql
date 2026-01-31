DROP INDEX IF EXISTS idx_services_monitoring_enabled;
ALTER TABLE services DROP COLUMN monitoring_enabled;
