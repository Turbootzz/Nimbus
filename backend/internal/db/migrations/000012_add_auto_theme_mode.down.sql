-- Revert 'auto' theme mode changes

-- Update any 'auto' values to 'light' before removing the option
UPDATE user_preferences SET theme_mode = 'light' WHERE theme_mode = 'auto';

-- Remove the new constraint
ALTER TABLE user_preferences DROP CONSTRAINT IF EXISTS user_preferences_theme_mode_check;

-- Add back the old constraint (only light and dark)
ALTER TABLE user_preferences ADD CONSTRAINT user_preferences_theme_mode_check
    CHECK (theme_mode IN ('light', 'dark'));

-- Change the default back to 'light'
ALTER TABLE user_preferences ALTER COLUMN theme_mode SET DEFAULT 'light';
