package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/stretchr/testify/assert"

	_ "github.com/mattn/go-sqlite3"
)

// setupAdminTestDB creates an in-memory SQLite database for admin handler testing
func setupAdminTestDB(t *testing.T) *sql.DB {
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

// createAdminTestUser inserts a test user directly into the database
func createAdminTestUser(t *testing.T, db *sql.DB, email, name, role string) *models.User {
	hashedPassword := "hashedpassword"
	user := &models.User{
		ID:        uuid.New().String(),
		Email:     email,
		Name:      name,
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

func TestAdminHandler_GetAllUsers(t *testing.T) {
	db := setupAdminTestDB(t)
	defer db.Close()

	// Create test users
	createAdminTestUser(t, db, "admin@example.com", "Admin User", "admin")
	createAdminTestUser(t, db, "user1@example.com", "User One", "user")
	createAdminTestUser(t, db, "user2@example.com", "User Two", "user")

	userRepo := repository.NewUserRepository(db)
	handler := NewAdminHandler(userRepo)

	tests := []struct {
		name         string
		queryParams  string
		expectStatus int
		expectUsers  int // Expected minimum number of users
	}{
		{
			name:         "Get all users without filters",
			queryParams:  "",
			expectStatus: fiber.StatusOK,
			expectUsers:  3,
		},
		{
			name:         "Filter by role=admin",
			queryParams:  "?role=admin",
			expectStatus: fiber.StatusOK,
			expectUsers:  1,
		},
		{
			name:         "Filter by role=user",
			queryParams:  "?role=user",
			expectStatus: fiber.StatusOK,
			expectUsers:  2,
		},
		{
			name:         "Search by email",
			queryParams:  "?search=user1",
			expectStatus: fiber.StatusOK,
			expectUsers:  1,
		},
		{
			name:         "Pagination with limit",
			queryParams:  "?page=1&limit=2",
			expectStatus: fiber.StatusOK,
			expectUsers:  2,
		},
		{
			name:         "Invalid page defaults to 1",
			queryParams:  "?page=0",
			expectStatus: fiber.StatusOK,
			expectUsers:  3,
		},
		{
			name:         "Excessive limit capped at 100",
			queryParams:  "?limit=999",
			expectStatus: fiber.StatusOK,
			expectUsers:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/admin/users", handler.GetAllUsers)

			req := httptest.NewRequest("GET", "/admin/users"+tt.queryParams, nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectStatus, resp.StatusCode)

			if resp.StatusCode == fiber.StatusOK {
				var result map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&result)
				assert.NoError(t, err)
				assert.Contains(t, result, "users")
				assert.Contains(t, result, "total")
			}
		})
	}
}

func TestAdminHandler_GetUserStats(t *testing.T) {
	db := setupAdminTestDB(t)
	defer db.Close()

	// Create test users
	createAdminTestUser(t, db, "admin@example.com", "Admin User", "admin")
	createAdminTestUser(t, db, "user1@example.com", "User One", "user")
	createAdminTestUser(t, db, "user2@example.com", "User Two", "user")

	userRepo := repository.NewUserRepository(db)
	handler := NewAdminHandler(userRepo)

	app := fiber.New()
	app.Get("/admin/stats", handler.GetUserStats)

	req := httptest.NewRequest("GET", "/admin/stats", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var stats map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&stats)
	assert.NoError(t, err)
	assert.Contains(t, stats, "total")
	assert.Contains(t, stats, "admins")
	assert.Contains(t, stats, "users")
}

func TestAdminHandler_UpdateUserRole(t *testing.T) {
	db := setupAdminTestDB(t)
	defer db.Close()

	admin := createAdminTestUser(t, db, "admin@example.com", "Admin User", "admin")
	user := createAdminTestUser(t, db, "user@example.com", "Regular User", "user")

	userRepo := repository.NewUserRepository(db)
	handler := NewAdminHandler(userRepo)

	tests := []struct {
		name           string
		userID         string
		currentUserID  string
		requestBody    map[string]string
		expectedStatus int
	}{
		{
			name:          "Successfully promote user to admin",
			userID:        user.ID,
			currentUserID: admin.ID,
			requestBody: map[string]string{
				"role": "admin",
			},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:          "Successfully demote admin to user",
			userID:        user.ID,
			currentUserID: admin.ID,
			requestBody: map[string]string{
				"role": "user",
			},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:          "Cannot change own role",
			userID:        admin.ID,
			currentUserID: admin.ID,
			requestBody: map[string]string{
				"role": "user",
			},
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:          "Invalid role",
			userID:        user.ID,
			currentUserID: admin.ID,
			requestBody: map[string]string{
				"role": "superadmin",
			},
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:           "Missing role in request",
			userID:         user.ID,
			currentUserID:  admin.ID,
			requestBody:    map[string]string{},
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:          "Empty user ID",
			userID:        "",
			currentUserID: admin.ID,
			requestBody: map[string]string{
				"role": "admin",
			},
			expectedStatus: fiber.StatusNotFound, // Fiber returns 404 for empty param
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Put("/admin/users/:id/role", func(c *fiber.Ctx) error {
				c.Locals("user_id", tt.currentUserID)
				return handler.UpdateUserRole(c)
			})

			bodyJSON, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("PUT", "/admin/users/"+tt.userID+"/role", bytes.NewReader(bodyJSON))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

func TestAdminHandler_DeleteUser(t *testing.T) {
	db := setupAdminTestDB(t)
	defer db.Close()

	admin := createAdminTestUser(t, db, "admin@example.com", "Admin User", "admin")
	user := createAdminTestUser(t, db, "user@example.com", "Regular User", "user")

	userRepo := repository.NewUserRepository(db)
	handler := NewAdminHandler(userRepo)

	tests := []struct {
		name           string
		userID         string
		currentUserID  string
		expectedStatus int
	}{
		{
			name:           "Successfully delete user",
			userID:         user.ID,
			currentUserID:  admin.ID,
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "Cannot delete own account",
			userID:         admin.ID,
			currentUserID:  admin.ID,
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:           "Delete non-existent user",
			userID:         "non-existent-id",
			currentUserID:  admin.ID,
			expectedStatus: fiber.StatusNotFound,
		},
		{
			name:           "Empty user ID",
			userID:         "",
			currentUserID:  admin.ID,
			expectedStatus: fiber.StatusNotFound, // Fiber returns 404 for empty param
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Delete("/admin/users/:id", func(c *fiber.Ctx) error {
				c.Locals("user_id", tt.currentUserID)
				return handler.DeleteUser(c)
			})

			req := httptest.NewRequest("DELETE", "/admin/users/"+tt.userID, nil)

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}
