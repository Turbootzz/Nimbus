-- New users start on the canvas layout; users who are already here keep
-- classic. Those without a preferences row get one first, or the new default
-- would reach them too. The other columns get their defaults, the same values
-- they saw without a row.
INSERT INTO user_preferences (user_id, layout_mode)
SELECT u.id, 'classic' FROM users u
WHERE NOT EXISTS (SELECT 1 FROM user_preferences p WHERE p.user_id = u.id);

ALTER TABLE user_preferences ALTER COLUMN layout_mode SET DEFAULT 'canvas';
