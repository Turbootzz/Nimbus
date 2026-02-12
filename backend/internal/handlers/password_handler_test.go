package handlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/services"

	_ "github.com/mattn/go-sqlite3"
)

// setupPasswordTestDB creates an in-memory SQLite database for password handler testing
func setupPasswordTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	schema := `
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

		CREATE TABLE IF NOT EXISTS password_reset_tokens (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token_hash TEXT NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			used_at TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS system_settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_by TEXT
		);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create test tables: %v", err)
	}

	return db
}

func setupPasswordHandler(t *testing.T) (*PasswordHandler, *sql.DB, *services.AuthService) {
	os.Setenv("JWT_SECRET", "test-secret-key-for-testing-minimum-32-chars")

	db := setupPasswordTestDB(t)
	userRepo := repository.NewUserRepository(db)
	authService := services.NewAuthService()
	settingsRepo := repository.NewSettingsRepository(db)
	emailService := services.NewEmailService(settingsRepo)
	passwordResetRepo := repository.NewPasswordResetRepository(db)

	handler := NewPasswordHandler(userRepo, authService, emailService, passwordResetRepo)

	return handler, db, authService
}

func TestChangePassword_Success(t *testing.T) {
	handler, db, authService := setupPasswordHandler(t)
	defer db.Close()

	// Create a local user
	hashedPassword, _ := authService.HashPassword("oldpassword123")
	now := time.Now()
	user := &models.User{
		ID:        "user-1",
		Email:     "test@example.com",
		Name:      "Test User",
		Password:  &hashedPassword,
		Role:      "user",
		Provider:  "local",
		CreatedAt: now,
		UpdatedAt: now,
	}
	createUserDirectly(t, db, user)

	app := fiber.New()
	app.Put("/change-password", func(c *fiber.Ctx) error {
		c.Locals("user_id", "user-1")
		c.Locals("email", "test@example.com")
		c.Locals("role", "user")
		return handler.ChangePassword(c)
	})

	body, _ := json.Marshal(map[string]string{
		"current_password": "oldpassword123",
		"new_password":     "newpassword123",
	})

	req := httptest.NewRequest(http.MethodPut, "/change-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify the new password works
	var dbPassword string
	err = db.QueryRow("SELECT password FROM users WHERE id = ?", "user-1").Scan(&dbPassword)
	if err != nil {
		t.Fatalf("Failed to query password: %v", err)
	}

	if err := authService.ComparePassword(dbPassword, "newpassword123"); err != nil {
		t.Error("New password should be valid after change")
	}
}

func TestChangePassword_WrongCurrentPassword(t *testing.T) {
	handler, db, authService := setupPasswordHandler(t)
	defer db.Close()

	hashedPassword, _ := authService.HashPassword("correctpassword")
	now := time.Now()
	user := &models.User{
		ID:        "user-1",
		Email:     "test@example.com",
		Name:      "Test User",
		Password:  &hashedPassword,
		Role:      "user",
		Provider:  "local",
		CreatedAt: now,
		UpdatedAt: now,
	}
	createUserDirectly(t, db, user)

	app := fiber.New()
	app.Put("/change-password", func(c *fiber.Ctx) error {
		c.Locals("user_id", "user-1")
		c.Locals("email", "test@example.com")
		c.Locals("role", "user")
		return handler.ChangePassword(c)
	})

	body, _ := json.Marshal(map[string]string{
		"current_password": "wrongpassword",
		"new_password":     "newpassword123",
	})

	req := httptest.NewRequest(http.MethodPut, "/change-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestChangePassword_ShortNewPassword(t *testing.T) {
	handler, db, authService := setupPasswordHandler(t)
	defer db.Close()

	hashedPassword, _ := authService.HashPassword("oldpassword123")
	now := time.Now()
	user := &models.User{
		ID:        "user-1",
		Email:     "test@example.com",
		Name:      "Test User",
		Password:  &hashedPassword,
		Role:      "user",
		Provider:  "local",
		CreatedAt: now,
		UpdatedAt: now,
	}
	createUserDirectly(t, db, user)

	app := fiber.New()
	app.Put("/change-password", func(c *fiber.Ctx) error {
		c.Locals("user_id", "user-1")
		c.Locals("email", "test@example.com")
		c.Locals("role", "user")
		return handler.ChangePassword(c)
	})

	body, _ := json.Marshal(map[string]string{
		"current_password": "oldpassword123",
		"new_password":     "short",
	})

	req := httptest.NewRequest(http.MethodPut, "/change-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestChangePassword_OAuthUser(t *testing.T) {
	handler, db, _ := setupPasswordHandler(t)
	defer db.Close()

	// Create an OAuth user without a password
	now := time.Now()
	providerID := "google-123"
	_, err := db.Exec(
		`INSERT INTO users (id, email, name, password, role, provider, provider_id, email_verified, created_at, updated_at)
		 VALUES (?, ?, ?, NULL, ?, ?, ?, ?, ?, ?)`,
		"user-oauth", "oauth@example.com", "OAuth User", "user", "google", providerID, true, now, now,
	)
	if err != nil {
		t.Fatalf("Failed to create OAuth user: %v", err)
	}

	app := fiber.New()
	app.Put("/change-password", func(c *fiber.Ctx) error {
		c.Locals("user_id", "user-oauth")
		c.Locals("email", "oauth@example.com")
		c.Locals("role", "user")
		return handler.ChangePassword(c)
	})

	body, _ := json.Marshal(map[string]string{
		"current_password": "anything",
		"new_password":     "newpassword123",
	})

	req := httptest.NewRequest(http.MethodPut, "/change-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400 for OAuth user, got %d", resp.StatusCode)
	}
}

func TestResetPassword_Success(t *testing.T) {
	handler, db, authService := setupPasswordHandler(t)
	defer db.Close()

	// Create a local user
	hashedPassword, _ := authService.HashPassword("oldpassword123")
	now := time.Now()
	user := &models.User{
		ID:        "user-1",
		Email:     "test@example.com",
		Name:      "Test User",
		Password:  &hashedPassword,
		Role:      "user",
		Provider:  "local",
		CreatedAt: now,
		UpdatedAt: now,
	}
	createUserDirectly(t, db, user)

	// Create a valid token
	rawToken := "abc123def456ghi789jkl012mno345pq"
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	_, err := db.Exec(
		`INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		"token-1", "user-1", tokenHash, now.Add(1*time.Hour), now,
	)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	app := fiber.New()
	app.Post("/reset-password", handler.ResetPassword)

	body, _ := json.Marshal(map[string]string{
		"token":        rawToken,
		"new_password": "newpassword123",
	})

	req := httptest.NewRequest(http.MethodPost, "/reset-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify password was changed
	var dbPassword string
	err = db.QueryRow("SELECT password FROM users WHERE id = ?", "user-1").Scan(&dbPassword)
	if err != nil {
		t.Fatalf("Failed to query password: %v", err)
	}

	if err := authService.ComparePassword(dbPassword, "newpassword123"); err != nil {
		t.Error("New password should be valid after reset")
	}

	// Verify token is marked as used
	var usedAt sql.NullTime
	err = db.QueryRow("SELECT used_at FROM password_reset_tokens WHERE id = ?", "token-1").Scan(&usedAt)
	if err != nil {
		t.Fatalf("Failed to query token: %v", err)
	}
	if !usedAt.Valid {
		t.Error("Token should be marked as used")
	}
}

func TestResetPassword_ExpiredToken(t *testing.T) {
	handler, db, authService := setupPasswordHandler(t)
	defer db.Close()

	// Create a local user
	hashedPassword, _ := authService.HashPassword("oldpassword123")
	now := time.Now()
	user := &models.User{
		ID:        "user-1",
		Email:     "test@example.com",
		Name:      "Test User",
		Password:  &hashedPassword,
		Role:      "user",
		Provider:  "local",
		CreatedAt: now,
		UpdatedAt: now,
	}
	createUserDirectly(t, db, user)

	// Create an expired token
	rawToken := "expired-token-value-32-chars-long"
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	_, err := db.Exec(
		`INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		"token-expired", "user-1", tokenHash, now.Add(-1*time.Hour), now.Add(-2*time.Hour),
	)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	app := fiber.New()
	app.Post("/reset-password", handler.ResetPassword)

	body, _ := json.Marshal(map[string]string{
		"token":        rawToken,
		"new_password": "newpassword123",
	})

	req := httptest.NewRequest(http.MethodPost, "/reset-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400 for expired token, got %d", resp.StatusCode)
	}
}

func TestResetPassword_InvalidToken(t *testing.T) {
	handler, db, _ := setupPasswordHandler(t)
	defer db.Close()

	app := fiber.New()
	app.Post("/reset-password", handler.ResetPassword)

	body, _ := json.Marshal(map[string]string{
		"token":        "this-token-does-not-exist-in-db",
		"new_password": "newpassword123",
	})

	req := httptest.NewRequest(http.MethodPost, "/reset-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid token, got %d", resp.StatusCode)
	}
}

func TestResetPassword_UsedToken(t *testing.T) {
	handler, db, authService := setupPasswordHandler(t)
	defer db.Close()

	// Create a local user
	hashedPassword, _ := authService.HashPassword("oldpassword123")
	now := time.Now()
	user := &models.User{
		ID:        "user-1",
		Email:     "test@example.com",
		Name:      "Test User",
		Password:  &hashedPassword,
		Role:      "user",
		Provider:  "local",
		CreatedAt: now,
		UpdatedAt: now,
	}
	createUserDirectly(t, db, user)

	// Create a used token
	rawToken := "used-token-value-at-least-32-chars"
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	_, err := db.Exec(
		`INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at, used_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		"token-used", "user-1", tokenHash, now.Add(1*time.Hour), now, now.Add(-30*time.Minute),
	)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	app := fiber.New()
	app.Post("/reset-password", handler.ResetPassword)

	body, _ := json.Marshal(map[string]string{
		"token":        rawToken,
		"new_password": "newpassword123",
	})

	req := httptest.NewRequest(http.MethodPost, "/reset-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400 for used token, got %d", resp.StatusCode)
	}
}

func TestForgotPassword_AlwaysReturns200(t *testing.T) {
	handler, db, _ := setupPasswordHandler(t)
	defer db.Close()

	app := fiber.New()
	app.Post("/forgot-password", handler.ForgotPassword)

	// Test with non-existent email - should still return 200
	body, _ := json.Marshal(map[string]string{
		"email": "nonexistent@example.com",
	})

	req := httptest.NewRequest(http.MethodPost, "/forgot-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for non-existent email (prevent enumeration), got %d", resp.StatusCode)
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)

	if result["message"] == "" {
		t.Error("Expected a message in the response")
	}
}

func TestForgotPassword_InvalidEmail(t *testing.T) {
	handler, db, _ := setupPasswordHandler(t)
	defer db.Close()

	app := fiber.New()
	app.Post("/forgot-password", handler.ForgotPassword)

	body, _ := json.Marshal(map[string]string{
		"email": "not-an-email",
	})

	req := httptest.NewRequest(http.MethodPost, "/forgot-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid email, got %d", resp.StatusCode)
	}
}

func TestForgotPassword_EmptyEmail(t *testing.T) {
	handler, db, _ := setupPasswordHandler(t)
	defer db.Close()

	app := fiber.New()
	app.Post("/forgot-password", handler.ForgotPassword)

	body, _ := json.Marshal(map[string]string{
		"email": "",
	})

	req := httptest.NewRequest(http.MethodPost, "/forgot-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400 for empty email, got %d", resp.StatusCode)
	}
}

func TestPasswordResetRepository_DeleteExpired(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-key-for-testing-minimum-32-chars")

	db := setupPasswordTestDB(t)
	defer db.Close()

	repo := repository.NewPasswordResetRepository(db)
	ctx := context.Background()

	now := time.Now()

	// Insert an expired token
	_, err := db.Exec(
		`INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		"expired-1", "user-1", "hash1", now.Add(-2*time.Hour), now.Add(-3*time.Hour),
	)
	if err != nil {
		// user-1 doesn't exist, create a dummy user first
		_, err = db.Exec(
			`INSERT INTO users (id, email, name, password, role, provider, email_verified, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			"user-1", "test@example.com", "Test", "hash", "user", "local", false, now, now,
		)
		if err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}

		_, err = db.Exec(
			`INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at, created_at)
			 VALUES (?, ?, ?, ?, ?)`,
			"expired-1", "user-1", "hash1", now.Add(-2*time.Hour), now.Add(-3*time.Hour),
		)
		if err != nil {
			t.Fatalf("Failed to insert expired token: %v", err)
		}
	}

	// Insert a valid token
	_, err = db.Exec(
		`INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		"valid-1", "user-1", "hash2", now.Add(1*time.Hour), now,
	)
	if err != nil {
		t.Fatalf("Failed to insert valid token: %v", err)
	}

	deleted, err := repo.DeleteExpired(ctx)
	if err != nil {
		t.Fatalf("DeleteExpired failed: %v", err)
	}

	if deleted != 1 {
		t.Errorf("Expected 1 deleted, got %d", deleted)
	}

	// Verify the valid token still exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM password_reset_tokens").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count tokens: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 remaining token, got %d", count)
	}
}
