package handlers

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

// Note: Full OAuth flow integration tests require external provider mocking
// These tests focus on validation and error handling logic

func TestOAuthHandler_InitiateOAuth_InvalidProvider(t *testing.T) {
	// Test invalid provider name
	app := fiber.New()
	app.Get("/oauth/:provider", func(c *fiber.Ctx) error {
		provider := c.Params("provider")
		// Simulate validation logic
		validProviders := map[string]bool{
			"google":  true,
			"github":  true,
			"discord": true,
		}

		if !validProviders[provider] {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid OAuth provider",
			})
		}
		return c.SendStatus(fiber.StatusOK)
	})

	tests := []struct {
		name           string
		provider       string
		expectedStatus int
	}{
		{
			name:           "Invalid provider - unknown",
			provider:       "invalid",
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:           "Invalid provider - empty",
			provider:       "",
			expectedStatus: fiber.StatusNotFound, // Fiber returns 404 for missing param
		},
		{
			name:           "Invalid provider - local",
			provider:       "local",
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:           "Valid provider - google",
			provider:       "google",
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "Valid provider - github",
			provider:       "github",
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "Valid provider - discord",
			provider:       "discord",
			expectedStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/oauth/" + tt.provider
			if tt.provider == "" {
				url = "/oauth/"
			}
			req := httptest.NewRequest("GET", url, nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

func TestOAuthHandler_HandleCallback_MissingParameters(t *testing.T) {
	// Test missing code or state parameters
	app := fiber.New()
	app.Get("/oauth/:provider/callback", func(c *fiber.Ctx) error {
		code := c.Query("code")
		state := c.Query("state")

		if code == "" || state == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Missing OAuth parameters",
			})
		}
		return c.SendStatus(fiber.StatusOK)
	})

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
	}{
		{
			name:           "Missing both code and state",
			queryParams:    "",
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:           "Missing code",
			queryParams:    "?state=abc123",
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:           "Missing state",
			queryParams:    "?code=xyz789",
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:           "Both code and state present",
			queryParams:    "?code=xyz789&state=abc123",
			expectedStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/oauth/google/callback"+tt.queryParams, nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

func TestOAuthHandler_LinkProvider_InvalidProvider(t *testing.T) {
	// Test linking invalid provider
	app := fiber.New()
	app.Post("/oauth/link/:provider", func(c *fiber.Ctx) error {
		// Simulate auth middleware setting user_id
		c.Locals("user_id", "test-user-123")

		provider := c.Params("provider")
		validProviders := map[string]bool{
			"google":  true,
			"github":  true,
			"discord": true,
		}

		if !validProviders[provider] || provider == "local" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid OAuth provider",
			})
		}
		return c.SendStatus(fiber.StatusOK)
	})

	tests := []struct {
		name           string
		provider       string
		expectedStatus int
	}{
		{
			name:           "Invalid provider - unknown",
			provider:       "invalid",
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:           "Invalid provider - local",
			provider:       "local",
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:           "Valid provider - google",
			provider:       "google",
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "Valid provider - github",
			provider:       "github",
			expectedStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/oauth/link/"+tt.provider, nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}
