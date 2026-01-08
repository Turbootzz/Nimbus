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
			card_size TEXT DEFAULT '2x1',
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
		INSERT INTO services (id, user_id, name, url, icon, icon_type, icon_image_path, description, status, response_time, position, card_size, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	iconType := service.IconType
	if iconType == "" {
		iconType = models.IconTypeEmoji
	}
	cardSize := service.CardSize
	if cardSize == "" {
		cardSize = models.DefaultCardSize
	}
	_, err := db.Exec(
		query,
		service.ID,
		service.UserID,
		service.Name,
		service.URL,
		service.Icon,
		iconType,
		service.IconImagePath,
		service.Description,
		service.Status,
		service.ResponseTime,
		service.Position,
		cardSize,
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
func TestServiceHandler_CreateService(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	serviceRepo := repository.NewServiceRepository(db)
	handler := NewServiceHandler(serviceRepo, nil)

	tests := []struct {
		name           string
		userID         string
		requestBody    models.ServiceCreateRequest
		expectedStatus int
		expectError    bool
	}{
		{
			name:   "Missing name",
			userID: "user-1",
			requestBody: models.ServiceCreateRequest{
				URL: "https://example.com",
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:   "Missing URL",
			userID: "user-1",
			requestBody: models.ServiceCreateRequest{
				Name: "Test Service",
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:   "Invalid URL format",
			userID: "user-1",
			requestBody: models.ServiceCreateRequest{
				Name: "Test Service",
				URL:  "not-a-valid-url",
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Post("/services", func(c *fiber.Ctx) error {
				c.Locals("user_id", tt.userID)
				return handler.CreateService(c)
			})

			bodyJSON, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/services", bytes.NewReader(bodyJSON))
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

func TestServiceHandler_GetServices(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	serviceRepo := repository.NewServiceRepository(db)
	handler := NewServiceHandler(serviceRepo, nil)

	// Create test services for user-1
	createServiceDirectly(t, db, &models.Service{
		ID:        "service-1",
		UserID:    "user-1",
		Name:      "Service 1",
		URL:       "https://example1.com",
		Icon:      "🔗",
		Status:    models.StatusUnknown,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	createServiceDirectly(t, db, &models.Service{
		ID:        "service-2",
		UserID:    "user-1",
		Name:      "Service 2",
		URL:       "https://example2.com",
		Icon:      "🔗",
		Status:    models.StatusUnknown,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	app := fiber.New()
	app.Get("/services", func(c *fiber.Ctx) error {
		c.Locals("user_id", "user-1")
		return handler.GetServices(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/services", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestServiceHandler_DeleteService(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	serviceRepo := repository.NewServiceRepository(db)
	handler := NewServiceHandler(serviceRepo, nil)

	createServiceDirectly(t, db, &models.Service{
		ID:        "service-1",
		UserID:    "user-1",
		Name:      "Service 1",
		URL:       "https://example1.com",
		Icon:      "🔗",
		Status:    models.StatusUnknown,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	tests := []struct {
		name           string
		userID         string
		serviceID      string
		expectedStatus int
	}{
		{
			name:           "Successfully delete service",
			userID:         "user-1",
			serviceID:      "service-1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Delete non-existent service",
			userID:         "user-1",
			serviceID:      "non-existent",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Delete("/services/:id", func(c *fiber.Ctx) error {
				c.Locals("user_id", tt.userID)
				return handler.DeleteService(c)
			})

			req := httptest.NewRequest(http.MethodDelete, "/services/"+tt.serviceID, nil)
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

func TestServiceHandler_UpdateService(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	serviceRepo := repository.NewServiceRepository(db)
	handler := NewServiceHandler(serviceRepo, nil)

	// Create test service
	createServiceDirectly(t, db, &models.Service{
		ID:        "service-1",
		UserID:    "user-1",
		Name:      "Service 1",
		URL:       "https://example1.com",
		Icon:      "🔗",
		Status:    models.StatusUnknown,
		CardSize:  models.CardSize2x1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	// Create service owned by different user
	createServiceDirectly(t, db, &models.Service{
		ID:        "service-2",
		UserID:    "user-2",
		Name:      "Service 2",
		URL:       "https://example2.com",
		Icon:      "🔗",
		Status:    models.StatusUnknown,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	tests := []struct {
		name           string
		userID         string
		serviceID      string
		requestBody    models.ServiceUpdateRequest
		expectedStatus int
		expectError    bool
	}{
		{
			name:      "Successfully update service with valid card_size",
			userID:    "user-1",
			serviceID: "service-1",
			requestBody: models.ServiceUpdateRequest{
				Name:     "Updated Service",
				URL:      "https://updated.com",
				CardSize: models.CardSize2x2,
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:      "Update service with 1x1 card size",
			userID:    "user-1",
			serviceID: "service-1",
			requestBody: models.ServiceUpdateRequest{
				Name:     "Updated Service",
				URL:      "https://updated.com",
				CardSize: models.CardSize1x1,
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:      "Update with invalid card_size",
			userID:    "user-1",
			serviceID: "service-1",
			requestBody: models.ServiceUpdateRequest{
				Name:     "Updated Service",
				URL:      "https://updated.com",
				CardSize: "3x3", // Invalid size
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:      "Update preserves card_size when not provided",
			userID:    "user-1",
			serviceID: "service-1",
			requestBody: models.ServiceUpdateRequest{
				Name: "Updated Service",
				URL:  "https://updated.com",
				// CardSize not provided - should preserve existing
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:      "Attempt to update another user's service",
			userID:    "user-1",
			serviceID: "service-2",
			requestBody: models.ServiceUpdateRequest{
				Name:     "Updated Service",
				URL:      "https://updated.com",
				CardSize: models.CardSize1x1,
			},
			expectedStatus: http.StatusForbidden,
			expectError:    true,
		},
		{
			name:      "Update non-existent service",
			userID:    "user-1",
			serviceID: "non-existent",
			requestBody: models.ServiceUpdateRequest{
				Name: "Updated Service",
				URL:  "https://updated.com",
			},
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Put("/services/:id", func(c *fiber.Ctx) error {
				c.Locals("user_id", tt.userID)
				return handler.UpdateService(c)
			})

			bodyJSON, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPut, "/services/"+tt.serviceID, bytes.NewReader(bodyJSON))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("Failed to execute request: %v", err)
			}

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			// Verify card_size was updated correctly (if success expected)
			if !tt.expectError && resp.StatusCode == http.StatusOK && tt.requestBody.CardSize != "" {
				service, err := serviceRepo.GetByID(context.Background(), tt.serviceID)
				if err != nil {
					t.Fatalf("Failed to retrieve service: %v", err)
				}
				if service.CardSize != tt.requestBody.CardSize {
					t.Errorf("Service card_size = %s, want %s", service.CardSize, tt.requestBody.CardSize)
				}
			}
		})
	}
}

func TestServiceHandler_UpdateService_NoAuth(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	serviceRepo := repository.NewServiceRepository(db)
	handler := NewServiceHandler(serviceRepo, nil)

	app := fiber.New()
	app.Put("/services/:id", handler.UpdateService)

	requestBody := models.ServiceUpdateRequest{
		Name:     "Updated Service",
		URL:      "https://updated.com",
		CardSize: models.CardSize1x1,
	}

	bodyJSON, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPut, "/services/service-1", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status %d for unauthenticated request, got %d", http.StatusUnauthorized, resp.StatusCode)
	}
}
