package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	_ "github.com/mattn/go-sqlite3"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tinyPNG is a valid 1x1 PNG
var tinyPNG = []byte{
	0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
	0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
	0x42, 0x60, 0x82,
}

// setupPreferencesApp serves the preferences routes for user-1 on an
// in-memory database, with uploads in a temp dir
func setupPreferencesApp(t *testing.T) (*fiber.App, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	_, err = db.Exec(`CREATE TABLE user_preferences (
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
		layout_mode TEXT NOT NULL DEFAULT 'classic',
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	)`)
	require.NoError(t, err)

	oldDir := WallpaperUploadDir
	WallpaperUploadDir = t.TempDir()
	t.Cleanup(func() { WallpaperUploadDir = oldDir })

	handler := NewPreferencesHandler(repository.NewPreferencesRepository(db))
	app := fiber.New(fiber.Config{BodyLimit: MaxRequestBodySize})
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", "user-1")
		return c.Next()
	})
	app.Get("/preferences", handler.GetPreferences)
	app.Put("/preferences", handler.UpdatePreferences)
	app.Put("/wallpaper", handler.UploadWallpaper)
	return app, db
}

func putJSON(t *testing.T, app *fiber.App, path, body string) (int, models.PreferencesResponse, string) {
	t.Helper()
	req := httptest.NewRequest("PUT", path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return send(t, app, req)
}

func uploadWallpaper(t *testing.T, app *fiber.App, data []byte, contentType string) (int, models.PreferencesResponse, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	header := make(map[string][]string)
	header["Content-Disposition"] = []string{`form-data; name="wallpaper"; filename="bg.png"`}
	header["Content-Type"] = []string{contentType}
	part, err := writer.CreatePart(header)
	require.NoError(t, err)
	_, err = part.Write(data)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	req := httptest.NewRequest("PUT", "/wallpaper", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return send(t, app, req)
}

func send(t *testing.T, app *fiber.App, req *http.Request) (int, models.PreferencesResponse, string) {
	t.Helper()
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var prefs models.PreferencesResponse
	_ = json.Unmarshal(raw, &prefs)
	return resp.StatusCode, prefs, string(raw)
}

func wallpaperFiles(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(WallpaperUploadDir)
	require.NoError(t, err)
	names := []string{}
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func TestPreferences_DefaultsMatchTheDatabase(t *testing.T) {
	app, _ := setupPreferencesApp(t)
	resp, err := app.Test(httptest.NewRequest("GET", "/preferences", nil))
	require.NoError(t, err)
	var prefs models.PreferencesResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&prefs))
	assert.Equal(t, "auto", prefs.ThemeMode)
	assert.Equal(t, "medium", prefs.CardScale)
	assert.Equal(t, 100, prefs.CardOpacity)
	assert.Equal(t, "classic", prefs.LayoutMode)
}

func TestPreferences_LayoutMode(t *testing.T) {
	app, _ := setupPreferencesApp(t)
	status, prefs, body := putJSON(t, app, "/preferences", `{"layout_mode":"canvas"}`)
	require.Equal(t, fiber.StatusOK, status, body)
	assert.Equal(t, "canvas", prefs.LayoutMode)

	status, _, body = putJSON(t, app, "/preferences", `{"layout_mode":"grid"}`)
	assert.Equal(t, fiber.StatusBadRequest, status)
	assert.Contains(t, body, "classic canvas")
}

func TestPreferences_GlassRanges(t *testing.T) {
	app, _ := setupPreferencesApp(t)

	status, prefs, body := putJSON(t, app, "/preferences", `{"wallpaper_blur":20,"wallpaper_dim":80,"card_opacity":0,"card_blur":40}`)
	require.Equal(t, fiber.StatusOK, status, body)
	assert.Equal(t, []int{20, 80, 0, 40}, []int{prefs.WallpaperBlur, prefs.WallpaperDim, prefs.CardOpacity, prefs.CardBlur})

	for _, bad := range []string{`{"wallpaper_blur":21}`, `{"wallpaper_dim":81}`, `{"card_opacity":-1}`, `{"card_blur":41}`} {
		status, _, body := putJSON(t, app, "/preferences", bad)
		assert.Equal(t, fiber.StatusBadRequest, status, bad)
		assert.Contains(t, body, "out of range", bad)
	}
}

func TestPreferences_WallpaperUpload(t *testing.T) {
	app, _ := setupPreferencesApp(t)

	status, prefs, body := uploadWallpaper(t, app, tinyPNG, "image/png")
	require.Equal(t, fiber.StatusOK, status, body)
	require.NotNil(t, prefs.ThemeBackground)
	first := *prefs.ThemeBackground
	assert.Regexp(t, `^/uploads/wallpapers/[0-9a-f]{32}\.png$`, first)
	assert.Equal(t, []string{filepath.Base(first)}, wallpaperFiles(t))

	// A new upload replaces the file
	status, prefs, _ = uploadWallpaper(t, app, tinyPNG, "image/png")
	require.Equal(t, fiber.StatusOK, status)
	assert.NotEqual(t, first, *prefs.ThemeBackground)
	assert.Equal(t, []string{filepath.Base(*prefs.ThemeBackground)}, wallpaperFiles(t))

	// The user's own path is accepted when sent back; any other is not, so
	// nobody can point at (and later get deleted) someone else's upload
	status, _, body = putJSON(t, app, "/preferences", `{"theme_background":"`+*prefs.ThemeBackground+`"}`)
	assert.Equal(t, fiber.StatusOK, status, body)
	assert.Len(t, wallpaperFiles(t), 1, "sending the same path keeps the file")
	status, _, body = putJSON(t, app, "/preferences", `{"theme_background":"/uploads/wallpapers/`+strings.Repeat("a", 32)+`.png"}`)
	assert.Equal(t, fiber.StatusBadRequest, status)
	assert.Contains(t, body, "Upload the image instead")
	status, _, _ = putJSON(t, app, "/preferences", `{"theme_background":"/uploads/avatars/x.png"}`)
	assert.Equal(t, fiber.StatusBadRequest, status)

	// Switching to a URL or clearing deletes the uploaded file
	status, _, _ = putJSON(t, app, "/preferences", `{"theme_background":"https://example.com/bg.jpg"}`)
	require.Equal(t, fiber.StatusOK, status)
	assert.Empty(t, wallpaperFiles(t))
}

func TestPreferences_WallpaperUploadRejects(t *testing.T) {
	app, _ := setupPreferencesApp(t)

	status, _, body := uploadWallpaper(t, app, []byte("<svg></svg>"), "image/svg+xml")
	assert.Equal(t, fiber.StatusBadRequest, status)
	assert.Contains(t, body, "Invalid file type")

	// A text file that claims to be a PNG is caught by the content sniffing
	status, _, _ = uploadWallpaper(t, app, []byte("not an image at all"), "image/png")
	assert.Equal(t, fiber.StatusBadRequest, status)

	big := append(append([]byte{}, tinyPNG...), make([]byte, MaxWallpaperSize)...)
	status, _, body = uploadWallpaper(t, app, big, "image/png")
	assert.Equal(t, fiber.StatusBadRequest, status)
	assert.Contains(t, body, "8 MB")
	assert.Empty(t, wallpaperFiles(t))
}

func TestPreferences_SweepKeepsOtherUsersWallpapers(t *testing.T) {
	app, db := setupPreferencesApp(t)

	// Another user's upload, and a file nothing points at
	theirs := "/uploads/wallpapers/" + strings.Repeat("b", 32) + ".jpg"
	_, err := db.Exec(`INSERT INTO user_preferences (user_id, theme_background, created_at, updated_at) VALUES ('user-2', ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, theirs)
	require.NoError(t, err)
	for _, name := range []string{filepath.Base(theirs), strings.Repeat("c", 32) + ".png"} {
		require.NoError(t, os.WriteFile(filepath.Join(WallpaperUploadDir, name), tinyPNG, 0o644))
	}

	status, _, _ := putJSON(t, app, "/preferences", `{"theme_background":null}`)
	require.Equal(t, fiber.StatusOK, status)
	assert.Equal(t, []string{filepath.Base(theirs)}, wallpaperFiles(t))
}

func TestPreferences_SweepWaitsForUploadsInProgress(t *testing.T) {
	_, db := setupPreferencesApp(t)
	require.NoError(t, os.WriteFile(filepath.Join(WallpaperUploadDir, strings.Repeat("d", 32)+".png"), tinyPNG, 0o644))
	repo := repository.NewPreferencesRepository(db)

	// A file written by an upload that hasn't stored it yet
	wallpaperSweep.RLock()
	PruneWallpapers(context.Background(), repo)
	assert.Len(t, wallpaperFiles(t), 1, "kept while an upload is running")

	wallpaperSweep.RUnlock()
	PruneWallpapers(context.Background(), repo)
	assert.Empty(t, wallpaperFiles(t))
}
