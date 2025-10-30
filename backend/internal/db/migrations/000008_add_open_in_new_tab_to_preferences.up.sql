-- Add open_in_new_tab preference to user_preferences table
ALTER TABLE user_preferences ADD COLUMN IF NOT EXISTS open_in_new_tab BOOLEAN NOT NULL DEFAULT true;

-- Add comment for clarity
COMMENT ON COLUMN user_preferences.open_in_new_tab IS 'Whether to open service links in a new tab (default: true)';
