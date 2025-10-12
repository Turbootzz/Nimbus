-- Drop the index first
DROP INDEX IF EXISTS idx_services_response_time;

-- Remove response_time column from services table
ALTER TABLE services DROP COLUMN IF EXISTS response_time;
