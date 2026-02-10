package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/nimbus/backend/internal/models"
)

type WebhookRepository struct {
	db *sql.DB
}

func NewWebhookRepository(sqlDB *sql.DB) *WebhookRepository {
	return &WebhookRepository{db: sqlDB}
}

// Create creates a new webhook
func (r *WebhookRepository) Create(ctx context.Context, webhook *models.Webhook) error {
	triggersJSON, err := json.Marshal(webhook.Triggers)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO webhooks (user_id, name, url, enabled, triggers, format, retry_count, retry_delay_seconds, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`

	webhook.CreatedAt = time.Now()
	webhook.UpdatedAt = webhook.CreatedAt

	err = r.db.QueryRowContext(
		ctx,
		query,
		webhook.UserID,
		webhook.Name,
		webhook.URL,
		webhook.Enabled,
		triggersJSON,
		webhook.Format,
		webhook.RetryCount,
		webhook.RetryDelaySeconds,
		webhook.CreatedAt,
		webhook.UpdatedAt,
	).Scan(&webhook.ID)

	return err
}

// GetByID retrieves a webhook by ID, scoped to the user
func (r *WebhookRepository) GetByID(ctx context.Context, id, userID string) (*models.Webhook, error) {
	webhook := &models.Webhook{}
	var triggersJSON []byte

	query := `
		SELECT id, user_id, name, url, enabled, triggers, format,
			   retry_count, retry_delay_seconds,
			   last_triggered_at, last_success_at, consecutive_failures,
			   total_sent, total_failed, created_at, updated_at
		FROM webhooks
		WHERE id = $1 AND user_id = $2
	`

	err := r.db.QueryRowContext(ctx, query, id, userID).Scan(
		&webhook.ID,
		&webhook.UserID,
		&webhook.Name,
		&webhook.URL,
		&webhook.Enabled,
		&triggersJSON,
		&webhook.Format,
		&webhook.RetryCount,
		&webhook.RetryDelaySeconds,
		&webhook.LastTriggeredAt,
		&webhook.LastSuccessAt,
		&webhook.ConsecutiveFailures,
		&webhook.TotalSent,
		&webhook.TotalFailed,
		&webhook.CreatedAt,
		&webhook.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(triggersJSON, &webhook.Triggers); err != nil {
		return nil, err
	}

	return webhook, nil
}

// GetAllByUserID retrieves all webhooks for a specific user
func (r *WebhookRepository) GetAllByUserID(ctx context.Context, userID string) ([]*models.Webhook, error) {
	query := `
		SELECT id, user_id, name, url, enabled, triggers, format,
			   retry_count, retry_delay_seconds,
			   last_triggered_at, last_success_at, consecutive_failures,
			   total_sent, total_failed, created_at, updated_at
		FROM webhooks
		WHERE user_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var webhooks []*models.Webhook
	for rows.Next() {
		webhook := &models.Webhook{}
		var triggersJSON []byte

		err := rows.Scan(
			&webhook.ID,
			&webhook.UserID,
			&webhook.Name,
			&webhook.URL,
			&webhook.Enabled,
			&triggersJSON,
			&webhook.Format,
			&webhook.RetryCount,
			&webhook.RetryDelaySeconds,
			&webhook.LastTriggeredAt,
			&webhook.LastSuccessAt,
			&webhook.ConsecutiveFailures,
			&webhook.TotalSent,
			&webhook.TotalFailed,
			&webhook.CreatedAt,
			&webhook.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(triggersJSON, &webhook.Triggers); err != nil {
			return nil, err
		}

		webhooks = append(webhooks, webhook)
	}

	return webhooks, rows.Err()
}

// GetEnabledByUserID retrieves all enabled webhooks for a user (for notification service)
func (r *WebhookRepository) GetEnabledByUserID(ctx context.Context, userID string) ([]*models.Webhook, error) {
	query := `
		SELECT id, user_id, name, url, enabled, triggers, format,
			   retry_count, retry_delay_seconds,
			   last_triggered_at, last_success_at, consecutive_failures,
			   total_sent, total_failed, created_at, updated_at
		FROM webhooks
		WHERE user_id = $1 AND enabled = true
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var webhooks []*models.Webhook
	for rows.Next() {
		webhook := &models.Webhook{}
		var triggersJSON []byte

		err := rows.Scan(
			&webhook.ID,
			&webhook.UserID,
			&webhook.Name,
			&webhook.URL,
			&webhook.Enabled,
			&triggersJSON,
			&webhook.Format,
			&webhook.RetryCount,
			&webhook.RetryDelaySeconds,
			&webhook.LastTriggeredAt,
			&webhook.LastSuccessAt,
			&webhook.ConsecutiveFailures,
			&webhook.TotalSent,
			&webhook.TotalFailed,
			&webhook.CreatedAt,
			&webhook.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(triggersJSON, &webhook.Triggers); err != nil {
			return nil, err
		}

		webhooks = append(webhooks, webhook)
	}

	return webhooks, rows.Err()
}

// CountByUserID returns the number of webhooks for a user (for limit check)
func (r *WebhookRepository) CountByUserID(ctx context.Context, userID string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM webhooks WHERE user_id = $1`
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&count)
	return count, err
}

// Update updates an existing webhook
func (r *WebhookRepository) Update(ctx context.Context, webhook *models.Webhook) error {
	triggersJSON, err := json.Marshal(webhook.Triggers)
	if err != nil {
		return err
	}

	webhook.UpdatedAt = time.Now()

	query := `
		UPDATE webhooks
		SET name = $1, url = $2, enabled = $3, triggers = $4, format = $5,
			retry_count = $6, retry_delay_seconds = $7, updated_at = $8
		WHERE id = $9 AND user_id = $10
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		webhook.Name,
		webhook.URL,
		webhook.Enabled,
		triggersJSON,
		webhook.Format,
		webhook.RetryCount,
		webhook.RetryDelaySeconds,
		webhook.UpdatedAt,
		webhook.ID,
		webhook.UserID,
	)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// Delete deletes a webhook
func (r *WebhookRepository) Delete(ctx context.Context, id, userID string) error {
	query := `DELETE FROM webhooks WHERE id = $1 AND user_id = $2`

	result, err := r.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// RecordDeliverySuccess updates webhook stats after a successful delivery
func (r *WebhookRepository) RecordDeliverySuccess(ctx context.Context, webhookID string) error {
	query := `
		UPDATE webhooks
		SET last_triggered_at = $1, last_success_at = $1,
			consecutive_failures = 0, total_sent = total_sent + 1,
			updated_at = $1
		WHERE id = $2
	`
	_, err := r.db.ExecContext(ctx, query, time.Now(), webhookID)
	return err
}

// RecordDeliveryFailure updates webhook stats after a failed delivery
func (r *WebhookRepository) RecordDeliveryFailure(ctx context.Context, webhookID string) error {
	query := `
		UPDATE webhooks
		SET last_triggered_at = $1, consecutive_failures = consecutive_failures + 1,
			total_sent = total_sent + 1, total_failed = total_failed + 1,
			updated_at = $1
		WHERE id = $2
	`
	_, err := r.db.ExecContext(ctx, query, time.Now(), webhookID)
	return err
}

// CreateLog creates a webhook delivery log entry
func (r *WebhookRepository) CreateLog(ctx context.Context, log *models.WebhookLog) error {
	query := `
		INSERT INTO webhook_logs (webhook_id, service_id, service_name, old_status, new_status,
								  success, status_code, error_message, response_time_ms, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`

	log.CreatedAt = time.Now()

	return r.db.QueryRowContext(
		ctx,
		query,
		log.WebhookID,
		log.ServiceID,
		log.ServiceName,
		log.OldStatus,
		log.NewStatus,
		log.Success,
		log.StatusCode,
		log.ErrorMessage,
		log.ResponseTimeMs,
		log.CreatedAt,
	).Scan(&log.ID)
}

// GetLogsByWebhookID retrieves recent logs for a webhook
func (r *WebhookRepository) GetLogsByWebhookID(ctx context.Context, webhookID string, limit int) ([]*models.WebhookLog, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT id, webhook_id, service_id, service_name, old_status, new_status,
			   success, status_code, error_message, response_time_ms, created_at
		FROM webhook_logs
		WHERE webhook_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, webhookID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*models.WebhookLog
	for rows.Next() {
		log := &models.WebhookLog{}
		err := rows.Scan(
			&log.ID,
			&log.WebhookID,
			&log.ServiceID,
			&log.ServiceName,
			&log.OldStatus,
			&log.NewStatus,
			&log.Success,
			&log.StatusCode,
			&log.ErrorMessage,
			&log.ResponseTimeMs,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	return logs, rows.Err()
}

// DeleteLogsOlderThan deletes logs older than the given cutoff time
func (r *WebhookRepository) DeleteLogsOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	query := `DELETE FROM webhook_logs WHERE created_at < $1`
	result, err := r.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
