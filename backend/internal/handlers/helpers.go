package handlers

import "github.com/gofiber/fiber/v2"

// RequireUserID extracts the user ID from context. If not found, it sends a 401 response
// and returns a fiber error that stops the handler chain.
func RequireUserID(c *fiber.Ctx) (string, error) {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return "", fiber.NewError(fiber.StatusUnauthorized, "Unauthorized: user ID not found")
	}
	return userID, nil
}
