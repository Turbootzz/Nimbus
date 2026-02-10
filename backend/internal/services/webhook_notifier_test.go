package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"

	_ "github.com/mattn/go-sqlite3"
)

func TestWebhookNotifier_BuildGenericPayload(t *testing.T) {
	notifier := &WebhookNotifier{}

	event := NotificationEvent{
		ServiceID:   "service-123",
		ServiceName: "My Service",
		ServiceURL:  "https://myservice.com",
		OldStatus:   models.StatusOnline,
		NewStatus:   models.StatusOffline,
		UserID:      "user-1",
		Timestamp:   time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC),
	}

	payload := notifier.buildGenericPayload(event)

	if payload["event"] != "service_status_changed" {
		t.Errorf("Expected event 'service_status_changed', got %v", payload["event"])
	}

	if payload["service_id"] != "service-123" {
		t.Errorf("Expected service_id 'service-123', got %v", payload["service_id"])
	}

	if payload["service_name"] != "My Service" {
		t.Errorf("Expected service_name 'My Service', got %v", payload["service_name"])
	}

	if payload["service_url"] != "https://myservice.com" {
		t.Errorf("Expected service_url 'https://myservice.com', got %v", payload["service_url"])
	}

	if payload["old_status"] != models.StatusOnline {
		t.Errorf("Expected old_status 'online', got %v", payload["old_status"])
	}

	if payload["new_status"] != models.StatusOffline {
		t.Errorf("Expected new_status 'offline', got %v", payload["new_status"])
	}

	if _, hasError := payload["error"]; hasError {
		t.Error("Expected no error field when ErrorMsg is nil")
	}
}

func TestWebhookNotifier_BuildGenericPayload_WithError(t *testing.T) {
	notifier := &WebhookNotifier{}

	errMsg := "Connection refused"
	event := NotificationEvent{
		ServiceID:   "service-123",
		ServiceName: "My Service",
		ServiceURL:  "https://myservice.com",
		OldStatus:   models.StatusOnline,
		NewStatus:   models.StatusOffline,
		UserID:      "user-1",
		Timestamp:   time.Now(),
		ErrorMsg:    &errMsg,
	}

	payload := notifier.buildGenericPayload(event)

	if payload["error"] != errMsg {
		t.Errorf("Expected error '%s', got %v", errMsg, payload["error"])
	}
}

func TestWebhookNotifier_BuildDiscordPayload_Offline(t *testing.T) {
	notifier := &WebhookNotifier{}

	errMsg := "Connection timeout"
	event := NotificationEvent{
		ServiceID:   "service-123",
		ServiceName: "My Service",
		ServiceURL:  "https://myservice.com",
		OldStatus:   models.StatusOnline,
		NewStatus:   models.StatusOffline,
		UserID:      "user-1",
		Timestamp:   time.Now(),
		ErrorMsg:    &errMsg,
	}

	payload := notifier.buildDiscordPayload(event)

	embeds, ok := payload["embeds"].([]map[string]interface{})
	if !ok || len(embeds) == 0 {
		t.Fatal("Expected embeds array in payload")
	}

	embed := embeds[0]

	// Check title contains service name and status
	title, ok := embed["title"].(string)
	if !ok {
		t.Fatal("Expected title to be string")
	}
	if title == "" {
		t.Error("Expected non-empty title")
	}

	// Check color is red for offline
	color, ok := embed["color"].(int)
	if !ok {
		t.Fatal("Expected color to be int")
	}
	if color != 0xED4245 {
		t.Errorf("Expected red color (0xED4245), got %x", color)
	}

	// Check URL is set
	if embed["url"] != "https://myservice.com" {
		t.Errorf("Expected URL 'https://myservice.com', got %v", embed["url"])
	}

	// Check description contains error
	desc, ok := embed["description"].(string)
	if !ok || desc == "" {
		t.Error("Expected description with error message")
	}

	// Check footer
	footer, ok := embed["footer"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected footer object")
	}
	if footer["text"] != "Nimbus Dashboard" {
		t.Errorf("Expected footer text 'Nimbus Dashboard', got %v", footer["text"])
	}
}

func TestWebhookNotifier_BuildDiscordPayload_Online(t *testing.T) {
	notifier := &WebhookNotifier{}

	event := NotificationEvent{
		ServiceID:   "service-123",
		ServiceName: "My Service",
		ServiceURL:  "https://myservice.com",
		OldStatus:   models.StatusOffline,
		NewStatus:   models.StatusOnline,
		UserID:      "user-1",
		Timestamp:   time.Now(),
	}

	payload := notifier.buildDiscordPayload(event)

	embeds := payload["embeds"].([]map[string]interface{})
	embed := embeds[0]

	// Check color is green for online
	color := embed["color"].(int)
	if color != 0x57F287 {
		t.Errorf("Expected green color (0x57F287), got %x", color)
	}

	// Check description for recovery message
	desc := embed["description"].(string)
	if desc != "Service is back up and responding normally." {
		t.Errorf("Expected recovery message, got %s", desc)
	}
}

func TestWebhookNotifier_BuildSlackPayload_Offline(t *testing.T) {
	notifier := &WebhookNotifier{}

	errMsg := "Connection refused"
	event := NotificationEvent{
		ServiceID:   "service-123",
		ServiceName: "My Service",
		ServiceURL:  "https://myservice.com",
		OldStatus:   models.StatusOnline,
		NewStatus:   models.StatusOffline,
		UserID:      "user-1",
		Timestamp:   time.Now(),
		ErrorMsg:    &errMsg,
	}

	payload := notifier.buildSlackPayload(event)

	attachments, ok := payload["attachments"].([]map[string]interface{})
	if !ok || len(attachments) == 0 {
		t.Fatal("Expected attachments array in payload")
	}

	attachment := attachments[0]

	// Check color is red
	color := attachment["color"].(string)
	if color != "#ED4245" {
		t.Errorf("Expected red color '#ED4245', got %s", color)
	}

	// Check blocks exist
	blocks, ok := attachment["blocks"].([]map[string]interface{})
	if !ok || len(blocks) == 0 {
		t.Fatal("Expected blocks array in attachment")
	}

	// First block should be the header section
	firstBlock := blocks[0]
	if firstBlock["type"] != "section" {
		t.Errorf("Expected first block type 'section', got %v", firstBlock["type"])
	}
}

func TestWebhookNotifier_BuildSlackPayload_Online(t *testing.T) {
	notifier := &WebhookNotifier{}

	event := NotificationEvent{
		ServiceID:   "service-123",
		ServiceName: "My Service",
		ServiceURL:  "https://myservice.com",
		OldStatus:   models.StatusOffline,
		NewStatus:   models.StatusOnline,
		UserID:      "user-1",
		Timestamp:   time.Now(),
	}

	payload := notifier.buildSlackPayload(event)

	attachments := payload["attachments"].([]map[string]interface{})
	attachment := attachments[0]

	// Check color is green
	color := attachment["color"].(string)
	if color != "#57F287" {
		t.Errorf("Expected green color '#57F287', got %s", color)
	}
}

func TestWebhookNotifier_BuildPayload_CorrectFormat(t *testing.T) {
	notifier := &WebhookNotifier{}

	event := NotificationEvent{
		ServiceID:   "service-123",
		ServiceName: "My Service",
		ServiceURL:  "https://myservice.com",
		OldStatus:   models.StatusOnline,
		NewStatus:   models.StatusOffline,
		UserID:      "user-1",
		Timestamp:   time.Now(),
	}

	tests := []struct {
		format      string
		checkKey    string
		description string
	}{
		{models.WebhookFormatGeneric, "event", "Generic format should have 'event' key"},
		{models.WebhookFormatDiscord, "embeds", "Discord format should have 'embeds' key"},
		{models.WebhookFormatSlack, "attachments", "Slack format should have 'attachments' key"},
		{"unknown", "event", "Unknown format should default to generic"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			payload := notifier.buildPayload(tt.format, event)

			payloadMap, ok := payload.(map[string]interface{})
			if !ok {
				t.Fatal("Expected payload to be map")
			}

			if _, exists := payloadMap[tt.checkKey]; !exists {
				t.Errorf("%s: expected key '%s' in payload", tt.description, tt.checkKey)
			}
		})
	}
}

func TestWebhookNotifier_ShouldTrigger(t *testing.T) {
	notifier := &WebhookNotifier{}

	tests := []struct {
		name     string
		webhook  *models.Webhook
		event    NotificationEvent
		expected bool
	}{
		{
			name: "Trigger on offline when enabled",
			webhook: &models.Webhook{
				Triggers: models.WebhookTriggers{OnOffline: true, OnOnline: false},
			},
			event:    NotificationEvent{NewStatus: models.StatusOffline},
			expected: true,
		},
		{
			name: "No trigger on offline when disabled",
			webhook: &models.Webhook{
				Triggers: models.WebhookTriggers{OnOffline: false, OnOnline: true},
			},
			event:    NotificationEvent{NewStatus: models.StatusOffline},
			expected: false,
		},
		{
			name: "Trigger on online when enabled",
			webhook: &models.Webhook{
				Triggers: models.WebhookTriggers{OnOffline: false, OnOnline: true},
			},
			event:    NotificationEvent{NewStatus: models.StatusOnline},
			expected: true,
		},
		{
			name: "No trigger on online when disabled",
			webhook: &models.Webhook{
				Triggers: models.WebhookTriggers{OnOffline: true, OnOnline: false},
			},
			event:    NotificationEvent{NewStatus: models.StatusOnline},
			expected: false,
		},
		{
			name: "Trigger on both when both enabled",
			webhook: &models.Webhook{
				Triggers: models.WebhookTriggers{OnOffline: true, OnOnline: true},
			},
			event:    NotificationEvent{NewStatus: models.StatusOffline},
			expected: true,
		},
		{
			name: "No trigger when both disabled",
			webhook: &models.Webhook{
				Triggers: models.WebhookTriggers{OnOffline: false, OnOnline: false},
			},
			event:    NotificationEvent{NewStatus: models.StatusOffline},
			expected: false,
		},
		{
			name: "No trigger for unknown status",
			webhook: &models.Webhook{
				Triggers: models.WebhookTriggers{OnOffline: true, OnOnline: true},
			},
			event:    NotificationEvent{NewStatus: models.StatusUnknown},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := notifier.shouldTrigger(tt.webhook, tt.event)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestWebhookNotifier_IsRateLimited(t *testing.T) {
	notifier := &WebhookNotifier{}

	webhookID := "webhook-test-rate-limit"
	serviceID := "service-test"
	status := "offline"

	// First call should not be rate limited
	if notifier.isRateLimited(webhookID, serviceID, status) {
		t.Error("First call should not be rate limited")
	}

	// Immediate second call should be rate limited
	if !notifier.isRateLimited(webhookID, serviceID, status) {
		t.Error("Immediate second call should be rate limited")
	}

	// Different service should not be rate limited
	if notifier.isRateLimited(webhookID, "other-service", status) {
		t.Error("Different service should not be rate limited")
	}

	// Different status should not be rate limited
	if notifier.isRateLimited(webhookID, serviceID, "online") {
		t.Error("Different status should not be rate limited")
	}

	// Different webhook should not be rate limited
	if notifier.isRateLimited("other-webhook", serviceID, status) {
		t.Error("Different webhook should not be rate limited")
	}
}

func TestCleanupRateLimitCache(t *testing.T) {
	// Clear the cache first
	rateLimitMu.Lock()
	rateLimitCache = make(map[rateLimitKey]time.Time)
	rateLimitMu.Unlock()

	// Add some entries
	notifier := &WebhookNotifier{}
	notifier.isRateLimited("webhook-1", "service-1", "offline")
	notifier.isRateLimited("webhook-2", "service-2", "online")

	// Cache should have 2 entries
	rateLimitMu.RLock()
	initialCount := len(rateLimitCache)
	rateLimitMu.RUnlock()

	if initialCount != 2 {
		t.Errorf("Expected 2 entries in cache, got %d", initialCount)
	}

	// Cleanup should remove nothing (entries are fresh)
	removed := CleanupRateLimitCache()
	if removed != 0 {
		t.Errorf("Expected 0 entries removed, got %d", removed)
	}
}

func TestWebhookNotifier_PayloadIsValidJSON(t *testing.T) {
	notifier := &WebhookNotifier{}

	event := NotificationEvent{
		ServiceID:   "service-123",
		ServiceName: "Test Service",
		ServiceURL:  "https://example.com",
		OldStatus:   models.StatusOnline,
		NewStatus:   models.StatusOffline,
		UserID:      "user-1",
		Timestamp:   time.Now(),
	}

	formats := []string{
		models.WebhookFormatGeneric,
		models.WebhookFormatDiscord,
		models.WebhookFormatSlack,
	}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			payload := notifier.buildPayload(format, event)

			// Should be valid JSON
			data, err := json.Marshal(payload)
			if err != nil {
				t.Errorf("Failed to marshal %s payload: %v", format, err)
			}

			// Should be able to unmarshal back
			var decoded interface{}
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Errorf("Failed to unmarshal %s payload: %v", format, err)
			}
		})
	}
}

func TestWebhookNotifier_DiscordFieldsStructure(t *testing.T) {
	notifier := &WebhookNotifier{}

	event := NotificationEvent{
		ServiceID:   "service-123",
		ServiceName: "Test Service",
		ServiceURL:  "https://example.com",
		OldStatus:   models.StatusOnline,
		NewStatus:   models.StatusOffline,
		UserID:      "user-1",
		Timestamp:   time.Now(),
	}

	payload := notifier.buildDiscordPayload(event)
	embeds := payload["embeds"].([]map[string]interface{})
	embed := embeds[0]
	fields := embed["fields"].([]map[string]interface{})

	// Should have Status and Service fields
	if len(fields) < 2 {
		t.Errorf("Expected at least 2 fields, got %d", len(fields))
	}

	// Check field structure
	for _, field := range fields {
		if _, ok := field["name"]; !ok {
			t.Error("Field missing 'name'")
		}
		if _, ok := field["value"]; !ok {
			t.Error("Field missing 'value'")
		}
		if _, ok := field["inline"]; !ok {
			t.Error("Field missing 'inline'")
		}
	}
}

func TestWebhookNotifier_SlackBlocksStructure(t *testing.T) {
	notifier := &WebhookNotifier{}

	event := NotificationEvent{
		ServiceID:   "service-123",
		ServiceName: "Test Service",
		ServiceURL:  "https://example.com",
		OldStatus:   models.StatusOnline,
		NewStatus:   models.StatusOffline,
		UserID:      "user-1",
		Timestamp:   time.Now(),
	}

	payload := notifier.buildSlackPayload(event)
	attachments := payload["attachments"].([]map[string]interface{})
	blocks := attachments[0]["blocks"].([]map[string]interface{})

	// Should have at least header, divider, and context blocks
	if len(blocks) < 3 {
		t.Errorf("Expected at least 3 blocks, got %d", len(blocks))
	}

	// Check first block is section with accessory button
	firstBlock := blocks[0]
	if firstBlock["type"] != "section" {
		t.Errorf("Expected first block type 'section', got %v", firstBlock["type"])
	}
	if _, ok := firstBlock["accessory"]; !ok {
		t.Error("First block should have accessory (button)")
	}

	// Check there's a divider
	hasDivider := false
	for _, block := range blocks {
		if block["type"] == "divider" {
			hasDivider = true
			break
		}
	}
	if !hasDivider {
		t.Error("Expected a divider block")
	}

	// Check there's a context block
	hasContext := false
	for _, block := range blocks {
		if block["type"] == "context" {
			hasContext = true
			break
		}
	}
	if !hasContext {
		t.Error("Expected a context block")
	}
}

// retryTestServiceRepo allows configuring GetByID responses for retry tests
type retryTestServiceRepo struct {
	MockServiceRepository
	status string
}

func (m *retryTestServiceRepo) GetByID(_ context.Context, id string) (*models.Service, error) {
	return &models.Service{ID: id, Status: m.status}, nil
}

func clearRetryPending() {
	retryPendingMu.Lock()
	retryPending = make(map[retryPendingKey]bool)
	retryPendingMu.Unlock()
}

func isKeyPending(webhookID, serviceID string) bool {
	retryPendingMu.Lock()
	defer retryPendingMu.Unlock()
	return retryPending[retryPendingKey{WebhookID: webhookID, ServiceID: serviceID}]
}

// setupWebhookNotifierTestDB creates an in-memory SQLite database for webhook notifier tests
func setupWebhookNotifierTestDB(t *testing.T) (*sql.DB, *repository.WebhookRepository) {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	schema := `
		CREATE TABLE IF NOT EXISTS webhooks (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			user_id TEXT NOT NULL, name TEXT NOT NULL, url TEXT NOT NULL,
			enabled INTEGER NOT NULL DEFAULT 1,
			triggers TEXT NOT NULL DEFAULT '{"on_offline":true,"on_online":false}',
			format TEXT NOT NULL DEFAULT 'generic',
			retry_count INTEGER NOT NULL DEFAULT 0,
			retry_delay_seconds INTEGER NOT NULL DEFAULT 30,
			last_triggered_at TIMESTAMP, last_success_at TIMESTAMP,
			consecutive_failures INTEGER NOT NULL DEFAULT 0,
			total_sent INTEGER NOT NULL DEFAULT 0, total_failed INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL, updated_at TIMESTAMP NOT NULL
		);
		CREATE TABLE IF NOT EXISTS webhook_logs (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			webhook_id TEXT NOT NULL, service_id TEXT NOT NULL, service_name TEXT NOT NULL,
			old_status TEXT NOT NULL, new_status TEXT NOT NULL,
			success INTEGER NOT NULL, status_code INTEGER,
			error_message TEXT, response_time_ms INTEGER,
			created_at TIMESTAMP NOT NULL
		);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create test tables: %v", err)
	}
	return db, repository.NewWebhookRepository(db)
}

func TestNotifyWithRetry_RecoveryKeepsPending(t *testing.T) {
	clearRetryPending()

	serviceRepo := &retryTestServiceRepo{status: models.StatusOnline}
	notifier := &WebhookNotifier{serviceRepo: serviceRepo}

	webhook := &models.Webhook{
		ID: "wh-recovery", Name: "Test", RetryCount: 1, RetryDelaySeconds: 0,
	}
	event := NotificationEvent{
		ServiceID: "svc-1", ServiceName: "Test Service",
		NewStatus: models.StatusOffline,
	}

	notifier.notifyWithRetry(context.Background(), webhook, event)

	// Key should remain pending — Notify() will clear it when the online event arrives
	if !isKeyPending("wh-recovery", "svc-1") {
		t.Error("Expected retry pending key to remain after recovery")
	}
}

func TestNotifyWithRetry_ConfirmedOfflineClearsPending(t *testing.T) {
	clearRetryPending()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	db, webhookRepo := setupWebhookNotifierTestDB(t)
	defer db.Close()

	serviceRepo := &retryTestServiceRepo{status: models.StatusOffline}
	notifier := &WebhookNotifier{
		webhookRepo: webhookRepo,
		serviceRepo: serviceRepo,
		httpClient:  &http.Client{Timeout: 5 * time.Second},
	}

	webhook := &models.Webhook{
		ID: "wh-confirmed", UserID: "user-1", Name: "Test", URL: server.URL,
		Format: models.WebhookFormatGeneric, RetryCount: 1, RetryDelaySeconds: 0,
	}
	event := NotificationEvent{
		ServiceID: "svc-1", ServiceName: "Test Service",
		OldStatus: models.StatusOnline, NewStatus: models.StatusOffline,
		UserID: "user-1", Timestamp: time.Now(),
	}

	notifier.notifyWithRetry(context.Background(), webhook, event)

	// Key should be cleared after confirmed offline and notification sent
	if isKeyPending("wh-confirmed", "svc-1") {
		t.Error("Expected retry pending key to be cleared after confirmed offline")
	}
}

func TestNotifyWithRetry_CancelledClearsPending(t *testing.T) {
	clearRetryPending()

	serviceRepo := &retryTestServiceRepo{status: models.StatusOffline}
	notifier := &WebhookNotifier{serviceRepo: serviceRepo}

	webhook := &models.Webhook{
		ID: "wh-cancel", Name: "Test", RetryCount: 3, RetryDelaySeconds: 60,
	}
	event := NotificationEvent{
		ServiceID: "svc-1", ServiceName: "Test Service",
		NewStatus: models.StatusOffline,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	notifier.notifyWithRetry(ctx, webhook, event)

	if isKeyPending("wh-cancel", "svc-1") {
		t.Error("Expected retry pending key to be cleared after cancellation")
	}
}

func TestNotify_SuppressesOnlineWhenRetryPending(t *testing.T) {
	clearRetryPending()

	db, webhookRepo := setupWebhookNotifierTestDB(t)
	defer db.Close()

	now := time.Now()
	webhook := &models.Webhook{
		UserID: "user-suppress", Name: "Test Webhook", URL: "https://example.com/hook",
		Enabled: true, Format: models.WebhookFormatGeneric,
		Triggers:  models.WebhookTriggers{OnOffline: true, OnOnline: true},
		CreatedAt: now, UpdatedAt: now,
	}
	if err := webhookRepo.Create(context.Background(), webhook); err != nil {
		t.Fatalf("Failed to create webhook: %v", err)
	}

	serviceRepo := &retryTestServiceRepo{status: models.StatusOnline}
	notifier := NewWebhookNotifier(webhookRepo, serviceRepo)

	// Simulate a pending retry for this webhook/service
	retryPendingMu.Lock()
	retryPending[retryPendingKey{WebhookID: webhook.ID, ServiceID: "svc-1"}] = true
	retryPendingMu.Unlock()

	event := NotificationEvent{
		ServiceID: "svc-1", ServiceName: "Test Service",
		OldStatus: models.StatusOffline, NewStatus: models.StatusOnline,
		UserID: "user-suppress", Timestamp: time.Now(),
	}

	if err := notifier.Notify(context.Background(), event); err != nil {
		t.Fatalf("Notify returned error: %v", err)
	}

	// Pending key should be cleared by the suppression logic
	if isKeyPending(webhook.ID, "svc-1") {
		t.Error("Expected pending key to be cleared after online suppression")
	}
}

func TestNotify_AllowsOnlineWhenNotPending(t *testing.T) {
	clearRetryPending()

	received := make(chan bool, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	db, webhookRepo := setupWebhookNotifierTestDB(t)
	defer db.Close()

	now := time.Now()
	webhook := &models.Webhook{
		UserID: "user-allow", Name: "Test Webhook", URL: server.URL,
		Enabled: true, Format: models.WebhookFormatGeneric,
		Triggers:  models.WebhookTriggers{OnOffline: true, OnOnline: true},
		CreatedAt: now, UpdatedAt: now,
	}
	if err := webhookRepo.Create(context.Background(), webhook); err != nil {
		t.Fatalf("Failed to create webhook: %v", err)
	}

	serviceRepo := &retryTestServiceRepo{status: models.StatusOnline}
	notifier := NewWebhookNotifier(webhookRepo, serviceRepo)

	// No pending retry — online notification should go through
	event := NotificationEvent{
		ServiceID: "svc-1", ServiceName: "Test Service",
		OldStatus: models.StatusOffline, NewStatus: models.StatusOnline,
		UserID: "user-allow", Timestamp: time.Now(),
	}

	if err := notifier.Notify(context.Background(), event); err != nil {
		t.Fatalf("Notify returned error: %v", err)
	}

	select {
	case <-received:
		// Webhook was sent
	case <-time.After(5 * time.Second):
		t.Error("Expected online webhook to be sent when no retry is pending")
	}
}

func TestNotify_RetryBypassesRateLimit(t *testing.T) {
	clearRetryPending()

	// Clear rate limit cache
	rateLimitMu.Lock()
	rateLimitCache = make(map[rateLimitKey]time.Time)
	rateLimitMu.Unlock()

	db, webhookRepo := setupWebhookNotifierTestDB(t)
	defer db.Close()

	now := time.Now()
	webhook := &models.Webhook{
		UserID: "user-bypass", Name: "Test Webhook", URL: "https://example.com/hook",
		Enabled: true, Format: models.WebhookFormatGeneric,
		Triggers:   models.WebhookTriggers{OnOffline: true, OnOnline: true},
		RetryCount: 1, RetryDelaySeconds: 0,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := webhookRepo.Create(context.Background(), webhook); err != nil {
		t.Fatalf("Failed to create webhook: %v", err)
	}

	// Pre-fill rate limit for this webhook/service/offline combo
	rateLimitMu.Lock()
	rateLimitCache[rateLimitKey{WebhookID: webhook.ID, ServiceID: "svc-1", Status: models.StatusOffline}] = time.Now()
	rateLimitMu.Unlock()

	serviceRepo := &retryTestServiceRepo{status: models.StatusOnline}
	notifier := NewWebhookNotifier(webhookRepo, serviceRepo)

	event := NotificationEvent{
		ServiceID: "svc-1", ServiceName: "Test Service",
		OldStatus: models.StatusOnline, NewStatus: models.StatusOffline,
		UserID: "user-bypass", Timestamp: time.Now(),
	}

	if err := notifier.Notify(context.Background(), event); err != nil {
		t.Fatalf("Notify returned error: %v", err)
	}

	// Give the goroutine a moment to run
	time.Sleep(50 * time.Millisecond)

	// Retry should have started despite rate limit (key set then cleared on recovery)
	// Service recovered so key stays pending for Notify to clean up
	if !isKeyPending(webhook.ID, "svc-1") {
		t.Error("Expected retry to start despite rate limit (pending key should exist)")
	}
}

func TestNotify_SkipsDuplicateRetry(t *testing.T) {
	clearRetryPending()

	db, webhookRepo := setupWebhookNotifierTestDB(t)
	defer db.Close()

	now := time.Now()
	webhook := &models.Webhook{
		UserID: "user-dup", Name: "Test Webhook", URL: "https://example.com/hook",
		Enabled: true, Format: models.WebhookFormatGeneric,
		Triggers:   models.WebhookTriggers{OnOffline: true, OnOnline: true},
		RetryCount: 2, RetryDelaySeconds: 60,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := webhookRepo.Create(context.Background(), webhook); err != nil {
		t.Fatalf("Failed to create webhook: %v", err)
	}

	// Simulate an already-running retry
	retryPendingMu.Lock()
	retryPending[retryPendingKey{WebhookID: webhook.ID, ServiceID: "svc-1"}] = true
	retryPendingMu.Unlock()

	serviceRepo := &retryTestServiceRepo{status: models.StatusOffline}
	notifier := NewWebhookNotifier(webhookRepo, serviceRepo)

	event := NotificationEvent{
		ServiceID: "svc-1", ServiceName: "Test Service",
		OldStatus: models.StatusOnline, NewStatus: models.StatusOffline,
		UserID: "user-dup", Timestamp: time.Now(),
	}

	if err := notifier.Notify(context.Background(), event); err != nil {
		t.Fatalf("Notify returned error: %v", err)
	}

	// Give time for any goroutine to run (there shouldn't be one)
	time.Sleep(50 * time.Millisecond)

	// Key should still be pending (original retry untouched, no duplicate spawned)
	if !isKeyPending(webhook.ID, "svc-1") {
		t.Error("Expected pending key to remain unchanged (no duplicate retry)")
	}
}

func TestRedactWebhookURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "Discord webhook URL",
			url:      "https://discord.com/api/webhooks/123456789/abcdefghijk",
			expected: "discord.com",
		},
		{
			name:     "Slack webhook URL",
			url:      "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXX",
			expected: "hooks.slack.com",
		},
		{
			name:     "Generic webhook URL",
			url:      "https://example.com:8080/webhook?token=secret123",
			expected: "example.com:8080",
		},
		{
			name:     "HTTP localhost",
			url:      "http://localhost:3000/notify",
			expected: "localhost:3000",
		},
		{
			name:     "Invalid URL",
			url:      "not-a-valid-url",
			expected: "<invalid-url>",
		},
		{
			name:     "URL with userinfo",
			url:      "https://user:pass@example.com/webhook",
			expected: "example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redactWebhookURL(tt.url)
			if result != tt.expected {
				t.Errorf("redactWebhookURL(%q) = %q, expected %q", tt.url, result, tt.expected)
			}
		})
	}
}
