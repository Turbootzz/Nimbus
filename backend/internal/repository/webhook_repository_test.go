package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/nimbus/backend/internal/models"

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

func TestWebhookRepository_Create(t *testing.T) {
	db := setupWebhookTestDB(t)
	defer db.Close()

	repo := NewWebhookRepository(db)
	ctx := context.Background()

	webhook := &models.Webhook{
		UserID:   "user-1",
		Name:     "Test Webhook",
		URL:      "https://example.com/webhook",
		Enabled:  true,
		Triggers: models.WebhookTriggers{OnOffline: true, OnOnline: false},
		Format:   models.WebhookFormatGeneric,
	}

	err := repo.Create(ctx, webhook)
	if err != nil {
		t.Fatalf("Failed to create webhook: %v", err)
	}

	if webhook.ID == "" {
		t.Error("Expected webhook ID to be set after creation")
	}

	if webhook.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}

	if webhook.UpdatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be set")
	}
}

func TestWebhookRepository_GetByID(t *testing.T) {
	db := setupWebhookTestDB(t)
	defer db.Close()

	repo := NewWebhookRepository(db)
	ctx := context.Background()

	// Create a webhook
	webhook := &models.Webhook{
		UserID:   "user-1",
		Name:     "Test Webhook",
		URL:      "https://example.com/webhook",
		Enabled:  true,
		Triggers: models.WebhookTriggers{OnOffline: true, OnOnline: true},
		Format:   models.WebhookFormatDiscord,
	}
	err := repo.Create(ctx, webhook)
	if err != nil {
		t.Fatalf("Failed to create webhook: %v", err)
	}

	// Retrieve it
	retrieved, err := repo.GetByID(ctx, webhook.ID, "user-1")
	if err != nil {
		t.Fatalf("Failed to get webhook: %v", err)
	}

	if retrieved.Name != webhook.Name {
		t.Errorf("Expected name %s, got %s", webhook.Name, retrieved.Name)
	}

	if retrieved.Format != models.WebhookFormatDiscord {
		t.Errorf("Expected format %s, got %s", models.WebhookFormatDiscord, retrieved.Format)
	}

	if !retrieved.Triggers.OnOffline || !retrieved.Triggers.OnOnline {
		t.Error("Triggers were not properly stored/retrieved")
	}
}

func TestWebhookRepository_GetByID_NotFound(t *testing.T) {
	db := setupWebhookTestDB(t)
	defer db.Close()

	repo := NewWebhookRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "non-existent", "user-1")
	if err != sql.ErrNoRows {
		t.Errorf("Expected sql.ErrNoRows, got %v", err)
	}
}

func TestWebhookRepository_GetByID_WrongUser(t *testing.T) {
	db := setupWebhookTestDB(t)
	defer db.Close()

	repo := NewWebhookRepository(db)
	ctx := context.Background()

	// Create webhook for user-1
	webhook := &models.Webhook{
		UserID:   "user-1",
		Name:     "Test Webhook",
		URL:      "https://example.com/webhook",
		Enabled:  true,
		Triggers: models.WebhookTriggers{OnOffline: true},
		Format:   models.WebhookFormatGeneric,
	}
	err := repo.Create(ctx, webhook)
	if err != nil {
		t.Fatalf("Failed to create webhook: %v", err)
	}

	// Try to retrieve as user-2
	_, err = repo.GetByID(ctx, webhook.ID, "user-2")
	if err != sql.ErrNoRows {
		t.Errorf("Expected sql.ErrNoRows when getting another user's webhook, got %v", err)
	}
}

func TestWebhookRepository_GetAllByUserID(t *testing.T) {
	db := setupWebhookTestDB(t)
	defer db.Close()

	repo := NewWebhookRepository(db)
	ctx := context.Background()

	// Create webhooks for user-1
	for i := 0; i < 3; i++ {
		webhook := &models.Webhook{
			UserID:   "user-1",
			Name:     "Webhook",
			URL:      "https://example.com/webhook",
			Enabled:  true,
			Triggers: models.WebhookTriggers{OnOffline: true},
			Format:   models.WebhookFormatGeneric,
		}
		if err := repo.Create(ctx, webhook); err != nil {
			t.Fatalf("Failed to create webhook: %v", err)
		}
	}

	// Create webhook for user-2
	webhook := &models.Webhook{
		UserID:   "user-2",
		Name:     "Other Webhook",
		URL:      "https://example.com/webhook",
		Enabled:  true,
		Triggers: models.WebhookTriggers{OnOffline: true},
		Format:   models.WebhookFormatGeneric,
	}
	if err := repo.Create(ctx, webhook); err != nil {
		t.Fatalf("Failed to create webhook: %v", err)
	}

	// Get user-1's webhooks
	webhooks, err := repo.GetAllByUserID(ctx, "user-1")
	if err != nil {
		t.Fatalf("Failed to get webhooks: %v", err)
	}

	if len(webhooks) != 3 {
		t.Errorf("Expected 3 webhooks, got %d", len(webhooks))
	}
}

func TestWebhookRepository_GetEnabledByUserID(t *testing.T) {
	db := setupWebhookTestDB(t)
	defer db.Close()

	repo := NewWebhookRepository(db)
	ctx := context.Background()

	// Create enabled webhook
	enabled := &models.Webhook{
		UserID:   "user-1",
		Name:     "Enabled Webhook",
		URL:      "https://example.com/webhook1",
		Enabled:  true,
		Triggers: models.WebhookTriggers{OnOffline: true},
		Format:   models.WebhookFormatGeneric,
	}
	if err := repo.Create(ctx, enabled); err != nil {
		t.Fatalf("Failed to create webhook: %v", err)
	}

	// Create disabled webhook
	disabled := &models.Webhook{
		UserID:   "user-1",
		Name:     "Disabled Webhook",
		URL:      "https://example.com/webhook2",
		Enabled:  false,
		Triggers: models.WebhookTriggers{OnOffline: true},
		Format:   models.WebhookFormatGeneric,
	}
	if err := repo.Create(ctx, disabled); err != nil {
		t.Fatalf("Failed to create webhook: %v", err)
	}

	// Get enabled webhooks only
	webhooks, err := repo.GetEnabledByUserID(ctx, "user-1")
	if err != nil {
		t.Fatalf("Failed to get enabled webhooks: %v", err)
	}

	if len(webhooks) != 1 {
		t.Errorf("Expected 1 enabled webhook, got %d", len(webhooks))
	}

	if webhooks[0].Name != "Enabled Webhook" {
		t.Errorf("Expected enabled webhook, got %s", webhooks[0].Name)
	}
}

func TestWebhookRepository_CountByUserID(t *testing.T) {
	db := setupWebhookTestDB(t)
	defer db.Close()

	repo := NewWebhookRepository(db)
	ctx := context.Background()

	// Initially should be 0
	count, err := repo.CountByUserID(ctx, "user-1")
	if err != nil {
		t.Fatalf("Failed to count webhooks: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 webhooks, got %d", count)
	}

	// Create 5 webhooks
	for i := 0; i < 5; i++ {
		webhook := &models.Webhook{
			UserID:   "user-1",
			Name:     "Webhook",
			URL:      "https://example.com/webhook",
			Enabled:  true,
			Triggers: models.WebhookTriggers{OnOffline: true},
			Format:   models.WebhookFormatGeneric,
		}
		if err := repo.Create(ctx, webhook); err != nil {
			t.Fatalf("Failed to create webhook: %v", err)
		}
	}

	count, err = repo.CountByUserID(ctx, "user-1")
	if err != nil {
		t.Fatalf("Failed to count webhooks: %v", err)
	}
	if count != 5 {
		t.Errorf("Expected 5 webhooks, got %d", count)
	}
}

func TestWebhookRepository_Update(t *testing.T) {
	db := setupWebhookTestDB(t)
	defer db.Close()

	repo := NewWebhookRepository(db)
	ctx := context.Background()

	// Create a webhook
	webhook := &models.Webhook{
		UserID:   "user-1",
		Name:     "Original Name",
		URL:      "https://example.com/webhook",
		Enabled:  true,
		Triggers: models.WebhookTriggers{OnOffline: true},
		Format:   models.WebhookFormatGeneric,
	}
	if err := repo.Create(ctx, webhook); err != nil {
		t.Fatalf("Failed to create webhook: %v", err)
	}

	// Update it
	webhook.Name = "Updated Name"
	webhook.Enabled = false
	webhook.Format = models.WebhookFormatSlack

	if err := repo.Update(ctx, webhook); err != nil {
		t.Fatalf("Failed to update webhook: %v", err)
	}

	// Retrieve and verify
	retrieved, err := repo.GetByID(ctx, webhook.ID, "user-1")
	if err != nil {
		t.Fatalf("Failed to get webhook: %v", err)
	}

	if retrieved.Name != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got %s", retrieved.Name)
	}

	if retrieved.Enabled {
		t.Error("Expected webhook to be disabled")
	}

	if retrieved.Format != models.WebhookFormatSlack {
		t.Errorf("Expected format %s, got %s", models.WebhookFormatSlack, retrieved.Format)
	}
}

func TestWebhookRepository_Update_NotFound(t *testing.T) {
	db := setupWebhookTestDB(t)
	defer db.Close()

	repo := NewWebhookRepository(db)
	ctx := context.Background()

	webhook := &models.Webhook{
		ID:       "non-existent",
		UserID:   "user-1",
		Name:     "Test",
		URL:      "https://example.com",
		Enabled:  true,
		Triggers: models.WebhookTriggers{OnOffline: true},
		Format:   models.WebhookFormatGeneric,
	}

	err := repo.Update(ctx, webhook)
	if err != sql.ErrNoRows {
		t.Errorf("Expected sql.ErrNoRows, got %v", err)
	}
}

func TestWebhookRepository_Delete(t *testing.T) {
	db := setupWebhookTestDB(t)
	defer db.Close()

	repo := NewWebhookRepository(db)
	ctx := context.Background()

	// Create a webhook
	webhook := &models.Webhook{
		UserID:   "user-1",
		Name:     "To Delete",
		URL:      "https://example.com/webhook",
		Enabled:  true,
		Triggers: models.WebhookTriggers{OnOffline: true},
		Format:   models.WebhookFormatGeneric,
	}
	if err := repo.Create(ctx, webhook); err != nil {
		t.Fatalf("Failed to create webhook: %v", err)
	}

	// Delete it
	if err := repo.Delete(ctx, webhook.ID, "user-1"); err != nil {
		t.Fatalf("Failed to delete webhook: %v", err)
	}

	// Verify it's gone
	_, err := repo.GetByID(ctx, webhook.ID, "user-1")
	if err != sql.ErrNoRows {
		t.Errorf("Expected webhook to be deleted, got error: %v", err)
	}
}

func TestWebhookRepository_Delete_WrongUser(t *testing.T) {
	db := setupWebhookTestDB(t)
	defer db.Close()

	repo := NewWebhookRepository(db)
	ctx := context.Background()

	// Create a webhook for user-1
	webhook := &models.Webhook{
		UserID:   "user-1",
		Name:     "Test",
		URL:      "https://example.com/webhook",
		Enabled:  true,
		Triggers: models.WebhookTriggers{OnOffline: true},
		Format:   models.WebhookFormatGeneric,
	}
	if err := repo.Create(ctx, webhook); err != nil {
		t.Fatalf("Failed to create webhook: %v", err)
	}

	// Try to delete as user-2
	err := repo.Delete(ctx, webhook.ID, "user-2")
	if err != sql.ErrNoRows {
		t.Errorf("Expected sql.ErrNoRows when deleting another user's webhook, got %v", err)
	}

	// Verify it still exists
	_, err = repo.GetByID(ctx, webhook.ID, "user-1")
	if err != nil {
		t.Error("Webhook should still exist")
	}
}

func TestWebhookRepository_RecordDeliverySuccess(t *testing.T) {
	db := setupWebhookTestDB(t)
	defer db.Close()

	repo := NewWebhookRepository(db)
	ctx := context.Background()

	// Create a webhook
	webhook := &models.Webhook{
		UserID:   "user-1",
		Name:     "Test",
		URL:      "https://example.com/webhook",
		Enabled:  true,
		Triggers: models.WebhookTriggers{OnOffline: true},
		Format:   models.WebhookFormatGeneric,
	}
	if err := repo.Create(ctx, webhook); err != nil {
		t.Fatalf("Failed to create webhook: %v", err)
	}

	// Manually set initial stats to simulate existing webhook with failures
	_, err := db.Exec(`UPDATE webhooks SET consecutive_failures = 3, total_sent = 5, total_failed = 3 WHERE id = ?`, webhook.ID)
	if err != nil {
		t.Fatalf("Failed to set initial stats: %v", err)
	}

	// Record a success
	if err := repo.RecordDeliverySuccess(ctx, webhook.ID); err != nil {
		t.Fatalf("Failed to record success: %v", err)
	}

	// Verify stats
	retrieved, err := repo.GetByID(ctx, webhook.ID, "user-1")
	if err != nil {
		t.Fatalf("Failed to get webhook: %v", err)
	}

	if retrieved.ConsecutiveFailures != 0 {
		t.Errorf("Expected consecutive_failures to be reset to 0, got %d", retrieved.ConsecutiveFailures)
	}

	if retrieved.TotalSent != 6 {
		t.Errorf("Expected total_sent to be 6, got %d", retrieved.TotalSent)
	}

	if retrieved.LastTriggeredAt == nil {
		t.Error("Expected last_triggered_at to be set")
	}

	if retrieved.LastSuccessAt == nil {
		t.Error("Expected last_success_at to be set")
	}
}

func TestWebhookRepository_RecordDeliveryFailure(t *testing.T) {
	db := setupWebhookTestDB(t)
	defer db.Close()

	repo := NewWebhookRepository(db)
	ctx := context.Background()

	webhook := &models.Webhook{
		UserID:   "user-1",
		Name:     "Test",
		URL:      "https://example.com/webhook",
		Enabled:  true,
		Triggers: models.WebhookTriggers{OnOffline: true},
		Format:   models.WebhookFormatGeneric,
	}
	if err := repo.Create(ctx, webhook); err != nil {
		t.Fatalf("Failed to create webhook: %v", err)
	}

	// Manually set initial stats to simulate existing webhook
	_, err := db.Exec(`UPDATE webhooks SET consecutive_failures = 2, total_sent = 5, total_failed = 2 WHERE id = ?`, webhook.ID)
	if err != nil {
		t.Fatalf("Failed to set initial stats: %v", err)
	}

	// Record a failure
	if err := repo.RecordDeliveryFailure(ctx, webhook.ID); err != nil {
		t.Fatalf("Failed to record failure: %v", err)
	}

	// Verify stats
	retrieved, err := repo.GetByID(ctx, webhook.ID, "user-1")
	if err != nil {
		t.Fatalf("Failed to get webhook: %v", err)
	}

	if retrieved.ConsecutiveFailures != 3 {
		t.Errorf("Expected consecutive_failures to be 3, got %d", retrieved.ConsecutiveFailures)
	}

	if retrieved.TotalSent != 6 {
		t.Errorf("Expected total_sent to be 6, got %d", retrieved.TotalSent)
	}

	if retrieved.TotalFailed != 3 {
		t.Errorf("Expected total_failed to be 3, got %d", retrieved.TotalFailed)
	}
}

func TestWebhookRepository_CreateLog(t *testing.T) {
	db := setupWebhookTestDB(t)
	defer db.Close()

	repo := NewWebhookRepository(db)
	ctx := context.Background()

	statusCode := 200
	responseTime := 150
	log := &models.WebhookLog{
		WebhookID:      "webhook-1",
		ServiceID:      "service-1",
		ServiceName:    "Test Service",
		OldStatus:      "online",
		NewStatus:      "offline",
		Success:        true,
		StatusCode:     &statusCode,
		ResponseTimeMs: &responseTime,
	}

	if err := repo.CreateLog(ctx, log); err != nil {
		t.Fatalf("Failed to create log: %v", err)
	}

	if log.ID == "" {
		t.Error("Expected log ID to be set")
	}
}

func TestWebhookRepository_GetLogsByWebhookID(t *testing.T) {
	db := setupWebhookTestDB(t)
	defer db.Close()

	repo := NewWebhookRepository(db)
	ctx := context.Background()

	// Create webhook first
	webhook := &models.Webhook{
		UserID:   "user-1",
		Name:     "Test",
		URL:      "https://example.com/webhook",
		Enabled:  true,
		Triggers: models.WebhookTriggers{OnOffline: true},
		Format:   models.WebhookFormatGeneric,
	}
	if err := repo.Create(ctx, webhook); err != nil {
		t.Fatalf("Failed to create webhook: %v", err)
	}

	// Create some logs
	for i := 0; i < 5; i++ {
		statusCode := 200
		log := &models.WebhookLog{
			WebhookID:   webhook.ID,
			ServiceID:   "service-1",
			ServiceName: "Test Service",
			OldStatus:   "online",
			NewStatus:   "offline",
			Success:     true,
			StatusCode:  &statusCode,
		}
		if err := repo.CreateLog(ctx, log); err != nil {
			t.Fatalf("Failed to create log: %v", err)
		}
	}

	// Get logs
	logs, err := repo.GetLogsByWebhookID(ctx, webhook.ID, 10)
	if err != nil {
		t.Fatalf("Failed to get logs: %v", err)
	}

	if len(logs) != 5 {
		t.Errorf("Expected 5 logs, got %d", len(logs))
	}
}

func TestWebhookRepository_GetLogsByWebhookID_WithLimit(t *testing.T) {
	db := setupWebhookTestDB(t)
	defer db.Close()

	repo := NewWebhookRepository(db)
	ctx := context.Background()

	webhookID := "webhook-1"

	// Create 10 logs
	for i := 0; i < 10; i++ {
		log := &models.WebhookLog{
			WebhookID:   webhookID,
			ServiceID:   "service-1",
			ServiceName: "Test Service",
			OldStatus:   "online",
			NewStatus:   "offline",
			Success:     true,
		}
		if err := repo.CreateLog(ctx, log); err != nil {
			t.Fatalf("Failed to create log: %v", err)
		}
	}

	// Get only 3 logs
	logs, err := repo.GetLogsByWebhookID(ctx, webhookID, 3)
	if err != nil {
		t.Fatalf("Failed to get logs: %v", err)
	}

	if len(logs) != 3 {
		t.Errorf("Expected 3 logs, got %d", len(logs))
	}
}

func TestWebhookRepository_DeleteLogsOlderThan(t *testing.T) {
	db := setupWebhookTestDB(t)
	defer db.Close()

	repo := NewWebhookRepository(db)
	ctx := context.Background()

	webhookID := "webhook-1"

	// Create a log with old timestamp
	oldTime := time.Now().Add(-48 * time.Hour)
	_, err := db.Exec(`
		INSERT INTO webhook_logs (id, webhook_id, service_id, service_name, old_status, new_status, success, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, "old-log", webhookID, "service-1", "Test", "online", "offline", 1, oldTime)
	if err != nil {
		t.Fatalf("Failed to create old log: %v", err)
	}

	// Create a recent log
	log := &models.WebhookLog{
		WebhookID:   webhookID,
		ServiceID:   "service-1",
		ServiceName: "Test Service",
		OldStatus:   "online",
		NewStatus:   "offline",
		Success:     true,
	}
	if err := repo.CreateLog(ctx, log); err != nil {
		t.Fatalf("Failed to create log: %v", err)
	}

	// Delete logs older than 24 hours
	cutoff := time.Now().Add(-24 * time.Hour)
	deleted, err := repo.DeleteLogsOlderThan(ctx, cutoff)
	if err != nil {
		t.Fatalf("Failed to delete old logs: %v", err)
	}

	if deleted != 1 {
		t.Errorf("Expected 1 log deleted, got %d", deleted)
	}

	// Verify only the recent log remains
	logs, err := repo.GetLogsByWebhookID(ctx, webhookID, 100)
	if err != nil {
		t.Fatalf("Failed to get logs: %v", err)
	}

	if len(logs) != 1 {
		t.Errorf("Expected 1 log remaining, got %d", len(logs))
	}
}

func TestWebhookRepository_TriggersJSONMarshaling(t *testing.T) {
	db := setupWebhookTestDB(t)
	defer db.Close()

	repo := NewWebhookRepository(db)
	ctx := context.Background()

	// Test various trigger combinations
	tests := []models.WebhookTriggers{
		{OnOffline: true, OnOnline: false},
		{OnOffline: false, OnOnline: true},
		{OnOffline: true, OnOnline: true},
		{OnOffline: false, OnOnline: false},
	}

	for _, triggers := range tests {
		webhook := &models.Webhook{
			UserID:   "user-1",
			Name:     "Test",
			URL:      "https://example.com/webhook",
			Enabled:  true,
			Triggers: triggers,
			Format:   models.WebhookFormatGeneric,
		}
		if err := repo.Create(ctx, webhook); err != nil {
			t.Fatalf("Failed to create webhook: %v", err)
		}

		retrieved, err := repo.GetByID(ctx, webhook.ID, "user-1")
		if err != nil {
			t.Fatalf("Failed to get webhook: %v", err)
		}

		triggersStr, _ := json.Marshal(triggers)
		retrievedStr, _ := json.Marshal(retrieved.Triggers)
		if string(triggersStr) != string(retrievedStr) {
			t.Errorf("Triggers mismatch: expected %s, got %s", triggersStr, retrievedStr)
		}
	}
}
