package handlers

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type statusPageFixture struct {
	app     *fiber.App
	db      *sql.DB
	admin   *models.User
	group   *models.Group
	logoDir string
}

// newStatusPageFixture wires the routes like main.go: the public one behind
// the rate limiter, the admin ones as the fixture's admin
func newStatusPageFixture(t *testing.T) *statusPageFixture {
	t.Helper()
	db := setupMetricsTestDB(t)
	t.Cleanup(func() { db.Close() })
	_, err := db.Exec(`
		CREATE TABLE system_settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_by TEXT
		);
		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			password TEXT,
			role TEXT NOT NULL DEFAULT 'user',
			provider TEXT NOT NULL DEFAULT 'local',
			provider_id TEXT,
			avatar_url TEXT,
			email_verified INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			last_activity_at TIMESTAMP
		);
	`)
	require.NoError(t, err)

	f := &statusPageFixture{db: db, logoDir: t.TempDir()}
	f.admin = createAdminTestUser(t, db, "admin@example.com", "Admin", "admin")
	f.group = f.createGroup(t, f.admin.ID, "Public")

	settingsRepo := repository.NewSettingsRepository(db)
	service := services.NewStatusPageService(
		settingsRepo,
		repository.NewGroupRepository(db),
		repository.NewServiceRepository(db),
		repository.NewStatusLogRepository(db),
		repository.NewUserRepository(db),
		f.logoDir,
	)
	handler := NewStatusPageHandler(service)
	settingsHandler := NewSettingsHandler(settingsRepo, services.NewEmailService(settingsRepo))

	// X-Test-Peer stands in for the address of the connecting peer
	f.app = fiber.New(fiber.Config{ProxyHeader: "X-Test-Peer"})
	f.app.Get("/public/status", handler.RateLimit(), handler.GetPublic)
	asAdmin := func(c *fiber.Ctx) error {
		c.Locals("user_id", f.admin.ID)
		c.Locals("role", "admin")
		return c.Next()
	}
	f.app.Get("/admin/status-page", asAdmin, handler.GetSettings)
	f.app.Put("/admin/status-page", asAdmin, handler.UpdateSettings)
	f.app.Put("/admin/settings/:key", asAdmin, settingsHandler.UpdateSetting)
	// Same body as the catch-all in main.go
	f.app.Use(func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Not found"})
	})

	resetStatusPageRateLimit(t)
	return f
}

func resetStatusPageRateLimit(t *testing.T) {
	t.Helper()
	reset := func() {
		forgotPasswordRateLimitMu.Lock()
		defer forgotPasswordRateLimitMu.Unlock()
		for key := range forgotPasswordRateLimit {
			if strings.HasPrefix(key, "status-page:") {
				delete(forgotPasswordRateLimit, key)
			}
		}
	}
	reset()
	t.Cleanup(reset)
}

func (f *statusPageFixture) createGroup(t *testing.T, userID, name string) *models.Group {
	t.Helper()
	group := &models.Group{
		ID: uuid.NewString(), UserID: userID, Name: name, MonitoringEnabled: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	createGroupDirectly(t, f.db, group)
	return group
}

func (f *statusPageFixture) createService(t *testing.T, group *models.Group, name, status string) *models.Service {
	t.Helper()
	service := &models.Service{
		ID: uuid.NewString(), UserID: group.UserID, Name: name, URL: "http://" + name + ".lan:8080",
		Icon: "🎬", Description: "private note about " + name, Status: status,
		GroupID: &group.ID, MonitoringEnabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	createTestService(t, f.db, service)
	return service
}

func (f *statusPageFixture) addLog(t *testing.T, service *models.Service, status string, age time.Duration) {
	t.Helper()
	_, err := f.db.Exec(
		`INSERT INTO service_status_logs (id, service_id, status, checked_at) VALUES (?, ?, ?, ?)`,
		uuid.NewString(), service.ID, status, time.Now().Add(-age),
	)
	require.NoError(t, err)
}

func (f *statusPageFixture) do(t *testing.T, method, path, body string, headers ...string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	resp, err := f.app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, string(data)
}

func (f *statusPageFixture) save(t *testing.T, settings models.StatusPageSettings) (int, string) {
	t.Helper()
	body, err := json.Marshal(settings)
	require.NoError(t, err)
	return f.do(t, http.MethodPut, "/admin/status-page", string(body))
}

func (f *statusPageFixture) enable(t *testing.T, settings models.StatusPageSettings) {
	t.Helper()
	settings.Enabled, settings.GroupID = true, f.group.ID
	status, body := f.save(t, settings)
	require.Equal(t, http.StatusOK, status, body)
}

func TestStatusPage_DisabledLooksLikeAMissingRoute(t *testing.T) {
	f := newStatusPageFixture(t)
	f.createService(t, f.group, "jellyfin", models.StatusOnline)

	missingStatus, missingBody := f.do(t, http.MethodGet, "/public/nothing-here", "")
	status, body := f.do(t, http.MethodGet, "/public/status", "")
	assert.Equal(t, http.StatusNotFound, status)
	assert.Equal(t, missingStatus, status)
	assert.Equal(t, missingBody, body)

	// Headers too: nothing the enabled page sets may show while it is off
	headersOf := func(path string) http.Header {
		resp, err := f.app.Test(httptest.NewRequest(http.MethodGet, path, nil), -1)
		require.NoError(t, err)
		defer resp.Body.Close()
		return resp.Header
	}
	assert.Equal(t, headersOf("/public/nothing-here"), headersOf("/public/status"))

	// Configured but switched off stays hidden
	saveStatus, saveBody := f.save(t, models.StatusPageSettings{GroupID: f.group.ID, Title: "Lab"})
	require.Equal(t, http.StatusOK, saveStatus, saveBody)
	status, body = f.do(t, http.MethodGet, "/public/status", "")
	assert.Equal(t, http.StatusNotFound, status)
	assert.Equal(t, missingBody, body)
}

func TestStatusPage_PublicReturnsOnlyWhitelistedFields(t *testing.T) {
	f := newStatusPageFixture(t)
	up := f.createService(t, f.group, "jellyfin", models.StatusOnline)
	down := f.createService(t, f.group, "nextcloud", models.StatusOffline)
	f.addLog(t, up, models.StatusOnline, 10*time.Minute)
	f.addLog(t, up, models.StatusOnline, 3*24*time.Hour)
	f.addLog(t, down, models.StatusOnline, 30*time.Hour)
	f.addLog(t, down, models.StatusOffline, 0)

	// Not on the page: another group of the admin, and another user's group
	f.createService(t, f.createGroup(t, f.admin.ID, "Private"), "vaultwarden", models.StatusOnline)
	other := createAdminTestUser(t, f.db, "other@example.com", "Other", "user")
	f.createService(t, f.createGroup(t, other.ID, "Theirs"), "pihole", models.StatusOnline)

	f.enable(t, models.StatusPageSettings{Title: "  Home lab  "})

	status, body := f.do(t, http.MethodGet, "/public/status", "")
	require.Equal(t, http.StatusOK, status, body)

	var raw map[string]any
	require.NoError(t, json.Unmarshal([]byte(body), &raw))
	assert.ElementsMatch(t, []string{"title", "overall", "updated_at", "services"}, keysOf(raw))
	assert.Equal(t, "Home lab", raw["title"])
	assert.Equal(t, models.StatusPageDegraded, raw["overall"])

	rows := raw["services"].([]any)
	require.Len(t, rows, 2)
	allowed := []string{"name", "icon", "icon_type", "status", "uptime_24h", "uptime_7d", "bars_24h", "bars_7d"}
	byName := map[string]map[string]any{}
	for _, row := range rows {
		service := row.(map[string]any)
		assert.ElementsMatch(t, allowed, keysOf(service))
		byName[service["name"].(string)] = service
	}

	assert.Equal(t, 100.0, byName["jellyfin"]["uptime_24h"])
	assert.Equal(t, 100.0, byName["jellyfin"]["uptime_7d"])
	assert.Equal(t, 0.0, byName["nextcloud"]["uptime_24h"])
	assert.Equal(t, 50.0, byName["nextcloud"]["uptime_7d"])
	assert.Len(t, byName["nextcloud"]["bars_24h"], 24)
	assert.Len(t, byName["nextcloud"]["bars_7d"], 28)
	assert.Equal(t, 0.0, byName["nextcloud"]["bars_24h"].([]any)[23], "the last bar is the running hour")

	for _, secret := range []string{".lan", "private note", "vaultwarden", "pihole", up.ID, f.group.ID, f.admin.ID, "admin@example.com"} {
		assert.NotContains(t, body, secret)
	}
}

func TestStatusPage_UptimeCountsExactWindows(t *testing.T) {
	f := newStatusPageFixture(t)
	nas := f.createService(t, f.group, "nas", models.StatusOnline)
	// The oldest check of each window falls in the hour before its first bar
	f.addLog(t, nas, models.StatusOnline, 10*time.Minute)
	f.addLog(t, nas, models.StatusOffline, 24*time.Hour-time.Minute)
	f.addLog(t, nas, models.StatusOffline, 24*time.Hour+time.Minute)
	f.addLog(t, nas, models.StatusOnline, 7*24*time.Hour-time.Minute)
	f.addLog(t, nas, models.StatusOffline, 7*24*time.Hour+time.Minute)
	f.enable(t, models.StatusPageSettings{})

	status, body := f.do(t, http.MethodGet, "/public/status", "")
	require.Equal(t, http.StatusOK, status, body)
	var page models.PublicStatusPage
	require.NoError(t, json.Unmarshal([]byte(body), &page))
	require.Len(t, page.Services, 1)
	row := page.Services[0]
	require.NotNil(t, row.Uptime24h)
	require.NotNil(t, row.Uptime7d)
	assert.Equal(t, 50.0, *row.Uptime24h, "last 24 hours: one online, one offline")
	assert.Equal(t, 50.0, *row.Uptime7d, "last 7 days: two online, two offline")
	assert.Len(t, row.Bars24h, 24)
	assert.Len(t, row.Bars7d, 28)
}

func TestStatusPage_ShowLinksExposesURLs(t *testing.T) {
	f := newStatusPageFixture(t)
	service := f.createService(t, f.group, "jellyfin", models.StatusOnline)
	f.enable(t, models.StatusPageSettings{ShowLinks: true})

	status, body := f.do(t, http.MethodGet, "/public/status", "")
	require.Equal(t, http.StatusOK, status, body)
	assert.Contains(t, body, service.URL)
	assert.NotContains(t, body, "private note")
}

func TestStatusPage_SaveValidation(t *testing.T) {
	f := newStatusPageFixture(t)
	other := createAdminTestUser(t, f.db, "other@example.com", "Other", "admin")
	theirs := f.createGroup(t, other.ID, "Theirs")
	f.createService(t, theirs, "pihole", models.StatusOnline)

	logo := strings.Repeat("a", 32) + ".png"
	require.NoError(t, os.WriteFile(filepath.Join(f.logoDir, logo), []byte("png"), 0o644))

	tests := []struct {
		name     string
		settings models.StatusPageSettings
		want     int
	}{
		{"another user's group", models.StatusPageSettings{Enabled: true, GroupID: theirs.ID}, http.StatusBadRequest},
		{"unknown group", models.StatusPageSettings{GroupID: uuid.NewString()}, http.StatusBadRequest},
		{"group that is no ID", models.StatusPageSettings{GroupID: "1 OR 1=1"}, http.StatusBadRequest},
		{"enabled without group", models.StatusPageSettings{Enabled: true}, http.StatusBadRequest},
		{"title too long", models.StatusPageSettings{Title: strings.Repeat("x", models.MaxStatusPageTitleLen+1)}, http.StatusBadRequest},
		{"title with control characters", models.StatusPageSettings{Title: "Lab\u0000\u001b[31m"}, http.StatusBadRequest},
		{"logo path traversal", models.StatusPageSettings{Logo: "../../etc/passwd"}, http.StatusBadRequest},
		{"logo that was never uploaded", models.StatusPageSettings{Logo: strings.Repeat("b", 32) + ".png"}, http.StatusBadRequest},
		{"remote logo", models.StatusPageSettings{Logo: "https://example.com/logo.png"}, http.StatusBadRequest},
		{"off and empty", models.StatusPageSettings{}, http.StatusOK},
		{"own group with logo", models.StatusPageSettings{Enabled: true, GroupID: f.group.ID, Logo: logo, Title: "Lab"}, http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := f.save(t, tt.settings)
			assert.Equal(t, tt.want, status, body)
		})
	}

	// The last save is what the admin reads back and what visitors get
	status, body := f.do(t, http.MethodGet, "/admin/status-page", "")
	require.Equal(t, http.StatusOK, status)
	var stored models.StatusPageSettings
	require.NoError(t, json.Unmarshal([]byte(body), &stored))
	assert.Equal(t, models.StatusPageSettings{Enabled: true, GroupID: f.group.ID, Logo: logo, Title: "Lab"}, stored)

	_, body = f.do(t, http.MethodGet, "/public/status", "")
	assert.Contains(t, body, logo)
	assert.NotContains(t, body, "pihole")
}

func TestStatusPage_GenericSettingsEndpointCannotBypassValidation(t *testing.T) {
	f := newStatusPageFixture(t)
	other := createAdminTestUser(t, f.db, "other@example.com", "Other", "user")
	theirs := f.createGroup(t, other.ID, "Theirs")

	for key, value := range map[string]string{"status_page_group_id": theirs.ID, "status_page_enabled": "true"} {
		status, _ := f.do(t, http.MethodPut, "/admin/settings/"+key, `{"value":"`+value+`"}`)
		assert.Equal(t, http.StatusBadRequest, status, key)
	}
	status, _ := f.do(t, http.MethodGet, "/public/status", "")
	assert.Equal(t, http.StatusNotFound, status)
}

func TestStatusPage_TurnsOffWhenGroupOrAdminGoesAway(t *testing.T) {
	t.Run("group deleted", func(t *testing.T) {
		f := newStatusPageFixture(t)
		f.enable(t, models.StatusPageSettings{})
		_, err := f.db.Exec(`DELETE FROM groups WHERE id = ?`, f.group.ID)
		require.NoError(t, err)
		status, _ := f.do(t, http.MethodGet, "/public/status", "")
		assert.Equal(t, http.StatusNotFound, status)
	})
	t.Run("owner demoted", func(t *testing.T) {
		f := newStatusPageFixture(t)
		f.enable(t, models.StatusPageSettings{})
		_, err := f.db.Exec(`UPDATE users SET role = 'user' WHERE id = ?`, f.admin.ID)
		require.NoError(t, err)
		status, _ := f.do(t, http.MethodGet, "/public/status", "")
		assert.Equal(t, http.StatusNotFound, status)
	})
}

func TestStatusPage_CachedUntilSettingsChange(t *testing.T) {
	f := newStatusPageFixture(t)
	service := f.createService(t, f.group, "jellyfin", models.StatusOnline)
	f.enable(t, models.StatusPageSettings{})

	_, first := f.do(t, http.MethodGet, "/public/status", "")
	require.Contains(t, first, "jellyfin")

	// A change in the database does not reach visitors while the copy is fresh
	_, err := f.db.Exec(`UPDATE services SET name = 'renamed' WHERE id = ?`, service.ID)
	require.NoError(t, err)
	_, cached := f.do(t, http.MethodGet, "/public/status", "")
	assert.Equal(t, first, cached)

	// Saving the settings drops the copy
	f.enable(t, models.StatusPageSettings{Title: "New title"})
	_, fresh := f.do(t, http.MethodGet, "/public/status", "")
	assert.Contains(t, fresh, "renamed")
	assert.Contains(t, fresh, "New title")

	// And turning the page off works at once
	status, body := f.save(t, models.StatusPageSettings{GroupID: f.group.ID})
	require.Equal(t, http.StatusOK, status, body)
	status, _ = f.do(t, http.MethodGet, "/public/status", "")
	assert.Equal(t, http.StatusNotFound, status)
}

func TestStatusPage_RateLimited(t *testing.T) {
	f := newStatusPageFixture(t)

	// While the page is off nothing is counted and nothing answers 429
	for i := 0; i < statusPageRateLimit+5; i++ {
		status, _ := f.do(t, http.MethodGet, "/public/status", "", "X-Test-Peer", "203.0.113.7")
		require.Equal(t, http.StatusNotFound, status)
	}

	f.enable(t, models.StatusPageSettings{})
	for i := 0; i < statusPageRateLimit; i++ {
		status, _ := f.do(t, http.MethodGet, "/public/status", "", "X-Test-Peer", "203.0.113.7")
		require.Equal(t, http.StatusOK, status, "request %d", i)
	}
	status, body := f.do(t, http.MethodGet, "/public/status", "", "X-Test-Peer", "203.0.113.7")
	assert.Equal(t, http.StatusTooManyRequests, status)
	assert.NotContains(t, body, "services")

	// Another visitor is not affected
	status, _ = f.do(t, http.MethodGet, "/public/status", "", "X-Test-Peer", "203.0.113.8")
	assert.Equal(t, http.StatusOK, status)
}

func TestVisitorIP(t *testing.T) {
	app := fiber.New(fiber.Config{ProxyHeader: "X-Test-Peer"})
	app.Get("/", func(c *fiber.Ctx) error { return c.SendString(visitorIP(c)) })

	tests := []struct {
		name, peer, forwardedFor, realIP, want string
	}{
		{"nginx on loopback passes the visitor on", "127.0.0.1", "", "198.51.100.4", "198.51.100.4"},
		{"frontend container on a private network", "172.18.0.3", "", "198.51.100.4", "198.51.100.4"},
		{"visitor behind the user's own reverse proxy", "127.0.0.1", "198.51.100.4, 172.18.0.2", "172.18.0.2", "198.51.100.4"},
		{"hops the visitor made up are skipped", "127.0.0.1", "8.8.8.8, 198.51.100.4, 10.0.0.2", "10.0.0.2", "198.51.100.4"},
		{"visitor on the local network", "127.0.0.1", "192.168.1.20", "192.168.1.20", "192.168.1.20"},
		{"proxy without headers", "127.0.0.1", "", "", "127.0.0.1"},
		{"proxy with garbage headers", "127.0.0.1", "nonsense, ", "not-an-ip", "127.0.0.1"},
		{"public peer cannot pick its own key", "203.0.113.9", "198.51.100.4", "198.51.100.4", "203.0.113.9"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("X-Test-Peer", tt.peer)
			if tt.forwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.forwardedFor)
			}
			if tt.realIP != "" {
				req.Header.Set("X-Real-IP", tt.realIP)
			}
			resp, err := app.Test(req, -1)
			require.NoError(t, err)
			defer resp.Body.Close()
			got, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(got))
		})
	}
}

func TestRateLimitNetwork(t *testing.T) {
	assert.Equal(t, "198.51.100.4", rateLimitNetwork("198.51.100.4"))
	assert.Equal(t, "2001:db8:1:2::", rateLimitNetwork("2001:db8:1:2:aaaa:bbbb:cccc:dddd"))
	assert.Equal(t, rateLimitNetwork("2001:db8:1:2::1"), rateLimitNetwork("2001:db8:1:2:ffff::9"))
	assert.NotEqual(t, rateLimitNetwork("2001:db8:1:2::1"), rateLimitNetwork("2001:db8:1:3::1"))
	assert.Equal(t, "nonsense", rateLimitNetwork("nonsense"))
}

// With every optional setting on, the response still has no key outside the
// documented set; a new field has to be added here on purpose
func TestStatusPage_PublicKeySetWithEverythingOn(t *testing.T) {
	f := newStatusPageFixture(t)
	logo := strings.Repeat("a", 32) + ".png"
	require.NoError(t, os.WriteFile(filepath.Join(f.logoDir, logo), []byte("png"), 0o644))

	service := f.createService(t, f.group, "jellyfin", models.StatusOnline)
	_, err := f.db.Exec(
		`UPDATE services SET url = ?, icon_type = ?, icon_image_path = ? WHERE id = ?`,
		"https://admin:hunter2@jellyfin.example.com/web", models.IconTypeImageUpload, strings.Repeat("c", 32)+".png", service.ID,
	)
	require.NoError(t, err)
	odd := f.createService(t, f.group, "odd", models.StatusOnline)
	_, err = f.db.Exec(`UPDATE services SET url = ? WHERE id = ?`, "javascript://x/%0aalert(1)", odd.ID)
	require.NoError(t, err)

	f.enable(t, models.StatusPageSettings{ShowLinks: true, Logo: logo, Title: "Lab"})
	req := httptest.NewRequest(http.MethodGet, "/public/status", nil)
	resp, err := f.app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "no-store", resp.Header.Get("Cache-Control"))
	assert.Equal(t, "noindex", resp.Header.Get("X-Robots-Tag"))

	var raw map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&raw))
	assert.ElementsMatch(t, []string{"title", "logo", "overall", "updated_at", "services"}, keysOf(raw))

	byName := map[string]map[string]any{}
	for _, row := range raw["services"].([]any) {
		row := row.(map[string]any)
		byName[row["name"].(string)] = row
	}
	assert.ElementsMatch(t,
		[]string{"name", "icon", "icon_type", "icon_image_path", "status", "url", "uptime_24h", "uptime_7d", "bars_24h", "bars_7d"},
		keysOf(byName["jellyfin"]))
	assert.Equal(t, "https://jellyfin.example.com/web", byName["jellyfin"]["url"], "credentials in a URL stay private")
	assert.NotContains(t, keysOf(byName["odd"]), "url", "only http(s) links are shown")
}

func keysOf(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}
