package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
)

// Rate limit: max 1 notification per service per status per webhook per 5 minutes
type rateLimitKey struct {
	WebhookID string
	ServiceID string
	Status    string
}

// Track services currently pending retry verification per webhook
type retryPendingKey struct {
	WebhookID string
	ServiceID string
}

// Package-level state shared across all WebhookNotifier instances.
// Tests that access these maps must NOT use t.Parallel().
var (
	rateLimitCache = make(map[rateLimitKey]time.Time)
	rateLimitMu    sync.RWMutex
	rateLimitTTL   = 5 * time.Minute

	retryPending   = make(map[retryPendingKey]bool)
	retryPendingMu sync.Mutex
)

// WebhookNotifier handles webhook delivery
type WebhookNotifier struct {
	webhookRepo *repository.WebhookRepository
	serviceRepo repository.ServiceRepositoryInterface
	httpClient  *http.Client
}

// NewWebhookNotifier creates a new webhook notifier
func NewWebhookNotifier(webhookRepo *repository.WebhookRepository, serviceRepo repository.ServiceRepositoryInterface) *WebhookNotifier {
	return &WebhookNotifier{
		webhookRepo: webhookRepo,
		serviceRepo: serviceRepo,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Notify sends notifications to all enabled webhooks for a user
func (w *WebhookNotifier) Notify(ctx context.Context, event NotificationEvent) error {
	webhooks, err := w.webhookRepo.GetEnabledByUserID(ctx, event.UserID)
	if err != nil {
		return err
	}

	for _, webhook := range webhooks {
		if !w.shouldTrigger(webhook, event) {
			continue
		}

		// Suppress online notifications if offline was never sent (still pending retry)
		if event.NewStatus == models.StatusOnline {
			key := retryPendingKey{WebhookID: webhook.ID, ServiceID: event.ServiceID}
			retryPendingMu.Lock()
			pending := retryPending[key]
			if pending {
				delete(retryPending, key)
			}
			retryPendingMu.Unlock()
			if pending {
				log.Printf("Webhook %s: suppressing online notification for %s (offline was never sent)", webhook.Name, event.ServiceName)
				continue
			}
		}

		// For offline events with retries, skip rate limit here (checked after verification)
		if event.NewStatus == models.StatusOffline && webhook.RetryCount > 0 {
			key := retryPendingKey{WebhookID: webhook.ID, ServiceID: event.ServiceID}
			retryPendingMu.Lock()
			if retryPending[key] {
				retryPendingMu.Unlock()
				continue
			}
			// Set pending while holding the lock to prevent duplicate goroutines
			retryPending[key] = true
			retryPendingMu.Unlock()
			go w.notifyWithRetry(context.Background(), webhook, event)
			continue
		}

		if w.isRateLimited(webhook.ID, event.ServiceID, event.NewStatus) {
			log.Printf("Webhook %s rate limited for service %s (status: %s)", webhook.ID, event.ServiceID, event.NewStatus)
			continue
		}

		go w.sendWebhook(context.Background(), webhook, event)
	}

	return nil
}

// notifyWithRetry waits and re-checks service status before sending offline notifications.
// This prevents false positive alerts from transient failures.
func (w *WebhookNotifier) notifyWithRetry(ctx context.Context, webhook *models.Webhook, event NotificationEvent) {
	key := retryPendingKey{WebhookID: webhook.ID, ServiceID: event.ServiceID}
	// retryPending[key] is already set by Notify() before spawning this goroutine

	delay := time.Duration(webhook.RetryDelaySeconds) * time.Second
	log.Printf("Webhook %s: service %s went offline, verifying with %d retries (delay: %ds)",
		webhook.Name, event.ServiceName, webhook.RetryCount, webhook.RetryDelaySeconds)

	for attempt := 1; attempt <= webhook.RetryCount; attempt++ {
		select {
		case <-ctx.Done():
			log.Printf("Webhook %s: retry cancelled for service %s", webhook.Name, event.ServiceName)
			retryPendingMu.Lock()
			delete(retryPending, key)
			retryPendingMu.Unlock()
			return
		case <-time.After(delay):
		}

		// Re-check service status from DB
		service, err := w.serviceRepo.GetByID(ctx, event.ServiceID)
		if err != nil {
			log.Printf("Webhook %s: retry %d/%d failed to fetch service %s: %v (continuing)",
				webhook.Name, attempt, webhook.RetryCount, event.ServiceID, err)
			continue
		}

		// Service recovered, skip notification
		if service.Status == models.StatusOnline {
			log.Printf("Webhook %s: service %s recovered on retry %d/%d, skipping notification",
				webhook.Name, event.ServiceName, attempt, webhook.RetryCount)
			// Don't clear pending here — Notify() will clear it when the online event arrives
			return
		}

		log.Printf("Webhook %s: service %s still offline on retry %d/%d",
			webhook.Name, event.ServiceName, attempt, webhook.RetryCount)
	}

	// Confirmed offline, clear pending and send
	retryPendingMu.Lock()
	delete(retryPending, key)
	retryPendingMu.Unlock()

	if w.isRateLimited(webhook.ID, event.ServiceID, event.NewStatus) {
		log.Printf("Webhook %s: service %s confirmed offline but rate limited, skipping",
			webhook.Name, event.ServiceName)
		return
	}

	log.Printf("Webhook %s: service %s confirmed offline after %d retries, sending notification",
		webhook.Name, event.ServiceName, webhook.RetryCount)
	w.sendWebhook(ctx, webhook, event)
}

// shouldTrigger checks if the event type should trigger the webhook
func (w *WebhookNotifier) shouldTrigger(webhook *models.Webhook, event NotificationEvent) bool {
	if event.NewStatus == models.StatusOffline && webhook.Triggers.OnOffline {
		return true
	}
	if event.NewStatus == models.StatusOnline && webhook.Triggers.OnOnline {
		return true
	}
	return false
}

// isRateLimited checks if this webhook/service/status combo is rate limited.
// Uses a single lock to prevent race conditions between check and update.
func (w *WebhookNotifier) isRateLimited(webhookID, serviceID, status string) bool {
	key := rateLimitKey{WebhookID: webhookID, ServiceID: serviceID, Status: status}

	rateLimitMu.Lock()
	defer rateLimitMu.Unlock()

	if lastSent, exists := rateLimitCache[key]; exists && time.Since(lastSent) < rateLimitTTL {
		return true
	}

	rateLimitCache[key] = time.Now()
	return false
}

// sendWebhook sends a single webhook notification
func (w *WebhookNotifier) sendWebhook(ctx context.Context, webhook *models.Webhook, event NotificationEvent) {
	startTime := time.Now()

	payload := w.buildPayload(webhook.Format, event)
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		w.recordResult(ctx, webhook, event, false, 0, time.Since(startTime), err.Error())
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook.URL, bytes.NewReader(payloadBytes))
	if err != nil {
		w.recordResult(ctx, webhook, event, false, 0, time.Since(startTime), err.Error())
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Nimbus-Webhook/1.0")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		w.recordResult(ctx, webhook, event, false, 0, time.Since(startTime), err.Error())
		return
	}
	defer resp.Body.Close()

	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	errMsg := ""
	if !success {
		errMsg = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	w.recordResult(ctx, webhook, event, success, resp.StatusCode, time.Since(startTime), errMsg)
}

// recordResult records the webhook delivery result
func (w *WebhookNotifier) recordResult(ctx context.Context, webhook *models.Webhook, event NotificationEvent, success bool, statusCode int, duration time.Duration, errMsg string) {
	responseTimeMs := int(duration.Milliseconds())

	// Update webhook stats
	if success {
		if err := w.webhookRepo.RecordDeliverySuccess(ctx, webhook.ID); err != nil {
			log.Printf("Failed to record webhook success: %v", err)
		}
	} else {
		if err := w.webhookRepo.RecordDeliveryFailure(ctx, webhook.ID); err != nil {
			log.Printf("Failed to record webhook failure: %v", err)
		}
	}

	// Create log entry
	logEntry := &models.WebhookLog{
		WebhookID:      webhook.ID,
		ServiceID:      event.ServiceID,
		ServiceName:    event.ServiceName,
		OldStatus:      event.OldStatus,
		NewStatus:      event.NewStatus,
		Success:        success,
		ResponseTimeMs: &responseTimeMs,
	}

	if statusCode > 0 {
		logEntry.StatusCode = &statusCode
	}

	if errMsg != "" {
		logEntry.ErrorMessage = &errMsg
	}

	if err := w.webhookRepo.CreateLog(ctx, logEntry); err != nil {
		log.Printf("Failed to create webhook log: %v", err)
	}

	// Log with redacted URL (host only) to avoid leaking secrets in webhook URLs
	redactedHost := redactWebhookURL(webhook.URL)
	if success {
		log.Printf("Webhook %s delivered successfully to %s in %dms", webhook.Name, redactedHost, responseTimeMs)
	} else {
		log.Printf("Webhook %s delivery failed to %s: %s", webhook.Name, redactedHost, errMsg)
	}
}

// redactWebhookURL extracts only the host from a webhook URL to avoid logging secrets
func redactWebhookURL(webhookURL string) string {
	parsed, err := url.Parse(webhookURL)
	if err != nil || parsed.Host == "" {
		return "<invalid-url>"
	}
	return parsed.Host
}

// buildPayload builds the webhook payload based on format
func (w *WebhookNotifier) buildPayload(format string, event NotificationEvent) interface{} {
	switch format {
	case models.WebhookFormatDiscord:
		return w.buildDiscordPayload(event)
	case models.WebhookFormatSlack:
		return w.buildSlackPayload(event)
	default:
		return w.buildGenericPayload(event)
	}
}

// buildGenericPayload creates a generic JSON payload
func (w *WebhookNotifier) buildGenericPayload(event NotificationEvent) map[string]interface{} {
	payload := map[string]interface{}{
		"event":        "service_status_changed",
		"service_id":   event.ServiceID,
		"service_name": event.ServiceName,
		"service_url":  event.ServiceURL,
		"old_status":   event.OldStatus,
		"new_status":   event.NewStatus,
		"timestamp":    event.Timestamp.Format(time.RFC3339),
	}

	if event.ErrorMsg != nil {
		payload["error"] = *event.ErrorMsg
	}

	return payload
}

// buildDiscordPayload creates a Discord webhook payload with embeds
func (w *WebhookNotifier) buildDiscordPayload(event NotificationEvent) map[string]interface{} {
	var color int
	var statusEmoji string
	var statusText string
	var description string

	if event.NewStatus == models.StatusOffline {
		color = 0xED4245 // Discord red
		statusEmoji = "🔴"
		statusText = "Offline"
		if event.ErrorMsg != nil {
			description = fmt.Sprintf("```%s```", *event.ErrorMsg)
		}
	} else {
		color = 0x57F287 // Discord green
		statusEmoji = "🟢"
		statusText = "Online"
		description = "Service is back up and responding normally."
	}

	title := fmt.Sprintf("%s %s is now %s", statusEmoji, event.ServiceName, strings.ToLower(statusText))

	embed := map[string]interface{}{
		"title":     title,
		"url":       event.ServiceURL,
		"color":     color,
		"timestamp": event.Timestamp.Format(time.RFC3339),
		"footer": map[string]interface{}{
			"text": "Nimbus Dashboard",
		},
	}

	if description != "" {
		embed["description"] = description
	}

	// Add fields for additional context
	fields := []map[string]interface{}{
		{"name": "Status", "value": fmt.Sprintf("`%s`", statusText), "inline": true},
		{"name": "Service", "value": fmt.Sprintf("[%s](%s)", event.ServiceName, event.ServiceURL), "inline": true},
	}

	embed["fields"] = fields

	return map[string]interface{}{
		"embeds": []map[string]interface{}{embed},
	}
}

// buildSlackPayload creates a Slack webhook payload with blocks
func (w *WebhookNotifier) buildSlackPayload(event NotificationEvent) map[string]interface{} {
	var emoji string
	var color string
	var statusText string

	if event.NewStatus == models.StatusOffline {
		emoji = ":red_circle:"
		color = "#ED4245" // Red
		statusText = "offline"
	} else {
		emoji = ":large_green_circle:"
		color = "#57F287" // Green
		statusText = "online"
	}

	headerText := fmt.Sprintf("%s *%s* is now %s", emoji, event.ServiceName, statusText)

	blocks := []map[string]interface{}{
		{
			"type": "section",
			"text": map[string]interface{}{
				"type": "mrkdwn",
				"text": headerText,
			},
			"accessory": map[string]interface{}{
				"type": "button",
				"text": map[string]interface{}{
					"type":  "plain_text",
					"text":  "View Service",
					"emoji": true,
				},
				"url": event.ServiceURL,
			},
		},
	}

	// Add error message or recovery message
	if event.NewStatus == models.StatusOffline && event.ErrorMsg != nil {
		blocks = append(blocks, map[string]interface{}{
			"type": "section",
			"text": map[string]interface{}{
				"type": "mrkdwn",
				"text": fmt.Sprintf("```%s```", *event.ErrorMsg),
			},
		})
	} else if event.NewStatus == models.StatusOnline {
		blocks = append(blocks, map[string]interface{}{
			"type": "section",
			"text": map[string]interface{}{
				"type": "mrkdwn",
				"text": "_Service is back up and responding normally._",
			},
		})
	}

	// Add divider and context
	blocks = append(blocks,
		map[string]interface{}{"type": "divider"},
		map[string]interface{}{
			"type": "context",
			"elements": []map[string]interface{}{
				{"type": "mrkdwn", "text": fmt.Sprintf("*Service:* <%s|%s>", event.ServiceURL, event.ServiceName)},
				{"type": "mrkdwn", "text": fmt.Sprintf("*Status:* `%s`", strings.ToUpper(statusText))},
				{"type": "mrkdwn", "text": fmt.Sprintf("*Time:* %s", event.Timestamp.Format(time.RFC822))},
			},
		},
	)

	// Use attachments for the color sidebar
	return map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"color":  color,
				"blocks": blocks,
			},
		},
	}
}

// TestWebhook sends a test notification to a webhook
func (w *WebhookNotifier) TestWebhook(ctx context.Context, webhookID, userID string) (*TestWebhookResult, error) {
	webhook, err := w.webhookRepo.GetByID(ctx, webhookID, userID)
	if err != nil {
		return nil, err
	}

	testEvent := NotificationEvent{
		ServiceID:   "test-service-id",
		ServiceName: "Test Service",
		ServiceURL:  "https://example.com",
		OldStatus:   models.StatusOnline,
		NewStatus:   models.StatusOffline,
		UserID:      userID,
		Timestamp:   time.Now(),
	}

	startTime := time.Now()

	payload := w.buildPayload(webhook.Format, testEvent)
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return &TestWebhookResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook.URL, bytes.NewReader(payloadBytes))
	if err != nil {
		return &TestWebhookResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Nimbus-Webhook/1.0")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return &TestWebhookResult{
			Success:        false,
			ResponseTimeMs: int(time.Since(startTime).Milliseconds()),
			Error:          err.Error(),
		}, nil
	}
	defer resp.Body.Close()

	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	result := &TestWebhookResult{
		Success:        success,
		StatusCode:     resp.StatusCode,
		ResponseTimeMs: int(time.Since(startTime).Milliseconds()),
	}

	if !success {
		result.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return result, nil
}

// CleanupRateLimitCache removes expired entries from the rate limit cache
func CleanupRateLimitCache() int {
	rateLimitMu.Lock()
	defer rateLimitMu.Unlock()

	now := time.Now()
	removed := 0
	for key, lastSent := range rateLimitCache {
		if now.Sub(lastSent) > rateLimitTTL {
			delete(rateLimitCache, key)
			removed++
		}
	}
	return removed
}
