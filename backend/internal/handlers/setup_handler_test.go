package handlers

import (
	"bytes"
	"database/sql"
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

// setupSetupTestDB creates an in-memory SQLite database for setup testing
func setupSetupTestDB(t *testing.T) *sql.DB {
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

func TestSetupHandler_GetSetupStatus_NoUsers(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-for-jwt-token-generation-minimum-32-chars")
	defer os.Unsetenv("JWT_SECRET")

	db := setupSetupTestDB(t)
	defer db.Close()

	userRepo := repository.NewUserRepository(db)
	authService := services.NewAuthService()
	handler := NewSetupHandler(userRepo, authService)

	app := fiber.New()
	app.Get("/setup/status", handler.GetSetupStatus)

	req := httptest.NewRequest(http.MethodGet, "/setup/status", nil)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var response map[string]bool
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !response["needs_setup"] {
		t.Error("Expected needs_setup to be true when no users exist")
	}
}

func TestSetupHandler_GetSetupStatus_UsersExist(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-for-jwt-token-generation-minimum-32-chars")
	defer os.Unsetenv("JWT_SECRET")

	db := setupSetupTestDB(t)
	defer db.Close()

	// Create a user
	password := "hashedpassword"
	_, err := db.Exec(`
		INSERT INTO users (id, email, name, password, role, provider, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, "user-1", "admin@example.com", "Admin", password, "admin", "local", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	userRepo := repository.NewUserRepository(db)
	authService := services.NewAuthService()
	handler := NewSetupHandler(userRepo, authService)

	app := fiber.New()
	app.Get("/setup/status", handler.GetSetupStatus)

	req := httptest.NewRequest(http.MethodGet, "/setup/status", nil)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var response map[string]bool
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["needs_setup"] {
		t.Error("Expected needs_setup to be false when users exist")
	}
}

func TestSetupHandler_CreateInitialAdmin_Success(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-for-jwt-token-generation-minimum-32-chars")
	defer os.Unsetenv("JWT_SECRET")

	os.Setenv("COOKIE_SECURE", "false")
	defer os.Unsetenv("COOKIE_SECURE")

	db := setupSetupTestDB(t)
	defer db.Close()

	userRepo := repository.NewUserRepository(db)
	authService := services.NewAuthService()
	handler := NewSetupHandler(userRepo, authService)

	app := fiber.New()
	app.Post("/setup/admin", handler.CreateInitialAdmin)

	registerReq := models.RegisterRequest{
		Name:     "Admin User",
		Email:    "admin@example.com",
		Password: "SecurePassword123!",
	}

	bodyJSON, _ := json.Marshal(registerReq)
	req := httptest.NewRequest(http.MethodPost, "/setup/admin", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, resp.StatusCode)
	}

	// Verify auth cookie is set
	cookies := resp.Cookies()
	var authCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "auth_token" {
			authCookie = cookie
			break
		}
	}

	if authCookie == nil {
		t.Fatal("auth_token cookie not found")
	}

	// Verify user was created with admin role
	var role string
	err = db.QueryRow("SELECT role FROM users WHERE email = ?", registerReq.Email).Scan(&role)
	if err != nil {
		t.Fatalf("Failed to query user: %v", err)
	}

	if role != "admin" {
		t.Errorf("Expected user role to be 'admin', got '%s'", role)
	}
}

func TestSetupHandler_CreateInitialAdmin_FailsWhenUsersExist(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-for-jwt-token-generation-minimum-32-chars")
	defer os.Unsetenv("JWT_SECRET")

	db := setupSetupTestDB(t)
	defer db.Close()

	// Create an existing user
	password := "hashedpassword"
	_, err := db.Exec(`
		INSERT INTO users (id, email, name, password, role, provider, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, "user-1", "existing@example.com", "Existing User", password, "user", "local", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	userRepo := repository.NewUserRepository(db)
	authService := services.NewAuthService()
	handler := NewSetupHandler(userRepo, authService)

	app := fiber.New()
	app.Post("/setup/admin", handler.CreateInitialAdmin)

	registerReq := models.RegisterRequest{
		Name:     "Admin User",
		Email:    "admin@example.com",
		Password: "SecurePassword123!",
	}

	bodyJSON, _ := json.Marshal(registerReq)
	req := httptest.NewRequest(http.MethodPost, "/setup/admin", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("Expected status %d when users already exist, got %d", http.StatusForbidden, resp.StatusCode)
	}
}

func TestSetupHandler_CreateInitialAdmin_ValidationErrors(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-for-jwt-token-generation-minimum-32-chars")
	defer os.Unsetenv("JWT_SECRET")

	db := setupSetupTestDB(t)
	defer db.Close()

	userRepo := repository.NewUserRepository(db)
	authService := services.NewAuthService()
	handler := NewSetupHandler(userRepo, authService)

	tests := []struct {
		name           string
		request        models.RegisterRequest
		expectedStatus int
	}{
		{
			name: "Missing name",
			request: models.RegisterRequest{
				Email:    "admin@example.com",
				Password: "SecurePassword123!",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Missing email",
			request: models.RegisterRequest{
				Name:     "Admin User",
				Password: "SecurePassword123!",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Missing password",
			request: models.RegisterRequest{
				Name:  "Admin User",
				Email: "admin@example.com",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Password too short",
			request: models.RegisterRequest{
				Name:     "Admin User",
				Email:    "admin@example.com",
				Password: "short",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset database for each test
			db.Exec("DELETE FROM users")

			app := fiber.New()
			app.Post("/setup/admin", handler.CreateInitialAdmin)

			bodyJSON, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(http.MethodPost, "/setup/admin", bytes.NewReader(bodyJSON))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("Failed to execute request: %v", err)
			}

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func TestSetupHandler_CreateInitialAdmin_InvalidJSON(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-for-jwt-token-generation-minimum-32-chars")
	defer os.Unsetenv("JWT_SECRET")

	db := setupSetupTestDB(t)
	defer db.Close()

	userRepo := repository.NewUserRepository(db)
	authService := services.NewAuthService()
	handler := NewSetupHandler(userRepo, authService)

	app := fiber.New()
	app.Post("/setup/admin", handler.CreateInitialAdmin)

	req := httptest.NewRequest(http.MethodPost, "/setup/admin", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status %d for invalid JSON, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}
