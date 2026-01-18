-- Remove enable_service_grouping from user_preferences
ALTER TABLE user_preferences DROP COLUMN IF EXISTS enable_service_grouping;
