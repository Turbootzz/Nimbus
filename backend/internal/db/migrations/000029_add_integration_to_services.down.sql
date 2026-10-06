DROP INDEX IF EXISTS idx_services_integration_id;
ALTER TABLE services DROP COLUMN IF EXISTS integration_id;
