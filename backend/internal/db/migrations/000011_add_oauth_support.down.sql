-- Rollback OAuth support from users table

-- Drop check constraint
ALTER TABLE users
DROP CONSTRAINT IF EXISTS check_auth_method;

-- Drop indexes
DROP INDEX IF EXISTS idx_users_provider;
DROP INDEX IF EXISTS idx_users_provider_id;

-- Drop unique constraint
ALTER TABLE users
DROP CONSTRAINT IF EXISTS unique_provider_user;

-- Delete OAuth users before setting password to NOT NULL
-- This prevents migration failure when OAuth users (with NULL passwords) exist
DELETE FROM users
WHERE provider != 'local' OR password IS NULL;

-- Make password NOT NULL again (safe now that OAuth users are deleted)
ALTER TABLE users
ALTER COLUMN password SET NOT NULL;

-- Remove OAuth columns
ALTER TABLE users
DROP COLUMN IF EXISTS email_verified;

ALTER TABLE users
DROP COLUMN IF EXISTS avatar_url;

ALTER TABLE users
DROP COLUMN IF EXISTS provider_id;

ALTER TABLE users
DROP COLUMN IF EXISTS provider;
