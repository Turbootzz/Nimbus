package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"
)

// RequireUserID extracts the user ID from context. If not found, it sends a 401 response
// and returns an error that stops the handler chain.
func RequireUserID(c *fiber.Ctx) (string, error) {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized: user ID not found"})
		return "", errors.New("unauthorized")
	}
	return userID, nil
}
