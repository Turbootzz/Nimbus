-- Revert default card_scale back to 'large'
ALTER TABLE user_preferences
  ALTER COLUMN card_scale SET DEFAULT 'large';
