-- Back to one dashboard per user: the tiles and groups of all boards stay
-- and show together. Only the default board's default group stays default.
DROP INDEX IF EXISTS idx_groups_default_per_board;
UPDATE groups SET is_default = FALSE
WHERE is_default = TRUE AND board_id NOT IN (SELECT id FROM boards WHERE is_default = TRUE);
CREATE UNIQUE INDEX IF NOT EXISTS idx_groups_default_per_user ON groups(user_id) WHERE is_default = TRUE;

DROP INDEX IF EXISTS idx_groups_board_id;
DROP INDEX IF EXISTS idx_services_board_id;
DROP INDEX IF EXISTS idx_widgets_board_id;
ALTER TABLE groups DROP COLUMN IF EXISTS board_id;
ALTER TABLE services DROP COLUMN IF EXISTS board_id;
ALTER TABLE widgets DROP COLUMN IF EXISTS board_id;
DROP TABLE IF EXISTS boards;
