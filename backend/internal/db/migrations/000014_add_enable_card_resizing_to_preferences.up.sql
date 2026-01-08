-- Add enable_card_resizing to user_preferences table
-- Default true to keep card resizing enabled for existing users
ALTER TABLE user_preferences ADD COLUMN IF NOT EXISTS enable_card_resizing BOOLEAN NOT NULL DEFAULT true;
