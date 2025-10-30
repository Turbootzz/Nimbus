-- Add open_in_new_tab preference to user_preferences table
-- Default: true (open service links in new tab)
ALTER TABLE user_preferences ADD COLUMN IF NOT EXISTS open_in_new_tab BOOLEAN NOT NULL DEFAULT true;
