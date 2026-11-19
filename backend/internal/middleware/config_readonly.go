package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// ConfigReadOnlyMessage is what an edit of a locked dashboard is answered with
const ConfigReadOnlyMessage = "This dashboard is managed by a config file (NIMBUS_CONFIG_FILE) and is read-only. Change the file instead."

// ConfigReadOnly refuses requests that would edit a dashboard the config
// file locks (NIMBUS_CONFIG_READONLY). locked tells whether the user's
// dashboard is. Use it after AuthMiddleware, on routes that write what the
// file manages. Reads pass, and so do the POSTs that only look: health
// checks, refreshes, connection tests, conversions and import previews.
func ConfigReadOnly(locked func(userID string) bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		switch c.Method() {
		case fiber.MethodGet, fiber.MethodHead, fiber.MethodOptions:
			return c.Next()
		}
		path := strings.TrimRight(c.Path(), "/")
		for _, action := range []string{"/check", "/refresh", "/test", "/convert"} {
			if strings.HasSuffix(path, action) {
				return c.Next()
			}
		}
		if strings.HasSuffix(path, "/import") && c.QueryBool("dry_run") {
			return c.Next()
		}
		if userID, _ := c.Locals("user_id").(string); userID == "" || !locked(userID) {
			return c.Next()
		}
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": ConfigReadOnlyMessage,
			"code":  "config_read_only",
		})
	}
}
