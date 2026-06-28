package handlers

import (
	"context"
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

// newResolveTestHandler builds a real OAuthHandler wired to in-memory repos.
// oauthService/authService are intentionally nil — resolveOAuthUser never uses
// them, which is exactly what lets us unit-test the security logic without any
// HTTP/provider calls.
func newResolveTestHandler(db *sql.DB) *OAuthHandler {
	return &OAuthHandler{
		userRepo:     repository.NewUserRepository(db),
		settingsRepo: repository.NewSettingsRepository(db),
	}
}

// seedResolveUser inserts a user row. Empty providerID/avatar are stored as NULL.
func seedResolveUser(t *testing.T, db *sql.DB, id, email, provider, providerID, avatar string) {
	t.Helper()
	var pid, av interface{}
	if providerID != "" {
		pid = providerID
	}
	if avatar != "" {
		av = avatar
	}
	now := time.Now()
	if _, err := db.Exec(`
		INSERT INTO users (id, email, name, provider, provider_id, avatar_url, role, created_at, updated_at)
		VALUES (?, ?, 'Test User', ?, ?, ?, 'user', ?, ?)
	`, id, email, provider, pid, av, now, now); err != nil {
		t.Fatalf("Failed to seed user: %v", err)
	}
}

func setRegistration(t *testing.T, db *sql.DB, enabled bool) {
	t.Helper()
	value := "true"
	if !enabled {
		value = "false"
	}
	if _, err := db.Exec(`
		INSERT INTO system_settings (key, value, updated_at) VALUES (?, ?, ?)
	`, "public_registration_enabled", value, time.Now()); err != nil {
		t.Fatalf("Failed to set registration: %v", err)
	}
}

// TestResolveOAuthUser_VerifiedEmail_Links is the positive case for #167: a
// provider-verified email may link into a matching pre-existing account.
func TestResolveOAuthUser_VerifiedEmail_Links(t *testing.T) {
	db := setupOAuthTestDB(t)
	defer db.Close()
	h := newResolveTestHandler(db)
	seedResolveUser(t, db, "victim-1", "victim@example.com", "local", "", "")

	user, err := h.resolveOAuthUser(context.Background(), models.ProviderGoogle, &models.OAuthUserInfo{
		ProviderID:    "g1",
		Email:         "victim@example.com",
		EmailVerified: true,
	})
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "victim-1", user.ID)

	linked, err := h.userRepo.GetByID("victim-1")
	assert.NoError(t, err)
	assert.Equal(t, "google", linked.Provider)
	assert.NotNil(t, linked.ProviderID)
	assert.Equal(t, "g1", *linked.ProviderID)
}

// TestResolveOAuthUser_UnverifiedEmail_DoesNotLink is the load-bearing security
// regression for #167: an unverified email must NOT link, and the existing
// account must be left completely untouched (no takeover).
func TestResolveOAuthUser_UnverifiedEmail_DoesNotLink(t *testing.T) {
	db := setupOAuthTestDB(t)
	defer db.Close()
	h := newResolveTestHandler(db)
	seedResolveUser(t, db, "victim-1", "victim@example.com", "local", "", "")

	user, err := h.resolveOAuthUser(context.Background(), models.ProviderGoogle, &models.OAuthUserInfo{
		ProviderID:    "g1",
		Email:         "victim@example.com",
		EmailVerified: false,
	})
	assert.Nil(t, user)
	assert.ErrorIs(t, err, errEmailNotVerified)

	// The victim's account must be unchanged: still local, no provider linked.
	unchanged, err := h.userRepo.GetByID("victim-1")
	assert.NoError(t, err)
	assert.Equal(t, "local", unchanged.Provider)
	assert.Nil(t, unchanged.ProviderID)
}

// TestResolveOAuthUser_EmptyEmail_DoesNotLink covers the defense-in-depth guard:
// a permissive IdP returning an empty email must not match/link an account.
func TestResolveOAuthUser_EmptyEmail_DoesNotLink(t *testing.T) {
	db := setupOAuthTestDB(t)
	defer db.Close()
	h := newResolveTestHandler(db)
	seedResolveUser(t, db, "ghost-1", "", "local", "", "")

	user, err := h.resolveOAuthUser(context.Background(), models.ProviderOIDC, &models.OAuthUserInfo{
		ProviderID:    "g9",
		Email:         "",
		EmailVerified: true,
	})
	assert.Nil(t, user)
	assert.ErrorIs(t, err, errEmailNotVerified)

	unchanged, err := h.userRepo.GetByID("ghost-1")
	assert.NoError(t, err)
	assert.Equal(t, "local", unchanged.Provider)
	assert.Nil(t, unchanged.ProviderID)
}

// TestResolveOAuthUser_EmptyEmail_NoUser_DoesNotCreate verifies that an empty
// provider email cannot fall through into the new-user creation branch (which
// would persist a malformed, NOT NULL/unique-colliding account).
func TestResolveOAuthUser_EmptyEmail_NoUser_DoesNotCreate(t *testing.T) {
	db := setupOAuthTestDB(t)
	defer db.Close()
	h := newResolveTestHandler(db) // empty users table, registration enabled by default

	user, err := h.resolveOAuthUser(context.Background(), models.ProviderOIDC, &models.OAuthUserInfo{
		ProviderID:    "g9",
		Email:         "",
		EmailVerified: true,
	})
	assert.Nil(t, user)
	assert.ErrorIs(t, err, errMissingEmail)

	// Nothing should have been created.
	_, err = h.userRepo.GetByEmail("")
	assert.ErrorIs(t, err, repository.ErrUserNotFound)
}

// TestResolveOAuthUser_NewVerifiedUser_Created confirms a brand-new verified
// login creates an account.
func TestResolveOAuthUser_NewVerifiedUser_Created(t *testing.T) {
	db := setupOAuthTestDB(t)
	defer db.Close()
	h := newResolveTestHandler(db) // registration defaults to enabled (no setting row)

	user, err := h.resolveOAuthUser(context.Background(), models.ProviderGoogle, &models.OAuthUserInfo{
		ProviderID:    "g2",
		Email:         "new@example.com",
		Name:          "New User",
		EmailVerified: true,
	})
	assert.NoError(t, err)
	assert.NotNil(t, user)

	created, err := h.userRepo.GetByEmail("new@example.com")
	assert.NoError(t, err)
	assert.Equal(t, "google", created.Provider)
	assert.True(t, created.EmailVerified)
}

// TestResolveOAuthUser_NewUnverifiedUser_Created confirms creating a fresh
// account is still allowed when unverified — there is no existing row to take
// over, so only LINKING (not creation) needs verification. The flag is persisted
// honestly as false.
func TestResolveOAuthUser_NewUnverifiedUser_Created(t *testing.T) {
	db := setupOAuthTestDB(t)
	defer db.Close()
	h := newResolveTestHandler(db)

	user, err := h.resolveOAuthUser(context.Background(), models.ProviderGoogle, &models.OAuthUserInfo{
		ProviderID:    "g3",
		Email:         "unverified@example.com",
		Name:          "Unverified User",
		EmailVerified: false,
	})
	assert.NoError(t, err)
	assert.NotNil(t, user)

	created, err := h.userRepo.GetByEmail("unverified@example.com")
	assert.NoError(t, err)
	assert.False(t, created.EmailVerified)
}

// TestResolveOAuthUser_RegistrationDisabled_Refused confirms new accounts are
// blocked when public registration is off.
func TestResolveOAuthUser_RegistrationDisabled_Refused(t *testing.T) {
	db := setupOAuthTestDB(t)
	defer db.Close()
	h := newResolveTestHandler(db)
	setRegistration(t, db, false)

	user, err := h.resolveOAuthUser(context.Background(), models.ProviderGoogle, &models.OAuthUserInfo{
		ProviderID:    "g4",
		Email:         "new@example.com",
		EmailVerified: true,
	})
	assert.Nil(t, user)
	assert.ErrorIs(t, err, errRegistrationDisabled)

	_, err = h.userRepo.GetByEmail("new@example.com")
	assert.ErrorIs(t, err, repository.ErrUserNotFound)
}

// TestResolveOAuthUser_ExistingProviderID_LogsIn proves a known provider id
// short-circuits before the email, verification, and registration checks.
func TestResolveOAuthUser_ExistingProviderID_LogsIn(t *testing.T) {
	db := setupOAuthTestDB(t)
	defer db.Close()
	h := newResolveTestHandler(db)
	setRegistration(t, db, false) // even with registration off...
	seedResolveUser(t, db, "user-1", "user@example.com", "google", "g1", "")

	user, err := h.resolveOAuthUser(context.Background(), models.ProviderGoogle, &models.OAuthUserInfo{
		ProviderID:    "g1",
		Email:         "user@example.com",
		EmailVerified: false, // ...and an unverified claim, this still logs in
	})
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "user-1", user.ID)
}

// TestResolveOAuthUser_EmailLink_PreservesLocalAvatar confirms a locally
// uploaded avatar survives an email-link.
func TestResolveOAuthUser_EmailLink_PreservesLocalAvatar(t *testing.T) {
	db := setupOAuthTestDB(t)
	defer db.Close()
	h := newResolveTestHandler(db)
	seedResolveUser(t, db, "user-1", "linker@example.com", "local", "", "/uploads/avatars/custom.png")

	_, err := h.resolveOAuthUser(context.Background(), models.ProviderDiscord, &models.OAuthUserInfo{
		ProviderID:    "d1",
		Email:         "linker@example.com",
		AvatarURL:     "https://cdn.discordapp.com/avatars/d1/hash.png",
		EmailVerified: true,
	})
	assert.NoError(t, err)

	linked, err := h.userRepo.GetByID("user-1")
	assert.NoError(t, err)
	assert.NotNil(t, linked.AvatarURL)
	assert.Equal(t, "/uploads/avatars/custom.png", *linked.AvatarURL)
}

// TestResolveOAuthUser_ProviderIDLogin_RefreshesStaleAvatar confirms a rotated
// provider avatar URL is refreshed on login.
func TestResolveOAuthUser_ProviderIDLogin_RefreshesStaleAvatar(t *testing.T) {
	db := setupOAuthTestDB(t)
	defer db.Close()
	h := newResolveTestHandler(db)
	seedResolveUser(t, db, "user-1", "disco@example.com", "discord", "d1", "https://cdn.discordapp.com/avatars/d1/old.png")

	_, err := h.resolveOAuthUser(context.Background(), models.ProviderDiscord, &models.OAuthUserInfo{
		ProviderID: "d1",
		Email:      "disco@example.com",
		AvatarURL:  "https://cdn.discordapp.com/avatars/d1/new.png",
	})
	assert.NoError(t, err)

	refreshed, err := h.userRepo.GetByID("user-1")
	assert.NoError(t, err)
	assert.NotNil(t, refreshed.AvatarURL)
	assert.Equal(t, "https://cdn.discordapp.com/avatars/d1/new.png", *refreshed.AvatarURL)
}

// TestShouldRefreshAvatar covers the avatar-refresh decision directly, including
// the "URL unchanged -> skip the write" branch (previously verified end-to-end
// by TestOAuthHandler_AvatarRefresh_MatchingURLIsSkipped).
func TestShouldRefreshAvatar(t *testing.T) {
	local := "/uploads/avatars/custom.png"
	same := "https://cdn/avatar.png"
	stale := "https://cdn/old.png"

	tests := []struct {
		name        string
		current     *string
		providerURL string
		want        bool
	}{
		{"nil current -> refresh", nil, "https://cdn/new.png", true},
		{"different URL -> refresh", &stale, "https://cdn/new.png", true},
		{"same URL -> skip", &same, same, false},
		{"local upload -> never clobber", &local, "https://cdn/new.png", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, shouldRefreshAvatar(tt.current, tt.providerURL))
		})
	}
}
