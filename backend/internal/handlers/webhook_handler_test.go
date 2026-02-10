package handlers

import (
	"bytes"
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

// setupWebhookTestDB creates an in-memory SQLite database with webhook tables
func setupWebhookTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	schema := `
		CREATE TABLE IF NOT EXISTS webhooks (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			user_id TEXT NOT NULL,
			name TEXT NOT NULL,
			url TEXT NOT NULL,
			enabled INTEGER NOT NULL DEFAULT 1,
			triggers TEXT NOT NULL DEFAULT '{"on_offline":true,"on_online":false}',
			format TEXT NOT NULL DEFAULT 'generic',
			retry_count INTEGER NOT NULL DEFAULT 0,
			retry_delay_seconds INTEGER NOT NULL DEFAULT 30,
			last_triggered_at TIMESTAMP,
			last_success_at TIMESTAMP,
			consecutive_failures INTEGER NOT NULL DEFAULT 0,
			total_sent INTEGER NOT NULL DEFAULT 0,
			total_failed INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		);

		CREATE TABLE IF NOT EXISTS webhook_logs (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			webhook_id TEXT NOT NULL,
			service_id TEXT NOT NULL,
			service_name TEXT NOT NULL,
			old_status TEXT NOT NULL,
			new_status TEXT NOT NULL,
			success INTEGER NOT NULL,
			status_code INTEGER,
			error_message TEXT,
			response_time_ms INTEGER,
			created_at TIMESTAMP NOT NULL
		);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create test tables: %v", err)
	}

	return db
}

// createWebhookDirectly inserts a webhook for testing
func createWebhookDirectly(t *testing.T, db *sql.DB, webhook *models.Webhook) {
	triggersJSON, _ := json.Marshal(webhook.Triggers)
	query := `
		INSERT INTO webhooks (id, user_id, name, url, enabled, triggers, format,
			last_triggered_at, last_success_at, consecutive_failures, total_sent, total_failed,
			created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	enabled := 0
	if webhook.Enabled {
		enabled = 1
	}

	_, err := db.Exec(
		query,
		webhook.ID,
		webhook.UserID,
		webhook.Name,
		webhook.URL,
		enabled,
		triggersJSON,
		webhook.Format,
		webhook.LastTriggeredAt,
		webhook.LastSuccessAt,
		webhook.ConsecutiveFailures,
		webhook.TotalSent,
		webhook.TotalFailed,
		webhook.CreatedAt,
		webhook.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("Failed to create webhook directly: %v", err)
	}
}

func TestWebhookHandler_CreateWebhook(t *testing.T) {
	db := setupWebhookTestDB(t)
	defer db.Close()

	webhookRepo := repository.NewWebhookRepository(db)
	handler := NewWebhookHandler(webhookRepo, nil)

	tests := []struct {
		name           string
		userID         string
		requestBody    models.WebhookCreateRequest
		expectedStatus int
		expectError    bool
	}{
		{
			name:   "Successfully create webhook",
			userID: "user-1",
			requestBody: models.WebhookCreateRequest{
				Name:   "Test Webhook",
				URL:    "https://discord.com/api/webhooks/123/abc",
				Format: models.WebhookFormatDiscord,
			},
			expectedStatus: http.StatusCreated,
			expectError:    false,
		},
		{
			name:   "Missing name",
			userID: "user-1",
			requestBody: models.WebhookCreateRequest{
				URL: "https://example.com/webhook",
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:   "Missing URL",
			userID: "user-1",
			requestBody: models.WebhookCreateRequest{
				Name: "Test Webhook",
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:   "Invalid format",
			userID: "user-1",
			requestBody: models.WebhookCreateRequest{
				Name:   "Test Webhook",
				URL:    "https://example.com/webhook",
				Format: "invalid",
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:   "Name too long",
			userID: "user-1",
			requestBody: models.WebhookCreateRequest{
				Name: string(make([]byte, 101)), // 101 chars
				URL:  "https://example.com/webhook",
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:   "Default format when not provided",
			userID: "user-1",
			requestBody: models.WebhookCreateRequest{
				Name: "Default Format Webhook",
				URL:  "https://example.com/webhook",
			},
			expectedStatus: http.StatusCreated,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Post("/webhooks", func(c *fiber.Ctx) error {
				c.Locals("user_id", tt.userID)
				return handler.CreateWebhook(c)
			})

			bodyJSON, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/webhooks", bytes.NewReader(bodyJSON))
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

func TestWebhookHandler_CreateWebhook_NoAuth(t *testing.T) {
	db := setupWebhookTestDB(t)
	defer db.Close()

	webhookRepo := repository.NewWebhookRepository(db)
	handler := NewWebhookHandler(webhookRepo, nil)

	app := fiber.New()
	app.Post("/webhooks", handler.CreateWebhook)

	requestBody := models.WebhookCreateRequest{
		Name: "Test Webhook",
		URL:  "https://example.com/webhook",
	}

	bodyJSON, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/webhooks", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status %d for unauthenticated request, got %d", http.StatusUnauthorized, resp.StatusCode)
	}
}

func TestWebhookHandler_CreateWebhook_MaxLimit(t *testing.T) {
	db := setupWebhookTestDB(t)
	defer db.Close()

	webhookRepo := repository.NewWebhookRepository(db)
	handler := NewWebhookHandler(webhookRepo, nil)

	// Create 10 webhooks (the limit)
	for i := 0; i < 10; i++ {
		createWebhookDirectly(t, db, &models.Webhook{
			ID:        "webhook-" + string(rune('0'+i)),
			UserID:    "user-1",
			Name:      "Webhook " + string(rune('0'+i)),
			URL:       "https://example.com/webhook",
			Enabled:   true,
			Triggers:  models.WebhookTriggers{OnOffline: true},
			Format:    models.WebhookFormatGeneric,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		})
	}

	app := fiber.New()
	app.Post("/webhooks", func(c *fiber.Ctx) error {
		c.Locals("user_id", "user-1")
		return handler.CreateWebhook(c)
	})

	requestBody := models.WebhookCreateRequest{
		Name: "Extra Webhook",
		URL:  "https://example.com/webhook",
	}

	bodyJSON, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/webhooks", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status %d for exceeding webhook limit, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

func TestWebhookHandler_GetWebhooks(t *testing.T) {
	db := setupWebhookTestDB(t)
	defer db.Close()

	webhookRepo := repository.NewWebhookRepository(db)
	handler := NewWebhookHandler(webhookRepo, nil)

	// Create test webhooks
	createWebhookDirectly(t, db, &models.Webhook{
		ID:        "webhook-1",
		UserID:    "user-1",
		Name:      "Webhook 1",
		URL:       "https://example.com/webhook1",
		Enabled:   true,
		Triggers:  models.WebhookTriggers{OnOffline: true},
		Format:    models.WebhookFormatGeneric,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	createWebhookDirectly(t, db, &models.Webhook{
		ID:        "webhook-2",
		UserID:    "user-1",
		Name:      "Webhook 2",
		URL:       "https://example.com/webhook2",
		Enabled:   false,
		Triggers:  models.WebhookTriggers{OnOnline: true},
		Format:    models.WebhookFormatDiscord,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	// Different user's webhook
	createWebhookDirectly(t, db, &models.Webhook{
		ID:        "webhook-3",
		UserID:    "user-2",
		Name:      "Other User Webhook",
		URL:       "https://example.com/webhook3",
		Enabled:   true,
		Triggers:  models.WebhookTriggers{OnOffline: true},
		Format:    models.WebhookFormatGeneric,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	app := fiber.New()
	app.Get("/webhooks", func(c *fiber.Ctx) error {
		c.Locals("user_id", "user-1")
		return handler.GetWebhooks(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/webhooks", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Decode response
	var webhooks []models.WebhookResponse
	if err := json.NewDecoder(resp.Body).Decode(&webhooks); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should only see user-1's webhooks
	if len(webhooks) != 2 {
		t.Errorf("Expected 2 webhooks, got %d", len(webhooks))
	}
}

func TestWebhookHandler_GetWebhook(t *testing.T) {
	db := setupWebhookTestDB(t)
	defer db.Close()

	webhookRepo := repository.NewWebhookRepository(db)
	handler := NewWebhookHandler(webhookRepo, nil)

	createWebhookDirectly(t, db, &models.Webhook{
		ID:        "webhook-1",
		UserID:    "user-1",
		Name:      "Test Webhook",
		URL:       "https://example.com/webhook",
		Enabled:   true,
		Triggers:  models.WebhookTriggers{OnOffline: true},
		Format:    models.WebhookFormatGeneric,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	tests := []struct {
		name           string
		userID         string
		webhookID      string
		expectedStatus int
	}{
		{
			name:           "Get existing webhook",
			userID:         "user-1",
			webhookID:      "webhook-1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Get non-existent webhook",
			userID:         "user-1",
			webhookID:      "non-existent",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Get other user's webhook",
			userID:         "user-2",
			webhookID:      "webhook-1",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/webhooks/:id", func(c *fiber.Ctx) error {
				c.Locals("user_id", tt.userID)
				return handler.GetWebhook(c)
			})

			req := httptest.NewRequest(http.MethodGet, "/webhooks/"+tt.webhookID, nil)
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

func TestWebhookHandler_UpdateWebhook(t *testing.T) {
	db := setupWebhookTestDB(t)
	defer db.Close()

	webhookRepo := repository.NewWebhookRepository(db)
	handler := NewWebhookHandler(webhookRepo, nil)

	createWebhookDirectly(t, db, &models.Webhook{
		ID:        "webhook-1",
		UserID:    "user-1",
		Name:      "Test Webhook",
		URL:       "https://example.com/webhook",
		Enabled:   true,
		Triggers:  models.WebhookTriggers{OnOffline: true},
		Format:    models.WebhookFormatGeneric,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	newName := "Updated Webhook"
	newURL := "https://updated.com/webhook"
	disabled := false

	tests := []struct {
		name           string
		userID         string
		webhookID      string
		requestBody    models.WebhookUpdateRequest
		expectedStatus int
	}{
		{
			name:      "Update name",
			userID:    "user-1",
			webhookID: "webhook-1",
			requestBody: models.WebhookUpdateRequest{
				Name: &newName,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:      "Update URL",
			userID:    "user-1",
			webhookID: "webhook-1",
			requestBody: models.WebhookUpdateRequest{
				URL: &newURL,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:      "Disable webhook",
			userID:    "user-1",
			webhookID: "webhook-1",
			requestBody: models.WebhookUpdateRequest{
				Enabled: &disabled,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:      "Update non-existent webhook",
			userID:    "user-1",
			webhookID: "non-existent",
			requestBody: models.WebhookUpdateRequest{
				Name: &newName,
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:      "Update other user's webhook",
			userID:    "user-2",
			webhookID: "webhook-1",
			requestBody: models.WebhookUpdateRequest{
				Name: &newName,
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Put("/webhooks/:id", func(c *fiber.Ctx) error {
				c.Locals("user_id", tt.userID)
				return handler.UpdateWebhook(c)
			})

			bodyJSON, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPut, "/webhooks/"+tt.webhookID, bytes.NewReader(bodyJSON))
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

func TestWebhookHandler_DeleteWebhook(t *testing.T) {
	db := setupWebhookTestDB(t)
	defer db.Close()

	webhookRepo := repository.NewWebhookRepository(db)
	handler := NewWebhookHandler(webhookRepo, nil)

	createWebhookDirectly(t, db, &models.Webhook{
		ID:        "webhook-1",
		UserID:    "user-1",
		Name:      "Test Webhook",
		URL:       "https://example.com/webhook",
		Enabled:   true,
		Triggers:  models.WebhookTriggers{OnOffline: true},
		Format:    models.WebhookFormatGeneric,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	tests := []struct {
		name           string
		userID         string
		webhookID      string
		expectedStatus int
	}{
		{
			name:           "Delete existing webhook",
			userID:         "user-1",
			webhookID:      "webhook-1",
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "Delete non-existent webhook",
			userID:         "user-1",
			webhookID:      "non-existent",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Delete("/webhooks/:id", func(c *fiber.Ctx) error {
				c.Locals("user_id", tt.userID)
				return handler.DeleteWebhook(c)
			})

			req := httptest.NewRequest(http.MethodDelete, "/webhooks/"+tt.webhookID, nil)
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

func TestWebhookHandler_InvalidJSON(t *testing.T) {
	db := setupWebhookTestDB(t)
	defer db.Close()

	webhookRepo := repository.NewWebhookRepository(db)
	handler := NewWebhookHandler(webhookRepo, nil)

	app := fiber.New()
	app.Post("/webhooks", func(c *fiber.Ctx) error {
		c.Locals("user_id", "user-1")
		return handler.CreateWebhook(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/webhooks", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status %d for invalid JSON, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}
