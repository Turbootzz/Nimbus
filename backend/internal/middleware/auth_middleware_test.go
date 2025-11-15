package middleware

import (
	"database/sql"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/services"
	"github.com/stretchr/testify/assert"

	_ "github.com/mattn/go-sqlite3"
)

// setupMiddlewareTestDB creates an in-memory SQLite database for middleware testing
func setupMiddlewareTestDB(t *testing.T) *sql.DB {
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
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	return db
}

// setupTestEnv sets up test environment variables
func setupTestEnv() {
	os.Setenv("JWT_SECRET", "test-secret-32-characters-long!!")
}

// createTestUser inserts a test user directly into the database
func createTestUser(t *testing.T, db *sql.DB, role string) *models.User {
	hashedPassword := "hashedpassword"
	user := &models.User{
		ID:        uuid.New().String(),
		Email:     "test@example.com",
		Name:      "Test User",
		Password:  &hashedPassword,
		Role:      role,
		Provider:  "local",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	query := `
		INSERT INTO users (id, email, name, password, role, provider, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := db.Exec(query, user.ID, user.Email, user.Name, user.Password, user.Role, user.Provider, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	return user
}

func TestAuthMiddleware_ValidToken_Cookie(t *testing.T) {
	setupTestEnv()
	// Setup
	setupTestEnv()
	db := setupMiddlewareTestDB(t)
	defer db.Close()

	user := createTestUser(t, db, "user")
	userRepo := repository.NewUserRepository(db)
	authService := services.NewAuthService()

	// Generate token
	token, err := authService.GenerateToken(user.ID, user.Email, user.Role)
	assert.NoError(t, err)

	// Create fiber app with middleware
	app := fiber.New()
	app.Use(AuthMiddleware(authService, userRepo))
	app.Get("/protected", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"user_id": c.Locals("user_id"),
			"email":   c.Locals("email"),
			"role":    c.Locals("role"),
		})
	})

	// Create request with cookie
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Cookie", "auth_token="+token)

	// Execute
	resp, err := app.Test(req)
	assert.NoError(t, err)

	// Assert
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestAuthMiddleware_ValidToken_BearerHeader(t *testing.T) {
	setupTestEnv()
	// Setup
	setupTestEnv()
	db := setupMiddlewareTestDB(t)
	defer db.Close()

	user := createTestUser(t, db, "user")
	userRepo := repository.NewUserRepository(db)
	authService := services.NewAuthService()

	// Generate token
	token, err := authService.GenerateToken(user.ID, user.Email, user.Role)
	assert.NoError(t, err)

	// Create fiber app with middleware
	app := fiber.New()
	app.Use(AuthMiddleware(authService, userRepo))
	app.Get("/protected", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"user_id": c.Locals("user_id"),
			"email":   c.Locals("email"),
			"role":    c.Locals("role"),
		})
	})

	// Create request with Bearer token
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	// Execute
	resp, err := app.Test(req)
	assert.NoError(t, err)

	// Assert
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestAuthMiddleware_MissingToken(t *testing.T) {
	setupTestEnv()
	// Setup
	db := setupMiddlewareTestDB(t)
	defer db.Close()

	userRepo := repository.NewUserRepository(db)
	authService := services.NewAuthService()

	// Create fiber app with middleware
	app := fiber.New()
	app.Use(AuthMiddleware(authService, userRepo))
	app.Get("/protected", func(c *fiber.Ctx) error {
		return c.SendString("Should not reach here")
	})

	// Create request without token
	req := httptest.NewRequest("GET", "/protected", nil)

	// Execute
	resp, err := app.Test(req)
	assert.NoError(t, err)

	// Assert
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	setupTestEnv()
	// Setup
	db := setupMiddlewareTestDB(t)
	defer db.Close()

	userRepo := repository.NewUserRepository(db)
	authService := services.NewAuthService()

	// Create fiber app with middleware
	app := fiber.New()
	app.Use(AuthMiddleware(authService, userRepo))
	app.Get("/protected", func(c *fiber.Ctx) error {
		return c.SendString("Should not reach here")
	})

	// Create request with invalid token
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Cookie", "auth_token=invalid.token.here")

	// Execute
	resp, err := app.Test(req)
	assert.NoError(t, err)

	// Assert
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	setupTestEnv()
	// Setup
	db := setupMiddlewareTestDB(t)
	defer db.Close()

	user := createTestUser(t, db, "user")
	userRepo := repository.NewUserRepository(db)
	authService := services.NewAuthService()

	// Generate token with 1 second expiry
	token, err := authService.GenerateTokenWithExpiration(user.ID, user.Email, user.Role, 1*time.Second)
	assert.NoError(t, err)

	// Wait for token to expire
	time.Sleep(2 * time.Second)

	// Create fiber app with middleware
	app := fiber.New()
	app.Use(AuthMiddleware(authService, userRepo))
	app.Get("/protected", func(c *fiber.Ctx) error {
		return c.SendString("Should not reach here")
	})

	// Create request with expired token
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Cookie", "auth_token="+token)

	// Execute
	resp, err := app.Test(req)
	assert.NoError(t, err)

	// Assert
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestAuthMiddleware_UserNotFound(t *testing.T) {
	setupTestEnv()
	// Setup
	db := setupMiddlewareTestDB(t)
	defer db.Close()

	userRepo := repository.NewUserRepository(db)
	authService := services.NewAuthService()

	// Generate token for non-existent user
	nonExistentUserID := uuid.New().String()
	token, err := authService.GenerateToken(nonExistentUserID, "ghost@example.com", "user")
	assert.NoError(t, err)

	// Create fiber app with middleware
	app := fiber.New()
	app.Use(AuthMiddleware(authService, userRepo))
	app.Get("/protected", func(c *fiber.Ctx) error {
		return c.SendString("Should not reach here")
	})

	// Create request
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Cookie", "auth_token="+token)

	// Execute
	resp, err := app.Test(req)
	assert.NoError(t, err)

	// Assert
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestOptionalAuthMiddleware_WithValidToken(t *testing.T) {
	setupTestEnv()
	// Setup
	db := setupMiddlewareTestDB(t)
	defer db.Close()

	user := createTestUser(t, db, "user")
	userRepo := repository.NewUserRepository(db)
	authService := services.NewAuthService()

	// Generate token
	token, err := authService.GenerateToken(user.ID, user.Email, user.Role)
	assert.NoError(t, err)

	// Create fiber app with middleware
	app := fiber.New()
	app.Use(OptionalAuthMiddleware(authService, userRepo))
	app.Get("/maybe-protected", func(c *fiber.Ctx) error {
		userID := c.Locals("user_id")
		if userID == nil {
			return c.JSON(fiber.Map{"authenticated": false})
		}
		return c.JSON(fiber.Map{
			"authenticated": true,
			"user_id":       userID,
		})
	})

	// Create request with token
	req := httptest.NewRequest("GET", "/maybe-protected", nil)
	req.Header.Set("Cookie", "auth_token="+token)

	// Execute
	resp, err := app.Test(req)
	assert.NoError(t, err)

	// Assert
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestOptionalAuthMiddleware_WithoutToken(t *testing.T) {
	setupTestEnv()
	// Setup
	db := setupMiddlewareTestDB(t)
	defer db.Close()

	userRepo := repository.NewUserRepository(db)
	authService := services.NewAuthService()

	// Create fiber app with middleware
	app := fiber.New()
	app.Use(OptionalAuthMiddleware(authService, userRepo))
	app.Get("/maybe-protected", func(c *fiber.Ctx) error {
		userID := c.Locals("user_id")
		if userID == nil {
			return c.JSON(fiber.Map{"authenticated": false})
		}
		return c.JSON(fiber.Map{"authenticated": true})
	})

	// Create request without token
	req := httptest.NewRequest("GET", "/maybe-protected", nil)

	// Execute
	resp, err := app.Test(req)
	assert.NoError(t, err)

	// Assert - should succeed without authentication
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestOptionalAuthMiddleware_WithInvalidToken(t *testing.T) {
	setupTestEnv()
	// Setup
	db := setupMiddlewareTestDB(t)
	defer db.Close()

	userRepo := repository.NewUserRepository(db)
	authService := services.NewAuthService()

	// Create fiber app with middleware
	app := fiber.New()
	app.Use(OptionalAuthMiddleware(authService, userRepo))
	app.Get("/maybe-protected", func(c *fiber.Ctx) error {
		userID := c.Locals("user_id")
		if userID == nil {
			return c.JSON(fiber.Map{"authenticated": false})
		}
		return c.JSON(fiber.Map{"authenticated": true})
	})

	// Create request with invalid token
	req := httptest.NewRequest("GET", "/maybe-protected", nil)
	req.Header.Set("Cookie", "auth_token=invalid.token.here")

	// Execute
	resp, err := app.Test(req)
	assert.NoError(t, err)

	// Assert - should succeed without authentication (invalid token is gracefully ignored)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestAdminOnly_WithAdminRole(t *testing.T) {
	setupTestEnv()
	// Setup
	app := fiber.New()

	// Simulate auth middleware setting role
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("role", "admin")
		return c.Next()
	})

	app.Use(AdminOnly())
	app.Get("/admin", func(c *fiber.Ctx) error {
		return c.SendString("Admin access granted")
	})

	// Create request
	req := httptest.NewRequest("GET", "/admin", nil)

	// Execute
	resp, err := app.Test(req)
	assert.NoError(t, err)

	// Assert
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestAdminOnly_WithUserRole(t *testing.T) {
	setupTestEnv()
	// Setup
	app := fiber.New()

	// Simulate auth middleware setting role
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("role", "user")
		return c.Next()
	})

	app.Use(AdminOnly())
	app.Get("/admin", func(c *fiber.Ctx) error {
		return c.SendString("Should not reach here")
	})

	// Create request
	req := httptest.NewRequest("GET", "/admin", nil)

	// Execute
	resp, err := app.Test(req)
	assert.NoError(t, err)

	// Assert
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

func TestAdminOnly_WithoutRole(t *testing.T) {
	setupTestEnv()
	// Setup
	app := fiber.New()
	app.Use(AdminOnly())
	app.Get("/admin", func(c *fiber.Ctx) error {
		return c.SendString("Should not reach here")
	})

	// Create request
	req := httptest.NewRequest("GET", "/admin", nil)

	// Execute
	resp, err := app.Test(req)
	assert.NoError(t, err)

	// Assert
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

func TestAuthMiddleware_ContextValues(t *testing.T) {
	setupTestEnv()
	// Setup
	db := setupMiddlewareTestDB(t)
	defer db.Close()

	user := createTestUser(t, db, "admin")
	userRepo := repository.NewUserRepository(db)
	authService := services.NewAuthService()

	// Generate token
	token, err := authService.GenerateToken(user.ID, user.Email, user.Role)
	assert.NoError(t, err)

	// Create fiber app with middleware
	var capturedUserID, capturedEmail, capturedRole string
	app := fiber.New()
	app.Use(AuthMiddleware(authService, userRepo))
	app.Get("/protected", func(c *fiber.Ctx) error {
		capturedUserID = c.Locals("user_id").(string)
		capturedEmail = c.Locals("email").(string)
		capturedRole = c.Locals("role").(string)
		return c.SendStatus(fiber.StatusOK)
	})

	// Create request
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Cookie", "auth_token="+token)

	// Execute
	resp, err := app.Test(req)
	assert.NoError(t, err)

	// Assert
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.Equal(t, user.ID, capturedUserID)
	assert.Equal(t, user.Email, capturedEmail)
	assert.Equal(t, user.Role, capturedRole)
}
