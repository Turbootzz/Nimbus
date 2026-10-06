-- The rows added by the up migration only hold defaults, so they stay
ALTER TABLE user_preferences ALTER COLUMN layout_mode SET DEFAULT 'classic';
