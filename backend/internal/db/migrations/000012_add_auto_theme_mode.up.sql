-- Add 'auto' option to theme_mode and change default to 'auto'

-- Remove the old constraint
ALTER TABLE user_preferences DROP CONSTRAINT IF EXISTS user_preferences_theme_mode_check;

-- Add new constraint that includes 'auto'
ALTER TABLE user_preferences ADD CONSTRAINT user_preferences_theme_mode_check
    CHECK (theme_mode IN ('light', 'dark', 'auto'));

-- Change the default value to 'auto'
ALTER TABLE user_preferences ALTER COLUMN theme_mode SET DEFAULT 'auto';

-- Update existing NULL values to 'auto' (if any exist)
UPDATE user_preferences SET theme_mode = 'auto' WHERE theme_mode IS NULL;
