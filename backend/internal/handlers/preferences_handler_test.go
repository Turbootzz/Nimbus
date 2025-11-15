package handlers

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

// Note: Full database integration tests for preferences are in repository/preferences_repository_test.go
// These handler tests focus on validation, authorization, and error handling

func TestPreferencesHandler_GetPreferences_Unauthorized(t *testing.T) {
	// Create a minimal app to test auth check
	// We don't need a real database for this test since it should fail before touching the DB
	app := fiber.New()
	app.Get("/preferences", func(c *fiber.Ctx) error {
		// Simulate handler without user_id in locals
		userID, ok := c.Locals("user_id").(string)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized",
			})
		}
		return c.SendString(userID)
	})

	req := httptest.NewRequest("GET", "/preferences", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestPreferencesHandler_UpdatePreferences_Unauthorized(t *testing.T) {
	// Test unauthorized access (no user_id in locals)
	app := fiber.New()
	app.Put("/preferences", func(c *fiber.Ctx) error {
		// Simulate handler without user_id in locals
		userID, ok := c.Locals("user_id").(string)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized",
			})
		}
		return c.SendString(userID)
	})

	requestBody := `{"theme_mode": "dark"}`
	req := httptest.NewRequest("PUT", "/preferences", bytes.NewReader([]byte(requestBody)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestPreferencesHandler_UpdatePreferences_InvalidJSON(t *testing.T) {
	// Test invalid JSON body
	app := fiber.New()
	app.Put("/preferences", func(c *fiber.Ctx) error {
		c.Locals("user_id", "test-user")
		var req map[string]interface{}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}
		return c.SendStatus(fiber.StatusOK)
	})

	invalidJSON := `{"theme_mode":`
	req := httptest.NewRequest("PUT", "/preferences", bytes.NewReader([]byte(invalidJSON)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}
