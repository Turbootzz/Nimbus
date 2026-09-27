package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/nimbus/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// snapshotTestSchema mirrors migration 000028 for SQLite, on top of the
// widget schema
const snapshotTestSchema = `
	CREATE TABLE integrations (id TEXT PRIMARY KEY);
	CREATE TABLE widget_snapshots (
		source_kind TEXT NOT NULL CHECK (source_kind IN ('widget', 'integration')),
		source_id TEXT NOT NULL,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		payload TEXT,
		error TEXT,
		fetched_at TIMESTAMP NOT NULL,
		PRIMARY KEY (source_kind, source_id)
	);
`

func setupSnapshotTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db := setupWidgetTestDB(t)
	_, err := db.Exec(snapshotTestSchema)
	require.NoError(t, err)
	return db
}

func TestSnapshotRepository_UpsertAndList(t *testing.T) {
	repo := NewSnapshotRepository(setupSnapshotTestDB(t))
	ctx := context.Background()
	at := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)

	snap := &models.Snapshot{SourceKind: "widget", SourceID: "w1", UserID: "user-1", Payload: json.RawMessage(`{"t":12}`), FetchedAt: at}
	require.NoError(t, repo.Upsert(ctx, snap))

	// Second write replaces the first: failed fetch, payload kept by the caller
	snap.Error = "timeout"
	snap.FetchedAt = at.Add(time.Minute)
	require.NoError(t, repo.Upsert(ctx, snap))
	require.NoError(t, repo.Upsert(ctx, &models.Snapshot{SourceKind: "integration", SourceID: "i1", UserID: "user-2", Error: "down", FetchedAt: at}))

	list, err := repo.ListAll(ctx)
	require.NoError(t, err)
	require.Len(t, list, 2)
	byKey := map[string]models.Snapshot{}
	for _, s := range list {
		byKey[s.Key()] = s
	}
	w := byKey["widget:w1"]
	assert.Equal(t, "user-1", w.UserID)
	assert.JSONEq(t, `{"t":12}`, string(w.Payload))
	assert.Equal(t, "timeout", w.Error)
	assert.True(t, w.FetchedAt.Equal(at.Add(time.Minute)))

	i := byKey["integration:i1"]
	assert.Nil(t, i.Payload, "no payload stays NULL")
	assert.Equal(t, "down", i.Error)
}

func TestSnapshotRepository_PruneOrphans(t *testing.T) {
	db := setupSnapshotTestDB(t)
	repo := NewSnapshotRepository(db)
	widgets := NewWidgetRepository(db)
	ctx := context.Background()

	live := newTestWidget("user-1", "weather")
	require.NoError(t, widgets.Create(ctx, live, 10))
	_, err := db.Exec(`INSERT INTO integrations (id) VALUES ('i-live')`)
	require.NoError(t, err)

	for _, s := range []models.Snapshot{
		{SourceKind: "widget", SourceID: live.ID},
		{SourceKind: "widget", SourceID: "w-deleted"},
		{SourceKind: "integration", SourceID: "i-live"},
		{SourceKind: "integration", SourceID: "i-deleted"},
	} {
		s.UserID = "user-1"
		s.FetchedAt = time.Now()
		require.NoError(t, repo.Upsert(ctx, &s))
	}

	pruned, err := repo.PruneOrphans(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), pruned)

	list, err := repo.ListAll(ctx)
	require.NoError(t, err)
	var keys []string
	for _, s := range list {
		keys = append(keys, s.Key())
	}
	assert.ElementsMatch(t, []string{"widget:" + live.ID, "integration:i-live"}, keys)
}

func TestWidgetRepository_ListEnabled(t *testing.T) {
	repo := NewWidgetRepository(setupWidgetTestDB(t))
	ctx := context.Background()

	on := newTestWidget("user-1", "weather")
	require.NoError(t, repo.Create(ctx, on, 10))
	other := newTestWidget("user-2", "weather")
	require.NoError(t, repo.Create(ctx, other, 10))
	off := newTestWidget("user-1", "weather")
	off.Enabled = false
	require.NoError(t, repo.Create(ctx, off, 10))

	list, err := repo.ListEnabled(ctx)
	require.NoError(t, err)
	var ids []string
	for _, w := range list {
		ids = append(ids, w.ID)
	}
	assert.ElementsMatch(t, []string{on.ID, other.ID}, ids, "all users, enabled only")
}
