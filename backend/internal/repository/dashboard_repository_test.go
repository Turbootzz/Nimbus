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

	_ "github.com/mattn/go-sqlite3"
)

// dashboardTestSchema mirrors the tables an import writes, for SQLite
const dashboardTestSchema = `
	CREATE TABLE users (id TEXT PRIMARY KEY);
	CREATE TABLE groups (
		id TEXT PRIMARY KEY, user_id TEXT NOT NULL, name TEXT NOT NULL, color TEXT, position INTEGER DEFAULT 0,
		is_default INTEGER DEFAULT 0, monitoring_enabled INTEGER DEFAULT 1, created_at TIMESTAMP, updated_at TIMESTAMP
	);
	CREATE UNIQUE INDEX idx_groups_default_per_user ON groups(user_id) WHERE is_default = TRUE;
	CREATE TABLE integrations (
		id TEXT PRIMARY KEY, user_id TEXT NOT NULL, kind TEXT NOT NULL, name TEXT NOT NULL, base_url TEXT NOT NULL,
		auth_type TEXT NOT NULL DEFAULT 'none', credentials_enc BLOB, verify_tls INTEGER NOT NULL DEFAULT 1,
		options TEXT NOT NULL DEFAULT '{}', refresh_seconds INTEGER NOT NULL DEFAULT 60,
		last_test_at TIMESTAMP, last_test_ok INTEGER, last_error TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE services (
		id TEXT PRIMARY KEY, user_id TEXT NOT NULL, name TEXT NOT NULL, url TEXT NOT NULL, icon TEXT,
		icon_type TEXT DEFAULT 'emoji', icon_image_path TEXT DEFAULT '', description TEXT, status TEXT NOT NULL,
		response_time INTEGER, position INTEGER DEFAULT 0, card_size TEXT DEFAULT '2x1', group_id TEXT,
		integration_id TEXT, monitoring_enabled INTEGER NOT NULL DEFAULT 1, created_at TIMESTAMP, updated_at TIMESTAMP
	);
	CREATE TABLE widgets (
		id TEXT PRIMARY KEY, user_id TEXT NOT NULL, type TEXT NOT NULL, title TEXT NOT NULL DEFAULT '',
		group_id TEXT, integration_id TEXT, config TEXT NOT NULL DEFAULT '{}', card_size TEXT NOT NULL DEFAULT '2x1',
		position INTEGER NOT NULL DEFAULT 0, refresh_seconds INTEGER NOT NULL DEFAULT 300, enabled INTEGER NOT NULL DEFAULT 1,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE user_preferences (
		id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))), user_id TEXT NOT NULL UNIQUE,
		theme_mode TEXT NOT NULL DEFAULT 'auto', created_at TIMESTAMP, updated_at TIMESTAMP
	);
	INSERT INTO users (id) VALUES ('user-1'), ('user-2');
`

var testDashboardLimits = DashboardLimits{Integrations: 2, Widgets: 2}

func setupDashboardTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	_, err = db.Exec(dashboardTestSchema)
	require.NoError(t, err)
	return db
}

// testDashboardRows is a small dashboard: a default group with a service and
// a widget in it, a second group, and an integration the service uses
func testDashboardRows(prefix string) *DashboardRows {
	now := time.Now()
	defaultGroup, integration := prefix+"-default", prefix+"-integration"
	return &DashboardRows{
		Groups: []*models.Group{
			{ID: defaultGroup, Name: "Main", Color: "#0ea5e9", IsDefault: true, MonitoringEnabled: true, CreatedAt: now, UpdatedAt: now},
			{ID: prefix + "-tools", Name: "Tools", Color: "#0ea5e9", MonitoringEnabled: true, CreatedAt: now, UpdatedAt: now},
		},
		Integrations: []*models.Integration{
			{ID: integration, Kind: "sonarr", Name: "Sonarr", BaseURL: "http://sonarr.lan", AuthType: models.IntegrationAuthAPIKey, VerifyTLS: true, RefreshSeconds: 60},
		},
		Services: []*models.Service{{
			ID: prefix + "-service", Name: "Sonarr", URL: "http://sonarr.lan", Icon: "📺", IconType: models.IconTypeImageUpload,
			IconImagePath: prefix + ".png", Status: models.StatusUnknown, Position: 0, CardSize: models.CardSize2x1,
			GroupID: &defaultGroup, IntegrationID: &integration, MonitoringEnabled: true, CreatedAt: now, UpdatedAt: now,
		}},
		Widgets: []*models.Widget{{
			ID: prefix + "-widget", Type: "markdown", Title: "Note", GroupID: &defaultGroup, Config: json.RawMessage(`{"content":"hi"}`),
			CardSize: models.CardSize2x2, Position: 1, RefreshSeconds: 300, Enabled: true,
		}},
	}
}

func TestDashboardApplyReplaceOnlyWipesTheUser(t *testing.T) {
	db := setupDashboardTestDB(t)
	repo := NewDashboardRepository(db)
	ctx := context.Background()
	_, err := repo.Apply(ctx, "user-1", true, testDashboardRows("old"), testDashboardLimits)
	require.NoError(t, err)
	_, err = repo.Apply(ctx, "user-2", true, testDashboardRows("other"), testDashboardLimits)
	require.NoError(t, err)

	rows := testDashboardRows("new")
	rows.Preferences = &models.PreferencesUpdateRequest{ThemeMode: stringPtr("dark")}
	applied, err := repo.Apply(ctx, "user-1", true, rows, testDashboardLimits)
	require.NoError(t, err)

	assert.Equal(t, DashboardState{Groups: 2, Services: 1, Widgets: 1, Integrations: 1, DefaultGroupID: "old-default"}, applied.Before)
	assert.Equal(t, []string{"old.png"}, applied.RemovedIcons)

	state, err := repo.State(ctx, "user-1")
	require.NoError(t, err)
	assert.Equal(t, DashboardState{Groups: 2, Services: 1, Widgets: 1, Integrations: 1, DefaultGroupID: "new-default"}, state)
	other, err := repo.State(ctx, "user-2")
	require.NoError(t, err)
	assert.Equal(t, DashboardState{Groups: 2, Services: 1, Widgets: 1, Integrations: 1, DefaultGroupID: "other-default"}, other)

	var themeMode string
	require.NoError(t, db.QueryRow(`SELECT theme_mode FROM user_preferences WHERE user_id = 'user-1'`).Scan(&themeMode))
	assert.Equal(t, "dark", themeMode)

	// Imported integrations never have credentials
	integration, err := NewIntegrationRepository(db).GetByID(ctx, "new-integration", "user-1")
	require.NoError(t, err)
	assert.False(t, integration.HasCredentials)
	assert.Equal(t, models.IntegrationAuthAPIKey, integration.AuthType)
}

func TestDashboardApplyMergeOffsetsPositionsAndReusesTheDefaultGroup(t *testing.T) {
	db := setupDashboardTestDB(t)
	repo := NewDashboardRepository(db)
	ctx := context.Background()
	limits := DashboardLimits{Integrations: 10, Widgets: 10}
	_, err := repo.Apply(ctx, "user-1", false, testDashboardRows("first"), limits)
	require.NoError(t, err)

	rows := testDashboardRows("second")
	applied, err := repo.Apply(ctx, "user-1", false, rows, limits)
	require.NoError(t, err)

	assert.Empty(t, applied.RemovedIcons)
	assert.Equal(t, "first-default", applied.Before.DefaultGroupID)
	// The second default group was not created; its tiles joined the first
	require.Len(t, rows.Groups, 1)
	assert.Equal(t, "first-default", *rows.Services[0].GroupID)
	assert.Equal(t, "first-default", *rows.Widgets[0].GroupID)

	var groupPositions, tilePositions []int
	scanInts(t, db, &groupPositions, `SELECT position FROM groups WHERE user_id = 'user-1' ORDER BY position`)
	assert.Equal(t, []int{0, 1, 2}, groupPositions)
	scanInts(t, db, &tilePositions, `SELECT position FROM (
		SELECT position FROM services WHERE user_id = 'user-1' UNION ALL SELECT position FROM widgets WHERE user_id = 'user-1'
	) ORDER BY position`)
	assert.Equal(t, []int{0, 1, 2, 3}, tilePositions)
}

func TestDashboardApplyRollsBackWhenALimitIsHit(t *testing.T) {
	db := setupDashboardTestDB(t)
	repo := NewDashboardRepository(db)
	ctx := context.Background()
	_, err := repo.Apply(ctx, "user-1", true, testDashboardRows("old"), testDashboardLimits)
	require.NoError(t, err)

	tooMany := testDashboardRows("new")
	for _, id := range []string{"w2", "w3"} {
		tooMany.Widgets = append(tooMany.Widgets, &models.Widget{ID: id, Type: "markdown", Config: json.RawMessage(`{}`), CardSize: models.CardSize2x1, RefreshSeconds: 300})
	}
	_, err = repo.Apply(ctx, "user-1", true, tooMany, testDashboardLimits)
	assert.ErrorIs(t, err, ErrWidgetLimitReached)

	// The wipe of the replace was rolled back with it
	state, err := repo.State(ctx, "user-1")
	require.NoError(t, err)
	assert.Equal(t, "old-default", state.DefaultGroupID)
	assert.Equal(t, 1, state.Services)

	// A merge counts what the user keeps
	_, err = repo.Apply(ctx, "user-1", false, testDashboardRows("more"), DashboardLimits{Integrations: 1, Widgets: 10})
	assert.ErrorIs(t, err, ErrIntegrationLimitReached)
}

func scanInts(t *testing.T, db *sql.DB, out *[]int, query string) {
	t.Helper()
	rows, err := db.Query(query)
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var n int
		require.NoError(t, rows.Scan(&n))
		*out = append(*out, n)
	}
	require.NoError(t, rows.Err())
}
