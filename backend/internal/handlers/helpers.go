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

// requireUserID extracts user ID and returns 401 error if not found
// Helper that combines getUserID with error response
func requireUserID(c *fiber.Ctx) (string, error) {
	userID, err := getUserID(c)
	if err != nil {
		return "", c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized: user ID not found",
		})
	}
	return userID, nil
}
