package repository

import (
	"context"
	"database/sql"
	"github.com/nimbus/backend/internal/testutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

// discoveryTestSchema mirrors migration 000037 for SQLite
const discoveryTestSchema = `
	CREATE TABLE users (id TEXT PRIMARY KEY, role TEXT NOT NULL DEFAULT 'user');
	CREATE TABLE services (id TEXT PRIMARY KEY);
	CREATE TABLE discovered_containers (
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		discovery_key TEXT NOT NULL,
		service_id TEXT REFERENCES services(id) ON DELETE SET NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (user_id, discovery_key)
	);
	INSERT INTO users (id, role) VALUES ('user-1', 'admin'), ('user-2', 'admin'), ('user-3', 'user');
	INSERT INTO services (id) VALUES ('service-1');
	CREATE TABLE integrations (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		kind TEXT NOT NULL,
		name TEXT NOT NULL,
		base_url TEXT NOT NULL,
		auth_type TEXT NOT NULL DEFAULT 'none',
		credentials_enc BLOB,
		verify_tls INTEGER NOT NULL DEFAULT 1,
		options TEXT NOT NULL DEFAULT '{}',
		refresh_seconds INTEGER NOT NULL DEFAULT 60,
		last_test_at TIMESTAMP,
		last_test_ok INTEGER,
		last_error TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	INSERT INTO integrations (id, user_id, kind, name, base_url) VALUES
		('i-1', 'user-1', 'docker', 'B', 'http://a.lan:2375'),
		('i-2', 'user-1', 'docker', 'A', 'http://b.lan:2375'),
		('i-3', 'user-1', 'sonarr', 'Sonarr', 'http://sonarr.lan'),
		('i-4', 'user-3', 'docker', 'Not an admin', 'http://c.lan:2375');
`

func setupDiscoveryTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", "file::memory:?_foreign_keys=on")
	require.NoError(t, err)
	db.SetMaxOpenConns(1) // every connection to :memory: is its own database
	t.Cleanup(func() { db.Close() })
	_, err = db.Exec(discoveryTestSchema)
	require.NoError(t, err)
	testutil.AddBoards(t, db)
	return db
}

func TestDiscoveryRepository_ClaimLinkRelease(t *testing.T) {
	db := setupDiscoveryTestDB(t)
	repo := NewDiscoveryRepository(db)
	ctx := context.Background()

	claimed, err := repo.Claim(ctx, "user-1", "i-1/web")
	require.NoError(t, err)
	assert.True(t, claimed)
	claimed, err = repo.Claim(ctx, "user-1", "i-1/web")
	require.NoError(t, err)
	assert.False(t, claimed, "only the first claim wins")
	claimed, err = repo.Claim(ctx, "user-2", "i-1/web")
	require.NoError(t, err)
	assert.True(t, claimed, "claims are per user")

	require.NoError(t, repo.SetService(ctx, "user-1", "i-1/web", "service-1"))
	handled, err := repo.List(ctx, "user-1")
	require.NoError(t, err)
	require.Len(t, handled, 1)
	require.NotNil(t, handled["i-1/web"])
	assert.Equal(t, "service-1", *handled["i-1/web"])

	// A container that became a service can't be released
	require.NoError(t, repo.Release(ctx, "user-1", "i-1/web"))
	handled, err = repo.List(ctx, "user-1")
	require.NoError(t, err)
	assert.Len(t, handled, 1)

	// Deleting the service leaves the container ignored, which can be released
	_, err = db.Exec(`DELETE FROM services WHERE id = 'service-1'`)
	require.NoError(t, err)
	handled, err = repo.List(ctx, "user-1")
	require.NoError(t, err)
	serviceID, isHandled := handled["i-1/web"]
	assert.True(t, isHandled)
	assert.Nil(t, serviceID)

	require.NoError(t, repo.Release(ctx, "user-1", "i-1/web"))
	handled, err = repo.List(ctx, "user-1")
	require.NoError(t, err)
	assert.Empty(t, handled)
	handled, err = repo.List(ctx, "user-2")
	require.NoError(t, err)
	assert.Len(t, handled, 1, "another user's rows are untouched")
}

func TestDiscoveryRepository_AdminIntegrations(t *testing.T) {
	repo := NewDiscoveryRepository(setupDiscoveryTestDB(t))

	list, err := repo.AdminIntegrations(context.Background(), "docker")
	require.NoError(t, err)
	require.Len(t, list, 2, "only Docker integrations of current admins")
	assert.Equal(t, []string{"i-2", "i-1"}, []string{list[0].ID, list[1].ID}, "by name")
	assert.Equal(t, "user-1", list[0].UserID)
}
