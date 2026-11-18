package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigReadOnly(t *testing.T) {
	app := fiber.New()
	// Stands in for AuthMiddleware
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", c.Get("X-User"))
		return c.Next()
	})
	app.Use(ConfigReadOnly(func(userID string) bool { return userID == "owner" }))
	app.All("/*", func(c *fiber.Ctx) error { return c.SendString("ok") })

	cases := []struct {
		method, path, user string
		status             int
	}{
		{http.MethodGet, "/services", "owner", http.StatusOK},
		{http.MethodPost, "/services", "owner", http.StatusConflict},
		{http.MethodPost, "/services/", "owner", http.StatusConflict},
		{http.MethodPut, "/services/reorder", "owner", http.StatusConflict},
		{http.MethodPut, "/groups/1", "owner", http.StatusConflict},
		{http.MethodDelete, "/widgets/1", "owner", http.StatusConflict},
		{http.MethodPatch, "/integrations/1", "owner", http.StatusConflict},
		{http.MethodPut, "/dashboard/reorder", "owner", http.StatusConflict},
		{http.MethodPost, "/discovery/docker/services", "owner", http.StatusConflict},
		{http.MethodPost, "/import?mode=replace", "owner", http.StatusConflict},
		{http.MethodPost, "/import?mode=merge&dry_run=0", "owner", http.StatusConflict},
		// Looking is fine
		{http.MethodPost, "/import?mode=merge&dry_run=1", "owner", http.StatusOK},
		{http.MethodPost, "/import/convert", "owner", http.StatusOK},
		{http.MethodPost, "/services/1/check", "owner", http.StatusOK},
		{http.MethodPost, "/widgets/1/refresh", "owner", http.StatusOK},
		{http.MethodPost, "/integrations/test", "owner", http.StatusOK},
		{http.MethodPost, "/integrations/1/test", "owner", http.StatusOK},
		// Only the owner's dashboard is locked
		{http.MethodPost, "/services", "someone-else", http.StatusOK},
		{http.MethodDelete, "/widgets/1", "", http.StatusOK},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		req.Header.Set("X-User", tc.user)
		resp, err := app.Test(req)
		require.NoError(t, err)
		body, _ := io.ReadAll(resp.Body)
		assert.Equal(t, tc.status, resp.StatusCode, "%s %s as %q", tc.method, tc.path, tc.user)
		if tc.status == http.StatusConflict {
			assert.Contains(t, string(body), "config_read_only")
			assert.Contains(t, string(body), "NIMBUS_CONFIG_FILE")
		}
	}
}

// A write token of the owner is locked like the owner's session: the lock
// goes by user, whatever the request authenticated with
func TestConfigReadOnly_LocksTokenWrites(t *testing.T) {
	setupTestEnv()
	db := setupMiddlewareTestDB(t)
	defer db.Close()
	owner := createTestUser(t, db, "admin")
	// A second user; createTestUser always uses the same email address
	other := &models.User{ID: "22222222-2222-2222-2222-222222222222"}
	_, err := db.Exec(`INSERT INTO users (id, email, name, role, created_at, updated_at) VALUES (?, 'other@example.com', 'Other', 'user', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, other.ID)
	require.NoError(t, err)

	app := fiber.New()
	auth := AuthMiddleware(services.NewAuthService(), repository.NewUserRepository(db), repository.NewAPITokenRepository(db), models.ScopeReadServices)
	app.Use(auth, ConfigReadOnly(func(userID string) bool { return userID == owner.ID }))
	app.All("/services", func(c *fiber.Ctx) error { return c.SendString("ok") })

	request := func(method string, user *models.User) int {
		req := httptest.NewRequest(method, "/services", nil)
		req.Header.Set("Authorization", "Bearer "+createTestAPIToken(t, db, user.ID, false))
		resp, err := app.Test(req)
		require.NoError(t, err)
		return resp.StatusCode
	}
	assert.Equal(t, http.StatusOK, request(http.MethodGet, owner))
	assert.Equal(t, http.StatusConflict, request(http.MethodPost, owner))
	assert.Equal(t, http.StatusOK, request(http.MethodPost, other))
}
