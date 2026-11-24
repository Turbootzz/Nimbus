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

// seedSyncTestDB gives both users the dashboard of testDashboardRows
func seedSyncTestDB(t *testing.T) (*sql.DB, *DashboardRepository) {
	t.Helper()
	db := setupDashboardTestDB(t)
	repo := NewDashboardRepository(db)
	for user, prefix := range map[string]string{"user-1": "mine", "user-2": "other"} {
		_, err := repo.Apply(context.Background(), user, true, testDashboardRows(prefix), testDashboardLimits)
		require.NoError(t, err)
	}
	return db, repo
}

func TestDashboardChangesEmpty(t *testing.T) {
	assert.True(t, (&DashboardChanges{Credentials: map[string][]byte{}}).Empty())

	var changes DashboardChanges
	changes.Delete.Widgets = []string{"w"}
	assert.False(t, changes.Empty())
	assert.False(t, (&DashboardChanges{Credentials: map[string][]byte{"i": nil}}).Empty())
	assert.False(t, (&DashboardChanges{Create: DashboardRows{Preferences: &models.PreferencesUpdateRequest{}}}).Empty())
	assert.False(t, (&DashboardChanges{Update: DashboardRows{Groups: []*models.Group{{}}}}).Empty())
}

func TestDashboardSyncWritesCreatesUpdatesAndDeletes(t *testing.T) {
	db, repo := seedSyncTestDB(t)
	ctx := context.Background()
	now := time.Now()
	tools, integration := "mine-tools", "mine-integration"

	changes := &DashboardChanges{
		Create: DashboardRows{
			Groups: []*models.Group{{ID: "new-group", Name: "New", Color: "#0ea5e9", Position: 5, MonitoringEnabled: true, TileStyle: models.TileStyleCards, CreatedAt: now, UpdatedAt: now}},
			Services: []*models.Service{{
				ID: "new-service", Name: "Radarr", URL: "http://radarr.lan", IconType: models.IconTypeEmoji, Status: models.StatusUnknown,
				Position: 4, CardSize: models.CardSize2x1, GroupID: &tools, MonitoringEnabled: true, CreatedAt: now, UpdatedAt: now,
			}},
			Preferences: &models.PreferencesUpdateRequest{ThemeMode: stringPtr("dark")},
		},
		Update: DashboardRows{
			Groups: []*models.Group{{ID: "mine-tools", Name: "Utilities", Color: "#111111", Position: 3, MonitoringEnabled: false, TileStyle: models.TileStyleChips, UpdatedAt: now}},
			Integrations: []*models.Integration{{
				ID: integration, Name: "Sonarr 4K", BaseURL: "http://sonarr4k.lan", AuthType: models.IntegrationAuthAPIKey,
				VerifyTLS: false, Options: json.RawMessage(`{"a":1}`), RefreshSeconds: 120,
			}},
			Services: []*models.Service{{
				ID: "mine-service", Name: "Sonarr", URL: "http://sonarr.example.com", IconType: models.IconTypeEmoji, Icon: "📺",
				Position: 7, CardSize: models.CardSize2x2, GroupID: &tools, IntegrationID: &integration, UpdatedAt: now,
			}},
		},
		Credentials: map[string][]byte{integration: []byte("encrypted")},
	}
	changes.Delete.Widgets = []string{"mine-widget"}
	require.NoError(t, repo.Sync(ctx, "user-1", changes, testDashboardLimits))

	state, err := repo.State(ctx, "user-1")
	require.NoError(t, err)
	assert.Equal(t, DashboardState{Groups: 3, Services: 2, Widgets: 0, Integrations: 1, DefaultGroupID: "mine-default"}, state)

	var name, color, groupID, status, tileStyle string
	var position int
	require.NoError(t, db.QueryRow(`SELECT name, color, position, tile_style FROM groups WHERE id = 'mine-tools'`).Scan(&name, &color, &position, &tileStyle))
	assert.Equal(t, []any{"Utilities", "#111111", 3, models.TileStyleChips}, []any{name, color, position, tileStyle})
	require.NoError(t, db.QueryRow(`SELECT url, position, group_id, status FROM services WHERE id = 'mine-service'`).Scan(&name, &position, &groupID, &status))
	assert.Equal(t, []any{"http://sonarr.example.com", 7, "mine-tools", models.StatusUnknown}, []any{name, position, groupID, status})

	integrations := NewIntegrationRepository(db)
	stored, err := integrations.GetByID(ctx, integration, "user-1")
	require.NoError(t, err)
	assert.Equal(t, "Sonarr 4K", stored.Name)
	assert.Equal(t, 120, stored.RefreshSeconds)
	assert.JSONEq(t, `{"a":1}`, string(stored.Options))
	blob, err := integrations.GetCredentials(ctx, integration, "user-1")
	require.NoError(t, err)
	assert.Equal(t, []byte("encrypted"), blob)

	var themeMode string
	require.NoError(t, db.QueryRow(`SELECT theme_mode FROM user_preferences WHERE user_id = 'user-1'`).Scan(&themeMode))
	assert.Equal(t, "dark", themeMode)

	// A nil value clears the credentials
	require.NoError(t, repo.Sync(ctx, "user-1", &DashboardChanges{Credentials: map[string][]byte{integration: nil}}, testDashboardLimits))
	blob, err = integrations.GetCredentials(ctx, integration, "user-1")
	require.NoError(t, err)
	assert.Nil(t, blob)
}

func TestDashboardSyncIsAllOrNothingAndOnlyTouchesTheUser(t *testing.T) {
	db, repo := seedSyncTestDB(t)
	ctx := context.Background()
	now := time.Now()
	dump := func() string {
		out := ""
		for _, table := range []string{"groups", "integrations", "services", "widgets"} {
			rows, err := db.Query(`SELECT id, user_id, ` + map[string]string{"groups": "name", "integrations": "name", "services": "name", "widgets": "title"}[table] + ` FROM ` + table + ` ORDER BY id`)
			require.NoError(t, err)
			for rows.Next() {
				var id, user, name string
				require.NoError(t, rows.Scan(&id, &user, &name))
				out += table + " " + id + " " + user + " " + name + "\n"
			}
			require.NoError(t, rows.Err())
			rows.Close()
		}
		return out
	}
	before := dump()

	newGroup := func() *models.Group {
		return &models.Group{ID: "new-group", Name: "New", Color: "#0ea5e9", MonitoringEnabled: true, TileStyle: models.TileStyleCards, CreatedAt: now, UpdatedAt: now}
	}
	cases := map[string]func(*DashboardChanges){
		// The create before it must be rolled back with it
		"an update of a row that is gone": func(c *DashboardChanges) {
			c.Update.Services = []*models.Service{{ID: "gone", Name: "X", URL: "http://x.lan", UpdatedAt: now}}
		},
		"a delete of a row that is gone": func(c *DashboardChanges) { c.Delete.Widgets = []string{"gone"} },
		"an update of another user's group": func(c *DashboardChanges) {
			c.Update.Groups = []*models.Group{{ID: "other-tools", Name: "Stolen", Color: "#000000", TileStyle: models.TileStyleCards, UpdatedAt: now}}
		},
		"an update of another user's integration": func(c *DashboardChanges) {
			c.Update.Integrations = []*models.Integration{{ID: "other-integration", Name: "Stolen", BaseURL: "http://evil.lan", AuthType: models.IntegrationAuthNone}}
		},
		"an update of another user's widget": func(c *DashboardChanges) {
			c.Update.Widgets = []*models.Widget{{ID: "other-widget", Title: "Stolen", Config: json.RawMessage(`{}`), CardSize: models.CardSize2x1}}
		},
		"credentials for another user's integration": func(c *DashboardChanges) {
			c.Credentials = map[string][]byte{"other-integration": []byte("x")}
		},
		"a delete of another user's service": func(c *DashboardChanges) { c.Delete.Services = []string{"other-service"} },
		"a delete of another user's group":   func(c *DashboardChanges) { c.Delete.Groups = []string{"other-tools"} },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			changes := &DashboardChanges{Create: DashboardRows{Groups: []*models.Group{newGroup()}}}
			change(changes)
			require.Error(t, repo.Sync(ctx, "user-1", changes, testDashboardLimits))
			assert.Equal(t, before, dump())
		})
	}
}

func TestDashboardSyncChecksTheLimitsUnderTheLock(t *testing.T) {
	_, repo := seedSyncTestDB(t)
	ctx := context.Background()
	widget := func(id string) *models.Widget {
		return &models.Widget{ID: id, Type: "markdown", Title: id, Config: json.RawMessage(`{"content":"hi"}`),
			CardSize: models.CardSize2x2, Position: 5, RefreshSeconds: 300, Enabled: true}
	}
	integration := func(id string) *models.Integration {
		return &models.Integration{ID: id, Kind: "sonarr", Name: id, BaseURL: "http://sonarr.lan", AuthType: models.IntegrationAuthAPIKey, RefreshSeconds: 60}
	}
	// The user has one widget and one integration, and adds one of each in
	// the UI after a sync was planned with room for one more
	require.NoError(t, repo.Sync(ctx, "user-1", &DashboardChanges{Create: DashboardRows{
		Widgets: []*models.Widget{widget("from-ui")}, Integrations: []*models.Integration{integration("from-ui")},
	}}, testDashboardLimits))

	changes := &DashboardChanges{Create: DashboardRows{Widgets: []*models.Widget{widget("from-file")}}}
	assert.ErrorIs(t, repo.Sync(ctx, "user-1", changes, testDashboardLimits), ErrWidgetLimitReached)
	changes = &DashboardChanges{Create: DashboardRows{Integrations: []*models.Integration{integration("from-file")}}}
	assert.ErrorIs(t, repo.Sync(ctx, "user-1", changes, testDashboardLimits), ErrIntegrationLimitReached)
	state, err := repo.State(ctx, "user-1")
	require.NoError(t, err)
	assert.Equal(t, 2, state.Widgets)
	assert.Equal(t, 2, state.Integrations)

	// What the same sync deletes makes room
	changes = &DashboardChanges{Create: DashboardRows{Widgets: []*models.Widget{widget("from-file")}}}
	changes.Delete.Widgets = []string{"mine-widget"}
	require.NoError(t, repo.Sync(ctx, "user-1", changes, testDashboardLimits))

	// The limits are per user: a widget on another board counts too
	onBoard := widget("on-board")
	onBoard.BoardID = "mine-board"
	changes = &DashboardChanges{Create: DashboardRows{
		Boards:  []*models.Board{{ID: "mine-board", Name: "Lab", Slug: "lab"}},
		Widgets: []*models.Widget{onBoard},
	}}
	changes.Delete.Widgets = []string{"from-file"}
	require.NoError(t, repo.Sync(ctx, "user-1", changes, testDashboardLimits))
	changes = &DashboardChanges{Create: DashboardRows{Widgets: []*models.Widget{widget("from-file-2")}}}
	assert.ErrorIs(t, repo.Sync(ctx, "user-1", changes, testDashboardLimits), ErrWidgetLimitReached)
}
