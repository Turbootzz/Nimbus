-- Add OAuth support to users table

-- Add provider column (local, google, github, discord)
ALTER TABLE users
ADD COLUMN provider VARCHAR(50) DEFAULT 'local';

-- Add provider_id column (external user ID from OAuth provider)
ALTER TABLE users
ADD COLUMN provider_id VARCHAR(255);

-- Add avatar_url column (profile picture URL from OAuth provider)
ALTER TABLE users
ADD COLUMN avatar_url TEXT;

-- Add email_verified column
ALTER TABLE users
ADD COLUMN email_verified BOOLEAN DEFAULT FALSE;

-- Make password nullable (OAuth users won't have passwords)
ALTER TABLE users
ALTER COLUMN password DROP NOT NULL;

-- Update existing users to have verified emails
UPDATE users
SET email_verified = TRUE
WHERE provider = 'local' AND password IS NOT NULL;

-- Create unique constraint on provider and provider_id combination
-- This ensures we can't have duplicate OAuth accounts
ALTER TABLE users
ADD CONSTRAINT unique_provider_user UNIQUE (provider, provider_id);

-- Create index on provider_id for faster lookups
CREATE INDEX idx_users_provider_id ON users(provider_id);

-- Create index on provider for filtering
CREATE INDEX idx_users_provider ON users(provider);

-- Add check constraint to ensure OAuth users have provider_id
-- and local users have password
ALTER TABLE users
ADD CONSTRAINT check_auth_method CHECK (
    (provider = 'local' AND password IS NOT NULL) OR
    (provider != 'local' AND provider_id IS NOT NULL)
);
