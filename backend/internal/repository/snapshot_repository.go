package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/nimbus/backend/internal/models"
)

// SnapshotRepository stores the latest data of polled widgets and integrations
type SnapshotRepository struct {
	db *sql.DB
}

func NewSnapshotRepository(db *sql.DB) *SnapshotRepository {
	return &SnapshotRepository{db: db}
}

// Upsert saves a snapshot, replacing the previous one of the same source
func (r *SnapshotRepository) Upsert(ctx context.Context, snap *models.Snapshot) error {
	query := `
		INSERT INTO widget_snapshots (source_kind, source_id, user_id, payload, error, fetched_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (source_kind, source_id) DO UPDATE
		SET payload = EXCLUDED.payload, error = EXCLUDED.error, fetched_at = EXCLUDED.fetched_at
	`
	var payload any
	if len(snap.Payload) > 0 {
		payload = string(snap.Payload)
	}
	var snapErr any
	if snap.Error != "" {
		snapErr = snap.Error
	}
	_, err := r.db.ExecContext(ctx, query, snap.SourceKind, snap.SourceID, snap.UserID, payload, snapErr, snap.FetchedAt)
	if err != nil {
		return fmt.Errorf("failed to save snapshot: %w", err)
	}
	return nil
}

// ListAll returns every stored snapshot, to warm the cache at boot
func (r *SnapshotRepository) ListAll(ctx context.Context) ([]models.Snapshot, error) {
	query := `SELECT source_kind, source_id, user_id, payload, error, fetched_at FROM widget_snapshots`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}
	defer rows.Close()

	snapshots := []models.Snapshot{}
	for rows.Next() {
		var snap models.Snapshot
		var payload []byte
		var snapErr sql.NullString
		if err := rows.Scan(&snap.SourceKind, &snap.SourceID, &snap.UserID, &payload, &snapErr, &snap.FetchedAt); err != nil {
			return nil, fmt.Errorf("failed to scan snapshot: %w", err)
		}
		if payload != nil {
			snap.Payload = json.RawMessage(payload)
		}
		snap.Error = snapErr.String
		snapshots = append(snapshots, snap)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate snapshots: %w", err)
	}
	return snapshots, nil
}

// PruneOrphans deletes snapshots whose widget or integration no longer exists
func (r *SnapshotRepository) PruneOrphans(ctx context.Context) (int64, error) {
	query := `
		DELETE FROM widget_snapshots
		WHERE (source_kind = 'widget' AND source_id NOT IN (SELECT id FROM widgets))
		   OR (source_kind = 'integration' AND source_id NOT IN (SELECT id FROM integrations))
	`
	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to prune snapshots: %w", err)
	}
	return result.RowsAffected()
}
