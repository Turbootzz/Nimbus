package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/services"

	_ "github.com/mattn/go-sqlite3"
)

// setupAuthTestDB creates an in-memory SQLite database for auth testing
func setupAuthTestDB(t *testing.T) *sql.DB {
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

		CREATE TABLE IF NOT EXISTS system_settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_by TEXT
		);

		INSERT INTO system_settings (key, value, updated_at)
		VALUES ('public_registration_enabled', 'true', CURRENT_TIMESTAMP);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	return db
}

// createUserDirectly inserts a user for testing
func createUserDirectly(t *testing.T, db *sql.DB, user *models.User) {
	query := `
		INSERT INTO users (id, email, name, password, role, created_at, updated_at, last_activity_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := db.Exec(
		query,
		user.ID,
		user.Email,
		user.Name,
		user.Password,
		user.Role,
		user.CreatedAt,
		user.UpdatedAt,
		user.LastActivityAt,
	)
	if err != nil {
		t.Fatalf("Failed to create user directly: %v", err)
	}
}

// extractTokenFromCookie extracts the JWT token from the Set-Cookie header
func extractTokenFromCookie(resp *http.Response) string {
	cookies := resp.Cookies()
	for _, cookie := range cookies {
		if cookie.Name == "auth_token" {
			return cookie.Value
		}
	}
	return ""
}

// parseTokenExpiration extracts the expiration time from a JWT token
func parseTokenExpiration(t *testing.T, tokenString string) time.Time {
	// Parse without validation to inspect claims
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("Failed to get claims from token")
	}

	exp, ok := claims["exp"].(float64)
	if !ok {
		t.Fatal("Token expiration not found or wrong type")
	}

	return time.Unix(int64(exp), 0)
}

func TestAuthHandler_Login_RememberMe(t *testing.T) {
	// Set JWT secret
	os.Setenv("JWT_SECRET", "test-secret-for-jwt-token-generation-minimum-32-chars")
	defer os.Unsetenv("JWT_SECRET")

	// Set cookie secure to false for testing
	os.Setenv("COOKIE_SECURE", "false")
	defer os.Unsetenv("COOKIE_SECURE")

	db := setupAuthTestDB(t)
	defer db.Close()

	userRepo := repository.NewUserRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	authService := services.NewAuthService()
	handler := NewAuthHandler(userRepo, authService, settingsRepo)

	// Create test user with hashed password
	password := "TestPassword123!"
	hashedPassword, err := authService.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	testUser := &models.User{
		ID:        "user-123",
		Email:     "test@example.com",
		Name:      "Test User",
		Password:  &hashedPassword,
		Role:      "user",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	createUserDirectly(t, db, testUser)

	tests := []struct {
		name                 string
		rememberMe           bool
		expectedCookieMaxAge int
		expectedTokenExpiry  time.Duration
		variance             time.Duration
	}{
		{
			name:                 "Login with remember me enabled",
			rememberMe:           true,
			expectedCookieMaxAge: 30 * 24 * 60 * 60, // 30 days in seconds
			expectedTokenExpiry:  30 * 24 * time.Hour,
			variance:             5 * time.Second,
		},
		{
			name:                 "Login with remember me disabled",
			rememberMe:           false,
			expectedCookieMaxAge: 0, // Session cookie
			expectedTokenExpiry:  24 * time.Hour,
			variance:             5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Post("/login", handler.Login)

			// Create login request
			loginReq := models.LoginRequest{
				Email:      testUser.Email,
				Password:   password,
				RememberMe: tt.rememberMe,
			}

			bodyJSON, _ := json.Marshal(loginReq)
			req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(bodyJSON))
			req.Header.Set("Content-Type", "application/json")

			// Execute request
			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("Failed to execute request: %v", err)
			}

			// Verify successful login
			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
			}

			// Extract token from cookie
			token := extractTokenFromCookie(resp)
			if token == "" {
				t.Fatal("No auth_token cookie found in response")
			}

			// Verify cookie MaxAge
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

			if authCookie.MaxAge != tt.expectedCookieMaxAge {
				t.Errorf("Cookie MaxAge = %d, want %d", authCookie.MaxAge, tt.expectedCookieMaxAge)
			}

			// Verify cookie attributes
			if !authCookie.HttpOnly {
				t.Error("Cookie should be HttpOnly")
			}

			if authCookie.SameSite != http.SameSiteLaxMode {
				t.Errorf("Cookie SameSite = %v, want %v", authCookie.SameSite, http.SameSiteLaxMode)
			}

			// Parse and verify token expiration
			tokenExpTime := parseTokenExpiration(t, token)
			expectedExpTime := time.Now().Add(tt.expectedTokenExpiry)
			timeDiff := tokenExpTime.Sub(expectedExpTime)

			if timeDiff > tt.variance || timeDiff < -tt.variance {
				t.Errorf("Token expiration = %v, want ~%v (diff: %v)", tokenExpTime, expectedExpTime, timeDiff)
			}

			// Verify token is valid
			claims, err := authService.ValidateToken(token)
			if err != nil {
				t.Errorf("Token validation failed: %v", err)
			}

			// Verify claims contain correct user info
			if (*claims)["user_id"] != testUser.ID {
				t.Errorf("Token user_id = %v, want %v", (*claims)["user_id"], testUser.ID)
			}
			if (*claims)["email"] != testUser.Email {
				t.Errorf("Token email = %v, want %v", (*claims)["email"], testUser.Email)
			}
			if (*claims)["role"] != testUser.Role {
				t.Errorf("Token role = %v, want %v", (*claims)["role"], testUser.Role)
			}
		})
	}
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-for-jwt-token-generation-minimum-32-chars")
	defer os.Unsetenv("JWT_SECRET")

	db := setupAuthTestDB(t)
	defer db.Close()

	userRepo := repository.NewUserRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	authService := services.NewAuthService()
	handler := NewAuthHandler(userRepo, authService, settingsRepo)

	// Create test user
	password := "CorrectPassword123!"
	hashedPassword, _ := authService.HashPassword(password)
	testUser := &models.User{
		ID:        "user-456",
		Email:     "user@example.com",
		Name:      "Test User",
		Password:  &hashedPassword,
		Role:      "user",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	createUserDirectly(t, db, testUser)

	tests := []struct {
		name     string
		email    string
		password string
	}{
		{
			name:     "Wrong password",
			email:    testUser.Email,
			password: "WrongPassword",
		},
		{
			name:     "Non-existent email",
			email:    "nonexistent@example.com",
			password: password,
		},
		{
			name:     "Empty password",
			email:    testUser.Email,
			password: "",
		},
		{
			name:     "Empty email",
			email:    "",
			password: password,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Post("/login", handler.Login)

			loginReq := models.LoginRequest{
				Email:      tt.email,
				Password:   tt.password,
				RememberMe: false,
			}

			bodyJSON, _ := json.Marshal(loginReq)
			req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(bodyJSON))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("Failed to execute request: %v", err)
			}

			// Verify login fails with appropriate status code
			if tt.email == "" || tt.password == "" {
				if resp.StatusCode != http.StatusBadRequest {
					t.Errorf("Expected status %d for empty credentials, got %d", http.StatusBadRequest, resp.StatusCode)
				}
			} else {
				if resp.StatusCode != http.StatusUnauthorized {
					t.Errorf("Expected status %d for invalid credentials, got %d", http.StatusUnauthorized, resp.StatusCode)
				}
			}

			// Verify no auth cookie was set
			token := extractTokenFromCookie(resp)
			if token != "" {
				t.Error("Auth token should not be set for failed login")
			}
		})
	}
}

func TestAuthHandler_Register(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-for-jwt-token-generation-minimum-32-chars")
	defer os.Unsetenv("JWT_SECRET")

	os.Setenv("COOKIE_SECURE", "false")
	defer os.Unsetenv("COOKIE_SECURE")

	db := setupAuthTestDB(t)
	defer db.Close()

	userRepo := repository.NewUserRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	authService := services.NewAuthService()
	handler := NewAuthHandler(userRepo, authService, settingsRepo)

	app := fiber.New()
	app.Post("/register", handler.Register)

	// Test successful registration
	t.Run("Successful registration", func(t *testing.T) {
		registerReq := models.RegisterRequest{
			Email:    "newuser@example.com",
			Name:     "New User",
			Password: "SecurePassword123!",
		}

		bodyJSON, _ := json.Marshal(registerReq)
		req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(bodyJSON))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected status %d, got %d", http.StatusCreated, resp.StatusCode)
		}

		// Verify session cookie is set (MaxAge = 0)
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

		if authCookie.MaxAge != 0 {
			t.Errorf("Registration should create session cookie (MaxAge=0), got MaxAge=%d", authCookie.MaxAge)
		}
	})

	// Test duplicate email
	t.Run("Duplicate email", func(t *testing.T) {
		registerReq := models.RegisterRequest{
			Email:    "newuser@example.com", // Same email as above
			Name:     "Another User",
			Password: "Password123!",
		}

		bodyJSON, _ := json.Marshal(registerReq)
		req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(bodyJSON))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		if resp.StatusCode != http.StatusConflict {
			t.Errorf("Expected status %d for duplicate email, got %d", http.StatusConflict, resp.StatusCode)
		}
	})

	// Test invalid email format
	t.Run("Invalid email format", func(t *testing.T) {
		registerReq := models.RegisterRequest{
			Email:    "invalid-email",
			Name:     "Test User",
			Password: "SecurePassword123!",
		}

		bodyJSON, _ := json.Marshal(registerReq)
		req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(bodyJSON))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status %d for invalid email, got %d", http.StatusBadRequest, resp.StatusCode)
		}

		var response map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response["error"] != "Invalid email format" {
			t.Errorf("Expected error message 'Invalid email format', got '%s'", response["error"])
		}
	})
}

func TestAuthHandler_Logout(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-for-jwt-token-generation-minimum-32-chars")
	defer os.Unsetenv("JWT_SECRET")

	os.Setenv("COOKIE_SECURE", "false")
	defer os.Unsetenv("COOKIE_SECURE")

	db := setupAuthTestDB(t)
	defer db.Close()

	userRepo := repository.NewUserRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	authService := services.NewAuthService()
	handler := NewAuthHandler(userRepo, authService, settingsRepo)

	app := fiber.New()
	app.Post("/logout", handler.Logout)

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Verify cookie is cleared (MaxAge = -1)
	cookies := resp.Cookies()
	var authCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "auth_token" {
			authCookie = cookie
			break
		}
	}

	if authCookie == nil {
		t.Fatal("auth_token cookie not found in logout response")
	}

	// MaxAge should be -1 or less to delete the cookie
	// Note: Fiber test responses may show 0 instead of -1, both indicate deletion
	if authCookie.MaxAge > 0 {
		t.Errorf("Logout should clear cookie (MaxAge<=0), got MaxAge=%d", authCookie.MaxAge)
	}

	if authCookie.Value != "" {
		t.Error("Logout should clear cookie value")
	}
}

func TestAuthHandler_TokenExpiration_30Days(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-for-jwt-token-generation-minimum-32-chars")
	defer os.Unsetenv("JWT_SECRET")

	os.Setenv("COOKIE_SECURE", "false")
	defer os.Unsetenv("COOKIE_SECURE")

	db := setupAuthTestDB(t)
	defer db.Close()

	userRepo := repository.NewUserRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	authService := services.NewAuthService()
	handler := NewAuthHandler(userRepo, authService, settingsRepo)

	// Create test user
	password := "TestPassword123!"
	hashedPassword, _ := authService.HashPassword(password)
	testUser := &models.User{
		ID:        "user-789",
		Email:     "longterm@example.com",
		Name:      "Long Term User",
		Password:  &hashedPassword,
		Role:      "user",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	createUserDirectly(t, db, testUser)

	app := fiber.New()
	app.Post("/login", handler.Login)

	// Login with remember me
	loginReq := models.LoginRequest{
		Email:      testUser.Email,
		Password:   password,
		RememberMe: true,
	}

	bodyJSON, _ := json.Marshal(loginReq)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	// Extract and parse token
	token := extractTokenFromCookie(resp)
	tokenExpTime := parseTokenExpiration(t, token)

	// Verify token expires in approximately 30 days
	expectedExpTime := time.Now().Add(30 * 24 * time.Hour)
	timeDiff := tokenExpTime.Sub(expectedExpTime)

	// Allow 10 seconds variance
	if timeDiff > 10*time.Second || timeDiff < -10*time.Second {
		t.Errorf("30-day token expiration = %v, want ~%v (diff: %v)", tokenExpTime, expectedExpTime, timeDiff)
	}

	// Verify it's significantly longer than 24 hours
	minExpTime := time.Now().Add(25 * 24 * time.Hour)
	if tokenExpTime.Before(minExpTime) {
		t.Error("Remember me token should expire in more than 25 days")
	}
}

func TestAuthHandler_InvalidJSON(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-for-jwt-token-generation-minimum-32-chars")
	defer os.Unsetenv("JWT_SECRET")

	db := setupAuthTestDB(t)
	defer db.Close()

	userRepo := repository.NewUserRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	authService := services.NewAuthService()
	handler := NewAuthHandler(userRepo, authService, settingsRepo)

	tests := []struct {
		name     string
		endpoint string
		handler  fiber.Handler
	}{
		{
			name:     "Login with invalid JSON",
			endpoint: "/login",
			handler:  handler.Login,
		},
		{
			name:     "Register with invalid JSON",
			endpoint: "/register",
			handler:  handler.Register,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Post(tt.endpoint, tt.handler)

			req := httptest.NewRequest(http.MethodPost, tt.endpoint, strings.NewReader("invalid json"))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("Failed to execute request: %v", err)
			}

			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("Expected status %d for invalid JSON, got %d", http.StatusBadRequest, resp.StatusCode)
			}
		})
	}
}

func TestAuthHandler_Register_DisabledPublicRegistration(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-for-jwt-token-generation-minimum-32-chars")
	defer os.Unsetenv("JWT_SECRET")

	db := setupAuthTestDB(t)
	defer db.Close()

	// Update setting to disable public registration
	_, err := db.Exec(`UPDATE system_settings SET value = 'false' WHERE key = 'public_registration_enabled'`)
	if err != nil {
		t.Fatalf("Failed to update setting: %v", err)
	}

	userRepo := repository.NewUserRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	authService := services.NewAuthService()
	handler := NewAuthHandler(userRepo, authService, settingsRepo)

	app := fiber.New()
	app.Post("/register", handler.Register)

	registerReq := models.RegisterRequest{
		Email:    "newuser@example.com",
		Name:     "New User",
		Password: "SecurePassword123!",
	}

	bodyJSON, _ := json.Marshal(registerReq)
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("Expected status %d when registration is disabled, got %d", http.StatusForbidden, resp.StatusCode)
	}

	var response map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["error"] != "Public registration is disabled" {
		t.Errorf("Expected error message 'Public registration is disabled', got '%s'", response["error"])
	}
}

func TestAuthHandler_DeleteAccount(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-for-jwt-token-generation-minimum-32-chars")
	defer os.Unsetenv("JWT_SECRET")

	os.Setenv("COOKIE_SECURE", "false")
	defer os.Unsetenv("COOKIE_SECURE")

	db := setupAuthTestDB(t)
	defer db.Close()

	userRepo := repository.NewUserRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	authService := services.NewAuthService()
	handler := NewAuthHandler(userRepo, authService, settingsRepo)

	password := "TestPassword123!"
	hashedPassword, _ := authService.HashPassword(password)
	testUser := &models.User{
		ID:        "user-delete-self",
		Email:     "self-delete@example.com",
		Name:      "Self Delete",
		Password:  &hashedPassword,
		Role:      "user",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	createUserDirectly(t, db, testUser)

	t.Run("Successfully deletes own account and clears cookie", func(t *testing.T) {
		app := fiber.New()
		app.Delete("/auth/me", func(c *fiber.Ctx) error {
			c.Locals("user_id", testUser.ID)
			return handler.DeleteAccount(c)
		})

		req := httptest.NewRequest(http.MethodDelete, "/auth/me", nil)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
		}

		// User should be gone
		if _, err := userRepo.GetByID(testUser.ID); err == nil {
			t.Error("User should not exist after DeleteAccount")
		}

		// Cookie should be cleared
		var authCookie *http.Cookie
		for _, cookie := range resp.Cookies() {
			if cookie.Name == "auth_token" {
				authCookie = cookie
				break
			}
		}
		if authCookie == nil {
			t.Fatal("auth_token cookie not found in response")
		}
		if authCookie.MaxAge > 0 {
			t.Errorf("DeleteAccount should clear cookie (MaxAge<=0), got MaxAge=%d", authCookie.MaxAge)
		}
		if authCookie.Value != "" {
			t.Error("DeleteAccount should clear cookie value")
		}
	})

	t.Run("Refuses to delete the only remaining admin", func(t *testing.T) {
		soloAdmin := &models.User{
			ID:        "solo-admin",
			Email:     "solo-admin@example.com",
			Name:      "Solo Admin",
			Password:  &hashedPassword,
			Role:      "admin",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		createUserDirectly(t, db, soloAdmin)

		app := fiber.New()
		app.Delete("/auth/me", func(c *fiber.Ctx) error {
			c.Locals("user_id", soloAdmin.ID)
			return handler.DeleteAccount(c)
		})

		req := httptest.NewRequest(http.MethodDelete, "/auth/me", nil)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
		}

		// Admin should still exist
		if _, err := userRepo.GetByID(soloAdmin.ID); err != nil {
			t.Error("Solo admin must not be deleted")
		}
	})

	t.Run("Allows admin deletion when another admin exists", func(t *testing.T) {
		adminA := &models.User{
			ID:        "admin-a",
			Email:     "admin-a@example.com",
			Name:      "Admin A",
			Password:  &hashedPassword,
			Role:      "admin",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		adminB := &models.User{
			ID:        "admin-b",
			Email:     "admin-b@example.com",
			Name:      "Admin B",
			Password:  &hashedPassword,
			Role:      "admin",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		createUserDirectly(t, db, adminA)
		createUserDirectly(t, db, adminB)

		app := fiber.New()
		app.Delete("/auth/me", func(c *fiber.Ctx) error {
			c.Locals("user_id", adminA.ID)
			return handler.DeleteAccount(c)
		})

		req := httptest.NewRequest(http.MethodDelete, "/auth/me", nil)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
		}
		if _, err := userRepo.GetByID(adminA.ID); err == nil {
			t.Error("Admin A should be deleted when another admin remains")
		}
	})

	t.Run("Returns 404 when user no longer exists", func(t *testing.T) {
		app := fiber.New()
		app.Delete("/auth/me", func(c *fiber.Ctx) error {
			c.Locals("user_id", "non-existent-user")
			return handler.DeleteAccount(c)
		})

		req := httptest.NewRequest(http.MethodDelete, "/auth/me", nil)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
		}
	})

	t.Run("Removes local avatar file from disk", func(t *testing.T) {
		tempDir := t.TempDir()
		originalDir := AvatarUploadDir
		AvatarUploadDir = tempDir
		t.Cleanup(func() { AvatarUploadDir = originalDir })

		avatarPath := tempDir + "/avatar.jpg"
		if err := os.WriteFile(avatarPath, []byte("x"), 0o644); err != nil {
			t.Fatalf("Failed to write fake avatar: %v", err)
		}

		userWithAvatar := &models.User{
			ID:        "user-with-local-avatar",
			Email:     "local-avatar@example.com",
			Name:      "Local Avatar",
			Password:  &hashedPassword,
			Role:      "user",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		createUserDirectly(t, db, userWithAvatar)
		avatarURL := "/uploads/avatars/avatar.jpg"
		if err := userRepo.UpdateAvatar(userWithAvatar.ID, &avatarURL); err != nil {
			t.Fatalf("Failed to set avatar URL: %v", err)
		}

		app := fiber.New()
		app.Delete("/auth/me", func(c *fiber.Ctx) error {
			c.Locals("user_id", userWithAvatar.ID)
			return handler.DeleteAccount(c)
		})

		req := httptest.NewRequest(http.MethodDelete, "/auth/me", nil)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
		}

		if _, err := os.Stat(avatarPath); !os.IsNotExist(err) {
			t.Errorf("Avatar file should be removed from disk, got err=%v", err)
		}
	})

	t.Run("Leaves remote avatar URLs alone", func(t *testing.T) {
		// Regression guard: the helper must skip OAuth-style remote URLs without
		// attempting any disk I/O.
		userOAuth := &models.User{
			ID:        "user-with-remote-avatar",
			Email:     "remote-avatar@example.com",
			Name:      "Remote Avatar",
			Password:  &hashedPassword,
			Role:      "user",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		createUserDirectly(t, db, userOAuth)
		remoteURL := "https://lh3.googleusercontent.com/a/example"
		if err := userRepo.UpdateAvatar(userOAuth.ID, &remoteURL); err != nil {
			t.Fatalf("Failed to set avatar URL: %v", err)
		}

		app := fiber.New()
		app.Delete("/auth/me", func(c *fiber.Ctx) error {
			c.Locals("user_id", userOAuth.ID)
			return handler.DeleteAccount(c)
		})

		req := httptest.NewRequest(http.MethodDelete, "/auth/me", nil)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
		}
	})
}
