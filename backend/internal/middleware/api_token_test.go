package middleware

import (
	"database/sql"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/services"
	"github.com/nimbus/backend/internal/utils"
	"github.com/stretchr/testify/assert"
)

// insertScopedToken stores a PAT with the given scopes and expiry and
// returns the plaintext token and its ID
func insertScopedToken(t *testing.T, db *sql.DB, userID string, scopes []string, expiresAt *time.Time) (token, id string) {
	t.Helper()

	token, hash, prefix, err := utils.GenerateAPIToken()
	if err != nil {
		t.Fatalf("Failed to generate api token: %v", err)
	}
	id = uuid.New().String()
	query := `
		INSERT INTO api_tokens (id, user_id, name, token_hash, token_prefix, scopes, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	if _, err := db.Exec(query, id, userID, "Scoped", hash, prefix, strings.Join(scopes, " "), expiresAt); err != nil {
		t.Fatalf("Failed to insert api token: %v", err)
	}
	return token, id
}

// newScopedTestApp mirrors how main.go guards its route groups: one group
// per read scope and one without a scope
func newScopedTestApp(db *sql.DB) *fiber.App {
	userRepo := repository.NewUserRepository(db)
	apiTokenRepo := repository.NewAPITokenRepository(db)
	authService := services.NewAuthService()
	ok := func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) }

	app := fiber.New()
	app.All("/services", AuthMiddleware(authService, userRepo, apiTokenRepo, models.ScopeReadServices), ok)
	app.All("/metrics", AuthMiddleware(authService, userRepo, apiTokenRepo, models.ScopeReadMetrics), ok)
	app.All("/widgets", AuthMiddleware(authService, userRepo, apiTokenRepo, models.ScopeReadWidgets), ok)
	app.All("/session-only", AuthMiddleware(authService, userRepo, apiTokenRepo), ok)
	return app
}

func statusFor(t *testing.T, app *fiber.App, method, path, token string) int {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	return resp.StatusCode
}

func TestAuthMiddleware_APIToken_Scopes(t *testing.T) {
	setupTestEnv()
	db := setupMiddlewareTestDB(t)
	defer db.Close()
	user := createTestUser(t, db, "user")
	app := newScopedTestApp(db)

	const ok, forbidden = fiber.StatusOK, fiber.StatusForbidden
	tests := []struct {
		name   string
		scopes []string
		method string
		path   string
		want   int
	}{
		{"read:services reads services", []string{models.ScopeReadServices}, "GET", "/services", ok},
		{"read:services cannot write", []string{models.ScopeReadServices}, "POST", "/services", forbidden},
		{"read:services cannot delete", []string{models.ScopeReadServices}, "DELETE", "/services", forbidden},
		{"read:services cannot read metrics", []string{models.ScopeReadServices}, "GET", "/metrics", forbidden},
		{"read:services cannot read widgets", []string{models.ScopeReadServices}, "GET", "/widgets", forbidden},
		{"read:metrics reads metrics", []string{models.ScopeReadMetrics}, "GET", "/metrics", ok},
		{"read:metrics cannot read services", []string{models.ScopeReadMetrics}, "GET", "/services", forbidden},
		{"read:widgets reads widgets", []string{models.ScopeReadWidgets}, "GET", "/widgets", ok},
		{"read:widgets cannot write widgets", []string{models.ScopeReadWidgets}, "PUT", "/widgets", forbidden},
		{"write:services writes services", []string{models.ScopeWriteServices}, "POST", "/services", ok},
		{"write:services implies read:services", []string{models.ScopeWriteServices}, "GET", "/services", ok},
		{"write:services alone cannot write widgets", []string{models.ScopeWriteServices}, "PUT", "/widgets", forbidden},
		{"write plus read:widgets still cannot write widgets", []string{models.ScopeReadWidgets, models.ScopeWriteServices}, "PUT", "/widgets", forbidden},
		{"write plus read:widgets reads widgets", []string{models.ScopeReadWidgets, models.ScopeWriteServices}, "GET", "/widgets", ok},
		{"mcp alone reads nothing", []string{models.ScopeMCP}, "GET", "/services", forbidden},
		{"mcp alone reads no metrics", []string{models.ScopeMCP}, "GET", "/metrics", forbidden},
		{"mcp alone reads no widgets", []string{models.ScopeMCP}, "GET", "/widgets", forbidden},
		{"mcp alone writes nothing", []string{models.ScopeMCP}, "POST", "/services", forbidden},
		{"no scopes at all", nil, "GET", "/services", forbidden},
		{"every scope is still shut out of session-only routes", models.APITokenScopes, "GET", "/session-only", forbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, _ := insertScopedToken(t, db, user.ID, tt.scopes, nil)
			assert.Equal(t, tt.want, statusFor(t, app, tt.method, tt.path, token))
		})
	}
}

// Tokens from before scopes existed keep the access they had
func TestAuthMiddleware_APIToken_LegacyScopes(t *testing.T) {
	setupTestEnv()
	db := setupMiddlewareTestDB(t)
	defer db.Close()
	user := createTestUser(t, db, "user")
	app := newScopedTestApp(db)

	readOnly, _ := insertScopedToken(t, db, user.ID, models.LegacyScopes(true), nil)
	fullAccess, _ := insertScopedToken(t, db, user.ID, models.LegacyScopes(false), nil)

	for _, path := range []string{"/services", "/metrics", "/widgets"} {
		assert.Equal(t, fiber.StatusOK, statusFor(t, app, "GET", path, readOnly), "read-only GET %s", path)
		assert.Equal(t, fiber.StatusForbidden, statusFor(t, app, "POST", path, readOnly), "read-only POST %s", path)
		assert.Equal(t, fiber.StatusOK, statusFor(t, app, "GET", path, fullAccess), "full GET %s", path)
	}
	// Full access now means services and groups; other areas are read-only
	assert.Equal(t, fiber.StatusOK, statusFor(t, app, "POST", "/services", fullAccess))
	assert.Equal(t, fiber.StatusForbidden, statusFor(t, app, "POST", "/widgets", fullAccess))
}

// An admin's token must not act as an admin: admin surfaces are session-only
func TestAuthMiddleware_APIToken_NeverAdmin(t *testing.T) {
	setupTestEnv()
	db := setupMiddlewareTestDB(t)
	defer db.Close()
	admin := createTestUser(t, db, "admin")

	app := fiber.New()
	app.Get("/services", AuthMiddleware(services.NewAuthService(), repository.NewUserRepository(db), repository.NewAPITokenRepository(db), models.ScopeReadServices), AdminOnly(), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	token, _ := insertScopedToken(t, db, admin.ID, models.APITokenScopes, nil)
	assert.Equal(t, fiber.StatusForbidden, statusFor(t, app, "GET", "/services", token))
}

func TestAuthMiddleware_APIToken_Expiry(t *testing.T) {
	setupTestEnv()
	db := setupMiddlewareTestDB(t)
	defer db.Close()
	user := createTestUser(t, db, "user")
	app := newScopedTestApp(db)

	past := time.Now().Add(-time.Minute).UTC()
	future := time.Now().Add(time.Hour).UTC()
	expired, _ := insertScopedToken(t, db, user.ID, models.ReadScopes(), &past)
	valid, _ := insertScopedToken(t, db, user.ID, models.ReadScopes(), &future)

	assert.Equal(t, fiber.StatusUnauthorized, statusFor(t, app, "GET", "/services", expired))
	assert.Equal(t, fiber.StatusOK, statusFor(t, app, "GET", "/services", valid))
}

func TestAuthMiddleware_APIToken_RateLimit(t *testing.T) {
	setupTestEnv()
	db := setupMiddlewareTestDB(t)
	defer db.Close()
	user := createTestUser(t, db, "user")
	app := newScopedTestApp(db)

	SetAPITokenRateLimit(2)
	t.Cleanup(func() { SetAPITokenRateLimit(DefaultAPITokenRateLimit) })

	limited, limitedID := insertScopedToken(t, db, user.ID, models.ReadScopes(), nil)
	other, _ := insertScopedToken(t, db, user.ID, models.ReadScopes(), nil)

	assert.Equal(t, fiber.StatusOK, statusFor(t, app, "GET", "/services", limited))
	// The budget is per token, not per route group
	assert.Equal(t, fiber.StatusOK, statusFor(t, app, "GET", "/metrics", limited))

	req := httptest.NewRequest("GET", "/services", nil)
	req.Header.Set("Authorization", "Bearer "+limited)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusTooManyRequests, resp.StatusCode)
	assert.Equal(t, "60", resp.Header.Get("Retry-After"))

	// Another token of the same user has its own budget
	assert.Equal(t, fiber.StatusOK, statusFor(t, app, "GET", "/services", other))

	// A finished window is dropped by the cleanup and the token may go again
	apiTokenRateMu.Lock()
	apiTokenRates[limitedID].start = time.Now().Add(-2 * apiTokenRateWindow)
	apiTokenRateMu.Unlock()
	assert.GreaterOrEqual(t, CleanupAPITokenRateLimit(), 1)
	assert.Equal(t, fiber.StatusOK, statusFor(t, app, "GET", "/services", limited))
}

func TestOptionalAuthMiddleware_APIToken_Scope(t *testing.T) {
	setupTestEnv()
	db := setupMiddlewareTestDB(t)
	defer db.Close()
	user := createTestUser(t, db, "user")

	app := fiber.New()
	app.Get("/prometheus", OptionalAuthMiddleware(services.NewAuthService(), repository.NewUserRepository(db), repository.NewAPITokenRepository(db), models.ScopeReadMetrics), func(c *fiber.Ctx) error {
		if c.Locals("user_id") == nil {
			return c.SendStatus(fiber.StatusUnauthorized)
		}
		return c.SendStatus(fiber.StatusOK)
	})

	withScope, _ := insertScopedToken(t, db, user.ID, []string{models.ScopeReadMetrics}, nil)
	withoutScope, _ := insertScopedToken(t, db, user.ID, []string{models.ScopeReadServices}, nil)

	assert.Equal(t, fiber.StatusOK, statusFor(t, app, "GET", "/prometheus", withScope))
	// The wrong scope falls through unauthenticated, like an unknown token
	assert.Equal(t, fiber.StatusUnauthorized, statusFor(t, app, "GET", "/prometheus", withoutScope))
	// A non-token bearer (e.g. PROMETHEUS_API_KEY) still reaches the handler
	assert.Equal(t, fiber.StatusUnauthorized, statusFor(t, app, "GET", "/prometheus", "some-static-key"))

	SetAPITokenRateLimit(1)
	t.Cleanup(func() { SetAPITokenRateLimit(DefaultAPITokenRateLimit) })
	limited, _ := insertScopedToken(t, db, user.ID, []string{models.ScopeReadMetrics}, nil)
	assert.Equal(t, fiber.StatusOK, statusFor(t, app, "GET", "/prometheus", limited))
	assert.Equal(t, fiber.StatusTooManyRequests, statusFor(t, app, "GET", "/prometheus", limited))
}

// When guarded groups are nested, the middleware runs twice for one request
func TestAuthMiddleware_APIToken_NestedGroupsCountOnce(t *testing.T) {
	setupTestEnv()
	db := setupMiddlewareTestDB(t)
	defer db.Close()
	user := createTestUser(t, db, "user")

	auth := AuthMiddleware(services.NewAuthService(), repository.NewUserRepository(db), repository.NewAPITokenRepository(db), models.ScopeReadWidgets)
	app := fiber.New()
	app.Group("/users/me", auth)
	app.Group("/users/me/preferences", auth).Get("/", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

	SetAPITokenRateLimit(2)
	t.Cleanup(func() { SetAPITokenRateLimit(DefaultAPITokenRateLimit) })
	token, _ := insertScopedToken(t, db, user.ID, []string{models.ScopeReadWidgets}, nil)

	assert.Equal(t, fiber.StatusOK, statusFor(t, app, "GET", "/users/me/preferences", token))
	assert.Equal(t, fiber.StatusOK, statusFor(t, app, "GET", "/users/me/preferences", token))
	assert.Equal(t, fiber.StatusTooManyRequests, statusFor(t, app, "GET", "/users/me/preferences", token))
}
