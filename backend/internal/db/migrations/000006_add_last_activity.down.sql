-- Remove last_activity_at column from users table
ALTER TABLE users DROP COLUMN IF EXISTS last_activity_at;
