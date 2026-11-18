// Package testutil holds helpers for the SQLite schemas of the tests
package testutil

import (
	"database/sql"
	"testing"
)

// AddBoards gives a SQLite test database what migration 000041 gives
// Postgres: the boards table and board_id on groups, services and widgets
// (those the schema has so far, with a user_id; call it again after creating
// more of them).
// Postgres requires board_id; here rows without one are put on the default
// board of their user, creating that board when needed: once for the rows
// that exist, like the migration's backfill, and by a trigger for rows a
// test inserts later.
func AddBoards(t testing.TB, db *sql.DB) {
	t.Helper()
	exec := func(query string) {
		t.Helper()
		if _, err := db.Exec(query); err != nil {
			t.Fatalf("Failed to add the boards schema: %v", err)
		}
	}
	exec(`
		CREATE TABLE IF NOT EXISTS boards (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			name TEXT NOT NULL,
			slug TEXT NOT NULL,
			position INTEGER NOT NULL DEFAULT 0,
			is_default INTEGER NOT NULL DEFAULT 0,
			is_shared INTEGER NOT NULL DEFAULT 0,
			visibility TEXT NOT NULL DEFAULT 'all',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_boards_user_slug ON boards(user_id, slug);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_boards_default_per_user ON boards(user_id) WHERE is_default = 1;
	`)
	for _, table := range []string{"groups", "services", "widgets"} {
		var todo int
		err := db.QueryRow(`
			SELECT COUNT(*) FROM sqlite_master m
			WHERE m.type = 'table' AND m.name = ?
			  AND EXISTS (SELECT 1 FROM pragma_table_info(m.name) WHERE name = 'user_id')
			  AND NOT EXISTS (SELECT 1 FROM pragma_table_info(m.name) WHERE name = 'board_id')`, table).Scan(&todo)
		if err != nil {
			t.Fatalf("Failed to look up table %s: %v", table, err)
		}
		if todo == 0 {
			continue
		}
		defaultBoard := `(SELECT id FROM boards WHERE user_id = ` + table + `.user_id AND is_default = 1)`
		exec(`
			ALTER TABLE ` + table + ` ADD COLUMN board_id TEXT;
			INSERT INTO boards (id, user_id, name, slug, is_default)
			SELECT lower(hex(randomblob(16))), user_id, 'Home', 'home', 1 FROM ` + table + `
			WHERE true GROUP BY user_id ON CONFLICT DO NOTHING;
			UPDATE ` + table + ` SET board_id = ` + defaultBoard + `;
			CREATE TRIGGER ` + table + `_default_board AFTER INSERT ON ` + table + `
			WHEN NEW.board_id IS NULL
			BEGIN
				INSERT INTO boards (id, user_id, name, slug, is_default)
				VALUES (lower(hex(randomblob(16))), NEW.user_id, 'Home', 'home', 1) ON CONFLICT DO NOTHING;
				UPDATE ` + table + ` SET board_id = ` + defaultBoard + ` WHERE rowid = NEW.rowid;
			END;
		`)
	}
}
