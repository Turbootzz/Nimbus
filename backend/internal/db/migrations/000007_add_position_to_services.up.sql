-- Add position column to services table for user-defined ordering
ALTER TABLE services ADD COLUMN IF NOT EXISTS position INTEGER DEFAULT 0;

-- Backfill existing services with positions based on creation order
-- Services created first get lower positions (0, 1, 2, ...)
WITH ranked_services AS (
    SELECT
        id,
        user_id,
        ROW_NUMBER() OVER (PARTITION BY user_id ORDER BY created_at ASC) - 1 AS new_position
    FROM services
    WHERE position = 0 OR position IS NULL
)
UPDATE services
SET position = ranked_services.new_position
FROM ranked_services
WHERE services.id = ranked_services.id;

-- Create index on position for efficient ordering queries
CREATE INDEX IF NOT EXISTS idx_services_position ON services(user_id, position);

-- Add comment for clarity
COMMENT ON COLUMN services.position IS 'User-defined position for dashboard ordering (0-based index)';
