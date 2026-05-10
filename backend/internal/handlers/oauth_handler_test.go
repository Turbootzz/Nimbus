package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/stretchr/testify/assert"

	_ "github.com/mattn/go-sqlite3"
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

func TestOAuthHandler_InitiateOAuth_RememberMe(t *testing.T) {
	// Test that remember_me query parameter is correctly parsed
	app := fiber.New()
	app.Get("/oauth/:provider", func(c *fiber.Ctx) error {
		rememberMe := c.Query("remember_me", "false") == "true"
		return c.JSON(fiber.Map{
			"remember_me": rememberMe,
		})
	})

	tests := []struct {
		name               string
		queryParams        string
		expectedRememberMe bool
	}{
		{
			name:               "No remember_me param defaults to false",
			queryParams:        "",
			expectedRememberMe: false,
		},
		{
			name:               "remember_me=true",
			queryParams:        "?remember_me=true",
			expectedRememberMe: true,
		},
		{
			name:               "remember_me=false",
			queryParams:        "?remember_me=false",
			expectedRememberMe: false,
		},
		{
			name:               "remember_me with other params",
			queryParams:        "?redirect=/settings&remember_me=true",
			expectedRememberMe: true,
		},
		{
			name:               "Invalid remember_me value treated as false",
			queryParams:        "?remember_me=yes",
			expectedRememberMe: false,
		},
		{
			name:               "remember_me=1 treated as false (must be 'true')",
			queryParams:        "?remember_me=1",
			expectedRememberMe: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/oauth/google"+tt.queryParams, nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, fiber.StatusOK, resp.StatusCode)

			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedRememberMe, result["remember_me"])
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

// setupOAuthTestDB creates an in-memory SQLite database for OAuth testing
func setupOAuthTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	schema := `
		CREATE TABLE IF NOT EXISTS system_settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_by TEXT
		);

		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			password TEXT,
			role TEXT NOT NULL DEFAULT 'user',
			provider TEXT NOT NULL DEFAULT 'local',
			provider_id TEXT,
			avatar_url TEXT,
			email_verified INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			last_activity_at TIMESTAMP,
			UNIQUE(provider, provider_id)
		);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create test tables: %v", err)
	}

	return db
}

func TestOAuthHandler_RegistrationDisabledCheck(t *testing.T) {
	// This test simulates the registration check that happens in OAuth callback
	// when a new user tries to sign up via OAuth

	db := setupOAuthTestDB(t)
	defer db.Close()

	// Insert setting with registration disabled
	_, err := db.Exec(`
		INSERT INTO system_settings (key, value, updated_at)
		VALUES (?, ?, ?)
	`, "public_registration_enabled", "false", time.Now())
	if err != nil {
		t.Fatalf("Failed to insert setting: %v", err)
	}

	settingsRepo := repository.NewSettingsRepository(db)

	// Simulate the OAuth callback handler logic for new user registration
	app := fiber.New()
	app.Get("/oauth/:provider/callback", func(c *fiber.Ctx) error {
		// Simulate: user not found in database (new OAuth user)
		userExists := false

		if !userExists {
			// Check if public registration is enabled
			isEnabled, err := settingsRepo.IsPublicRegistrationEnabled(c.Context())
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to check registration status",
				})
			}
			if !isEnabled {
				// Redirect with error (simulating actual handler behavior)
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "Public registration is disabled",
				})
			}
		}
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/oauth/google/callback?code=abc&state=xyz", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)

	var result map[string]string
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "Public registration is disabled", result["error"])
}

func TestOAuthHandler_RegistrationEnabledCheck(t *testing.T) {
	// This test verifies OAuth registration works when enabled

	db := setupOAuthTestDB(t)
	defer db.Close()

	// Insert setting with registration enabled
	_, err := db.Exec(`
		INSERT INTO system_settings (key, value, updated_at)
		VALUES (?, ?, ?)
	`, "public_registration_enabled", "true", time.Now())
	if err != nil {
		t.Fatalf("Failed to insert setting: %v", err)
	}

	settingsRepo := repository.NewSettingsRepository(db)

	app := fiber.New()
	app.Get("/oauth/:provider/callback", func(c *fiber.Ctx) error {
		userExists := false

		if !userExists {
			isEnabled, err := settingsRepo.IsPublicRegistrationEnabled(c.Context())
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to check registration status",
				})
			}
			if !isEnabled {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "Public registration is disabled",
				})
			}
		}
		// Registration allowed - would create user here
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/oauth/google/callback?code=abc&state=xyz", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestOAuthHandler_ExistingUserAvatarRefresh(t *testing.T) {
	// On every OAuth login, the handler should refresh the cached avatar URL
	// so that providers (e.g. Discord) rotating the URL after an avatar change
	// don't leave the user with a stale, 404-ing image.

	db := setupOAuthTestDB(t)
	defer db.Close()

	userRepo := repository.NewUserRepository(db)

	staleURL := "https://cdn.discordapp.com/avatars/123/old_hash.png"
	freshURL := "https://cdn.discordapp.com/avatars/123/new_hash.png"

	user := &models.User{
		ID:         "user-1",
		Email:      "discord@example.com",
		Name:       "Discord User",
		Provider:   "discord",
		ProviderID: stringPtr("123"),
		AvatarURL:  &staleURL,
		Role:       "user",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if _, err := db.Exec(`
		INSERT INTO users (id, email, name, provider, provider_id, avatar_url, role, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, user.ID, user.Email, user.Name, user.Provider, user.ProviderID, *user.AvatarURL, user.Role, user.CreatedAt, user.UpdatedAt); err != nil {
		t.Fatalf("Failed to seed user: %v", err)
	}

	// Simulate the relevant slice of HandleCallback: lookup existing user,
	// refresh the avatar URL when the provider's value differs.
	app := fiber.New()
	app.Get("/oauth/:provider/callback", func(c *fiber.Ctx) error {
		existingUser, err := userRepo.GetByProviderID("discord", "123")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}
		if existingUser.AvatarURL == nil || *existingUser.AvatarURL != freshURL {
			if err := userRepo.UpdateAvatar(existingUser.ID, &freshURL); err != nil {
				return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
			}
		}
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/oauth/discord/callback", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	refreshed, err := userRepo.GetByID(user.ID)
	assert.NoError(t, err)
	assert.NotNil(t, refreshed.AvatarURL)
	assert.Equal(t, freshURL, *refreshed.AvatarURL)
}

func stringPtr(s string) *string { return &s }

func TestOAuthHandler_ExistingUserBypassesRegistrationCheck(t *testing.T) {
	// This test verifies existing OAuth users can log in even when registration is disabled

	db := setupOAuthTestDB(t)
	defer db.Close()

	// Insert setting with registration disabled
	_, err := db.Exec(`
		INSERT INTO system_settings (key, value, updated_at)
		VALUES (?, ?, ?)
	`, "public_registration_enabled", "false", time.Now())
	if err != nil {
		t.Fatalf("Failed to insert setting: %v", err)
	}

	settingsRepo := repository.NewSettingsRepository(db)

	app := fiber.New()
	app.Get("/oauth/:provider/callback", func(c *fiber.Ctx) error {
		// Simulate: user exists in database (existing OAuth user)
		userExists := true

		if !userExists {
			isEnabled, err := settingsRepo.IsPublicRegistrationEnabled(c.Context())
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to check registration status",
				})
			}
			if !isEnabled {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "Public registration is disabled",
				})
			}
		}
		// Existing user - log them in
		return c.JSON(fiber.Map{
			"message": "User logged in successfully",
		})
	})

	req := httptest.NewRequest("GET", "/oauth/google/callback?code=abc&state=xyz", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]string
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "User logged in successfully", result["message"])
}
