package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/nimbus/backend/internal/models"
)

// Sentinel errors for password reset repository
var (
	ErrTokenNotFound = errors.New("reset token not found or expired")
)

type PasswordResetRepository struct {
	db *sql.DB
}

func NewPasswordResetRepository(db *sql.DB) *PasswordResetRepository {
	return &PasswordResetRepository{db: db}
}

// Create stores a new password reset token
func (r *PasswordResetRepository) Create(ctx context.Context, token *models.PasswordResetToken) error {
	query := `
		INSERT INTO password_reset_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`

	err := r.db.QueryRowContext(ctx, query,
		token.UserID,
		token.TokenHash,
		token.ExpiresAt,
	).Scan(&token.ID, &token.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create password reset token: %w", err)
	}

	return nil
}

// GetByTokenHash finds a valid (unexpired, unused) token by its hash
func (r *PasswordResetRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*models.PasswordResetToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, used_at, created_at
		FROM password_reset_tokens
		WHERE token_hash = $1 AND used_at IS NULL AND expires_at > $2
	`

	token := &models.PasswordResetToken{}
	err := r.db.QueryRowContext(ctx, query, tokenHash, time.Now()).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.UsedAt,
		&token.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrTokenNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get password reset token: %w", err)
	}

	return token, nil
}

// MarkUsed marks a token as used
func (r *PasswordResetRepository) MarkUsed(ctx context.Context, id string) error {
	query := `UPDATE password_reset_tokens SET used_at = CURRENT_TIMESTAMP WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to mark token as used: %w", err)
	}

	return nil
}

// InvalidateForUser marks all unused tokens for a user as used
func (r *PasswordResetRepository) InvalidateForUser(ctx context.Context, userID string) error {
	query := `UPDATE password_reset_tokens SET used_at = CURRENT_TIMESTAMP WHERE user_id = $1 AND used_at IS NULL`

	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to invalidate tokens for user: %w", err)
	}

	return nil
}

// DeleteExpired removes expired tokens (for cleanup)
func (r *PasswordResetRepository) DeleteExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM password_reset_tokens WHERE expires_at < $1`

	result, err := r.db.ExecContext(ctx, query, time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired tokens: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}
