package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/nimbus/backend/internal/models"
)

// DiscoveryRepository remembers which discovered containers a user has
// added or ignored
type DiscoveryRepository struct {
	db *sql.DB
}

func NewDiscoveryRepository(db *sql.DB) *DiscoveryRepository {
	return &DiscoveryRepository{db: db}
}

// List returns the user's handled containers by key. The value is the
// service the container became; nil means ignored.
func (r *DiscoveryRepository) List(ctx context.Context, userID string) (map[string]*string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT discovery_key, service_id FROM discovered_containers WHERE user_id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list discovered containers: %w", err)
	}
	defer rows.Close()

	handled := map[string]*string{}
	for rows.Next() {
		var key string
		var serviceID *string
		if err := rows.Scan(&key, &serviceID); err != nil {
			return nil, fmt.Errorf("failed to scan discovered container: %w", err)
		}
		handled[key] = serviceID
	}
	return handled, rows.Err()
}

// Claim marks a container as handled (ignored, until SetService). It returns
// false when the container was handled already, so only one caller gets to
// add it.
func (r *DiscoveryRepository) Claim(ctx context.Context, userID, key string) (bool, error) {
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO discovered_containers (user_id, discovery_key) VALUES ($1, $2)
		ON CONFLICT (user_id, discovery_key) DO NOTHING
	`, userID, key)
	if err != nil {
		return false, fmt.Errorf("failed to claim discovered container: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get rows affected: %w", err)
	}
	return affected == 1, nil
}

// SetService records the service a claimed container became
func (r *DiscoveryRepository) SetService(ctx context.Context, userID, key, serviceID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE discovered_containers SET service_id = $1 WHERE user_id = $2 AND discovery_key = $3
	`, serviceID, userID, key)
	if err != nil {
		return fmt.Errorf("failed to link discovered container: %w", err)
	}
	return nil
}

// Release offers an ignored container again. A container that became a
// service stays handled.
func (r *DiscoveryRepository) Release(ctx context.Context, userID, key string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM discovered_containers WHERE user_id = $1 AND discovery_key = $2 AND service_id IS NULL
	`, userID, key)
	if err != nil {
		return fmt.Errorf("failed to release discovered container: %w", err)
	}
	return nil
}

// AdminIntegrations lists every integration of a kind whose owner is an
// admin now: a demoted admin's Docker integration is no longer read.
func (r *DiscoveryRepository) AdminIntegrations(ctx context.Context, kind string) ([]models.Integration, error) {
	query := `SELECT ` + integrationColumns + ` FROM integrations
		WHERE kind = $1 AND user_id IN (SELECT id FROM users WHERE role = 'admin')
		ORDER BY user_id, name, id`

	rows, err := r.db.QueryContext(ctx, query, kind)
	if err != nil {
		return nil, fmt.Errorf("failed to list admin integrations: %w", err)
	}
	defer rows.Close()

	integrations := []models.Integration{}
	for rows.Next() {
		integration, err := scanIntegration(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan integration: %w", err)
		}
		integrations = append(integrations, *integration)
	}
	return integrations, rows.Err()
}
