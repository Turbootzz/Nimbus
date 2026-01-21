-- Add card_scale to user_preferences (default 'large')
-- Values: 'small', 'medium', 'large'
-- Add view_mode to user_preferences (default 'grid')
-- Values: 'grid', 'list'
ALTER TABLE user_preferences
  ADD COLUMN IF NOT EXISTS card_scale VARCHAR(10) NOT NULL DEFAULT 'large',
  ADD COLUMN IF NOT EXISTS view_mode VARCHAR(10) NOT NULL DEFAULT 'grid';
