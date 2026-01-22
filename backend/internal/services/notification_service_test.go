package services

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"

	_ "github.com/mattn/go-sqlite3"
)

// setupNotificationTestDB creates an in-memory SQLite database with webhook tables
func setupNotificationTestDB(t *testing.T) *sql.DB {
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

func TestNewNotificationService(t *testing.T) {
	db := setupNotificationTestDB(t)
	defer db.Close()

	webhookRepo := repository.NewWebhookRepository(db)
	service := NewNotificationService(webhookRepo)

	if service == nil {
		t.Fatal("Expected notification service to be created")
	}

	if service.webhookNotifier == nil {
		t.Fatal("Expected webhook notifier to be initialized")
	}
}

func TestNotificationService_NotifyStatusChange_NoWebhooks(t *testing.T) {
	db := setupNotificationTestDB(t)
	defer db.Close()

	webhookRepo := repository.NewWebhookRepository(db)
	service := NewNotificationService(webhookRepo)

	event := NotificationEvent{
		ServiceID:   "service-1",
		ServiceName: "Test Service",
		ServiceURL:  "https://example.com",
		OldStatus:   models.StatusOnline,
		NewStatus:   models.StatusOffline,
		UserID:      "user-1",
		Timestamp:   time.Now(),
	}

	// Should not panic even with no webhooks
	service.NotifyStatusChange(context.Background(), event)
}

func TestNotificationEvent_Fields(t *testing.T) {
	errMsg := "Connection timeout"
	event := NotificationEvent{
		ServiceID:   "service-123",
		ServiceName: "My Service",
		ServiceURL:  "https://myservice.com",
		OldStatus:   models.StatusOnline,
		NewStatus:   models.StatusOffline,
		UserID:      "user-456",
		Timestamp:   time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		ErrorMsg:    &errMsg,
	}

	if event.ServiceID != "service-123" {
		t.Errorf("Expected ServiceID 'service-123', got %s", event.ServiceID)
	}

	if event.ServiceName != "My Service" {
		t.Errorf("Expected ServiceName 'My Service', got %s", event.ServiceName)
	}

	if event.ServiceURL != "https://myservice.com" {
		t.Errorf("Expected ServiceURL 'https://myservice.com', got %s", event.ServiceURL)
	}

	if event.OldStatus != models.StatusOnline {
		t.Errorf("Expected OldStatus 'online', got %s", event.OldStatus)
	}

	if event.NewStatus != models.StatusOffline {
		t.Errorf("Expected NewStatus 'offline', got %s", event.NewStatus)
	}

	if event.UserID != "user-456" {
		t.Errorf("Expected UserID 'user-456', got %s", event.UserID)
	}

	if event.ErrorMsg == nil || *event.ErrorMsg != errMsg {
		t.Error("Expected ErrorMsg to be set")
	}
}

func TestTestWebhookResult_Fields(t *testing.T) {
	result := TestWebhookResult{
		Success:        true,
		StatusCode:     200,
		ResponseTimeMs: 150,
		Error:          "",
	}

	if !result.Success {
		t.Error("Expected Success to be true")
	}

	if result.StatusCode != 200 {
		t.Errorf("Expected StatusCode 200, got %d", result.StatusCode)
	}

	if result.ResponseTimeMs != 150 {
		t.Errorf("Expected ResponseTimeMs 150, got %d", result.ResponseTimeMs)
	}

	if result.Error != "" {
		t.Errorf("Expected empty Error, got %s", result.Error)
	}
}

func TestTestWebhookResult_Failure(t *testing.T) {
	result := TestWebhookResult{
		Success:        false,
		StatusCode:     500,
		ResponseTimeMs: 50,
		Error:          "Internal server error",
	}

	if result.Success {
		t.Error("Expected Success to be false")
	}

	if result.StatusCode != 500 {
		t.Errorf("Expected StatusCode 500, got %d", result.StatusCode)
	}

	if result.Error != "Internal server error" {
		t.Errorf("Expected Error 'Internal server error', got %s", result.Error)
	}
}

func TestNotificationEvent_OptionalErrorMsg(t *testing.T) {
	event := NotificationEvent{
		ServiceID:   "service-1",
		ServiceName: "Test Service",
		ServiceURL:  "https://example.com",
		OldStatus:   models.StatusOffline,
		NewStatus:   models.StatusOnline,
		UserID:      "user-1",
		Timestamp:   time.Now(),
		ErrorMsg:    nil, // No error for recovery events
	}

	if event.ErrorMsg != nil {
		t.Error("Expected ErrorMsg to be nil for recovery events")
	}
}

func TestNotificationService_TestWebhook_NotFound(t *testing.T) {
	db := setupNotificationTestDB(t)
	defer db.Close()

	webhookRepo := repository.NewWebhookRepository(db)
	service := NewNotificationService(webhookRepo)

	// Try to test a non-existent webhook
	_, err := service.TestWebhook(context.Background(), "non-existent", "user-1")
	if err == nil {
		t.Error("Expected error for non-existent webhook")
	}
}
