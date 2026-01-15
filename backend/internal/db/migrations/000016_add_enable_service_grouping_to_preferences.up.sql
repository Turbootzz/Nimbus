-- Add enable_service_grouping to user_preferences (default true)
ALTER TABLE user_preferences ADD COLUMN IF NOT EXISTS enable_service_grouping BOOLEAN NOT NULL DEFAULT TRUE;
