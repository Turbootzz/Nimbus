package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nimbus/backend/internal/models"
)

type WebhookRepository struct {
	db *sql.DB
}

func NewWebhookRepository(sqlDB *sql.DB) *WebhookRepository {
	return &WebhookRepository{db: sqlDB}
}

// webhookColumns is the column list every webhook read selects
const webhookColumns = `id, user_id, name, url, enabled, triggers, format, options, credentials_enc,
			   verify_tls, retry_count, retry_delay_seconds,
			   last_triggered_at, last_success_at, consecutive_failures,
			   total_sent, total_failed, created_at, updated_at`

// scanWebhook reads one row selected with webhookColumns
func scanWebhook(row interface{ Scan(dest ...any) error }) (*models.Webhook, error) {
	webhook := &models.Webhook{}
	var triggersJSON, optionsJSON []byte

	err := row.Scan(
		&webhook.ID,
		&webhook.UserID,
		&webhook.Name,
		&webhook.URL,
		&webhook.Enabled,
		&triggersJSON,
		&webhook.Format,
		&optionsJSON,
		&webhook.CredentialsEnc,
		&webhook.VerifyTLS,
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
	if err := json.Unmarshal(optionsJSON, &webhook.Options); err != nil {
		return nil, err
	}

	return webhook, nil
}

// queryWebhooks runs a query selecting webhookColumns and scans every row
func (r *WebhookRepository) queryWebhooks(ctx context.Context, query string, args ...any) ([]*models.Webhook, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var webhooks []*models.Webhook
	for rows.Next() {
		webhook, err := scanWebhook(rows)
		if err != nil {
			return nil, err
		}
		webhooks = append(webhooks, webhook)
	}

	return webhooks, rows.Err()
}

// Create creates a new webhook. A preset ID is kept, since encrypted
// credentials are bound to it.
func (r *WebhookRepository) Create(ctx context.Context, webhook *models.Webhook) error {
	triggersJSON, err := json.Marshal(webhook.Triggers)
	if err != nil {
		return err
	}
	optionsJSON, err := json.Marshal(webhook.Options)
	if err != nil {
		return err
	}

	if webhook.ID == "" {
		webhook.ID = uuid.New().String()
	}

	query := `
		INSERT INTO webhooks (id, user_id, name, url, enabled, triggers, format, options, credentials_enc,
							  verify_tls, retry_count, retry_delay_seconds, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	webhook.CreatedAt = time.Now()
	webhook.UpdatedAt = webhook.CreatedAt

	_, err = r.db.ExecContext(
		ctx,
		query,
		webhook.ID,
		webhook.UserID,
		webhook.Name,
		webhook.URL,
		webhook.Enabled,
		triggersJSON,
		webhook.Format,
		optionsJSON,
		nullableBytes(webhook.CredentialsEnc),
		webhook.VerifyTLS,
		webhook.RetryCount,
		webhook.RetryDelaySeconds,
		webhook.CreatedAt,
		webhook.UpdatedAt,
	)

	return err
}

// GetByID retrieves a webhook by ID, scoped to the user
func (r *WebhookRepository) GetByID(ctx context.Context, id, userID string) (*models.Webhook, error) {
	query := `
		SELECT ` + webhookColumns + `
		FROM webhooks
		WHERE id = $1 AND user_id = $2
	`
	return scanWebhook(r.db.QueryRowContext(ctx, query, id, userID))
}

// GetAllByUserID retrieves all webhooks for a specific user
func (r *WebhookRepository) GetAllByUserID(ctx context.Context, userID string) ([]*models.Webhook, error) {
	query := `
		SELECT ` + webhookColumns + `
		FROM webhooks
		WHERE user_id = $1
		ORDER BY created_at ASC
	`
	return r.queryWebhooks(ctx, query, userID)
}

// GetEnabledByUserID retrieves all enabled webhooks for a user (for notification service)
func (r *WebhookRepository) GetEnabledByUserID(ctx context.Context, userID string) ([]*models.Webhook, error) {
	query := `
		SELECT ` + webhookColumns + `
		FROM webhooks
		WHERE user_id = $1 AND enabled = true
		ORDER BY created_at ASC
	`
	return r.queryWebhooks(ctx, query, userID)
}

// CountByUserID returns the number of webhooks for a user (for limit check)
func (r *WebhookRepository) CountByUserID(ctx context.Context, userID string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM webhooks WHERE user_id = $1`
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&count)
	return count, err
}

// Update updates an existing webhook. The target (format, URL, options,
// verify_tls) and the credentials are only written when the caller changed
// them: an update of something else, like the enabled toggle, then can't put
// back a target or secret that another request replaced after this one read it.
func (r *WebhookRepository) Update(ctx context.Context, webhook *models.Webhook, writeTarget, writeCredentials bool) error {
	triggersJSON, err := json.Marshal(webhook.Triggers)
	if err != nil {
		return err
	}

	webhook.UpdatedAt = time.Now()

	set := "name = $1, enabled = $2, triggers = $3, retry_count = $4, retry_delay_seconds = $5, updated_at = $6"
	args := []interface{}{
		webhook.Name,
		webhook.Enabled,
		triggersJSON,
		webhook.RetryCount,
		webhook.RetryDelaySeconds,
		webhook.UpdatedAt,
	}
	// Placeholders are numbered in the order they appear
	add := func(column string, value interface{}) {
		args = append(args, value)
		set += fmt.Sprintf(", %s = $%d", column, len(args))
	}
	if writeTarget {
		optionsJSON, err := json.Marshal(webhook.Options)
		if err != nil {
			return err
		}
		add("url", webhook.URL)
		add("format", webhook.Format)
		add("options", optionsJSON)
		add("verify_tls", webhook.VerifyTLS)
	}
	if writeCredentials {
		add("credentials_enc", nullableBytes(webhook.CredentialsEnc))
	}
	args = append(args, webhook.ID, webhook.UserID)
	query := fmt.Sprintf(`UPDATE webhooks SET %s WHERE id = $%d AND user_id = $%d`, set, len(args)-1, len(args))

	result, err := r.db.ExecContext(ctx, query, args...)
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
