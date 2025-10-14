package repository

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/nimbus/backend/internal/models"
)

type InvitationRepository struct {
	db *sql.DB
}

func NewInvitationRepository(db *sql.DB) *InvitationRepository {
	return &InvitationRepository{db: db}
}

// Create creates a new invitation
func (r *InvitationRepository) Create(ctx context.Context, email, invitedBy string, expiresInHours int) (*models.UserInvitation, error) {
	// Generate secure random token
	token, err := generateSecureToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	expiresAt := time.Now().Add(time.Duration(expiresInHours) * time.Hour)

	query := `
		INSERT INTO user_invitations (email, token, invited_by, expires_at, created_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
		RETURNING id, created_at
	`

	inv := &models.UserInvitation{
		Email:     email,
		Token:     token,
		InvitedBy: invitedBy,
		ExpiresAt: expiresAt,
	}

	err = r.db.QueryRowContext(ctx, query, email, token, invitedBy, expiresAt).
		Scan(&inv.ID, &inv.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create invitation: %w", err)
	}

	return inv, nil
}

// GetByToken retrieves an invitation by token
func (r *InvitationRepository) GetByToken(ctx context.Context, token string) (*models.UserInvitation, error) {
	query := `
		SELECT id, email, token, invited_by, expires_at, accepted_at, created_at
		FROM user_invitations
		WHERE token = $1
	`

	inv := &models.UserInvitation{}
	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&inv.ID,
		&inv.Email,
		&inv.Token,
		&inv.InvitedBy,
		&inv.ExpiresAt,
		&inv.AcceptedAt,
		&inv.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invitation not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get invitation: %w", err)
	}

	return inv, nil
}

// GetAll retrieves all invitations
func (r *InvitationRepository) GetAll(ctx context.Context) ([]*models.UserInvitation, error) {
	query := `
		SELECT id, email, token, invited_by, expires_at, accepted_at, created_at
		FROM user_invitations
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get invitations: %w", err)
	}
	defer rows.Close()

	var invitations []*models.UserInvitation
	for rows.Next() {
		inv := &models.UserInvitation{}
		err := rows.Scan(
			&inv.ID,
			&inv.Email,
			&inv.Token,
			&inv.InvitedBy,
			&inv.ExpiresAt,
			&inv.AcceptedAt,
			&inv.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan invitation: %w", err)
		}
		invitations = append(invitations, inv)
	}

	return invitations, nil
}

// MarkAsAccepted marks an invitation as accepted
func (r *InvitationRepository) MarkAsAccepted(ctx context.Context, token string) error {
	query := `
		UPDATE user_invitations
		SET accepted_at = CURRENT_TIMESTAMP
		WHERE token = $1
	`

	result, err := r.db.ExecContext(ctx, query, token)
	if err != nil {
		return fmt.Errorf("failed to mark invitation as accepted: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("invitation not found")
	}

	return nil
}

// Delete deletes an invitation
func (r *InvitationRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM user_invitations WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete invitation: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("invitation not found")
	}

	return nil
}

// generateSecureToken generates a cryptographically secure random token
func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
