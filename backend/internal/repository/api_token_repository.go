package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/nimbus/backend/internal/models"
)

// Sentinel errors for api token repository
var (
	ErrAPITokenNotFound     = errors.New("api token not found")
	ErrAPITokenLimitReached = errors.New("api token limit reached")
)

type APITokenRepository struct {
	db *sql.DB
}

func NewAPITokenRepository(db *sql.DB) *APITokenRepository {
	return &APITokenRepository{db: db}
}

// Create stores a new API token if the user is under maxPerUser. A row lock
// on the owning user serializes concurrent creates so the limit can't be
// exceeded. The ID is generated in Go so tests can run on SQLite, which lacks
// gen_random_uuid().
func (r *APITokenRepository) Create(ctx context.Context, token *models.APIToken, maxPerUser int) error {
	token.ID = uuid.New().String()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Self-assign UPDATE takes an exclusive row lock on the owner until commit,
	// serializing concurrent creates per user. Portable across Postgres and
	// SQLite, unlike SELECT ... FOR UPDATE.
	if _, err := tx.ExecContext(ctx, `UPDATE users SET id = id WHERE id = $1`, token.UserID); err != nil {
		return fmt.Errorf("failed to lock user for token create: %w", err)
	}

	insert := `
		INSERT INTO api_tokens (id, user_id, name, token_hash, token_prefix, read_only)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at
	`
	err = tx.QueryRowContext(ctx, insert,
		token.ID,
		token.UserID,
		token.Name,
		token.TokenHash,
		token.TokenPrefix,
		token.ReadOnly,
	).Scan(&token.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create api token: %w", err)
	}

	var count int
	countQuery := `SELECT COUNT(*) FROM api_tokens WHERE user_id = $1`
	if err := tx.QueryRowContext(ctx, countQuery, token.UserID).Scan(&count); err != nil {
		return fmt.Errorf("failed to count api tokens: %w", err)
	}
	if count > maxPerUser {
		return ErrAPITokenLimitReached
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit api token: %w", err)
	}

	return nil
}

// GetByTokenHash finds a token by its SHA-256 hash
func (r *APITokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*models.APIToken, error) {
	query := `
		SELECT id, user_id, name, token_hash, token_prefix, read_only, last_used_at, created_at
		FROM api_tokens
		WHERE token_hash = $1
	`

	token := &models.APIToken{}
	err := r.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&token.ID,
		&token.UserID,
		&token.Name,
		&token.TokenHash,
		&token.TokenPrefix,
		&token.ReadOnly,
		&token.LastUsedAt,
		&token.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrAPITokenNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get api token: %w", err)
	}

	return token, nil
}

// ListByUserID returns all tokens for a user, newest first
func (r *APITokenRepository) ListByUserID(ctx context.Context, userID string) ([]models.APIToken, error) {
	query := `
		SELECT id, user_id, name, token_hash, token_prefix, read_only, last_used_at, created_at
		FROM api_tokens
		WHERE user_id = $1
		ORDER BY created_at DESC, id
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list api tokens: %w", err)
	}
	defer rows.Close()

	tokens := []models.APIToken{}
	for rows.Next() {
		var token models.APIToken
		if err := rows.Scan(
			&token.ID,
			&token.UserID,
			&token.Name,
			&token.TokenHash,
			&token.TokenPrefix,
			&token.ReadOnly,
			&token.LastUsedAt,
			&token.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan api token: %w", err)
		}
		tokens = append(tokens, token)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate api tokens: %w", err)
	}

	return tokens, nil
}

// Delete removes a token scoped to its owner
func (r *APITokenRepository) Delete(ctx context.Context, id, userID string) error {
	query := `DELETE FROM api_tokens WHERE id = $1 AND user_id = $2`

	result, err := r.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete api token: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrAPITokenNotFound
	}

	return nil
}

// UpdateLastUsed stamps the token's last usage time
func (r *APITokenRepository) UpdateLastUsed(ctx context.Context, id string) error {
	query := `UPDATE api_tokens SET last_used_at = CURRENT_TIMESTAMP WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to update api token last used: %w", err)
	}

	return nil
}
