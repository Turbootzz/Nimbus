package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
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
