-- Add last_activity_at to users table with timezone support
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_activity_at TIMESTAMPTZ;

-- Set initial value to created_at for existing users
UPDATE users SET last_activity_at = created_at WHERE last_activity_at IS NULL;
