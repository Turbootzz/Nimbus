package handlers

import (
	"errors"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/nimbus/backend/internal/services"
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

// IsAdmin reports whether the request comes from an admin
func IsAdmin(c *fiber.Ctx) bool {
	role, _ := c.Locals("role").(string)
	return role == "admin"
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

// serviceError maps errors from the services layer to responses without
// leaking internals. notFound maps repository sentinel errors to 404 messages.
func serviceError(c *fiber.Ctx, err error, action string, notFound map[error]string) error {
	var vErr *services.ValidationError
	if errors.As(err, &vErr) {
		return BadRequest(c, vErr.Message)
	}
	for sentinel, message := range notFound {
		if errors.Is(err, sentinel) {
			return NotFound(c, message)
		}
	}
	log.Printf("Failed to %s: %v", action, err)
	return InternalError(c, "Failed to "+action)
}
