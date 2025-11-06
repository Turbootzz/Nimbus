package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"
)

// getUserID extracts and validates the user ID from the request context
// Returns an error if the user ID is not found or invalid
func getUserID(c *fiber.Ctx) (string, error) {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return "", errors.New("user ID not found in context")
	}
	return userID, nil
}
