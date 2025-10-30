-- Remove position column from services table
DROP INDEX IF EXISTS idx_services_position;
ALTER TABLE services DROP COLUMN IF EXISTS position;
