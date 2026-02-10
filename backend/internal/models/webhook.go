package models

import "time"

// Webhook format constants
const (
	WebhookFormatGeneric = "generic"
	WebhookFormatDiscord = "discord"
	WebhookFormatSlack   = "slack"
)

// IsValidWebhookFormat checks if the given format is valid
func IsValidWebhookFormat(format string) bool {
	switch format {
	case WebhookFormatGeneric, WebhookFormatDiscord, WebhookFormatSlack:
		return true
	}
	return false
}

// WebhookTriggers defines what events trigger the webhook
type WebhookTriggers struct {
	OnOffline bool `json:"on_offline"`
	OnOnline  bool `json:"on_online"`
}

// Webhook represents a user's webhook configuration
type Webhook struct {
	ID                  string          `json:"id" db:"id"`
	UserID              string          `json:"user_id" db:"user_id"`
	Name                string          `json:"name" db:"name"`
	URL                 string          `json:"url" db:"url"`
	Enabled             bool            `json:"enabled" db:"enabled"`
	Triggers            WebhookTriggers `json:"triggers" db:"triggers"`
	Format              string          `json:"format" db:"format"`
	RetryCount          int             `json:"retry_count" db:"retry_count"`
	RetryDelaySeconds   int             `json:"retry_delay_seconds" db:"retry_delay_seconds"`
	LastTriggeredAt     *time.Time      `json:"last_triggered_at" db:"last_triggered_at"`
	LastSuccessAt       *time.Time      `json:"last_success_at" db:"last_success_at"`
	ConsecutiveFailures int             `json:"consecutive_failures" db:"consecutive_failures"`
	TotalSent           int             `json:"total_sent" db:"total_sent"`
	TotalFailed         int             `json:"total_failed" db:"total_failed"`
	CreatedAt           time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at" db:"updated_at"`
}

// WebhookCreateRequest for creating a new webhook
type WebhookCreateRequest struct {
	Name              string           `json:"name" validate:"required,max=100"`
	URL               string           `json:"url" validate:"required,url"`
	Enabled           *bool            `json:"enabled"`
	Triggers          *WebhookTriggers `json:"triggers"`
	Format            string           `json:"format" validate:"omitempty,oneof=generic discord slack"`
	RetryCount        *int             `json:"retry_count" validate:"omitempty,min=0,max=5"`
	RetryDelaySeconds *int             `json:"retry_delay_seconds" validate:"omitempty,min=10,max=300"`
}

// WebhookUpdateRequest for updating a webhook
type WebhookUpdateRequest struct {
	Name              *string          `json:"name" validate:"omitempty,max=100"`
	URL               *string          `json:"url" validate:"omitempty,url"`
	Enabled           *bool            `json:"enabled"`
	Triggers          *WebhookTriggers `json:"triggers"`
	Format            *string          `json:"format" validate:"omitempty,oneof=generic discord slack"`
	RetryCount        *int             `json:"retry_count" validate:"omitempty,min=0,max=5"`
	RetryDelaySeconds *int             `json:"retry_delay_seconds" validate:"omitempty,min=10,max=300"`
}

// WebhookResponse is the safe data to return to clients
type WebhookResponse struct {
	ID                  string          `json:"id"`
	Name                string          `json:"name"`
	URL                 string          `json:"url"`
	Enabled             bool            `json:"enabled"`
	Triggers            WebhookTriggers `json:"triggers"`
	Format              string          `json:"format"`
	RetryCount          int             `json:"retry_count"`
	RetryDelaySeconds   int             `json:"retry_delay_seconds"`
	LastTriggeredAt     *time.Time      `json:"last_triggered_at,omitempty"`
	LastSuccessAt       *time.Time      `json:"last_success_at,omitempty"`
	ConsecutiveFailures int             `json:"consecutive_failures"`
	TotalSent           int             `json:"total_sent"`
	TotalFailed         int             `json:"total_failed"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

// ToResponse converts Webhook to WebhookResponse
func (w *Webhook) ToResponse() WebhookResponse {
	return WebhookResponse{
		ID:                  w.ID,
		Name:                w.Name,
		URL:                 w.URL,
		Enabled:             w.Enabled,
		Triggers:            w.Triggers,
		Format:              w.Format,
		RetryCount:          w.RetryCount,
		RetryDelaySeconds:   w.RetryDelaySeconds,
		LastTriggeredAt:     w.LastTriggeredAt,
		LastSuccessAt:       w.LastSuccessAt,
		ConsecutiveFailures: w.ConsecutiveFailures,
		TotalSent:           w.TotalSent,
		TotalFailed:         w.TotalFailed,
		CreatedAt:           w.CreatedAt,
		UpdatedAt:           w.UpdatedAt,
	}
}

// WebhookLog represents a delivery attempt record
type WebhookLog struct {
	ID             string    `json:"id" db:"id"`
	WebhookID      string    `json:"webhook_id" db:"webhook_id"`
	ServiceID      string    `json:"service_id" db:"service_id"`
	ServiceName    string    `json:"service_name" db:"service_name"`
	OldStatus      string    `json:"old_status" db:"old_status"`
	NewStatus      string    `json:"new_status" db:"new_status"`
	Success        bool      `json:"success" db:"success"`
	StatusCode     *int      `json:"status_code" db:"status_code"`
	ErrorMessage   *string   `json:"error_message" db:"error_message"`
	ResponseTimeMs *int      `json:"response_time_ms" db:"response_time_ms"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// WebhookLogResponse is the safe data to return to clients
type WebhookLogResponse struct {
	ID             string    `json:"id"`
	WebhookID      string    `json:"webhook_id"`
	ServiceID      string    `json:"service_id"`
	ServiceName    string    `json:"service_name"`
	OldStatus      string    `json:"old_status"`
	NewStatus      string    `json:"new_status"`
	Success        bool      `json:"success"`
	StatusCode     *int      `json:"status_code,omitempty"`
	ErrorMessage   *string   `json:"error_message,omitempty"`
	ResponseTimeMs *int      `json:"response_time_ms,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// ToResponse converts WebhookLog to WebhookLogResponse
func (l *WebhookLog) ToResponse() WebhookLogResponse {
	return WebhookLogResponse{
		ID:             l.ID,
		WebhookID:      l.WebhookID,
		ServiceID:      l.ServiceID,
		ServiceName:    l.ServiceName,
		OldStatus:      l.OldStatus,
		NewStatus:      l.NewStatus,
		Success:        l.Success,
		StatusCode:     l.StatusCode,
		ErrorMessage:   l.ErrorMessage,
		ResponseTimeMs: l.ResponseTimeMs,
		CreatedAt:      l.CreatedAt,
	}
}

// WebhookTestResult represents the result of a webhook test
type WebhookTestResult struct {
	Success        bool   `json:"success"`
	Message        string `json:"message,omitempty"`
	StatusCode     *int   `json:"status_code,omitempty"`
	ResponseTimeMs *int   `json:"response_time_ms,omitempty"`
	Error          string `json:"error,omitempty"`
}
