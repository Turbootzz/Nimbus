package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"

	_ "github.com/mattn/go-sqlite3"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	schema := `
		CREATE TABLE IF NOT EXISTS services (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			name TEXT NOT NULL,
			url TEXT NOT NULL,
			icon TEXT,
			icon_type TEXT DEFAULT 'emoji',
			icon_image_path TEXT DEFAULT '',
			description TEXT,
			status TEXT NOT NULL,
			response_time INTEGER,
			position INTEGER DEFAULT 0,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	return db
}

// createServiceDirectly inserts a service for testing
func createServiceDirectly(t *testing.T, db *sql.DB, service *models.Service) {
	query := `
		INSERT INTO services (id, user_id, name, url, icon, description, status, response_time, position, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := db.Exec(
		query,
		service.ID,
		service.UserID,
		service.Name,
		service.URL,
		service.Icon,
		service.Description,
		service.Status,
		service.ResponseTime,
		service.Position,
		service.CreatedAt,
		service.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("Failed to create service directly: %v", err)
	}
}

func TestServiceHandler_ReorderServices(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	serviceRepo := repository.NewServiceRepository(db)
	handler := NewServiceHandler(serviceRepo, nil)

	// Create test services
	services := []*models.Service{
		{
			ID:        "service-1",
			UserID:    "user-1",
			Name:      "Service 1",
			URL:       "https://example1.com",
			Icon:      "🔗",
			Status:    models.StatusUnknown,
			Position:  0,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "service-2",
			UserID:    "user-1",
			Name:      "Service 2",
			URL:       "https://example2.com",
			Icon:      "🔗",
			Status:    models.StatusUnknown,
			Position:  1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "service-3",
			UserID:    "user-2",
			Name:      "Service 3",
			URL:       "https://example3.com",
			Icon:      "🔗",
			Status:    models.StatusUnknown,
			Position:  0,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, s := range services {
		createServiceDirectly(t, db, s)
	}

	tests := []struct {
		name           string
		userID         string
		requestBody    models.ServiceReorderRequest
		expectedStatus int
		expectError    bool
	}{
		{
			name:   "Successfully reorder services",
			userID: "user-1",
			requestBody: models.ServiceReorderRequest{
				Services: []models.ServicePosition{
					{ID: "service-1", Position: 1},
					{ID: "service-2", Position: 0},
				},
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:   "Reorder single service",
			userID: "user-1",
			requestBody: models.ServiceReorderRequest{
				Services: []models.ServicePosition{
					{ID: "service-1", Position: 5},
				},
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:   "Attempt to reorder another user's service",
			userID: "user-1",
			requestBody: models.ServiceReorderRequest{
				Services: []models.ServicePosition{
					{ID: "service-3", Position: 10},
				},
			},
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
		{
			name:   "Empty service ID",
			userID: "user-1",
			requestBody: models.ServiceReorderRequest{
				Services: []models.ServicePosition{
					{ID: "", Position: 0},
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:   "Negative position",
			userID: "user-1",
			requestBody: models.ServiceReorderRequest{
				Services: []models.ServicePosition{
					{ID: "service-1", Position: -1},
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:   "Empty services array",
			userID: "user-1",
			requestBody: models.ServiceReorderRequest{
				Services: []models.ServicePosition{},
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:   "Non-existent service",
			userID: "user-1",
			requestBody: models.ServiceReorderRequest{
				Services: []models.ServicePosition{
					{ID: "non-existent", Position: 0},
				},
			},
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()

			// Setup route with auth middleware mock
			app.Put("/services/reorder", func(c *fiber.Ctx) error {
				// Mock authentication by setting user_id in locals
				c.Locals("user_id", tt.userID)
				return handler.ReorderServices(c)
			})

			// Create request
			bodyJSON, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPut, "/services/reorder", bytes.NewReader(bodyJSON))
			req.Header.Set("Content-Type", "application/json")

			// Execute request
			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("Failed to execute request: %v", err)
			}

			// Check status code
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			// Verify positions were updated correctly (if success expected)
			if !tt.expectError && resp.StatusCode == http.StatusOK {
				ctx := context.Background()
				for _, sp := range tt.requestBody.Services {
					service, err := serviceRepo.GetByID(ctx, sp.ID)
					if err != nil {
						t.Fatalf("Failed to retrieve service %s: %v", sp.ID, err)
					}
					if service.Position != sp.Position {
						t.Errorf("Service %s position = %d, want %d", sp.ID, service.Position, sp.Position)
					}
				}
			}
		})
	}
}

func TestServiceHandler_ReorderServices_NoAuth(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	serviceRepo := repository.NewServiceRepository(db)
	handler := NewServiceHandler(serviceRepo, nil)

	app := fiber.New()

	// Setup route WITHOUT setting user_id in locals (no auth)
	app.Put("/services/reorder", handler.ReorderServices)

	requestBody := models.ServiceReorderRequest{
		Services: []models.ServicePosition{
			{ID: "service-1", Position: 0},
		},
	}

	bodyJSON, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPut, "/services/reorder", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status %d for unauthenticated request, got %d", http.StatusUnauthorized, resp.StatusCode)
	}
}

func TestServiceHandler_ReorderServices_InvalidJSON(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	serviceRepo := repository.NewServiceRepository(db)
	handler := NewServiceHandler(serviceRepo, nil)

	app := fiber.New()

	app.Put("/services/reorder", func(c *fiber.Ctx) error {
		c.Locals("user_id", "user-1")
		return handler.ReorderServices(c)
	})

	// Invalid JSON
	req := httptest.NewRequest(http.MethodPut, "/services/reorder", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status %d for invalid JSON, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}
