package handlers

import (
	"github.com/gofiber/fiber/v2"
)

// RequireUserID extracts the user ID from context. If not found, it returns
// a fiber.Error with 401 status that will be handled by Fiber's error handler.
func RequireUserID(c *fiber.Ctx) (string, error) {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return "", fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}
	return userID, nil
}
