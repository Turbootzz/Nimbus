package services

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/nimbus/backend/internal/testutil"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nimbus/backend/internal/dashdoc"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

const (
	transferAdmin = "11111111-1111-1111-1111-111111111111"
	transferUser  = "22222222-2222-2222-2222-222222222222"
	transferFresh = "33333333-3333-3333-3333-333333333333"
)

// transferSchema mirrors the migrations for SQLite, including the one
// default group per user index
const transferSchema = `
	CREATE TABLE users (id TEXT PRIMARY KEY);
	CREATE TABLE groups (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		name TEXT NOT NULL,
		color TEXT DEFAULT '#0ea5e9',
		position INTEGER DEFAULT 0,
		is_default INTEGER DEFAULT 0,
		monitoring_enabled INTEGER NOT NULL DEFAULT 1,
		tile_style TEXT NOT NULL DEFAULT 'cards' CHECK (tile_style IN ('cards', 'chips')),
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE UNIQUE INDEX idx_groups_default_per_user ON groups(user_id) WHERE is_default = TRUE;
	CREATE TABLE integrations (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
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
	CREATE TABLE services (
		id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
		user_id TEXT NOT NULL,
		name TEXT NOT NULL,
		url TEXT NOT NULL,
		icon TEXT,
		icon_type TEXT DEFAULT 'emoji',
		icon_image_path TEXT DEFAULT '',
		description TEXT,
		status TEXT NOT NULL,
		response_time INTEGER,
		position INTEGER DEFAULT 0,
		card_size TEXT DEFAULT '2x1',
		group_id TEXT,
		integration_id TEXT,
		monitoring_enabled INTEGER NOT NULL DEFAULT 1,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	);
	CREATE TABLE widgets (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		type TEXT NOT NULL,
		title TEXT NOT NULL DEFAULT '',
		group_id TEXT,
		integration_id TEXT,
		config TEXT NOT NULL DEFAULT '{}',
		card_size TEXT NOT NULL DEFAULT '2x1',
		position INTEGER NOT NULL DEFAULT 0,
		refresh_seconds INTEGER NOT NULL DEFAULT 300,
		enabled INTEGER NOT NULL DEFAULT 1,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE user_preferences (
		id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
		user_id TEXT NOT NULL UNIQUE,
		theme_mode TEXT NOT NULL DEFAULT 'auto',
		theme_background TEXT,
		theme_accent_color TEXT,
		open_in_new_tab BOOLEAN NOT NULL DEFAULT 1,
		enable_card_resizing BOOLEAN NOT NULL DEFAULT 1,
		enable_service_grouping BOOLEAN NOT NULL DEFAULT 1,
		card_scale TEXT NOT NULL DEFAULT 'medium',
		view_mode TEXT NOT NULL DEFAULT 'grid',
		wallpaper_blur INTEGER NOT NULL DEFAULT 0,
		wallpaper_dim INTEGER NOT NULL DEFAULT 0,
		card_opacity INTEGER NOT NULL DEFAULT 100,
		card_blur INTEGER NOT NULL DEFAULT 0,
		layout_mode TEXT NOT NULL DEFAULT 'canvas',
		group_display TEXT CHECK (group_display IN ('tabs', 'sections')),
		status_strip TEXT NOT NULL DEFAULT '{"enabled": false, "chips": []}',
		theme_preset TEXT NOT NULL DEFAULT 'default',
		theme_font TEXT NOT NULL DEFAULT 'default',
		kpi_mono BOOLEAN NOT NULL DEFAULT 0,
		icon_source TEXT NOT NULL DEFAULT 'dashboard-icons-svg',
		custom_css TEXT NOT NULL DEFAULT '',
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	INSERT INTO users (id) VALUES
		('11111111-1111-1111-1111-111111111111'),
		('22222222-2222-2222-2222-222222222222'),
		('33333333-3333-3333-3333-333333333333');
`

var pngImage = append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{1}, 32)...)

// fakeImages is an in-memory ImageStore that only takes PNGs
type fakeImages struct {
	files  map[string][]byte
	next   int
	pruned int
}

func (f *fakeImages) Check(_ ImageKind, data []byte) error {
	if !bytes.HasPrefix(data, pngImage[:8]) {
		return errors.New("image content does not match allowed types")
	}
	return nil
}

func (f *fakeImages) Save(kind ImageKind, data []byte) (string, func(), error) {
	if err := f.Check(kind, data); err != nil {
		return "", nil, err
	}
	f.next++
	name := fmt.Sprintf("icon-%d.png", f.next)
	if kind == ImageWallpaper {
		name = fmt.Sprintf("/uploads/wallpapers/wall-%d.png", f.next)
	}
	f.files[name] = data
	return name, func() {}, nil
}

func (f *fakeImages) Load(_ ImageKind, name string) ([]byte, error) {
	data, ok := f.files[name]
	if !ok {
		return nil, errors.New("no such file")
	}
	return data, nil
}

func (f *fakeImages) Remove(_ ImageKind, name string) { delete(f.files, name) }
func (f *fakeImages) PruneWallpapers(context.Context) { f.pruned++ }

type fakeUsers map[string]string

func (f fakeUsers) GetByID(id string) (*models.User, error) {
	role, ok := f[id]
	if !ok {
		return nil, repository.ErrUserNotFound
	}
	return &models.User{ID: id, Role: role}, nil
}

// transferEnv is a transfer service on a real (SQLite) database
type transferEnv struct {
	db           *sql.DB
	svc          *DashboardTransferService
	integrations *IntegrationService
	widgets      *WidgetService
	services     *repository.ServiceRepository
	preferences  *repository.PreferencesRepository
	images       *fakeImages
}

func newTransferEnv(t *testing.T) *transferEnv {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	// One connection: every connection to :memory: is its own database
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	_, err = db.Exec(transferSchema)
	require.NoError(t, err)
	testutil.AddBoards(t, db)
	return newTransferEnvOn(t, db)
}

// newTransferEnvOn is a transfer service on a database that has the schema
func newTransferEnvOn(t *testing.T, db *sql.DB) *transferEnv {
	t.Helper()
	cipher, err := utils.NewCipher(bytes.Repeat([]byte{9}, 32))
	require.NoError(t, err)
	integrationRepo := repository.NewIntegrationRepository(db)
	groupRepo := repository.NewGroupRepository(db)
	widgetRepo := repository.NewWidgetRepository(db)
	env := &transferEnv{
		db:           db,
		integrations: NewIntegrationService(integrationRepo, cipher, ""),
		widgets:      NewWidgetService(widgetRepo, groupRepo, integrationRepo),
		services:     repository.NewServiceRepository(db),
		preferences:  repository.NewPreferencesRepository(db),
		images:       &fakeImages{files: map[string][]byte{}},
	}
	env.svc = NewDashboardTransferService(DashboardTransferDeps{
		Dashboard:          repository.NewDashboardRepository(db),
		Groups:             groupRepo,
		Services:           env.services,
		Widgets:            widgetRepo,
		Integrations:       integrationRepo,
		Preferences:        env.preferences,
		Users:              fakeUsers{transferAdmin: "admin", transferUser: "user", transferFresh: "admin"},
		IntegrationService: env.integrations,
		Images:             env.images,
	})
	return env
}

func (e *transferEnv) addGroup(t *testing.T, userID, name string, position int, isDefault bool) string {
	t.Helper()
	id := uuid.New().String()
	_, err := e.db.Exec(`INSERT INTO groups (id, user_id, name, color, position, is_default) VALUES (?, ?, ?, '#ff0000', ?, ?)`,
		id, userID, name, position, isDefault)
	require.NoError(t, err)
	return id
}

func (e *transferEnv) addService(t *testing.T, userID string, service models.Service) {
	t.Helper()
	service.UserID, service.Status = userID, models.StatusOnline
	service.CreatedAt, service.UpdatedAt = time.Now(), time.Now()
	if service.IconType == "" {
		service.IconType, service.Icon = models.IconTypeEmoji, "🎬"
	}
	if service.CardSize == "" {
		service.CardSize = models.CardSize2x1
	}
	require.NoError(t, e.services.Create(context.Background(), &service))
}

func (e *transferEnv) addWidget(t *testing.T, userID string, req models.WidgetRequest) *models.Widget {
	t.Helper()
	widget, err := e.widgets.Create(context.Background(), userID, true, &req)
	require.NoError(t, err)
	return widget
}

// seedDashboard gives the admin a dashboard that uses everything a document
// can hold: groups, integrations with credentials, services with every icon
// type, widgets with secrets and integration lists, and preferences with an
// uploaded wallpaper and status strip chips
func (e *transferEnv) seedDashboard(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	defaultGroup := e.addGroup(t, transferAdmin, "Services", 0, true)
	media := e.addGroup(t, transferAdmin, "Media", 3, false)
	_, err := e.db.Exec(`UPDATE groups SET tile_style = 'chips' WHERE id = ?`, media)
	require.NoError(t, err)

	sonarr, err := e.integrations.Create(ctx, transferAdmin, true, &models.IntegrationRequest{
		Kind: "sonarr", Name: "Sonarr", BaseURL: "http://192.168.1.10:8989",
		Credentials: &models.IntegrationCredentials{APIKey: "sonarr-secret-key"},
	})
	require.NoError(t, err)
	_, err = e.integrations.Create(ctx, transferAdmin, true, &models.IntegrationRequest{
		Kind: "svc-fake", Name: "Plain", BaseURL: "http://192.168.1.11", AuthType: models.IntegrationAuthNone,
		VerifyTLS: new(bool), Options: json.RawMessage(`{"node":"pve"}`),
	})
	require.NoError(t, err)

	icon, _, err := e.images.Save(ImageIcon, pngImage)
	require.NoError(t, err)
	e.addService(t, transferAdmin, models.Service{
		Name: "Sonarr", URL: "http://192.168.1.10:8989", Description: "Series", GroupID: &media, IntegrationID: &sonarr.ID,
		Icon: "📺", IconType: models.IconTypeImageUpload, IconImagePath: icon, CardSize: models.CardSize2x2, MonitoringEnabled: true,
	})
	e.addService(t, transferAdmin, models.Service{Name: "Plex", URL: "https://plex.example.com", GroupID: &defaultGroup})
	e.addService(t, transferAdmin, models.Service{
		Name: "Docs", URL: "https://docs.example.com", MonitoringEnabled: true,
		IconType: models.IconTypeImageURL, IconImagePath: "https://cdn.example.com/docs.png",
	})

	e.addWidget(t, transferAdmin, models.WidgetRequest{
		Type: "markdown", Title: new("Notes"), GroupID: &media, Config: json.RawMessage(`{"content":"hello"}`),
	})
	e.addWidget(t, transferAdmin, models.WidgetRequest{
		Type: "calendar", Title: new("Upcoming"),
		Config: json.RawMessage(fmt.Sprintf(`{"integrations":[%q],"ical_urls":["https://192.168.1.20/private/abc123.ics"],"days":7}`, sonarr.ID)),
	})
	api := e.addWidget(t, transferAdmin, models.WidgetRequest{
		Type: "custom_api", Title: new("Temperature"), Enabled: new(bool),
		Config: json.RawMessage(`{"url":"http://192.168.1.30/api","headers":[{"name":"X-Key","value":"header-s3cret"}],"fields":[{"label":"Temp","path":"temp","unit":"C"}]}`),
	})

	wallpaper, _, err := e.images.Save(ImageWallpaper, pngImage)
	require.NoError(t, err)
	require.NoError(t, e.preferences.Upsert(ctx, transferAdmin, &models.PreferencesUpdateRequest{
		ThemeMode:        new("dark"),
		ThemeBackground:  models.NullableString{Value: &wallpaper, Set: true},
		ThemeAccentColor: models.NullableString{Value: new("#3B82F6"), Set: true},
		OpenInNewTab:     new(bool),
		CardScale:        new("large"),
		WallpaperBlur:    new(4),
		CardOpacity:      new(80),
		LayoutMode:       new("classic"),
		GroupDisplay:     new("sections"),
		ThemePreset:      new("nord"),
		ThemeFont:        new("inter"),
		KpiMono:          new(true),
		IconSource:       new("selfhst"),
		CustomCSS:        new(seededCSS),
		StatusStrip: &models.StatusStrip{Enabled: true, Chips: []models.StatusChip{
			{Source: "integration", ID: sonarr.ID, KPI: "queue"},
			{Source: "widget", ID: api.ID, KPI: "Temp"},
		}},
	}))
}

const seededCSS = `.glass-card { border-radius: 0 }`

func (e *transferEnv) export(t *testing.T, userID string) *dashdoc.Document {
	t.Helper()
	doc, err := e.svc.Export(context.Background(), userID)
	require.NoError(t, err)
	return doc
}

func toJSON(t *testing.T, v any) string {
	t.Helper()
	raw, err := json.MarshalIndent(v, "", "  ")
	require.NoError(t, err)
	return string(raw)
}

// reparse sends a document through JSON, as a file would
func reparse(t *testing.T, doc *dashdoc.Document) *dashdoc.Document {
	t.Helper()
	var out dashdoc.Document
	require.NoError(t, json.Unmarshal([]byte(toJSON(t, doc)), &out))
	return &out
}

func issuePaths(t *testing.T, err error) []string {
	t.Helper()
	var invalid *dashdoc.ValidationError
	require.ErrorAs(t, err, &invalid)
	paths := make([]string, len(invalid.Issues))
	for i, issue := range invalid.Issues {
		paths[i] = issue.Path
	}
	return paths
}

func (e *transferEnv) count(t *testing.T, table, userID string) int {
	t.Helper()
	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM `+table+` WHERE user_id = ?`, userID).Scan(&n))
	return n
}

func TestExportUsesRefsAndCarriesNoSecrets(t *testing.T) {
	env := newTransferEnv(t)
	env.seedDashboard(t)

	doc := env.export(t, transferAdmin)
	raw := toJSON(t, doc)

	assert.Equal(t, 1, doc.Version)
	assert.NotRegexp(t, regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`), raw, "no database IDs")
	for _, secret := range []string{"sonarr-secret-key", "header-s3cret", "abc123", "icon-1.png", "wall-"} {
		assert.NotContains(t, raw, secret)
	}

	require.Len(t, doc.Groups, 2)
	assert.Equal(t, dashdoc.Group{Ref: "services", Name: "Services", Color: "#ff0000", Default: true, MonitoringEnabled: new(true), TileStyle: "cards"}, doc.Groups[0])
	assert.Equal(t, "media", doc.Groups[1].Ref)
	assert.Equal(t, "chips", doc.Groups[1].TileStyle)
	assert.Equal(t, new("sections"), doc.Preferences.GroupDisplay)
	assert.Equal(t, new("nord"), doc.Preferences.ThemePreset)
	assert.Equal(t, new("inter"), doc.Preferences.ThemeFont)
	assert.Equal(t, new(true), doc.Preferences.KpiMono)
	assert.Equal(t, new("selfhst"), doc.Preferences.IconSource)
	assert.Equal(t, new(seededCSS), doc.Preferences.CustomCSS)

	require.Len(t, doc.Integrations, 2)
	assert.Equal(t, "plain", doc.Integrations[0].Ref)
	assert.False(t, doc.Integrations[0].NeedsCredentials)
	assert.False(t, *doc.Integrations[0].VerifyTLS)
	assert.JSONEq(t, `{"node":"pve"}`, string(doc.Integrations[0].Options))
	assert.Equal(t, "sonarr", doc.Integrations[1].Ref)
	assert.True(t, doc.Integrations[1].NeedsCredentials)

	require.Len(t, doc.Services, 3)
	sonarr := doc.Services[0]
	assert.Equal(t, "media", sonarr.GroupRef)
	assert.Equal(t, "sonarr", sonarr.IntegrationRef)
	assert.Equal(t, models.IconTypeImageUpload, sonarr.IconType)
	assert.Equal(t, base64.StdEncoding.EncodeToString(pngImage), sonarr.IconData)
	assert.Equal(t, "services", doc.Services[1].GroupRef)
	assert.Equal(t, "https://cdn.example.com/docs.png", doc.Services[2].IconURL)

	require.Len(t, doc.Widgets, 3)
	calendar := doc.Widgets[1]
	assert.True(t, calendar.NeedsSecrets)
	assert.JSONEq(t, `{"integrations":["sonarr"],"ical_urls":["https://192.168.1.20/********"],"days":7,"verify_tls":true}`, string(calendar.Config))
	assert.True(t, doc.Widgets[2].NeedsSecrets)
	assert.False(t, *doc.Widgets[2].Enabled)
	assert.False(t, doc.Widgets[0].NeedsSecrets)

	prefs := doc.Preferences
	require.NotNil(t, prefs)
	assert.Nil(t, prefs.ThemeBackground)
	assert.Equal(t, base64.StdEncoding.EncodeToString(pngImage), prefs.WallpaperData)
	assert.Equal(t, []dashdoc.Chip{
		{Source: "integration", Ref: "sonarr", KPI: "queue"},
		{Source: "widget", Ref: "temperature", KPI: "Temp"},
	}, prefs.StatusStrip.Chips)
	assert.Zero(t, doc.OmittedImages)
}

func TestExportLeavesOutMissingImages(t *testing.T) {
	env := newTransferEnv(t)
	env.seedDashboard(t)
	env.images.files = map[string][]byte{}

	doc := env.export(t, transferAdmin)

	assert.Equal(t, 2, doc.OmittedImages)
	assert.Equal(t, models.IconTypeEmoji, doc.Services[0].IconType)
	assert.Empty(t, doc.Services[0].IconData)
	// Neither a URL nor data: an import keeps the wallpaper it finds
	assert.Nil(t, doc.Preferences.ThemeBackground)
	assert.Empty(t, doc.Preferences.WallpaperData)
}

// A widget that lost its integration can't be imported; an export that held
// it could never be restored
func TestExportLeavesOutWidgetsWithoutTheirIntegration(t *testing.T) {
	env := newTransferEnv(t)
	ctx := context.Background()
	app, err := env.integrations.Create(ctx, transferAdmin, true, &models.IntegrationRequest{
		Kind: "svc-fake", Name: "App", BaseURL: "http://192.168.1.11", AuthType: models.IntegrationAuthNone,
	})
	require.NoError(t, err)
	sonarr, err := env.integrations.Create(ctx, transferAdmin, true, &models.IntegrationRequest{
		Kind: "sonarr", Name: "Sonarr", BaseURL: "http://192.168.1.10:8989",
		Credentials: &models.IntegrationCredentials{APIKey: "key"},
	})
	require.NoError(t, err)
	env.addWidget(t, transferAdmin, models.WidgetRequest{Type: "svc-queue", IntegrationID: &app.ID})
	env.addWidget(t, transferAdmin, models.WidgetRequest{
		Type: "calendar", Config: json.RawMessage(fmt.Sprintf(`{"integrations":[%q]}`, sonarr.ID)),
	})
	env.addWidget(t, transferAdmin, models.WidgetRequest{
		Type: "calendar", Title: new("Feeds"),
		Config: json.RawMessage(fmt.Sprintf(`{"integrations":[%q],"ical_urls":["https://192.168.1.20/a.ics"]}`, sonarr.ID)),
	})
	env.addWidget(t, transferAdmin, models.WidgetRequest{Type: "markdown", Title: new("Notes")})
	require.Len(t, env.export(t, transferAdmin).Widgets, 4)

	// SQLite has no ON DELETE SET NULL here; do what Postgres does
	require.NoError(t, env.integrations.Delete(ctx, app.ID, transferAdmin))
	require.NoError(t, env.integrations.Delete(ctx, sonarr.ID, transferAdmin))
	_, err = env.db.Exec(`UPDATE widgets SET integration_id = NULL`)
	require.NoError(t, err)

	doc := env.export(t, transferAdmin)

	titles := []string{}
	for _, widget := range doc.Widgets {
		titles = append(titles, widget.Title)
	}
	assert.ElementsMatch(t, []string{"Feeds", "Notes"}, titles)
	_, err = env.svc.Apply(ctx, transferAdmin, reparse(t, doc), dashdoc.ModeReplace)
	require.NoError(t, err)
}

func TestHasNUL(t *testing.T) {
	assert.True(t, hasNUL([]byte(`{"a":"x\u0000"}`)))
	assert.True(t, hasNUL([]byte(`{"a":"\\\u0000"}`)))
	// An escaped backslash followed by the text u0000
	assert.False(t, hasNUL([]byte(`{"a":"\\u0000"}`)))
	assert.False(t, hasNUL([]byte(`{"a":"\u0001"}`)))
}

func TestExportRenumbersPositionsADocumentCannotHold(t *testing.T) {
	env := newTransferEnv(t)
	env.addService(t, transferUser, models.Service{Name: "First", URL: "http://first.lan"})
	env.addService(t, transferUser, models.Service{Name: "Last", URL: "http://last.lan"})
	env.addWidget(t, transferUser, models.WidgetRequest{Type: "markdown"})
	_, err := env.db.Exec(`UPDATE services SET position = 2000000000 WHERE name = 'Last'`)
	require.NoError(t, err)

	doc := env.export(t, transferUser)

	assert.Equal(t, []int{0, 2}, []int{doc.Services[0].Position, doc.Services[1].Position})
	assert.Equal(t, 1, doc.Widgets[0].Position)
	_, err = env.svc.Apply(context.Background(), transferUser, doc, dashdoc.ModeReplace)
	require.NoError(t, err)
	assert.Equal(t, toJSON(t, doc), toJSON(t, env.export(t, transferUser)))
}

func TestRoundTripReplaceGivesIdenticalDashboard(t *testing.T) {
	env := newTransferEnv(t)
	env.seedDashboard(t)
	ctx := context.Background()
	first := env.export(t, transferAdmin)

	summary, err := env.svc.Apply(ctx, transferAdmin, reparse(t, first), dashdoc.ModeReplace)
	require.NoError(t, err)

	assert.Equal(t, &dashdoc.Summary{
		Mode:             dashdoc.ModeReplace,
		Groups:           dashdoc.Count{Created: 2, Removed: 2},
		Services:         dashdoc.Count{Created: 3, Removed: 3},
		Widgets:          dashdoc.Count{Created: 3, Removed: 3},
		Integrations:     dashdoc.Count{Created: 2, Removed: 2},
		Preferences:      true,
		NeedsCredentials: 1,
		NeedsSecrets:     2,
		CustomCSS:        seededCSS,
	}, summary)
	assert.Equal(t, toJSON(t, first), toJSON(t, env.export(t, transferAdmin)))

	// The old icon is gone, the new images are there, and the sweep that
	// takes the old wallpaper has run
	assert.NotContains(t, env.images.files, "icon-1.png")
	assert.Contains(t, env.images.files, "icon-3.png")
	assert.Contains(t, env.images.files, "/uploads/wallpapers/wall-4.png")
	assert.Equal(t, 1, env.images.pruned)
}

func TestRoundTripIntoEmptyAccount(t *testing.T) {
	env := newTransferEnv(t)
	env.seedDashboard(t)
	first := env.export(t, transferAdmin)

	_, err := env.svc.Apply(context.Background(), transferFresh, reparse(t, first), dashdoc.ModeReplace)
	require.NoError(t, err)

	assert.Equal(t, toJSON(t, first), toJSON(t, env.export(t, transferFresh)))
	// The source account is untouched
	assert.Equal(t, toJSON(t, first), toJSON(t, env.export(t, transferAdmin)))
}

// An export from before tile styles and the group display has neither field
func TestImportWithoutTileStyleAndGroupDisplay(t *testing.T) {
	env := newTransferEnv(t)
	env.seedDashboard(t)
	ctx := context.Background()
	groupDisplay := func() *string {
		prefs, err := env.preferences.GetByUserID(ctx, transferAdmin)
		require.NoError(t, err)
		return prefs.GroupDisplay
	}
	doc := func(raw string) *dashdoc.Document {
		var d dashdoc.Document
		require.NoError(t, json.Unmarshal([]byte(raw), &d))
		return &d
	}

	// No preferences at all: they stay as they are
	_, err := env.svc.Apply(ctx, transferAdmin, doc(`{"version": 1, "groups": [{"name": "Old"}]}`), dashdoc.ModeReplace)
	require.NoError(t, err)
	assert.Equal(t, "cards", env.export(t, transferAdmin).Groups[0].TileStyle)
	assert.Equal(t, new("sections"), groupDisplay())

	// Preferences without group_display: the groups follow the layout
	_, err = env.svc.Apply(ctx, transferAdmin, doc(`{"version": 1, "preferences": {"theme_mode": "light"}}`), dashdoc.ModeReplace)
	require.NoError(t, err)
	assert.Nil(t, groupDisplay())
	exported := env.export(t, transferAdmin)
	assert.Nil(t, exported.Preferences.GroupDisplay)
	assert.NotContains(t, toJSON(t, exported), "group_display")
}

// An export from before the theme settings has none of them: they keep
// their values, and a file can still clear the CSS with ""
func TestImportWithoutThemeSettings(t *testing.T) {
	env := newTransferEnv(t)
	env.seedDashboard(t)
	ctx := context.Background()
	apply := func(raw string) (*models.UserPreferences, error) {
		var doc dashdoc.Document
		require.NoError(t, json.Unmarshal([]byte(raw), &doc))
		_, applyErr := env.svc.Apply(ctx, transferAdmin, &doc, dashdoc.ModeReplace)
		prefs, err := env.preferences.GetByUserID(ctx, transferAdmin)
		require.NoError(t, err)
		return prefs, applyErr
	}

	prefs, err := apply(`{"version": 1, "preferences": {"theme_mode": "light"}}`)
	require.NoError(t, err)
	assert.Equal(t, "light", prefs.ThemeMode)
	assert.Equal(t, "nord", prefs.ThemePreset)
	assert.Equal(t, "inter", prefs.ThemeFont)
	assert.True(t, prefs.KpiMono)
	assert.Equal(t, "selfhst", prefs.IconSource)
	assert.Equal(t, seededCSS, prefs.CustomCSS)

	// CSS that loads from another host refuses the whole import
	prefs, err = apply(`{"version": 1, "preferences": {"theme_mode": "dark", "theme_preset": "dracula",
		"custom_css": "body { background: \\75rl(https://evil.example/a.png) }"}}`)
	var invalid *dashdoc.ValidationError
	require.ErrorAs(t, err, &invalid)
	assert.Equal(t, "preferences.custom_css", invalid.Issues[0].Path)
	assert.Equal(t, "light", prefs.ThemeMode)
	assert.Equal(t, "nord", prefs.ThemePreset)
	assert.Equal(t, seededCSS, prefs.CustomCSS)

	// The preview shows the CSS under whatever spelling the server accepts,
	// and a merge, which leaves the preferences alone, has none to show
	var shouted dashdoc.Document
	require.NoError(t, json.Unmarshal([]byte(`{"version": 1, "PREFERENCES": {"CUSTOM_CSS": "a { color: red }"}}`), &shouted))
	summary, err := env.svc.Preview(ctx, transferAdmin, &shouted, dashdoc.ModeReplace)
	require.NoError(t, err)
	assert.Equal(t, "a { color: red }", summary.CustomCSS)
	summary, err = env.svc.Preview(ctx, transferAdmin, &shouted, dashdoc.ModeMerge)
	require.NoError(t, err)
	assert.Empty(t, summary.CustomCSS)

	prefs, err = apply(`{"version": 1, "preferences": {"custom_css": ""}}`)
	require.NoError(t, err)
	assert.Empty(t, prefs.CustomCSS)
}

func TestImportedIntegrationHasNoCredentialsAndIsNotPolled(t *testing.T) {
	env := newTransferEnv(t)
	env.seedDashboard(t)
	ctx := context.Background()
	_, err := env.svc.Apply(ctx, transferAdmin, env.export(t, transferAdmin), dashdoc.ModeReplace)
	require.NoError(t, err)

	list, err := env.integrations.List(ctx, transferAdmin)
	require.NoError(t, err)
	require.Len(t, list, 2)
	sonarr := list[1]
	assert.Equal(t, "Sonarr", sonarr.Name)
	assert.False(t, sonarr.HasCredentials)

	_, err = env.integrations.Fetch(ctx, &sonarr)
	assert.ErrorIs(t, err, errMissingCredentials)
	result, err := env.integrations.TestSaved(ctx, sonarr.ID, transferAdmin)
	require.NoError(t, err)
	assert.Contains(t, result.Error, "Credentials are missing")

	// Entering them makes it a normal integration again
	updated, err := env.integrations.Update(ctx, sonarr.ID, transferAdmin, true, &models.IntegrationRequest{
		Credentials: &models.IntegrationCredentials{APIKey: "new-key"},
	})
	require.NoError(t, err)
	assert.True(t, updated.HasCredentials)
}

func TestReplaceOnlyTouchesTheUsersOwnRows(t *testing.T) {
	env := newTransferEnv(t)
	env.seedDashboard(t)
	ctx := context.Background()
	otherGroup := env.addGroup(t, transferUser, "Theirs", 0, true)
	env.addService(t, transferUser, models.Service{Name: "Their service", URL: "http://their.lan", GroupID: &otherGroup})
	env.addWidget(t, transferUser, models.WidgetRequest{Type: "markdown", Config: json.RawMessage(`{"content":"theirs"}`)})
	_, err := env.integrations.Create(ctx, transferUser, false, &models.IntegrationRequest{
		Kind: "svc-fake", Name: "Theirs", BaseURL: "http://192.168.1.50", AuthType: models.IntegrationAuthNone,
	})
	require.NoError(t, err)
	theirs := toJSON(t, env.export(t, transferUser))
	assert.NotContains(t, theirs, "Sonarr", "an export only holds the user's own rows")

	empty := &dashdoc.Document{Version: 1}
	summary, err := env.svc.Apply(ctx, transferAdmin, empty, dashdoc.ModeReplace)
	require.NoError(t, err)

	assert.Equal(t, 3, summary.Services.Removed)
	for _, table := range []string{"groups", "services", "widgets", "integrations"} {
		assert.Zero(t, env.count(t, table, transferAdmin), table)
	}
	assert.Equal(t, theirs, toJSON(t, env.export(t, transferUser)))
}

// fakeStatusPage is a status page that shows one group; failing makes
// UseGroup fail
type fakeStatusPage struct {
	models.StatusPageSettings
	failing bool
}

func (f *fakeStatusPage) Settings(context.Context) (models.StatusPageSettings, error) {
	return f.StatusPageSettings, nil
}

func (f *fakeStatusPage) UseGroup(_ context.Context, _, groupID string) error {
	if f.failing {
		return errors.New("database is down")
	}
	f.GroupID = groupID
	return nil
}

func (e *transferEnv) groupID(t *testing.T, userID, name string) string {
	t.Helper()
	var id string
	require.NoError(t, e.db.QueryRow(`SELECT id FROM groups WHERE user_id = ? AND name = ?`, userID, name).Scan(&id))
	return id
}

func TestReplaceKeepsTheStatusPageOnItsGroup(t *testing.T) {
	env := newTransferEnv(t)
	env.seedDashboard(t)
	ctx := context.Background()
	media := env.groupID(t, transferAdmin, "Media")
	page := &fakeStatusPage{StatusPageSettings: models.StatusPageSettings{Enabled: true, GroupID: media}}
	env.svc.SetStatusPage(page)
	doc := env.export(t, transferAdmin)

	// Another user's replace leaves the page alone
	summary, err := env.svc.Apply(ctx, transferFresh, reparse(t, doc), dashdoc.ModeReplace)
	require.NoError(t, err)
	assert.Nil(t, summary.StatusPage)
	assert.Equal(t, media, page.GroupID)

	preview, err := env.svc.Preview(ctx, transferAdmin, reparse(t, doc), dashdoc.ModeReplace)
	require.NoError(t, err)
	assert.Equal(t, &dashdoc.StatusPageImpact{Group: "Media", Kept: true}, preview.StatusPage)
	assert.Equal(t, media, page.GroupID, "a preview moves nothing")

	summary, err = env.svc.Apply(ctx, transferAdmin, reparse(t, doc), dashdoc.ModeReplace)
	require.NoError(t, err)
	assert.Equal(t, preview.StatusPage, summary.StatusPage)
	recreated := env.groupID(t, transferAdmin, "Media")
	assert.NotEqual(t, media, recreated, "a replace creates new groups")
	assert.Equal(t, recreated, page.GroupID, "the page follows its group")

	// A page that is off follows too, but is not worth a note
	page.Enabled = false
	summary, err = env.svc.Apply(ctx, transferAdmin, reparse(t, doc), dashdoc.ModeReplace)
	require.NoError(t, err)
	assert.Nil(t, summary.StatusPage)
	assert.Equal(t, env.groupID(t, transferAdmin, "Media"), page.GroupID)

	// Without a group of that name it stays on the removed one: offline
	page.Enabled = true
	for i := range doc.Groups {
		if doc.Groups[i].Name == "Media" {
			doc.Groups[i].Name = "Movies"
		}
	}
	before := page.GroupID
	summary, err = env.svc.Apply(ctx, transferAdmin, reparse(t, doc), dashdoc.ModeReplace)
	require.NoError(t, err)
	assert.Equal(t, &dashdoc.StatusPageImpact{Group: "Media", Kept: false}, summary.StatusPage)
	assert.Equal(t, before, page.GroupID)
}

// The page belongs to an admin: the replace of a demoted owner leaves it
func TestReplaceOfANonAdminLeavesTheStatusPage(t *testing.T) {
	env := newTransferEnv(t)
	ctx := context.Background()
	group := env.addGroup(t, transferUser, "Media", 0, true)
	page := &fakeStatusPage{StatusPageSettings: models.StatusPageSettings{Enabled: true, GroupID: group}}
	env.svc.SetStatusPage(page)

	doc := &dashdoc.Document{Version: 1, Groups: []dashdoc.Group{{Ref: "media", Name: "Media", Default: true}}}
	summary, err := env.svc.Apply(ctx, transferUser, doc, dashdoc.ModeReplace)
	require.NoError(t, err)
	assert.Nil(t, summary.StatusPage)
	assert.Equal(t, group, page.GroupID)
}

// A page that could not be moved shows the removed group, and the summary says so
func TestReplaceSaysWhenTheStatusPageCouldNotMove(t *testing.T) {
	env := newTransferEnv(t)
	env.seedDashboard(t)
	media := env.groupID(t, transferAdmin, "Media")
	page := &fakeStatusPage{StatusPageSettings: models.StatusPageSettings{Enabled: true, GroupID: media}, failing: true}
	env.svc.SetStatusPage(page)

	summary, err := env.svc.Apply(context.Background(), transferAdmin, reparse(t, env.export(t, transferAdmin)), dashdoc.ModeReplace)
	require.NoError(t, err, "the import itself is done")
	assert.Equal(t, &dashdoc.StatusPageImpact{Group: "Media", Kept: false}, summary.StatusPage)
	assert.Equal(t, media, page.GroupID)
}

func TestMergeAddsBehindWhatTheUserHas(t *testing.T) {
	env := newTransferEnv(t)
	ctx := context.Background()
	defaultGroup := env.addGroup(t, transferUser, "Services", 0, true)
	env.addGroup(t, transferUser, "Mine", 1, false)
	env.addService(t, transferUser, models.Service{Name: "Mine", URL: "http://mine.lan", GroupID: &defaultGroup})
	env.addWidget(t, transferUser, models.WidgetRequest{Type: "markdown", Config: json.RawMessage(`{"content":"mine"}`)})
	require.NoError(t, env.preferences.Upsert(ctx, transferUser, &models.PreferencesUpdateRequest{ThemeMode: new("light")}))

	doc := &dashdoc.Document{
		Version: 1,
		Groups: []dashdoc.Group{
			{Ref: "main", Name: "Their default", Default: true},
			{Ref: "tools", Name: "Tools"},
		},
		Integrations: []dashdoc.Integration{{Ref: "app", Kind: "svc-fake", Name: "App", BaseURL: "http://192.168.1.60"}},
		Services: []dashdoc.Service{
			{Name: "Gitea", URL: "http://gitea.lan", Position: 40, GroupRef: "tools", IntegrationRef: "app"},
			{Name: "Wiki", URL: "http://wiki.lan", Position: 10, GroupRef: "main"},
		},
		Widgets:     []dashdoc.Widget{{Type: "markdown", Position: 40, Config: json.RawMessage(`{"content":"imported"}`)}},
		Preferences: &dashdoc.Preferences{ThemeMode: new("dark")},
	}
	summary, err := env.svc.Apply(ctx, transferUser, doc, dashdoc.ModeMerge)
	require.NoError(t, err)

	assert.Equal(t, &dashdoc.Summary{
		Mode: dashdoc.ModeMerge,
		// The document's default group became the one the user has
		Groups:           dashdoc.Count{Created: 1},
		Services:         dashdoc.Count{Created: 2},
		Widgets:          dashdoc.Count{Created: 1},
		Integrations:     dashdoc.Count{Created: 1},
		NeedsCredentials: 1,
	}, summary)

	merged := env.export(t, transferUser)
	require.Len(t, merged.Groups, 3)
	assert.Equal(t, []string{"Services", "Mine", "Tools"}, []string{merged.Groups[0].Name, merged.Groups[1].Name, merged.Groups[2].Name})

	// Existing tiles keep positions 0 and 1; the imported ones follow in
	// their own order, and tiles that shared a position still do
	positions := map[string]int{}
	for _, service := range merged.Services {
		positions[service.Name] = service.Position
	}
	assert.Equal(t, map[string]int{"Mine": 0, "Wiki": 2, "Gitea": 3}, positions)
	assert.Equal(t, []int{1, 3}, []int{merged.Widgets[0].Position, merged.Widgets[1].Position})

	groupOf := map[string]string{}
	for _, service := range merged.Services {
		groupOf[service.Name] = service.GroupRef
	}
	assert.Equal(t, map[string]string{"Mine": "services", "Wiki": "services", "Gitea": "tools"}, groupOf)
	assert.Equal(t, "light", *merged.Preferences.ThemeMode, "a merge keeps the user's preferences")

	// Merging the same document again adds everything once more
	_, err = env.svc.Apply(ctx, transferUser, doc, dashdoc.ModeMerge)
	require.NoError(t, err)
	assert.Equal(t, 5, env.count(t, "services", transferUser))
}

func TestMergeCreatesTheDefaultGroupWhenTheUserHasNone(t *testing.T) {
	env := newTransferEnv(t)
	doc := &dashdoc.Document{Version: 1, Groups: []dashdoc.Group{{Ref: "main", Name: "Main", Default: true}}}

	summary, err := env.svc.Apply(context.Background(), transferUser, doc, dashdoc.ModeMerge)
	require.NoError(t, err)

	assert.Equal(t, 1, summary.Groups.Created)
	assert.True(t, env.export(t, transferUser).Groups[0].Default)
}

func TestPreviewWritesNothing(t *testing.T) {
	env := newTransferEnv(t)
	env.seedDashboard(t)
	ctx := context.Background()
	before := env.export(t, transferAdmin)
	files := len(env.images.files)

	for _, mode := range []dashdoc.Mode{dashdoc.ModeMerge, dashdoc.ModeReplace} {
		preview, err := env.svc.Preview(ctx, transferAdmin, before, mode)
		require.NoError(t, err)
		assert.True(t, preview.DryRun)
		assert.Equal(t, toJSON(t, before), toJSON(t, env.export(t, transferAdmin)), mode)
		assert.Len(t, env.images.files, files)

		if mode == dashdoc.ModeReplace {
			applied, err := env.svc.Apply(ctx, transferAdmin, before, mode)
			require.NoError(t, err)
			preview.DryRun = false
			assert.Equal(t, preview, applied, "the preview tells what the import does")
		} else {
			assert.Equal(t, dashdoc.Count{Created: 1}, preview.Groups)
			assert.False(t, preview.Preferences)
		}
	}
}

func TestFailedImportLeavesEverythingAsItWas(t *testing.T) {
	env := newTransferEnv(t)
	env.seedDashboard(t)
	before := toJSON(t, env.export(t, transferAdmin))
	files := len(env.images.files)
	// The last insert of the transaction fails
	_, err := env.db.Exec(`CREATE TRIGGER fail_widget BEFORE INSERT ON widgets WHEN NEW.title = 'boom'
		BEGIN SELECT RAISE(ABORT, 'boom'); END`)
	require.NoError(t, err)

	doc := env.export(t, transferAdmin)
	doc.Widgets = append(doc.Widgets, dashdoc.Widget{Type: "markdown", Title: "boom", Position: 99})
	_, err = env.svc.Apply(context.Background(), transferAdmin, doc, dashdoc.ModeReplace)

	require.Error(t, err)
	var invalid *dashdoc.ValidationError
	assert.False(t, errors.As(err, &invalid))
	assert.Equal(t, before, toJSON(t, env.export(t, transferAdmin)))
	assert.Len(t, env.images.files, files, "images written for the import are removed again")
	assert.Zero(t, env.images.pruned)
}

func TestAdminOnlyEntitiesNeedAnAdmin(t *testing.T) {
	env := newTransferEnv(t)
	doc := &dashdoc.Document{
		Version:      1,
		Integrations: []dashdoc.Integration{{Ref: "docker", Kind: "docker", Name: "Docker", BaseURL: "http://192.168.1.70:2375"}},
		Widgets: []dashdoc.Widget{
			{Type: "markdown", Config: json.RawMessage(`{"content":"fine"}`)},
			{Type: "custom_api", Position: 1, Config: json.RawMessage(`{"url":"http://192.168.1.30/api","fields":[{"label":"Temp","path":"temp"}]}`)},
			{Type: "system_stats", Position: 2},
		},
	}
	ctx := context.Background()

	for _, run := range []func(context.Context, string, *dashdoc.Document, dashdoc.Mode) (*dashdoc.Summary, error){env.svc.Preview, env.svc.Apply} {
		_, err := run(ctx, transferUser, doc, dashdoc.ModeMerge)
		assert.Equal(t, []string{"integrations[0]", "widgets[1]", "widgets[2]"}, issuePaths(t, err))
	}
	assert.Zero(t, env.count(t, "widgets", transferUser), "nothing is imported when part of it is refused")

	summary, err := env.svc.Apply(ctx, transferAdmin, doc, dashdoc.ModeMerge)
	require.NoError(t, err)
	assert.Equal(t, 3, summary.Widgets.Created)
}

func TestInvalidDocuments(t *testing.T) {
	env := newTransferEnv(t)
	service := func(change func(*dashdoc.Service)) *dashdoc.Document {
		s := dashdoc.Service{Name: "A", URL: "http://a.lan"}
		change(&s)
		return &dashdoc.Document{Version: 1, Services: []dashdoc.Service{s}}
	}
	widget := func(w dashdoc.Widget) *dashdoc.Document {
		return &dashdoc.Document{Version: 1, Widgets: []dashdoc.Widget{w}}
	}
	prefs := func(p dashdoc.Preferences) *dashdoc.Document {
		return &dashdoc.Document{Version: 1, Preferences: &p}
	}

	tests := []struct {
		name string
		doc  *dashdoc.Document
		path string
	}{
		{"unknown version", &dashdoc.Document{Version: 2}, "version"},
		{"missing version", &dashdoc.Document{}, "version"},
		{"group without name", &dashdoc.Document{Version: 1, Groups: []dashdoc.Group{{Ref: "g"}}}, "groups[0]"},
		{"group tile style", &dashdoc.Document{Version: 1, Groups: []dashdoc.Group{{Name: "G", TileStyle: "grid"}}}, "groups[0]"},
		{"group display", prefs(dashdoc.Preferences{GroupDisplay: new("grid")}), "preferences"},
		{"theme preset", prefs(dashdoc.Preferences{ThemePreset: new("solarized")}), "preferences"},
		{"theme font", prefs(dashdoc.Preferences{ThemeFont: new("comic-sans")}), "preferences"},
		{"icon source", prefs(dashdoc.Preferences{IconSource: new("https://evil.example")}), "preferences"},
		{"custom css with a url", prefs(dashdoc.Preferences{CustomCSS: new("a { background: url(https://evil.example) }")}), "preferences.custom_css"},
		{"custom css with an import", prefs(dashdoc.Preferences{CustomCSS: new(`@import "https://evil.example/x.css";`)}), "preferences.custom_css"},
		{"custom css too large", prefs(dashdoc.Preferences{CustomCSS: new(strings.Repeat("a", utils.MaxCustomCSSBytes+1))}), "preferences.custom_css"},
		{"group color", &dashdoc.Document{Version: 1, Groups: []dashdoc.Group{{Name: "G", Color: "red"}}}, "groups[0]"},
		{"unknown integration kind", &dashdoc.Document{Version: 1, Integrations: []dashdoc.Integration{{Kind: "nope", Name: "X", BaseURL: "http://192.168.1.2"}}}, "integrations[0]"},
		{"integration url with credentials", &dashdoc.Document{Version: 1, Integrations: []dashdoc.Integration{{Kind: "svc-fake", Name: "X", BaseURL: "http://user:pass@192.168.1.2"}}}, "integrations[0]"},
		{"service without url", service(func(s *dashdoc.Service) { s.URL = "" }), "services[0]"},
		{"service url without scheme", service(func(s *dashdoc.Service) { s.URL = "a.lan" }), "services[0]"},
		{"service card size", service(func(s *dashdoc.Service) { s.CardSize = "9x9" }), "services[0]"},
		{"service dangling group", service(func(s *dashdoc.Service) { s.GroupRef = "nope" }), "services[0].group_ref"},
		{"service dangling integration", service(func(s *dashdoc.Service) { s.IntegrationRef = "nope" }), "services[0].integration_ref"},
		{"service name too long", service(func(s *dashdoc.Service) { s.Name = strings.Repeat("a", models.MaxServiceNameLen+1) }), "services[0]"},
		{"service icon too long", service(func(s *dashdoc.Service) { s.Icon = strings.Repeat("a", models.MaxServiceIconLen+1) }), "services[0]"},
		{"NUL character", service(func(s *dashdoc.Service) { s.Description = "a\x00b" }), "document"},
		{"NUL character in a config", widget(dashdoc.Widget{Type: "markdown", Config: json.RawMessage(`{"content":"a\u0000b"}`)}), "document"},
		{"service position", service(func(s *dashdoc.Service) { s.Position = -1 }), "services[0].position"},
		{"icon not base64", service(func(s *dashdoc.Service) { s.IconData = "not base64!" }), "services[0].icon_data"},
		{"icon not an image", service(func(s *dashdoc.Service) { s.IconData = base64.StdEncoding.EncodeToString([]byte("<html>")) }), "services[0].icon_data"},
		{"uploaded icon without data", service(func(s *dashdoc.Service) { s.IconType = models.IconTypeImageUpload }), "services[0].icon_data"},
		{"icon url scheme", service(func(s *dashdoc.Service) { s.IconType, s.IconURL = models.IconTypeImageURL, "javascript:alert(1)" }), "services[0]"},
		{"unknown widget type", widget(dashdoc.Widget{Type: "nope"}), "widgets[0]"},
		{"widget config", widget(dashdoc.Widget{Type: "markdown", Config: json.RawMessage(`[1]`)}), "widgets[0]"},
		{"widget size", widget(dashdoc.Widget{Type: "calendar", CardSize: "1x1", Config: json.RawMessage(`{"ical_urls":["https://192.168.1.20/a.ics"]}`)}), "widgets[0]"},
		{"widget lists unknown integration", widget(dashdoc.Widget{Type: "calendar", Config: json.RawMessage(`{"integrations":["nope"]}`)}), "widgets[0].config"},
		{"widget integration it can't use", &dashdoc.Document{
			Version:      1,
			Integrations: []dashdoc.Integration{{Ref: "app", Kind: "svc-fake", Name: "App", BaseURL: "http://192.168.1.60"}},
			Widgets:      []dashdoc.Widget{{Type: "markdown", IntegrationRef: "app"}},
		}, "widgets[0]"},
		{"uploaded wallpaper path", prefs(dashdoc.Preferences{ThemeBackground: new("/uploads/wallpapers/0123456789abcdef0123456789abcdef.png")}), "preferences.theme_background"},
		{"wallpaper not an image", prefs(dashdoc.Preferences{WallpaperData: base64.StdEncoding.EncodeToString([]byte("nope"))}), "preferences.wallpaper_data"},
		{"accent color", prefs(dashdoc.Preferences{ThemeAccentColor: new("blue")}), "preferences.theme_accent_color"},
		{"accent color with alpha", prefs(dashdoc.Preferences{ThemeAccentColor: new("#11223344")}), "preferences.theme_accent_color"},
		{"theme mode", prefs(dashdoc.Preferences{ThemeMode: new("neon")}), "preferences"},
		{"wallpaper blur", prefs(dashdoc.Preferences{WallpaperBlur: new(99)}), "preferences"},
		{"chip without a target", prefs(dashdoc.Preferences{StatusStrip: &dashdoc.StatusStrip{Chips: []dashdoc.Chip{{Source: "integration", Ref: "nope", KPI: "queue"}}}}), "preferences.status_strip.chips[0].ref"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := env.svc.Apply(context.Background(), transferAdmin, tt.doc, dashdoc.ModeReplace)
			assert.Equal(t, []string{tt.path}, issuePaths(t, err))
			var invalid *dashdoc.ValidationError
			require.ErrorAs(t, err, &invalid)
			assert.NotEmpty(t, invalid.Issues[0].Message)
		})
	}
	assert.Zero(t, env.count(t, "services", transferAdmin))

	_, err := env.svc.Apply(context.Background(), transferAdmin, &dashdoc.Document{Version: 1}, "sideways")
	var vErr *ValidationError
	assert.ErrorAs(t, err, &vErr)
}

// The accent takes what the preferences endpoint takes, as written
func TestImportKeepsTheAccentColorAsGiven(t *testing.T) {
	env := newTransferEnv(t)
	ctx := context.Background()
	for _, color := range []string{"#abc", "#3B82F6"} {
		_, err := env.svc.Apply(ctx, transferUser, &dashdoc.Document{Version: 1, Preferences: &dashdoc.Preferences{ThemeAccentColor: &color}}, dashdoc.ModeReplace)
		require.NoError(t, err)
		prefs, err := env.preferences.GetByUserID(ctx, transferUser)
		require.NoError(t, err)
		require.NotNil(t, prefs.ThemeAccentColor)
		assert.Equal(t, color, *prefs.ThemeAccentColor)
	}

	_, err := env.svc.Apply(ctx, transferUser, &dashdoc.Document{Version: 1, Preferences: &dashdoc.Preferences{ThemeAccentColor: new("")}}, dashdoc.ModeReplace)
	require.NoError(t, err)
	prefs, err := env.preferences.GetByUserID(ctx, transferUser)
	require.NoError(t, err)
	assert.Nil(t, prefs.ThemeAccentColor)
}

func TestImportStaysUnderThePerUserLimits(t *testing.T) {
	env := newTransferEnv(t)
	ctx := context.Background()
	env.addWidget(t, transferUser, models.WidgetRequest{Type: "markdown"})
	doc := &dashdoc.Document{Version: 1}
	for i := range maxWidgetsPerUser {
		doc.Widgets = append(doc.Widgets, dashdoc.Widget{Type: "markdown", Position: i})
	}

	_, err := env.svc.Preview(ctx, transferUser, doc, dashdoc.ModeMerge)
	assert.Equal(t, []string{"widgets"}, issuePaths(t, err))
	_, err = env.svc.Apply(ctx, transferUser, doc, dashdoc.ModeMerge)
	assert.Equal(t, []string{"widgets"}, issuePaths(t, err))
	assert.Equal(t, 1, env.count(t, "widgets", transferUser))

	// A replace makes room first
	summary, err := env.svc.Apply(ctx, transferUser, doc, dashdoc.ModeReplace)
	require.NoError(t, err)
	assert.Equal(t, dashdoc.Count{Created: maxWidgetsPerUser, Removed: 1}, summary.Widgets)
}
