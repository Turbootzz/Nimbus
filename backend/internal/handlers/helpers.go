package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
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

// RequireUUIDParam reads a route param and validates it parses as a UUID.
// Returns a fiber.Error with 404 if invalid so non-UUID strings stay out of
// SQL — Postgres would otherwise log "invalid input syntax for type uuid"
// and the client would see a 500. Use for any :id-style path param.
//
// Empty -> 400; non-UUID -> 404; valid UUID (any case) -> returned as-is.
func RequireUUIDParam(c *fiber.Ctx, key string) (string, error) {
	v := c.Params(key)
	if v == "" {
		return "", fiber.NewError(fiber.StatusBadRequest, key+" is required")
	}
	if _, err := uuid.Parse(v); err != nil {
		return "", fiber.NewError(fiber.StatusNotFound, "Not found")
	}
	return v, nil
}
