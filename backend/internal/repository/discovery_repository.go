package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/nimbus/backend/internal/models"
)

// An add that takes longer than this has died, as when the server stopped
// between Claim and SetService: Release may then give its claim back
const staleClaimAge = 5 * time.Minute

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

// Claim marks a container as handled: ignored, or pending for an add until
// SetService or Unclaim. It returns false when the container was handled
// already, so only one caller gets to add it.
func (r *DiscoveryRepository) Claim(ctx context.Context, userID, key string, pending bool) (bool, error) {
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO discovered_containers (user_id, discovery_key, pending) VALUES ($1, $2, $3)
		ON CONFLICT (user_id, discovery_key) DO NOTHING
	`, userID, key, pending)
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
		UPDATE discovered_containers SET service_id = $1, pending = FALSE WHERE user_id = $2 AND discovery_key = $3
	`, serviceID, userID, key)
	if err != nil {
		return fmt.Errorf("failed to link discovered container: %w", err)
	}
	return nil
}

// Release offers an ignored container again. A container that became a
// service stays handled, and so does one an add is still creating.
func (r *DiscoveryRepository) Release(ctx context.Context, userID, key string) error {
	// UTC, so SQLite in the tests compares it as text with CURRENT_TIMESTAMP
	staleBefore := time.Now().Add(-staleClaimAge).UTC()
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM discovered_containers
		WHERE user_id = $1 AND discovery_key = $2 AND service_id IS NULL AND (NOT pending OR created_at < $3)
	`, userID, key, staleBefore)
	if err != nil {
		return fmt.Errorf("failed to release discovered container: %w", err)
	}
	return nil
}

// Unclaim gives back the claim of an add that failed
func (r *DiscoveryRepository) Unclaim(ctx context.Context, userID, key string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM discovered_containers WHERE user_id = $1 AND discovery_key = $2 AND pending
	`, userID, key)
	if err != nil {
		return fmt.Errorf("failed to unclaim discovered container: %w", err)
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
