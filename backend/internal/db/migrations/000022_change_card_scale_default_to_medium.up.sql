-- Change default card_scale from 'large' to 'medium' for new users
-- This only affects new rows; existing users keep their current setting
ALTER TABLE user_preferences
  ALTER COLUMN card_scale SET DEFAULT 'medium';
