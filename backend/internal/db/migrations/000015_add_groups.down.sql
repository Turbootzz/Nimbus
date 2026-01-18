-- Remove group_id from services first (due to foreign key)
ALTER TABLE services DROP COLUMN IF EXISTS group_id;

-- Drop indexes
DROP INDEX IF EXISTS idx_groups_default_per_user;
DROP INDEX IF EXISTS idx_services_group_id;
DROP INDEX IF EXISTS idx_groups_position;
DROP INDEX IF EXISTS idx_groups_user_id;

-- Drop groups table
DROP TABLE IF EXISTS groups;
