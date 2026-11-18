-- Boards: a user can have several dashboards. user_id is the owner, also of
-- a shared board (is_shared: every user can see it, only the owner edits it).
-- Safe to run twice: the runner sends this file as one batch, so it is one
-- transaction, and every statement skips what is already there.
CREATE TABLE IF NOT EXISTS boards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL,
    -- URL name, /board/<slug>; unique per owner
    slug VARCHAR(40) NOT NULL,
    position INTEGER NOT NULL DEFAULT 0,
    -- The board /dashboard shows, and where requests without a board go
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    is_shared BOOLEAN NOT NULL DEFAULT FALSE,
    -- Who sees a shared board; only 'all' exists so far
    visibility VARCHAR(10) NOT NULL DEFAULT 'all' CHECK (visibility IN ('all')),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_boards_user_slug ON boards(user_id, slug);
CREATE UNIQUE INDEX IF NOT EXISTS idx_boards_default_per_user ON boards(user_id) WHERE is_default = TRUE;
CREATE INDEX IF NOT EXISTS idx_boards_shared ON boards(is_shared) WHERE is_shared = TRUE;

-- Every existing user gets a default board, also users without any tiles
INSERT INTO boards (user_id, name, slug, is_default)
SELECT u.id, 'Home', 'home', TRUE
FROM users u
WHERE NOT EXISTS (SELECT 1 FROM boards b WHERE b.user_id = u.id AND b.is_default = TRUE);

-- Nullable first, so rows that exist can be moved before the column is required
ALTER TABLE groups ADD COLUMN IF NOT EXISTS board_id UUID REFERENCES boards(id) ON DELETE CASCADE;
ALTER TABLE services ADD COLUMN IF NOT EXISTS board_id UUID REFERENCES boards(id) ON DELETE CASCADE;
ALTER TABLE widgets ADD COLUMN IF NOT EXISTS board_id UUID REFERENCES boards(id) ON DELETE CASCADE;

UPDATE groups t SET board_id = b.id FROM boards b
WHERE t.board_id IS NULL AND b.user_id = t.user_id AND b.is_default = TRUE;
UPDATE services t SET board_id = b.id FROM boards b
WHERE t.board_id IS NULL AND b.user_id = t.user_id AND b.is_default = TRUE;
UPDATE widgets t SET board_id = b.id FROM boards b
WHERE t.board_id IS NULL AND b.user_id = t.user_id AND b.is_default = TRUE;

ALTER TABLE groups ALTER COLUMN board_id SET NOT NULL;
ALTER TABLE services ALTER COLUMN board_id SET NOT NULL;
ALTER TABLE widgets ALTER COLUMN board_id SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_groups_board_id ON groups(board_id);
CREATE INDEX IF NOT EXISTS idx_services_board_id ON services(board_id);
CREATE INDEX IF NOT EXISTS idx_widgets_board_id ON widgets(board_id);

-- Every board has its own default group
DROP INDEX IF EXISTS idx_groups_default_per_user;
CREATE UNIQUE INDEX IF NOT EXISTS idx_groups_default_per_board ON groups(board_id) WHERE is_default = TRUE;
