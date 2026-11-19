-- Everyone who is already here keeps what they see: users without a
-- preferences row get one first, so the new defaults below only reach
-- accounts created from now on.
INSERT INTO user_preferences (user_id)
SELECT u.id FROM users u
WHERE NOT EXISTS (SELECT 1 FROM user_preferences p WHERE p.user_id = u.id);

-- How groups show on the dashboard. NULL follows the layout: tabs in
-- classic, sections in canvas. Existing users keep their tabs.
ALTER TABLE user_preferences
  ADD COLUMN IF NOT EXISTS group_display VARCHAR(10) CHECK (group_display IN ('tabs', 'sections'));
UPDATE user_preferences SET group_display = 'tabs';

-- New users start on the canvas, which wants glass cards
ALTER TABLE user_preferences ALTER COLUMN card_opacity SET DEFAULT 75;
ALTER TABLE user_preferences ALTER COLUMN card_blur SET DEFAULT 12;

-- Tiles of a group are full cards or one-line chips
ALTER TABLE groups
  ADD COLUMN IF NOT EXISTS tile_style VARCHAR(10) NOT NULL DEFAULT 'cards' CHECK (tile_style IN ('cards', 'chips'));
