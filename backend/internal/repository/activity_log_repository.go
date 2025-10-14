package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/nimbus/backend/internal/models"
)

type ActivityLogRepository struct {
	db *sql.DB
}

func NewActivityLogRepository(db *sql.DB) *ActivityLogRepository {
	return &ActivityLogRepository{db: db}
}

// Create logs a new activity
func (r *ActivityLogRepository) Create(ctx context.Context, log *models.UserActivityLog) error {
	// Convert details map to JSON
	detailsJSON, err := json.Marshal(log.Details)
	if err != nil {
		return fmt.Errorf("failed to marshal details: %w", err)
	}

	query := `
		INSERT INTO user_activity_logs (user_id, actor_id, action, details, ip_address, created_at)
		VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP)
	`

	_, err = r.db.ExecContext(ctx, query, log.UserID, log.ActorID, log.Action, detailsJSON, log.IPAddress)
	if err != nil {
		return fmt.Errorf("failed to create activity log: %w", err)
	}

	return nil
}

// GetByUserID retrieves activity logs for a specific user
func (r *ActivityLogRepository) GetByUserID(ctx context.Context, userID string, limit int) ([]*models.UserActivityLog, error) {
	if limit == 0 {
		limit = 50 // Default limit
	}

	query := `
		SELECT id, user_id, actor_id, action, details, ip_address, created_at
		FROM user_activity_logs
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity logs: %w", err)
	}
	defer rows.Close()

	var logs []*models.UserActivityLog
	for rows.Next() {
		log := &models.UserActivityLog{}
		var detailsJSON []byte

		err := rows.Scan(
			&log.ID,
			&log.UserID,
			&log.ActorID,
			&log.Action,
			&detailsJSON,
			&log.IPAddress,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan activity log: %w", err)
		}

		// Unmarshal details
		if err := json.Unmarshal(detailsJSON, &log.Details); err != nil {
			log.Details = make(map[string]interface{})
		}

		logs = append(logs, log)
	}

	return logs, nil
}

// GetRecent retrieves recent activity logs across all users (admin view)
func (r *ActivityLogRepository) GetRecent(ctx context.Context, limit int) ([]*models.UserActivityLog, error) {
	if limit == 0 {
		limit = 100
	}

	query := `
		SELECT id, user_id, actor_id, action, details, ip_address, created_at
		FROM user_activity_logs
		ORDER BY created_at DESC
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent activity logs: %w", err)
	}
	defer rows.Close()

	var logs []*models.UserActivityLog
	for rows.Next() {
		log := &models.UserActivityLog{}
		var detailsJSON []byte

		err := rows.Scan(
			&log.ID,
			&log.UserID,
			&log.ActorID,
			&log.Action,
			&detailsJSON,
			&log.IPAddress,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan activity log: %w", err)
		}

		// Unmarshal details
		if err := json.Unmarshal(detailsJSON, &log.Details); err != nil {
			log.Details = make(map[string]interface{})
		}

		logs = append(logs, log)
	}

	return logs, nil
}
