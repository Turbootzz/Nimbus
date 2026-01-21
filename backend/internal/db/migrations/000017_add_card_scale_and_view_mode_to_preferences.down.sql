ALTER TABLE user_preferences
  DROP COLUMN IF EXISTS card_scale,
  DROP COLUMN IF EXISTS view_mode;
