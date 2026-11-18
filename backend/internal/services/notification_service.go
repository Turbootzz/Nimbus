package services

import (
	"context"
	"log"
	"time"

	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/utils"
)

// NotificationEvent represents a service status change event
type NotificationEvent struct {
	ServiceID   string
	ServiceName string
	ServiceURL  string
	OldStatus   string
	NewStatus   string
	UserID      string
	Timestamp   time.Time
	ErrorMsg    *string
}

// NotificationService orchestrates all notification channels
type NotificationService struct {
	webhookNotifier *WebhookNotifier
}

// NewNotificationService creates a new notification service
func NewNotificationService(webhookRepo *repository.WebhookRepository, serviceRepo repository.ServiceRepositoryInterface) *NotificationService {
	return &NotificationService{
		webhookNotifier: NewWebhookNotifier(webhookRepo, serviceRepo),
	}
}

// SetCipher enables encrypted credentials for notification targets (ntfy,
// Gotify, Telegram, Pushover). Plain webhooks work without it.
func (n *NotificationService) SetCipher(cipher *utils.Cipher) {
	n.webhookNotifier.cipher = cipher
}

// EncryptWebhookCredentials returns the blob to store for a webhook's
// secrets, bound to its ID; nil when there are none.
func (n *NotificationService) EncryptWebhookCredentials(creds models.WebhookCredentials, webhookID string) ([]byte, error) {
	if creds == (models.WebhookCredentials{}) {
		return nil, nil
	}
	if n == nil {
		return nil, errNoCipher
	}
	return n.webhookNotifier.encryptCredentials(creds, webhookID)
}

// NotifyStatusChange sends notifications for a service status change
func (n *NotificationService) NotifyStatusChange(ctx context.Context, event NotificationEvent) {
	// Send webhook notifications
	if err := n.webhookNotifier.Notify(ctx, event); err != nil {
		log.Printf("Webhook notification error for service %s: %v", event.ServiceID, err)
	}
}

// TestWebhook sends a test notification to a specific webhook
func (n *NotificationService) TestWebhook(ctx context.Context, webhookID, userID string) (*TestWebhookResult, error) {
	return n.webhookNotifier.TestWebhook(ctx, webhookID, userID)
}

// TestWebhookResult contains the result of a webhook test
type TestWebhookResult struct {
	Success        bool
	StatusCode     int
	ResponseTimeMs int
	Error          string
}
