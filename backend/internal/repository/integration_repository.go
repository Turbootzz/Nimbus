package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/nimbus/backend/internal/models"
)

// Sentinel errors for integration repository
var (
	ErrIntegrationNotFound     = errors.New("integration not found")
	ErrIntegrationLimitReached = errors.New("integration limit reached")
)

// integrationColumns never selects credentials_enc itself, only whether it is
// set, so secrets can't end up in a response by accident.
const integrationColumns = `
	id, user_id, kind, name, base_url, auth_type,
	credentials_enc IS NOT NULL, verify_tls, options, refresh_seconds,
	last_test_at, last_test_ok, last_error, created_at, updated_at
`

type IntegrationRepository struct {
	db *sql.DB
}

func NewIntegrationRepository(db *sql.DB) *IntegrationRepository {
	return &IntegrationRepository{db: db}
}

// rowScanner is satisfied by *sql.Row and *sql.Rows
type rowScanner interface {
	Scan(dest ...any) error
}

func scanIntegration(row rowScanner) (*models.Integration, error) {
	var integration models.Integration
	var options []byte
	err := row.Scan(
		&integration.ID,
		&integration.UserID,
		&integration.Kind,
		&integration.Name,
		&integration.BaseURL,
		&integration.AuthType,
		&integration.HasCredentials,
		&integration.VerifyTLS,
		&options,
		&integration.RefreshSeconds,
		&integration.LastTestAt,
		&integration.LastTestOK,
		&integration.LastError,
		&integration.CreatedAt,
		&integration.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	integration.Options = json.RawMessage(options)
	return &integration, nil
}

// nullableBytes turns a nil slice into SQL NULL. Drivers would otherwise
// store an empty value, which reads back as "has credentials".
func nullableBytes(b []byte) any {
	if b == nil {
		return nil
	}
	return b
}

// optionsJSON defaults empty options to an empty object
func optionsJSON(options json.RawMessage) string {
	if len(options) == 0 {
		return "{}"
	}
	return string(options)
}

// Create stores a new integration if the user is under maxPerUser. Uses the
// same per-user row lock as APITokenRepository.Create to keep the limit exact.
func (r *IntegrationRepository) Create(ctx context.Context, integration *models.Integration, credentialsEnc []byte, maxPerUser int) error {
	integration.ID = uuid.New().String()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `UPDATE users SET id = id WHERE id = $1`, integration.UserID); err != nil {
		return fmt.Errorf("failed to lock user for integration create: %w", err)
	}

	insert := `
		INSERT INTO integrations (id, user_id, kind, name, base_url, auth_type,
			credentials_enc, verify_tls, options, refresh_seconds)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING created_at, updated_at
	`
	err = tx.QueryRowContext(ctx, insert,
		integration.ID,
		integration.UserID,
		integration.Kind,
		integration.Name,
		integration.BaseURL,
		integration.AuthType,
		nullableBytes(credentialsEnc),
		integration.VerifyTLS,
		optionsJSON(integration.Options),
		integration.RefreshSeconds,
	).Scan(&integration.CreatedAt, &integration.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create integration: %w", err)
	}

	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM integrations WHERE user_id = $1`, integration.UserID).Scan(&count); err != nil {
		return fmt.Errorf("failed to count integrations: %w", err)
	}
	if count > maxPerUser {
		return ErrIntegrationLimitReached
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit integration: %w", err)
	}

	integration.HasCredentials = credentialsEnc != nil
	integration.Options = json.RawMessage(optionsJSON(integration.Options))
	return nil
}

// GetByID returns an integration scoped to its owner
func (r *IntegrationRepository) GetByID(ctx context.Context, id, userID string) (*models.Integration, error) {
	query := `SELECT ` + integrationColumns + ` FROM integrations WHERE id = $1 AND user_id = $2`

	integration, err := scanIntegration(r.db.QueryRowContext(ctx, query, id, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrIntegrationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get integration: %w", err)
	}
	return integration, nil
}

// GetCredentials returns the encrypted credentials blob (nil when unset).
// This is the only query that reads credentials_enc.
func (r *IntegrationRepository) GetCredentials(ctx context.Context, id, userID string) ([]byte, error) {
	query := `SELECT credentials_enc FROM integrations WHERE id = $1 AND user_id = $2`

	var blob []byte
	err := r.db.QueryRowContext(ctx, query, id, userID).Scan(&blob)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrIntegrationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get integration credentials: %w", err)
	}
	return blob, nil
}

// ListByUserID returns all integrations for a user, sorted by name
func (r *IntegrationRepository) ListByUserID(ctx context.Context, userID string) ([]models.Integration, error) {
	query := `SELECT ` + integrationColumns + ` FROM integrations WHERE user_id = $1 ORDER BY name, id`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list integrations: %w", err)
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
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate integrations: %w", err)
	}

	return integrations, nil
}

// Update saves all editable fields, including the credentials blob (nil
// clears it). Kind and test results are not touched.
func (r *IntegrationRepository) Update(ctx context.Context, integration *models.Integration, credentialsEnc []byte) error {
	query := `
		UPDATE integrations
		SET name = $1, base_url = $2, auth_type = $3, credentials_enc = $4,
			verify_tls = $5, options = $6, refresh_seconds = $7,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $8 AND user_id = $9
		RETURNING updated_at
	`
	err := r.db.QueryRowContext(ctx, query,
		integration.Name,
		integration.BaseURL,
		integration.AuthType,
		nullableBytes(credentialsEnc),
		integration.VerifyTLS,
		optionsJSON(integration.Options),
		integration.RefreshSeconds,
		integration.ID,
		integration.UserID,
	).Scan(&integration.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrIntegrationNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to update integration: %w", err)
	}

	integration.HasCredentials = credentialsEnc != nil
	integration.Options = json.RawMessage(optionsJSON(integration.Options))
	return nil
}

// UpdateTestResult records the outcome of a connection test
func (r *IntegrationRepository) UpdateTestResult(ctx context.Context, id, userID string, ok bool, lastError *string) error {
	query := `
		UPDATE integrations
		SET last_test_at = CURRENT_TIMESTAMP, last_test_ok = $1, last_error = $2
		WHERE id = $3 AND user_id = $4
	`
	result, err := r.db.ExecContext(ctx, query, ok, lastError, id, userID)
	if err != nil {
		return fmt.Errorf("failed to update integration test result: %w", err)
	}
	return requireAffected(result, ErrIntegrationNotFound)
}

// Delete removes an integration scoped to its owner
func (r *IntegrationRepository) Delete(ctx context.Context, id, userID string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM integrations WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete integration: %w", err)
	}
	return requireAffected(result, ErrIntegrationNotFound)
}

// requireAffected returns notFound when a write matched no rows
func requireAffected(result sql.Result, notFound error) error {
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return notFound
	}
	return nil
}
