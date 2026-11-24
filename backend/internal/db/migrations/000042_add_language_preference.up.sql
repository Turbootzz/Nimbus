-- The interface language per user. NULL follows the browser's language.
ALTER TABLE user_preferences
  ADD COLUMN IF NOT EXISTS language VARCHAR(10);
