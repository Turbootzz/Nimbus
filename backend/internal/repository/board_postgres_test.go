package repository

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/nimbus/backend/internal/db"
	"github.com/nimbus/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/lib/pq"
)

const boardsMigration = "000041_add_boards"

// TestBoardsMigrationOnPostgres runs migration 000041 on a real PostgreSQL
// that already holds users, groups, services and widgets, and then the board
// queries the SQLite tests can't check (ON CONFLICT targets, foreign keys,
// uuid columns).
//
// It only runs when NIMBUS_TEST_POSTGRES_DSN points at an EMPTY, throwaway
// database, which it migrates and writes to:
//
//	docker run -d --name nimbus-pg-test -e POSTGRES_PASSWORD=test -p 127.0.0.1:55432:5432 postgres:18
//	NIMBUS_TEST_POSTGRES_DSN='postgres://postgres:test@127.0.0.1:55432/postgres?sslmode=disable' \
//	  go test ./internal/repository/ -run TestBoardsMigrationOnPostgres -v
func TestBoardsMigrationOnPostgres(t *testing.T) {
	dsn := os.Getenv("NIMBUS_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("NIMBUS_TEST_POSTGRES_DSN is not set")
	}
	database, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { database.Close() })
	ctx := context.Background()
	exec := func(query string, args ...any) {
		t.Helper()
		_, err := database.Exec(query, args...)
		require.NoError(t, err, query)
	}
	count := func(query string, args ...any) int {
		t.Helper()
		var n int
		require.NoError(t, database.QueryRow(query, args...).Scan(&n), query)
		return n
	}

	// The migration runner reads its files relative to the backend directory
	t.Chdir("../..")
	dir := filepath.Join("internal", "db", "migrations")
	files, err := filepath.Glob(filepath.Join(dir, "*.up.sql"))
	require.NoError(t, err)
	sort.Strings(files)
	exec(`CREATE TABLE schema_migrations (version VARCHAR(255) PRIMARY KEY, applied_at TIMESTAMP NOT NULL DEFAULT NOW())`)
	for _, file := range files {
		version := strings.TrimSuffix(filepath.Base(file), ".up.sql")
		if version >= boardsMigration {
			continue
		}
		raw, err := os.ReadFile(file)
		require.NoError(t, err)
		exec(string(raw))
		exec(`INSERT INTO schema_migrations (version) VALUES ($1)`, version)
	}

	// A database as it is before boards: users with nothing, with only a
	// default group, with tiles in and out of groups, and without preferences
	const (
		empty   = "10000000-0000-0000-0000-000000000001"
		grouped = "10000000-0000-0000-0000-000000000002"
		full    = "10000000-0000-0000-0000-000000000003"
		admin   = "10000000-0000-0000-0000-000000000004"
	)
	for i, id := range []string{empty, grouped, full, admin} {
		role := "user"
		if id == admin {
			role = "admin"
		}
		exec(`INSERT INTO users (id, email, name, password, role) VALUES ($1, $2, 'User', 'x', $3)`, id, string(rune('a'+i))+"@example.com", role)
	}
	exec(`INSERT INTO groups (id, user_id, name, is_default) VALUES
		('20000000-0000-0000-0000-000000000002', $1, 'Services', TRUE),
		('20000000-0000-0000-0000-000000000003', $2, 'Services', TRUE),
		('20000000-0000-0000-0000-000000000013', $2, 'Media', FALSE)`, grouped, full)
	exec(`INSERT INTO integrations (id, user_id, kind, name, base_url, options) VALUES
		('50000000-0000-0000-0000-000000000003', $1, 'sonarr', 'Sonarr', 'http://sonarr.lan', '{"library":"tv","empty":""}')`, full)
	exec(`INSERT INTO services (id, user_id, name, url, icon, description, status, position, group_id, integration_id) VALUES
		('30000000-0000-0000-0000-000000000001', $1, 'Plex', 'http://plex.lan', '', '', 'online', 0, '20000000-0000-0000-0000-000000000013', '50000000-0000-0000-0000-000000000003'),
		('30000000-0000-0000-0000-000000000002', $1, 'Router', 'http://router.lan', '', '', 'unknown', 1, NULL, NULL),
		('30000000-0000-0000-0000-000000000003', $2, 'NAS', 'http://nas.lan', '', '', 'offline', 0, NULL, NULL)`, full, admin)
	exec(`INSERT INTO widgets (id, user_id, type, title, group_id, position) VALUES
		('40000000-0000-0000-0000-000000000001', $1, 'clock', 'Clock', NULL, 2),
		('40000000-0000-0000-0000-000000000002', $1, 'markdown', 'Notes', '20000000-0000-0000-0000-000000000013', 3)`, full)

	require.NoError(t, db.RunMigrations(database))

	// Every user has one default board, and every row is on its owner's
	assert.Equal(t, 4, count(`SELECT COUNT(*) FROM boards`))
	assert.Equal(t, 4, count(`SELECT COUNT(*) FROM boards WHERE is_default AND NOT is_shared AND slug = 'home' AND name = 'Home'`))
	for _, table := range []string{"groups", "services", "widgets"} {
		assert.Zero(t, count(`SELECT COUNT(*) FROM `+table+` t JOIN boards b ON b.id = t.board_id WHERE b.user_id <> t.user_id OR NOT b.is_default`), table)
		assert.Zero(t, count(`SELECT COUNT(*) FROM `+table+` WHERE board_id IS NULL`), table)
		_, err := database.Exec(`UPDATE ` + table + ` SET board_id = NULL`)
		assert.Error(t, err, "%s.board_id is required", table)
	}
	assert.Equal(t, 3, count(`SELECT COUNT(*) FROM groups`))
	assert.Equal(t, 3, count(`SELECT COUNT(*) FROM services`))
	assert.Equal(t, 2, count(`SELECT COUNT(*) FROM widgets`))
	assert.Equal(t, 1, count(`SELECT COUNT(*) FROM pg_indexes WHERE indexname = 'idx_groups_default_per_board'`))
	assert.Zero(t, count(`SELECT COUNT(*) FROM pg_indexes WHERE indexname = 'idx_groups_default_per_user'`))

	// Running it again changes nothing
	up, err := os.ReadFile(filepath.Join(dir, boardsMigration+".up.sql"))
	require.NoError(t, err)
	exec(string(up))
	assert.Equal(t, 4, count(`SELECT COUNT(*) FROM boards`))

	// The repositories on the real schema
	boards, groups, services, widgets := NewBoardRepository(database), NewGroupRepository(database), NewServiceRepository(database), NewWidgetRepository(database)
	home, err := boards.EnsureDefault(ctx, full)
	require.NoError(t, err)
	assert.True(t, home.IsDefault)
	assert.Equal(t, 4, count(`SELECT COUNT(*) FROM boards`), "the board the migration made is found, not made again")

	// A user who registers after the migration gets a board on first use
	const late = "10000000-0000-0000-0000-000000000005"
	exec(`INSERT INTO users (id, email, name, password, role) VALUES ($1, 'late@example.com', 'Late', 'x', 'user')`, late)
	defaultGroup, err := groups.EnsureDefaultGroup(ctx, late)
	require.NoError(t, err)
	again, err := groups.EnsureDefaultGroup(ctx, late)
	require.NoError(t, err)
	assert.Equal(t, defaultGroup.ID, again.ID, "the ON CONFLICT target is the per-board index")
	lateHome, err := boards.EnsureDefault(ctx, late)
	require.NoError(t, err)
	assert.Equal(t, lateHome.ID, defaultGroup.BoardID)
	now := time.Now()
	router := &models.Service{UserID: late, Name: "Router", URL: "http://router.lan", Status: models.StatusUnknown, IconType: models.IconTypeEmoji, CardSize: models.CardSize2x1, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, services.Create(ctx, router))
	assert.Equal(t, lateHome.ID, router.BoardID)

	// A shared board of the admin, with tiles created on it and moved to it
	shared := &models.Board{UserID: admin, Name: "Family", Slug: "family", IsShared: true}
	require.NoError(t, boards.Create(ctx, shared, false))
	twin := &models.Board{UserID: admin, Name: "Family", Slug: "family"}
	assert.ErrorIs(t, boards.Create(ctx, twin, false), ErrBoardSlugTaken)
	require.NoError(t, boards.Create(ctx, twin, true))
	assert.Equal(t, "family-2", twin.Slug)
	sharedGroup, err := groups.EnsureDefaultGroupOn(ctx, admin, shared.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, count(`SELECT COUNT(*) FROM groups WHERE board_id = $1`, shared.ID), "created with the board, not twice")
	media := &models.Group{UserID: admin, BoardID: shared.ID, Name: "Media", Color: models.DefaultGroupColor, MonitoringEnabled: true, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, groups.Create(ctx, media))
	exec(`INSERT INTO integrations (id, user_id, kind, name, base_url, options) VALUES
		('50000000-0000-0000-0000-000000000004', $1, 'sonarr', 'Sonarr', 'http://sonarr.lan', '{"library":"tv","empty":""}')`, admin)
	integration := "50000000-0000-0000-0000-000000000004"
	jellyfin := &models.Service{UserID: admin, BoardID: shared.ID, GroupID: &media.ID, IntegrationID: &integration, Name: "Jellyfin", URL: "http://jellyfin.lan", Status: models.StatusUnknown, IconType: models.IconTypeEmoji, CardSize: models.CardSize2x1, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, services.Create(ctx, jellyfin))
	clock := &models.Widget{UserID: admin, BoardID: shared.ID, Type: "clock", Config: []byte(`{}`), CardSize: "2x1", RefreshSeconds: 300, Enabled: true}
	require.NoError(t, widgets.Create(ctx, clock, 10))
	position, err := boards.MoveTile(ctx, admin, models.TileKindService, "30000000-0000-0000-0000-000000000003", shared.ID)
	require.NoError(t, err)
	assert.Greater(t, position, clock.Position)
	_, err = boards.MoveTile(ctx, full, models.TileKindService, jellyfin.ID, home.ID)
	assert.ErrorIs(t, err, ErrTileNotFound, "someone else's tile does not move")
	// Nor is it reordered: not found (404), as on the other by-id routes
	assert.ErrorIs(t, groups.UpdatePositions(ctx, full, map[string]int{media.ID: 0}), sql.ErrNoRows)
	assert.ErrorIs(t, services.UpdatePositions(ctx, full, map[string]int{jellyfin.ID: 0}), sql.ErrNoRows)

	onBoard, err := services.GetAllByBoardID(ctx, shared.ID)
	require.NoError(t, err)
	require.Len(t, onBoard, 2)
	assert.Equal(t, []string{"Jellyfin", "NAS"}, []string{onBoard[0].Name, onBoard[1].Name})
	boardWidgets, err := widgets.ListByBoardID(ctx, shared.ID)
	require.NoError(t, err)
	assert.Len(t, boardWidgets, 1)
	boardGroups, err := groups.GetAllByBoardID(ctx, shared.ID)
	require.NoError(t, err)
	assert.Len(t, boardGroups, 2)
	onDefault, err := groups.GetAllOnDefaultBoard(ctx, full)
	require.NoError(t, err)
	assert.Len(t, onDefault, 2)

	// Who sees what
	visible, err := boards.ListVisible(ctx, full)
	require.NoError(t, err)
	require.Len(t, visible, 2)
	assert.Equal(t, []string{home.ID, shared.ID}, []string{visible[0].ID, visible[1].ID})
	_, err = boards.GetVisible(ctx, twin.ID, full)
	assert.ErrorIs(t, err, ErrBoardNotFound)
	keys, err := boards.ViewerKeys(ctx, shared.ID)
	require.NoError(t, err)
	assert.Equal(t, map[string]bool{
		"service:" + jellyfin.ID: true, "service:30000000-0000-0000-0000-000000000003": true,
		"widget:" + clock.ID: true, "integration:" + integration: true,
	}, keys)
	shown, err := boards.Integrations(ctx, shared.ID)
	require.NoError(t, err)
	assert.Equal(t, []models.BoardIntegration{{ID: integration, Kind: "sonarr", Options: map[string]bool{"library": true}}}, shown)
	exec(`UPDATE users SET role = 'user' WHERE id = $1`, admin)
	visible, err = boards.ListVisible(ctx, full)
	require.NoError(t, err)
	assert.Len(t, visible, 1, "shared needs an admin owner")
	exec(`UPDATE users SET role = 'admin' WHERE id = $1`, admin)

	// The import's view of a dashboard: what a replace of the default and
	// the twin board removes, and what it keeps
	scope, err := NewDashboardRepository(database).Scope(ctx, admin, []string{twin.ID})
	require.NoError(t, err)
	assert.Equal(t, DashboardState{Groups: 1, Integrations: 1}, scope.State)
	assert.Equal(t, DashboardState{Widgets: 1, Integrations: 1}, scope.Kept)
	assert.Equal(t, sharedGroup.ID, scope.DefaultGroups[shared.ID])

	// Update, reorder, delete
	rename := func(slug string) func(*models.Board) error {
		return func(b *models.Board) error {
			b.Name, b.Slug = "Family night", slug
			return nil
		}
	}
	_, err = boards.Update(ctx, shared.ID, admin, rename("family-2"))
	assert.ErrorIs(t, err, ErrBoardSlugTaken)
	updated, err := boards.Update(ctx, shared.ID, admin, rename("night"))
	require.NoError(t, err)
	assert.Equal(t, "night", updated.Slug)
	assert.True(t, updated.IsShared)
	adminHome, err := boards.EnsureDefault(ctx, admin)
	require.NoError(t, err)
	require.NoError(t, boards.Reorder(ctx, admin, []string{twin.ID, shared.ID, adminHome.ID}))
	assert.ErrorIs(t, boards.Reorder(ctx, admin, []string{home.ID}), ErrBoardNotFound)
	own, err := boards.ListByUserID(ctx, admin)
	require.NoError(t, err)
	assert.Equal(t, []string{twin.ID, shared.ID, adminHome.ID}, []string{own[0].ID, own[1].ID, own[2].ID})

	assert.ErrorIs(t, boards.Delete(ctx, adminHome.ID, admin, ""), ErrCannotDeleteDefaultBoard)
	assert.ErrorIs(t, boards.Delete(ctx, shared.ID, full, ""), ErrBoardNotFound)
	assert.ErrorIs(t, boards.Delete(ctx, shared.ID, admin, home.ID), ErrBoardNotFound, "not to someone else's board")
	require.NoError(t, boards.Delete(ctx, shared.ID, admin, twin.ID))
	assert.Equal(t, 2, count(`SELECT COUNT(*) FROM services WHERE board_id = $1`, twin.ID))
	assert.Equal(t, 1, count(`SELECT COUNT(*) FROM services WHERE board_id = $1 AND group_id = $2`, twin.ID, media.ID), "the group moved along")
	assert.Equal(t, 1, count(`SELECT COUNT(*) FROM widgets WHERE board_id = $1`, twin.ID))
	assert.Zero(t, count(`SELECT COUNT(*) FROM groups WHERE id = $1`, sharedGroup.ID), "its default group went with the board")
	require.NoError(t, boards.Delete(ctx, twin.ID, admin, ""))
	assert.Zero(t, count(`SELECT COUNT(*) FROM services WHERE user_id = $1`, admin))
	assert.Zero(t, count(`SELECT COUNT(*) FROM widgets WHERE user_id = $1`, admin))
	// The foreign key alone does the same (a deleted user takes their boards along)
	cascade := &models.Board{UserID: late, Name: "Gone", Slug: "gone"}
	require.NoError(t, boards.Create(ctx, cascade, false))
	router.ID, router.BoardID = "", cascade.ID
	require.NoError(t, services.Create(ctx, router))
	exec(`DELETE FROM users WHERE id = $1`, late)
	assert.Zero(t, count(`SELECT COUNT(*) FROM boards WHERE user_id = $1`, late))
	assert.Zero(t, count(`SELECT COUNT(*) FROM services WHERE user_id = $1`, late))

	// Down with several boards and default groups, then up again
	extra := &models.Board{UserID: full, Name: "Lab", Slug: "lab"}
	require.NoError(t, boards.Create(ctx, extra, false))
	exec(`UPDATE widgets SET board_id = $1 WHERE id = '40000000-0000-0000-0000-000000000001'`, extra.ID)
	down, err := os.ReadFile(filepath.Join(dir, boardsMigration+".down.sql"))
	require.NoError(t, err)
	exec(string(down))
	assert.Equal(t, 2, count(`SELECT COUNT(*) FROM widgets WHERE user_id = $1`, full), "tiles of every board stay")
	assert.Equal(t, 1, count(`SELECT COUNT(*) FROM groups WHERE user_id = $1 AND is_default`, full))
	assert.Equal(t, 3, count(`SELECT COUNT(*) FROM groups WHERE user_id = $1`, full))
	exec(string(up))
	assert.Equal(t, 4, count(`SELECT COUNT(*) FROM boards WHERE is_default`))
	assert.Zero(t, count(`SELECT COUNT(*) FROM widgets w JOIN boards b ON b.id = w.board_id WHERE NOT b.is_default`))
}
