package handlers

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/nimbus/backend/internal/testutil"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/dashdoc"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/services"
	"github.com/nimbus/backend/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// transferTestSchema mirrors the tables of a dashboard for SQLite
const transferTestSchema = `
	CREATE TABLE users (id TEXT PRIMARY KEY);
	CREATE TABLE groups (
		id TEXT PRIMARY KEY, user_id TEXT NOT NULL, name TEXT NOT NULL, color TEXT, position INTEGER DEFAULT 0,
		is_default INTEGER DEFAULT 0, monitoring_enabled INTEGER NOT NULL DEFAULT 1,
		tile_style TEXT NOT NULL DEFAULT 'cards' CHECK (tile_style IN ('cards', 'chips')),
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
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
		integration_id TEXT, monitoring_enabled INTEGER NOT NULL DEFAULT 1,
		created_at TIMESTAMP NOT NULL, updated_at TIMESTAMP NOT NULL
	);
	CREATE TABLE widgets (
		id TEXT PRIMARY KEY, user_id TEXT NOT NULL, type TEXT NOT NULL, title TEXT NOT NULL DEFAULT '',
		group_id TEXT, integration_id TEXT, config TEXT NOT NULL DEFAULT '{}', card_size TEXT NOT NULL DEFAULT '2x1',
		position INTEGER NOT NULL DEFAULT 0, refresh_seconds INTEGER NOT NULL DEFAULT 300, enabled INTEGER NOT NULL DEFAULT 1,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE user_preferences (
		id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))), user_id TEXT NOT NULL UNIQUE,
		theme_mode TEXT NOT NULL DEFAULT 'auto', theme_background TEXT, theme_accent_color TEXT,
		open_in_new_tab BOOLEAN NOT NULL DEFAULT 1, enable_card_resizing BOOLEAN NOT NULL DEFAULT 1,
		enable_service_grouping BOOLEAN NOT NULL DEFAULT 1, card_scale TEXT NOT NULL DEFAULT 'medium',
		view_mode TEXT NOT NULL DEFAULT 'grid', wallpaper_blur INTEGER NOT NULL DEFAULT 0,
		wallpaper_dim INTEGER NOT NULL DEFAULT 0, card_opacity INTEGER NOT NULL DEFAULT 100,
		card_blur INTEGER NOT NULL DEFAULT 0, layout_mode TEXT NOT NULL DEFAULT 'canvas',
		group_display TEXT CHECK (group_display IN ('tabs', 'sections')),
		status_strip TEXT NOT NULL DEFAULT '{"enabled": false, "chips": []}',
		theme_preset TEXT NOT NULL DEFAULT 'default', theme_font TEXT NOT NULL DEFAULT 'default',
		kpi_mono BOOLEAN NOT NULL DEFAULT 0, icon_source TEXT NOT NULL DEFAULT 'dashboard-icons-svg',
		custom_css TEXT NOT NULL DEFAULT '', language TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	INSERT INTO users (id) VALUES ('dddddddd-dddd-dddd-dddd-ddddddddddd1'), ('dddddddd-dddd-dddd-dddd-ddddddddddd2');
`

// transferTestUsers resolves roles like the user repository would
type transferTestUsers map[string]string

func (u transferTestUsers) GetByID(id string) (*models.User, error) {
	role, ok := u[id]
	if !ok {
		return nil, repository.ErrUserNotFound
	}
	return &models.User{ID: id, Role: role}, nil
}

// setupTransferTestApp mirrors the export/import routes in main.go with a
// stub auth middleware; the owner is an admin, the other user is not
func setupTransferTestApp(t *testing.T, userID string) (*fiber.App, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	_, err = db.Exec(transferTestSchema)
	require.NoError(t, err)
	testutil.AddBoards(t, db)

	cleanUploadDir(t)
	oldDir := WallpaperUploadDir
	WallpaperUploadDir = t.TempDir()
	t.Cleanup(func() { WallpaperUploadDir = oldDir })

	cipher, err := utils.NewCipher(bytes.Repeat([]byte{7}, 32))
	require.NoError(t, err)
	integrationRepo := repository.NewIntegrationRepository(db)
	preferencesRepo := repository.NewPreferencesRepository(db)
	service := services.NewDashboardTransferService(services.DashboardTransferDeps{
		Dashboard:          repository.NewDashboardRepository(db),
		Groups:             repository.NewGroupRepository(db),
		Services:           repository.NewServiceRepository(db),
		Widgets:            repository.NewWidgetRepository(db),
		Integrations:       integrationRepo,
		Preferences:        preferencesRepo,
		Users:              transferTestUsers{integrationOwnerID: "admin", integrationOtherID: "user"},
		IntegrationService: services.NewIntegrationService(integrationRepo, cipher, ""),
		Images:             NewUploadStore(preferencesRepo),
	})
	handler := NewDashboardTransferHandler(service)

	app := fiber.New(fiber.Config{BodyLimit: MaxImportSize + 1024*1024})
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", userID)
		return c.Next()
	})
	app.Get("/export", handler.Export)
	app.Post("/import", handler.Import)
	app.Post("/import/convert", handler.Convert)
	return app, db
}

func transferRequest(t *testing.T, app *fiber.App, method, path, body string) (int, http.Header, []byte) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, resp.Header, raw
}

func uploadedIcons(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(UploadDir)
	if os.IsNotExist(err) {
		return nil
	}
	require.NoError(t, err)
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}
	return names
}

func transferTestDocument() string {
	image := base64.StdEncoding.EncodeToString(tinyPNG)
	return `{
		"version": 1,
		"groups": [{"ref": "media", "name": "Media", "color": "#ff0000", "default": true, "tile_style": "chips"}],
		"integrations": [{"ref": "sonarr", "kind": "handler-fake", "name": "Sonarr", "base_url": "http://192.168.1.10:8989"}],
		"services": [
			{"name": "Sonarr", "url": "http://192.168.1.10:8989", "position": 0, "group_ref": "media",
			 "integration_ref": "sonarr", "icon_type": "image_upload", "icon_data": "` + image + `"},
			{"name": "Plex", "url": "https://plex.example.com", "position": 2, "icon": "🎬"}
		],
		"widgets": [{"ref": "note", "type": "markdown", "title": "Note", "position": 1, "config": {"content": "hi"}}],
		"preferences": {
			"theme_mode": "dark",
			"group_display": "sections",
			"theme_preset": "nord",
			"language": "fr",
			"custom_css": ".glass-card { border-radius: 0 }",
			"wallpaper_data": "` + image + `",
			"status_strip": {"enabled": true, "chips": [{"source": "integration", "ref": "sonarr", "kpi": "queued"}]}
		}
	}`
}

func TestTransfer_ImportExportRoundTrip(t *testing.T) {
	app, db := setupTransferTestApp(t, integrationOwnerID)

	// A dry run tells what would happen and writes nothing
	status, _, body := transferRequest(t, app, http.MethodPost, "/import?mode=replace&dry_run=1", transferTestDocument())
	require.Equal(t, http.StatusOK, status, string(body))
	var preview dashdoc.Summary
	require.NoError(t, json.Unmarshal(body, &preview))
	assert.True(t, preview.DryRun)
	assert.Equal(t, dashdoc.Count{Created: 2}, preview.Services)
	assert.Empty(t, uploadedIcons(t))
	var services int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM services`).Scan(&services))
	assert.Zero(t, services)

	status, _, body = transferRequest(t, app, http.MethodPost, "/import?mode=replace", transferTestDocument())
	require.Equal(t, http.StatusOK, status, string(body))
	var summary dashdoc.Summary
	require.NoError(t, json.Unmarshal(body, &summary))
	preview.DryRun = false
	assert.Equal(t, preview, summary)
	assert.Equal(t, 1, summary.NeedsCredentials)

	// The images went through the upload path: sniffed type, generated name
	firstIcons, firstWallpapers := uploadedIcons(t), wallpaperFiles(t)
	require.Len(t, firstIcons, 1)
	require.Len(t, firstWallpapers, 1)
	assert.Regexp(t, `^[0-9a-f]{32}\.png$`, firstIcons[0])
	var background string
	require.NoError(t, db.QueryRow(`SELECT theme_background FROM user_preferences`).Scan(&background))
	assert.Equal(t, wallpaperURLPrefix+firstWallpapers[0], background)

	status, header, first := transferRequest(t, app, http.MethodGet, "/export", "")
	require.Equal(t, http.StatusOK, status)
	assert.Regexp(t, `^attachment; filename="nimbus-dashboard-\d{4}-\d{2}-\d{2}\.json"$`, header.Get("Content-Disposition"))
	assert.NotContains(t, string(first), firstIcons[0])
	assert.Contains(t, string(first), `"tile_style":"chips"`)
	assert.Contains(t, string(first), `"group_display":"sections"`)
	assert.Contains(t, string(first), `"theme_preset":"nord"`)
	assert.Contains(t, string(first), `"language":"fr"`)
	assert.Contains(t, string(first), `"custom_css":".glass-card { border-radius: 0 }"`)

	// Importing the export replaces the dashboard by an identical one
	status, _, body = transferRequest(t, app, http.MethodPost, "/import?mode=replace", string(first))
	require.Equal(t, http.StatusOK, status, string(body))
	_, _, second := transferRequest(t, app, http.MethodGet, "/export", "")
	assert.JSONEq(t, string(first), string(second))

	// The replaced images are gone from disk
	icons, wallpapers := uploadedIcons(t), wallpaperFiles(t)
	require.Len(t, icons, 1)
	require.Len(t, wallpapers, 1)
	assert.NotEqual(t, firstIcons, icons)
	assert.NotEqual(t, firstWallpapers, wallpapers)
}

// Whatever an export holds, the browser saves a file an import takes: images
// that would make it too large are left out
func TestTransfer_LargeExportStaysImportable(t *testing.T) {
	require.Equal(t, services.MaxDocumentSize, MaxImportSize)
	app, db := setupTransferTestApp(t, integrationOwnerID)
	addService := func(id, description, icon string) {
		iconType := models.IconTypeEmoji
		if icon != "" {
			iconType = models.IconTypeImageUpload
		}
		_, err := db.Exec(`INSERT INTO services (id, user_id, name, url, icon, icon_type, icon_image_path, description, status, created_at, updated_at)
			VALUES (?, ?, ?, 'http://a.lan', '🔗', ?, ?, ?, 'unknown', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
			id, integrationOwnerID, "Service "+id, iconType, icon, description)
		require.NoError(t, err)
	}
	// 4 MiB of text and 6 MiB of icons: 12 MiB with every icon as base64
	addService("text", strings.Repeat("a", 4*1024*1024), "")
	icon := append(append([]byte{}, tinyPNG...), make([]byte, MaxUploadSize-len(tinyPNG))...)
	for i := range 12 {
		name, _, err := NewUploadStore(nil).Save(services.ImageIcon, icon)
		require.NoError(t, err)
		addService(fmt.Sprint(i), "", name)
	}

	status, _, body := transferRequest(t, app, http.MethodGet, "/export", "")
	require.Equal(t, http.StatusOK, status)
	var doc dashdoc.Document
	require.NoError(t, json.Unmarshal(body, &doc))
	saved, err := json.MarshalIndent(&doc, "", "  ")
	require.NoError(t, err)

	assert.LessOrEqual(t, len(saved), MaxImportSize)
	assert.Equal(t, 4, doc.OmittedImages)
	status, _, body = transferRequest(t, app, http.MethodPost, "/import?mode=replace&dry_run=1", string(saved))
	assert.Equal(t, http.StatusOK, status, string(body))
}

func TestTransfer_ImportRejects(t *testing.T) {
	app, db := setupTransferTestApp(t, integrationOwnerID)
	valid := `{"version":1,"services":[{"name":"A","url":"http://a.lan"}]}`

	tests := []struct {
		name   string
		path   string
		body   string
		status int
		want   string
	}{
		{"no mode", "/import", valid, http.StatusBadRequest, "mode must be merge or replace"},
		{"unknown mode", "/import?mode=overwrite", valid, http.StatusBadRequest, "mode must be merge or replace"},
		{"not json", "/import?mode=merge", `version: 1`, http.StatusBadRequest, "Invalid dashboard document"},
		{"unknown field", "/import?mode=merge", `{"version":1,"servces":[]}`, http.StatusBadRequest, `unknown field \"servces\"`},
		{"a file name for an icon", "/import?mode=merge", `{"version":1,"services":[{"name":"A","url":"http://a.lan","icon_image_path":"../../etc/passwd"}]}`, http.StatusBadRequest, "unknown field"},
		{"trailing data", "/import?mode=merge", valid + valid, http.StatusBadRequest, "unexpected data"},
		{"trailing bracket", "/import?mode=merge", valid + "]", http.StatusBadRequest, "unexpected data"},
		{"too large", "/import?mode=merge", `{"version":1,"services":[{"name":"` + strings.Repeat("a", MaxImportSize) + `"}]}`, http.StatusRequestEntityTooLarge, "maximum size"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, _, body := transferRequest(t, app, http.MethodPost, tt.path, tt.body)
			assert.Equal(t, tt.status, status)
			assert.Contains(t, string(body), tt.want)
		})
	}

	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM services`).Scan(&count))
	assert.Zero(t, count)
}

// A form posted from another site can't set this content type
func TestTransfer_ImportNeedsJSONContentType(t *testing.T) {
	app, db := setupTransferTestApp(t, integrationOwnerID)
	req := httptest.NewRequest(http.MethodPost, "/import?mode=replace", strings.NewReader(`{"version":1,"services":[{"name":"A","url":"http://a.lan"}]}`))
	req.Header.Set("Content-Type", "text/plain")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnsupportedMediaType, resp.StatusCode)

	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM services`).Scan(&count))
	assert.Zero(t, count)
}

func TestTransfer_ValidationErrorsNameEntityAndField(t *testing.T) {
	app, _ := setupTransferTestApp(t, integrationOwnerID)
	doc := `{"version":1,
		"services":[{"name":"A","url":"http://a.lan","group_ref":"nope"},{"name":"B","url":"http://b.lan","position":-1}]}`

	status, _, body := transferRequest(t, app, http.MethodPost, "/import?mode=merge", doc)

	assert.Equal(t, http.StatusBadRequest, status)
	var out struct {
		Error   string          `json:"error"`
		Details []dashdoc.Issue `json:"details"`
	}
	require.NoError(t, json.Unmarshal(body, &out))
	assert.Equal(t, "The dashboard document is not valid", out.Error)
	require.Len(t, out.Details, 2)
	assert.Equal(t, "services[0].group_ref", out.Details[0].Path)
	assert.Contains(t, out.Details[0].Message, `"nope"`)
	assert.Equal(t, "services[1].position", out.Details[1].Path)

	status, _, body = transferRequest(t, app, http.MethodPost, "/import?mode=merge", `{"version":7}`)
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), "unsupported version 7")
}

func TestTransfer_NonAdminCannotImportAdminOnlyEntities(t *testing.T) {
	app, db := setupTransferTestApp(t, integrationOtherID)
	doc := `{"version":1,
		"integrations":[{"ref":"docker","kind":"docker","name":"Docker","base_url":"http://192.168.1.70:2375"}],
		"widgets":[{"type":"system_stats"}]}`

	status, _, body := transferRequest(t, app, http.MethodPost, "/import?mode=replace", doc)

	assert.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, string(body), `"path":"integrations[0]"`)
	assert.Contains(t, string(body), "Only admins")
	assert.Contains(t, string(body), `"path":"widgets[0]"`)
	var count int
	require.NoError(t, db.QueryRow(`SELECT (SELECT COUNT(*) FROM integrations) + (SELECT COUNT(*) FROM widgets)`).Scan(&count))
	assert.Zero(t, count)
}

func TestUploadStoreOnlyTouchesUploadedFiles(t *testing.T) {
	_, _ = setupTransferTestApp(t, integrationOwnerID)
	store := NewUploadStore(nil)

	for _, name := range []string{"", ".", "..", "../secret.png", "dir/a.png", `dir\a.png`, "/etc/passwd"} {
		_, err := store.Load(services.ImageIcon, name)
		assert.Error(t, err, name)
	}
	for _, name := range []string{"a.png", "/uploads/wallpapers/../../a.png", "https://example.com/a.png", "/uploads/avatars/" + strings.Repeat("a", 32) + ".png"} {
		_, err := store.Load(services.ImageWallpaper, name)
		assert.Error(t, err, name)
	}

	assert.ErrorIs(t, store.Check(services.ImageIcon, []byte("<html>")), errImageInvalidType)
	assert.ErrorIs(t, store.Check(services.ImageIcon, append(append([]byte{}, tinyPNG...), make([]byte, MaxUploadSize)...)), errImageTooLarge)
	// Fetched favicons are stored as icons too, so their types are accepted
	assert.NoError(t, store.Check(services.ImageIcon, []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`)))
	assert.ErrorIs(t, store.Check(services.ImageWallpaper, []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`)), errImageInvalidType)

	name, release, err := store.Save(services.ImageIcon, tinyPNG)
	require.NoError(t, err)
	release()
	data, err := store.Load(services.ImageIcon, name)
	require.NoError(t, err)
	assert.Equal(t, tinyPNG, data)
	store.Remove(services.ImageIcon, name)
	_, err = store.Load(services.ImageIcon, name)
	assert.Error(t, err)
}
