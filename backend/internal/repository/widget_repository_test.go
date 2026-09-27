package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	"github.com/nimbus/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

// widgetTestSchema mirrors migration 000027 for SQLite, plus the services
// columns that share the position space
const widgetTestSchema = `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY
	);

	CREATE TABLE IF NOT EXISTS services (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		position INTEGER DEFAULT 0,
		updated_at TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS widgets (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		type TEXT NOT NULL,
		title TEXT NOT NULL DEFAULT '',
		group_id TEXT,
		integration_id TEXT,
		config TEXT NOT NULL DEFAULT '{}',
		card_size TEXT NOT NULL DEFAULT '2x1',
		position INTEGER NOT NULL DEFAULT 0,
		refresh_seconds INTEGER NOT NULL DEFAULT 300 CHECK (refresh_seconds >= 10),
		enabled INTEGER NOT NULL DEFAULT 1,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	INSERT INTO users (id) VALUES ('user-1'), ('user-2');
`

func setupWidgetTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec(widgetTestSchema)
	require.NoError(t, err)
	return db
}

func newTestWidget(userID, typ string) *models.Widget {
	return &models.Widget{
		UserID:         userID,
		Type:           typ,
		Title:          "My " + typ,
		Config:         json.RawMessage(`{"content":"hi"}`),
		CardSize:       models.CardSize2x2,
		RefreshSeconds: 300,
		Enabled:        true,
	}
}

func servicePosition(t *testing.T, db *sql.DB, id string) int {
	t.Helper()
	var pos int
	require.NoError(t, db.QueryRow(`SELECT position FROM services WHERE id = ?`, id).Scan(&pos))
	return pos
}

func TestWidgetRepository_CreateAppendsAfterServices(t *testing.T) {
	db := setupWidgetTestDB(t)
	repo := NewWidgetRepository(db)
	ctx := context.Background()

	_, err := db.Exec(`INSERT INTO services (id, user_id, position) VALUES ('s1', 'user-1', 0), ('s2', 'user-1', 4), ('s3', 'user-2', 9)`)
	require.NoError(t, err)

	first := newTestWidget("user-1", "markdown")
	require.NoError(t, repo.Create(ctx, first, 10))
	assert.NotEmpty(t, first.ID)
	assert.Equal(t, 5, first.Position, "after the user's last service, ignoring other users")
	assert.False(t, first.CreatedAt.IsZero())

	second := newTestWidget("user-1", "clock")
	require.NoError(t, repo.Create(ctx, second, 10))
	assert.Equal(t, 6, second.Position, "after the last widget")

	other := newTestWidget("user-2", "clock")
	require.NoError(t, repo.Create(ctx, other, 10))
	assert.Equal(t, 10, other.Position)

	fresh := newTestWidget("user-1", "clock")
	_, err = db.Exec(`DELETE FROM services; DELETE FROM widgets`)
	require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, fresh, 10))
	assert.Equal(t, 0, fresh.Position, "an empty grid starts at 0")
}

func TestWidgetRepository_CreateLimit(t *testing.T) {
	repo := NewWidgetRepository(setupWidgetTestDB(t))
	ctx := context.Background()

	for range 2 {
		require.NoError(t, repo.Create(ctx, newTestWidget("user-1", "clock"), 2))
	}
	err := repo.Create(ctx, newTestWidget("user-1", "clock"), 2)
	assert.ErrorIs(t, err, ErrWidgetLimitReached)

	require.NoError(t, repo.Create(ctx, newTestWidget("user-2", "clock"), 2), "limit is per user")
}

func TestWidgetRepository_GetListScopedToOwner(t *testing.T) {
	repo := NewWidgetRepository(setupWidgetTestDB(t))
	ctx := context.Background()

	groupID := "group-1"
	a := newTestWidget("user-1", "markdown")
	a.GroupID = &groupID
	require.NoError(t, repo.Create(ctx, a, 10))
	b := newTestWidget("user-1", "clock")
	require.NoError(t, repo.Create(ctx, b, 10))
	require.NoError(t, repo.Create(ctx, newTestWidget("user-2", "clock"), 10))

	got, err := repo.GetByID(ctx, a.ID, "user-1")
	require.NoError(t, err)
	assert.Equal(t, "markdown", got.Type)
	assert.Equal(t, "My markdown", got.Title)
	assert.Equal(t, &groupID, got.GroupID)
	assert.Nil(t, got.IntegrationID)
	assert.JSONEq(t, `{"content":"hi"}`, string(got.Config))
	assert.Equal(t, models.CardSize2x2, got.CardSize)
	assert.True(t, got.Enabled)

	_, err = repo.GetByID(ctx, a.ID, "user-2")
	assert.ErrorIs(t, err, ErrWidgetNotFound)

	list, err := repo.ListByUserID(ctx, "user-1")
	require.NoError(t, err)
	if assert.Len(t, list, 2) {
		assert.Equal(t, a.ID, list[0].ID, "sorted by position")
		assert.Equal(t, b.ID, list[1].ID)
	}

	empty, err := repo.ListByUserID(ctx, "nobody")
	require.NoError(t, err)
	assert.NotNil(t, empty, "empty list must encode as [] not null")
}

func TestWidgetRepository_Update(t *testing.T) {
	repo := NewWidgetRepository(setupWidgetTestDB(t))
	ctx := context.Background()

	w := newTestWidget("user-1", "markdown")
	require.NoError(t, repo.Create(ctx, w, 10))

	w.Title = "Renamed"
	w.Config = json.RawMessage(`{"content":"new"}`)
	w.CardSize = models.CardSize1x1
	w.Enabled = false
	w.Type = "clock" // ignored: type never changes
	require.NoError(t, repo.Update(ctx, w))

	got, err := repo.GetByID(ctx, w.ID, "user-1")
	require.NoError(t, err)
	assert.Equal(t, "markdown", got.Type)
	assert.Equal(t, "Renamed", got.Title)
	assert.JSONEq(t, `{"content":"new"}`, string(got.Config))
	assert.Equal(t, models.CardSize1x1, got.CardSize)
	assert.False(t, got.Enabled)

	w.UserID = "user-2"
	assert.ErrorIs(t, repo.Update(ctx, w), ErrWidgetNotFound, "can't update another user's widget")
}

func TestWidgetRepository_Delete(t *testing.T) {
	repo := NewWidgetRepository(setupWidgetTestDB(t))
	ctx := context.Background()

	w := newTestWidget("user-1", "clock")
	require.NoError(t, repo.Create(ctx, w, 10))

	assert.ErrorIs(t, repo.Delete(ctx, w.ID, "user-2"), ErrWidgetNotFound)
	require.NoError(t, repo.Delete(ctx, w.ID, "user-1"))
	assert.ErrorIs(t, repo.Delete(ctx, w.ID, "user-1"), ErrWidgetNotFound)
}

func TestWidgetRepository_ReorderTiles(t *testing.T) {
	db := setupWidgetTestDB(t)
	repo := NewWidgetRepository(db)
	ctx := context.Background()

	_, err := db.Exec(`INSERT INTO services (id, user_id, position) VALUES ('s1', 'user-1', 0), ('s2', 'user-2', 0)`)
	require.NoError(t, err)
	w := newTestWidget("user-1", "clock")
	require.NoError(t, repo.Create(ctx, w, 10))
	assert.Equal(t, 1, w.Position)

	// Widget first, service second
	err = repo.ReorderTiles(ctx, "user-1", []models.TilePosition{
		{ID: w.ID, Kind: models.TileKindWidget, Position: 0},
		{ID: "s1", Kind: models.TileKindService, Position: 1},
	})
	require.NoError(t, err)

	got, err := repo.GetByID(ctx, w.ID, "user-1")
	require.NoError(t, err)
	assert.Equal(t, 0, got.Position)
	assert.Equal(t, 1, servicePosition(t, db, "s1"))

	// Another user's service rolls back the whole batch
	err = repo.ReorderTiles(ctx, "user-1", []models.TilePosition{
		{ID: w.ID, Kind: models.TileKindWidget, Position: 7},
		{ID: "s2", Kind: models.TileKindService, Position: 8},
	})
	assert.True(t, errors.Is(err, ErrTileNotFound))
	got, err = repo.GetByID(ctx, w.ID, "user-1")
	require.NoError(t, err)
	assert.Equal(t, 0, got.Position, "nothing changes when one tile fails")
	assert.Equal(t, 0, servicePosition(t, db, "s2"))

	// A widget ID sent as a service is not found either
	err = repo.ReorderTiles(ctx, "user-1", []models.TilePosition{{ID: w.ID, Kind: models.TileKindService, Position: 3}})
	assert.ErrorIs(t, err, ErrTileNotFound)

	err = repo.ReorderTiles(ctx, "user-1", []models.TilePosition{{ID: w.ID, Kind: "group", Position: 3}})
	assert.Error(t, err)
}
