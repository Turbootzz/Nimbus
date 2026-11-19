-- The rows added by the up migration only hold defaults, so they stay
ALTER TABLE groups DROP COLUMN IF EXISTS tile_style;
ALTER TABLE user_preferences ALTER COLUMN card_blur SET DEFAULT 0;
ALTER TABLE user_preferences ALTER COLUMN card_opacity SET DEFAULT 100;
ALTER TABLE user_preferences DROP COLUMN IF EXISTS group_display;
