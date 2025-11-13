package handlers

// Comprehensive tests for Prometheus metrics endpoint with API key authentication
//
// Test Coverage:
// - API key authentication (Authorization: Bearer header)
// - Alternative API key header (X-API-Key)
// - JWT authentication fallback
// - User isolation (users can only see their own services)
// - Admin access (admins can see any user's metrics)
// - Invalid/missing API keys
// - Empty service lists
// - Prometheus output format validation

import (
	"database/sql"
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

// setupMetricsTestDB creates an in-memory SQLite database for metrics testing
func setupMetricsTestDB(t *testing.T) *sql.DB {
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

		CREATE TABLE IF NOT EXISTS service_status_logs (
			id TEXT PRIMARY KEY,
			service_id TEXT NOT NULL,
			status TEXT NOT NULL,
			response_time INTEGER,
			checked_at TIMESTAMP NOT NULL,
			FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE
		);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create test tables: %v", err)
	}

	return db
}

// createTestService inserts a test service
func createTestService(t *testing.T, db *sql.DB, service *models.Service) {
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
		t.Fatalf("Failed to create test service: %v", err)
	}
}

func TestGetUserPrometheusMetrics_WithAPIKey(t *testing.T) {
	db := setupMetricsTestDB(t)
	defer db.Close()

	serviceRepo := repository.NewServiceRepository(db)
	statusLogRepo := repository.NewStatusLogRepository(db)
	metricsService := services.NewMetricsService(statusLogRepo, serviceRepo)
	handler := NewMetricsHandler(metricsService, serviceRepo)

	// Set API key in environment
	testAPIKey := "test-api-key-12345"
	os.Setenv("PROMETHEUS_API_KEY", testAPIKey)
	defer os.Unsetenv("PROMETHEUS_API_KEY")

	// Create test services for user-1
	responseTime := 100
	createTestService(t, db, &models.Service{
		ID:           "service-1",
		UserID:       "user-1",
		Name:         "Test Service 1",
		URL:          "https://example1.com",
		Icon:         "🔗",
		Status:       models.StatusOnline,
		ResponseTime: &responseTime,
		Position:     0,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	})

	createTestService(t, db, &models.Service{
		ID:        "service-2",
		UserID:    "user-1",
		Name:      "Test Service 2",
		URL:       "https://example2.com",
		Icon:      "🔗",
		Status:    models.StatusOffline,
		Position:  1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	// Create test service for user-2 (should not appear in user-1's metrics)
	createTestService(t, db, &models.Service{
		ID:        "service-3",
		UserID:    "user-2",
		Name:      "User 2 Service",
		URL:       "https://example3.com",
		Icon:      "🔗",
		Status:    models.StatusOnline,
		Position:  0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	app := fiber.New()
	app.Get("/api/v1/prometheus/metrics/user/:userID", handler.GetUserPrometheusMetrics)

	tests := []struct {
		name           string
		userID         string
		apiKey         string
		expectedStatus int
		checkBody      func(t *testing.T, body string)
	}{
		{
			name:           "Valid API key - should return user's metrics",
			userID:         "user-1",
			apiKey:         testAPIKey,
			expectedStatus: 200,
			checkBody: func(t *testing.T, body string) {
				// Check for Prometheus format headers
				if !containsString(body, "# HELP nimbus_service_up") {
					t.Error("Response should contain Prometheus HELP comment")
				}
				if !containsString(body, "# TYPE nimbus_service_up gauge") {
					t.Error("Response should contain Prometheus TYPE comment")
				}
				// Check for user-1's services
				if !containsString(body, "Test Service 1") {
					t.Error("Response should contain Test Service 1")
				}
				if !containsString(body, "Test Service 2") {
					t.Error("Response should contain Test Service 2")
				}
				// Ensure user-2's service is NOT included
				if containsString(body, "User 2 Service") {
					t.Error("Response should NOT contain services from other users")
				}
			},
		},
		{
			name:           "Invalid API key - should return 401",
			userID:         "user-1",
			apiKey:         "wrong-api-key",
			expectedStatus: 401,
			checkBody: func(t *testing.T, body string) {
				if !containsString(body, "Unauthorized") {
					t.Error("Response should contain Unauthorized message")
				}
			},
		},
		{
			name:           "Missing API key - should return 401",
			userID:         "user-1",
			apiKey:         "",
			expectedStatus: 401,
			checkBody: func(t *testing.T, body string) {
				if !containsString(body, "Unauthorized") {
					t.Error("Response should contain Unauthorized message")
				}
			},
		},
		{
			name:           "Valid API key for different user - should return their metrics",
			userID:         "user-2",
			apiKey:         testAPIKey,
			expectedStatus: 200,
			checkBody: func(t *testing.T, body string) {
				// Should only contain user-2's service
				if !containsString(body, "User 2 Service") {
					t.Error("Response should contain User 2 Service")
				}
				// Should not contain user-1's services
				if containsString(body, "Test Service 1") || containsString(body, "Test Service 2") {
					t.Error("Response should NOT contain services from other users")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/prometheus/metrics/user/"+tt.userID, nil)
			if tt.apiKey != "" {
				req.Header.Set("Authorization", "Bearer "+tt.apiKey)
			}

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to perform request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			// Read response body
			body := make([]byte, 10000)
			n, _ := resp.Body.Read(body)
			bodyStr := string(body[:n])

			if tt.checkBody != nil {
				tt.checkBody(t, bodyStr)
			}
		})
	}
}

func TestGetUserPrometheusMetrics_WithXAPIKeyHeader(t *testing.T) {
	db := setupMetricsTestDB(t)
	defer db.Close()

	serviceRepo := repository.NewServiceRepository(db)
	statusLogRepo := repository.NewStatusLogRepository(db)
	metricsService := services.NewMetricsService(statusLogRepo, serviceRepo)
	handler := NewMetricsHandler(metricsService, serviceRepo)

	// Set API key in environment
	testAPIKey := "test-x-api-key-67890"
	os.Setenv("PROMETHEUS_API_KEY", testAPIKey)
	defer os.Unsetenv("PROMETHEUS_API_KEY")

	// Create test service
	createTestService(t, db, &models.Service{
		ID:        "service-1",
		UserID:    "user-1",
		Name:      "Test Service",
		URL:       "https://example.com",
		Icon:      "🔗",
		Status:    models.StatusOnline,
		Position:  0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	app := fiber.New()
	app.Get("/api/v1/prometheus/metrics/user/:userID", handler.GetUserPrometheusMetrics)

	// Test with X-API-Key header
	req := httptest.NewRequest("GET", "/api/v1/prometheus/metrics/user/user-1", nil)
	req.Header.Set("X-API-Key", testAPIKey)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Read and verify response
	body := make([]byte, 10000)
	n, _ := resp.Body.Read(body)
	bodyStr := string(body[:n])

	if !containsString(bodyStr, "Test Service") {
		t.Error("Response should contain the test service")
	}
}

func TestGetUserPrometheusMetrics_WithJWTAuthentication(t *testing.T) {
	db := setupMetricsTestDB(t)
	defer db.Close()

	serviceRepo := repository.NewServiceRepository(db)
	statusLogRepo := repository.NewStatusLogRepository(db)
	metricsService := services.NewMetricsService(statusLogRepo, serviceRepo)
	handler := NewMetricsHandler(metricsService, serviceRepo)

	// Set API key (but we won't use it in this test)
	os.Setenv("PROMETHEUS_API_KEY", "some-key")
	defer os.Unsetenv("PROMETHEUS_API_KEY")

	// Create test services
	createTestService(t, db, &models.Service{
		ID:        "service-1",
		UserID:    "user-1",
		Name:      "User 1 Service",
		URL:       "https://example1.com",
		Icon:      "🔗",
		Status:    models.StatusOnline,
		Position:  0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	createTestService(t, db, &models.Service{
		ID:        "service-2",
		UserID:    "user-2",
		Name:      "User 2 Service",
		URL:       "https://example2.com",
		Icon:      "🔗",
		Status:    models.StatusOnline,
		Position:  0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	app := fiber.New()

	// Middleware to simulate JWT authentication
	app.Use(func(c *fiber.Ctx) error {
		// Simulate authenticated user (would normally come from JWT)
		c.Locals("user_id", "user-1")
		c.Locals("role", "user")
		return c.Next()
	})

	app.Get("/api/v1/prometheus/metrics/user/:userID", handler.GetUserPrometheusMetrics)

	tests := []struct {
		name           string
		requestUserID  string
		expectedStatus int
		expectService  string
	}{
		{
			name:           "User accessing their own metrics - should succeed",
			requestUserID:  "user-1",
			expectedStatus: 200,
			expectService:  "User 1 Service",
		},
		{
			name:           "User accessing another user's metrics - should be forbidden",
			requestUserID:  "user-2",
			expectedStatus: 403,
			expectService:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/prometheus/metrics/user/"+tt.requestUserID, nil)

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to perform request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if tt.expectService != "" {
				body := make([]byte, 10000)
				n, _ := resp.Body.Read(body)
				bodyStr := string(body[:n])

				if !containsString(bodyStr, tt.expectService) {
					t.Errorf("Response should contain '%s'", tt.expectService)
				}
			}
		})
	}
}

func TestGetUserPrometheusMetrics_AdminAccess(t *testing.T) {
	db := setupMetricsTestDB(t)
	defer db.Close()

	serviceRepo := repository.NewServiceRepository(db)
	statusLogRepo := repository.NewStatusLogRepository(db)
	metricsService := services.NewMetricsService(statusLogRepo, serviceRepo)
	handler := NewMetricsHandler(metricsService, serviceRepo)

	// Create test service for user-2
	createTestService(t, db, &models.Service{
		ID:        "service-1",
		UserID:    "user-2",
		Name:      "User 2 Service",
		URL:       "https://example.com",
		Icon:      "🔗",
		Status:    models.StatusOnline,
		Position:  0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	app := fiber.New()

	// Middleware to simulate admin JWT authentication
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", "admin-user")
		c.Locals("role", "admin")
		return c.Next()
	})

	app.Get("/api/v1/prometheus/metrics/user/:userID", handler.GetUserPrometheusMetrics)

	// Admin should be able to access any user's metrics
	req := httptest.NewRequest("GET", "/api/v1/prometheus/metrics/user/user-2", nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Admin should be able to access any user's metrics, got status %d", resp.StatusCode)
	}

	body := make([]byte, 10000)
	n, _ := resp.Body.Read(body)
	bodyStr := string(body[:n])

	if !containsString(bodyStr, "User 2 Service") {
		t.Error("Admin should see user-2's service")
	}
}

func TestGetUserPrometheusMetrics_EmptyServices(t *testing.T) {
	db := setupMetricsTestDB(t)
	defer db.Close()

	serviceRepo := repository.NewServiceRepository(db)
	statusLogRepo := repository.NewStatusLogRepository(db)
	metricsService := services.NewMetricsService(statusLogRepo, serviceRepo)
	handler := NewMetricsHandler(metricsService, serviceRepo)

	// Set API key
	testAPIKey := "test-key"
	os.Setenv("PROMETHEUS_API_KEY", testAPIKey)
	defer os.Unsetenv("PROMETHEUS_API_KEY")

	app := fiber.New()
	app.Get("/api/v1/prometheus/metrics/user/:userID", handler.GetUserPrometheusMetrics)

	// Request metrics for user with no services
	req := httptest.NewRequest("GET", "/api/v1/prometheus/metrics/user/user-no-services", nil)
	req.Header.Set("Authorization", "Bearer "+testAPIKey)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Should return 200 even with no services, got %d", resp.StatusCode)
	}

	body := make([]byte, 10000)
	n, _ := resp.Body.Read(body)
	bodyStr := string(body[:n])

	// Should still have Prometheus format headers
	if !containsString(bodyStr, "# HELP nimbus_service_up") {
		t.Error("Response should contain Prometheus headers even with no services")
	}

	// Should show 0 total services
	if !containsString(bodyStr, "nimbus_total_services 0") {
		t.Error("Response should show 0 total services")
	}
}

func TestGetUserPrometheusMetrics_MissingUserID(t *testing.T) {
	db := setupMetricsTestDB(t)
	defer db.Close()

	serviceRepo := repository.NewServiceRepository(db)
	statusLogRepo := repository.NewStatusLogRepository(db)
	metricsService := services.NewMetricsService(statusLogRepo, serviceRepo)
	handler := NewMetricsHandler(metricsService, serviceRepo)

	os.Setenv("PROMETHEUS_API_KEY", "test-key")
	defer os.Unsetenv("PROMETHEUS_API_KEY")

	app := fiber.New()
	app.Get("/api/v1/prometheus/metrics/user/:userID", handler.GetUserPrometheusMetrics)

	// Request without user ID in path (empty string)
	req := httptest.NewRequest("GET", "/api/v1/prometheus/metrics/user/", nil)
	req.Header.Set("Authorization", "Bearer test-key")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}
	defer resp.Body.Close()

	// Should return 404 or 400 (depending on routing)
	if resp.StatusCode != 404 && resp.StatusCode != 400 {
		t.Errorf("Expected 404 or 400 for missing user ID, got %d", resp.StatusCode)
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
