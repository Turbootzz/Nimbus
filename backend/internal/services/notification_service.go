package services

import (
	"context"
	"log"
	"time"

	"github.com/nimbus/backend/internal/repository"
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
