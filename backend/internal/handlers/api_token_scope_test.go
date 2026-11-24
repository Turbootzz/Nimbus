package handlers

import (
	"database/sql"
	"github.com/nimbus/backend/internal/testutil"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/nimbus/backend/internal/middleware"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/services"
	"github.com/nimbus/backend/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const scopeTestUserID = "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeee1"

// setupScopeTestApp wires the real service handler behind the real auth
// middleware, the way main.go guards the services group
func setupScopeTestApp(t *testing.T) (*fiber.App, *sql.DB) {
	t.Helper()
	t.Setenv("JWT_SECRET", "test-secret-32-characters-long!!")

	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	_, err = db.Exec(`
		CREATE TABLE users (
			id TEXT PRIMARY KEY, email TEXT UNIQUE NOT NULL, name TEXT NOT NULL, password TEXT,
			role TEXT NOT NULL DEFAULT 'user', provider TEXT NOT NULL DEFAULT 'local', provider_id TEXT,
			avatar_url TEXT, email_verified INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL, updated_at TIMESTAMP NOT NULL, last_activity_at TIMESTAMP
		);
		CREATE TABLE api_tokens (
			id TEXT PRIMARY KEY, user_id TEXT NOT NULL, name TEXT NOT NULL,
			token_hash TEXT NOT NULL UNIQUE, token_prefix TEXT NOT NULL,
			read_only INTEGER NOT NULL DEFAULT 1, scopes TEXT NOT NULL DEFAULT '',
			expires_at TIMESTAMP, last_used_at TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE groups (id TEXT PRIMARY KEY, user_id TEXT NOT NULL, name TEXT NOT NULL,
			color TEXT, position INTEGER DEFAULT 0, is_default INTEGER DEFAULT 0,
			monitoring_enabled INTEGER DEFAULT 1, created_at TIMESTAMP, updated_at TIMESTAMP);
		CREATE TABLE widgets (id TEXT PRIMARY KEY, user_id TEXT NOT NULL, position INTEGER NOT NULL DEFAULT 0);
		CREATE TABLE services (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			user_id TEXT NOT NULL, name TEXT NOT NULL, url TEXT NOT NULL,
			icon TEXT DEFAULT '', icon_type TEXT DEFAULT 'emoji', icon_image_path TEXT DEFAULT '',
			description TEXT DEFAULT '', status TEXT NOT NULL, response_time INTEGER,
			position INTEGER DEFAULT 0, card_size TEXT DEFAULT '2x1', group_id TEXT, integration_id TEXT,
			monitoring_enabled INTEGER NOT NULL DEFAULT 1,
			created_at TIMESTAMP NOT NULL, updated_at TIMESTAMP NOT NULL
		);
		INSERT INTO users (id, email, name, created_at, updated_at)
			VALUES ('` + scopeTestUserID + `', 'scope@example.com', 'Scope', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
		INSERT INTO services (id, user_id, name, url, status, created_at, updated_at)
			VALUES ('eeeeeeee-eeee-eeee-eeee-eeeeeeeeee01', '` + scopeTestUserID + `', 'Jellyfin', 'http://jellyfin.lan', 'online', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
	`)
	require.NoError(t, err)
	testutil.AddBoards(t, db)

	handler := NewServiceHandler(repository.NewServiceRepository(db), repository.NewGroupRepository(db), nil, repository.NewIntegrationRepository(db))
	auth := middleware.AuthMiddleware(services.NewAuthService(), repository.NewUserRepository(db), repository.NewAPITokenRepository(db), models.ScopeReadServices)

	app := fiber.New()
	group := app.Group("/services", auth)
	group.Get("/", handler.GetServices)
	group.Post("/", handler.CreateService)
	group.Delete("/:id", handler.DeleteService)
	return app, db
}

func insertScopeTestToken(t *testing.T, db *sql.DB, scopes []string, expiresAt *time.Time) string {
	t.Helper()
	token, hash, prefix, err := utils.GenerateAPIToken()
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO api_tokens (id, user_id, name, token_hash, token_prefix, scopes, expires_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		uuid.New().String(), scopeTestUserID, "Scoped", hash, prefix, strings.Join(scopes, " "), expiresAt)
	require.NoError(t, err)
	return token
}

func scopeRequest(t *testing.T, app *fiber.App, method, path, token, body string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	return resp
}

const newServiceBody = `{"name":"Sonarr","url":"http://sonarr.lan"}`

func TestServiceRoutes_ReadScopeCanReadButNotWrite(t *testing.T) {
	app, db := setupScopeTestApp(t)
	token := insertScopeTestToken(t, db, []string{models.ScopeReadServices}, nil)

	resp := scopeRequest(t, app, http.MethodGet, "/services", token, "")
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "Jellyfin")

	assert.Equal(t, fiber.StatusForbidden, scopeRequest(t, app, http.MethodPost, "/services", token, newServiceBody).StatusCode)
	assert.Equal(t, fiber.StatusForbidden, scopeRequest(t, app, http.MethodDelete, "/services/eeeeeeee-eeee-eeee-eeee-eeeeeeeeee01", token, "").StatusCode)

	// Nothing was written
	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM services`).Scan(&count))
	assert.Equal(t, 1, count)
}

func TestServiceRoutes_WriteScopeCanWrite(t *testing.T) {
	app, db := setupScopeTestApp(t)
	token := insertScopeTestToken(t, db, []string{models.ScopeWriteServices}, nil)

	assert.Equal(t, fiber.StatusCreated, scopeRequest(t, app, http.MethodPost, "/services", token, newServiceBody).StatusCode)
	assert.Equal(t, fiber.StatusOK, scopeRequest(t, app, http.MethodGet, "/services", token, "").StatusCode)
}

func TestServiceRoutes_OtherScopesAreForbidden(t *testing.T) {
	app, db := setupScopeTestApp(t)
	token := insertScopeTestToken(t, db, []string{models.ScopeReadMetrics, models.ScopeReadWidgets, models.ScopeMCP}, nil)

	assert.Equal(t, fiber.StatusForbidden, scopeRequest(t, app, http.MethodGet, "/services", token, "").StatusCode)
	assert.Equal(t, fiber.StatusForbidden, scopeRequest(t, app, http.MethodPost, "/services", token, newServiceBody).StatusCode)
}

func TestServiceRoutes_ExpiredTokenIsRejected(t *testing.T) {
	app, db := setupScopeTestApp(t)
	past := time.Now().Add(-time.Second).UTC()
	future := time.Now().Add(time.Hour).UTC()
	expired := insertScopeTestToken(t, db, models.LegacyScopes(false), &past)
	valid := insertScopeTestToken(t, db, models.LegacyScopes(false), &future)

	assert.Equal(t, fiber.StatusUnauthorized, scopeRequest(t, app, http.MethodGet, "/services", expired, "").StatusCode)
	assert.Equal(t, fiber.StatusUnauthorized, scopeRequest(t, app, http.MethodPost, "/services", expired, newServiceBody).StatusCode)
	assert.Equal(t, fiber.StatusOK, scopeRequest(t, app, http.MethodGet, "/services", valid, "").StatusCode)
}
