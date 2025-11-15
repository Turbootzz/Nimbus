-- Add OAuth support to users table

-- Add provider_id column (external user ID from OAuth provider)
ALTER TABLE users
ADD COLUMN provider_id VARCHAR(255);

-- Add avatar_url column (profile picture URL from OAuth provider)
ALTER TABLE users
ADD COLUMN avatar_url TEXT;

-- Add email_verified column
ALTER TABLE users
ADD COLUMN email_verified BOOLEAN DEFAULT FALSE;

-- Add provider column with NOT NULL constraint and default 'local'
-- This ensures the provider column is never NULL, making check constraints reliable
ALTER TABLE users
ADD COLUMN provider VARCHAR(50) NOT NULL DEFAULT 'local';

-- Make password nullable (OAuth users won't have passwords)
ALTER TABLE users
ALTER COLUMN password DROP NOT NULL;

-- Update existing users to have verified emails
UPDATE users
SET email_verified = TRUE
WHERE provider = 'local' AND password IS NOT NULL;

-- Create unique constraint on provider and provider_id combination
-- This ensures we can't have duplicate OAuth accounts
-- Note: NULL values are excluded from unique constraints, so local users with NULL provider_id won't conflict
ALTER TABLE users
ADD CONSTRAINT unique_provider_user UNIQUE (provider, provider_id);

-- Create index on provider_id for faster lookups
CREATE INDEX idx_users_provider_id ON users(provider_id);

-- Create index on provider for filtering
CREATE INDEX idx_users_provider ON users(provider);

-- Add check constraint to ensure OAuth users have provider_id
-- and local users have password
-- Since provider is NOT NULL, this constraint will properly validate all rows
ALTER TABLE users
ADD CONSTRAINT check_auth_method CHECK (
    (provider = 'local' AND password IS NOT NULL) OR
    (provider != 'local' AND provider_id IS NOT NULL)
);
