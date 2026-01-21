package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/nimbus/backend/internal/db"
	"github.com/nimbus/backend/internal/models"
)

var ErrCannotDeleteDefaultGroup = errors.New("cannot delete default group")

type GroupRepository struct {
	db           *sql.DB
	isPostgreSQL bool
}

func NewGroupRepository(sqlDB *sql.DB) *GroupRepository {
	return &GroupRepository{
		db:           sqlDB,
		isPostgreSQL: db.IsPostgreSQL(sqlDB),
	}
}

// Create creates a new group with auto-assigned position
func (r *GroupRepository) Create(ctx context.Context, group *models.Group) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get the max position for this user
	var maxPos sql.NullInt64
	posQuery := `
		SELECT position
		FROM groups
		WHERE user_id = $1
		ORDER BY position DESC
		LIMIT 1
		FOR UPDATE
	`
	err = tx.QueryRowContext(ctx, posQuery, group.UserID).Scan(&maxPos)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if maxPos.Valid {
		group.Position = int(maxPos.Int64) + 1
	} else {
		group.Position = 0
	}

	query := `
		INSERT INTO groups (user_id, name, color, position, is_default, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	err = tx.QueryRowContext(
		ctx,
		query,
		group.UserID,
		group.Name,
		group.Color,
		group.Position,
		group.IsDefault,
		group.CreatedAt,
		group.UpdatedAt,
	).Scan(&group.ID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetByID retrieves a group by ID
func (r *GroupRepository) GetByID(ctx context.Context, id string) (*models.Group, error) {
	group := &models.Group{}
	query := `
		SELECT id, user_id, name, color, position, is_default, created_at, updated_at
		FROM groups
		WHERE id = $1
	`

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&group.ID,
		&group.UserID,
		&group.Name,
		&group.Color,
		&group.Position,
		&group.IsDefault,
		&group.CreatedAt,
		&group.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}

	return group, err
}

// CountByUserID returns the number of groups for a specific user (efficient for validation)
func (r *GroupRepository) CountByUserID(ctx context.Context, userID string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM groups WHERE user_id = $1`
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&count)
	return count, err
}

// GetAllByUserID retrieves all groups for a specific user ordered by position
func (r *GroupRepository) GetAllByUserID(ctx context.Context, userID string) ([]*models.Group, error) {
	query := `
		SELECT id, user_id, name, color, position, is_default, created_at, updated_at
		FROM groups
		WHERE user_id = $1
		ORDER BY position ASC, created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []*models.Group
	for rows.Next() {
		group := &models.Group{}
		err := rows.Scan(
			&group.ID,
			&group.UserID,
			&group.Name,
			&group.Color,
			&group.Position,
			&group.IsDefault,
			&group.CreatedAt,
			&group.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}

	return groups, rows.Err()
}

// GetDefaultByUserID retrieves the default group for a user
func (r *GroupRepository) GetDefaultByUserID(ctx context.Context, userID string) (*models.Group, error) {
	group := &models.Group{}
	query := `
		SELECT id, user_id, name, color, position, is_default, created_at, updated_at
		FROM groups
		WHERE user_id = $1 AND is_default = TRUE
	`

	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&group.ID,
		&group.UserID,
		&group.Name,
		&group.Color,
		&group.Position,
		&group.IsDefault,
		&group.CreatedAt,
		&group.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}

	return group, err
}

// EnsureDefaultGroup creates a default group for the user if one doesn't exist (atomic)
func (r *GroupRepository) EnsureDefaultGroup(ctx context.Context, userID string) (*models.Group, error) {
	now := time.Now()
	group := &models.Group{
		ID:        uuid.New().String(),
		UserID:    userID,
		Name:      models.DefaultGroupName,
		Color:     models.DefaultGroupColor,
		Position:  0,
		IsDefault: true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if r.isPostgreSQL {
		// Use INSERT ... ON CONFLICT for atomic upsert
		query := `
			INSERT INTO groups (id, user_id, name, color, position, is_default, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (user_id) WHERE is_default = TRUE
			DO NOTHING
		`
		result, err := r.db.ExecContext(ctx, query,
			group.ID, group.UserID, group.Name, group.Color,
			group.Position, group.IsDefault, group.CreatedAt, group.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return nil, err
		}

		if rowsAffected == 0 {
			// ON CONFLICT DO NOTHING means a default already exists
			return r.GetDefaultByUserID(ctx, userID)
		}
		return group, nil
	}

	// SQLite fallback - use check-then-create pattern
	existing, err := r.GetDefaultByUserID(ctx, userID)
	if err == nil {
		return existing, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	if err := r.Create(ctx, group); err != nil {
		// Check if another goroutine created it (race condition)
		existing, checkErr := r.GetDefaultByUserID(ctx, userID)
		if checkErr == nil {
			return existing, nil
		}
		return nil, err
	}

	return group, nil
}

// Update updates an existing group (name and color only)
func (r *GroupRepository) Update(ctx context.Context, group *models.Group) error {
	query := `
		UPDATE groups
		SET name = $1, color = $2, updated_at = $3
		WHERE id = $4 AND user_id = $5
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		group.Name,
		group.Color,
		group.UpdatedAt,
		group.ID,
		group.UserID,
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

// Delete deletes a group (cannot delete default group)
// If deleteServices is true, also deletes all services in the group
func (r *GroupRepository) Delete(ctx context.Context, id, userID string, deleteServices bool) error {
	// First check if it's the default group
	group, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if group.UserID != userID {
		return sql.ErrNoRows // Access denied
	}

	if group.IsDefault {
		return ErrCannotDeleteDefaultGroup
	}

	// Use transaction to ensure atomicity
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// If deleteServices is true, delete all services in this group
	if deleteServices {
		deleteServicesQuery := `DELETE FROM services WHERE group_id = $1 AND user_id = $2`
		_, err = tx.ExecContext(ctx, deleteServicesQuery, id, userID)
		if err != nil {
			return err
		}
	}

	// Delete the group
	query := `DELETE FROM groups WHERE id = $1 AND user_id = $2`

	result, err := tx.ExecContext(ctx, query, id, userID)
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

	return tx.Commit()
}

// UpdatePositions updates positions for multiple groups in a transaction
func (r *GroupRepository) UpdatePositions(ctx context.Context, userID string, positions map[string]int) error {
	if r.isPostgreSQL {
		return r.bulkUpdatePositionsPostgreSQL(ctx, userID, positions)
	}
	return r.loopUpdatePositions(ctx, userID, positions)
}

// bulkUpdatePositionsPostgreSQL uses PostgreSQL array operations
func (r *GroupRepository) bulkUpdatePositionsPostgreSQL(ctx context.Context, userID string, positions map[string]int) error {
	if len(positions) == 0 {
		return nil
	}

	groupIDs := make([]string, 0, len(positions))
	positionValues := make([]int, 0, len(positions))

	for id, pos := range positions {
		groupIDs = append(groupIDs, id)
		positionValues = append(positionValues, pos)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		UPDATE groups
		SET position = data.new_position,
		    updated_at = CURRENT_TIMESTAMP
		FROM (
			SELECT unnest($1::uuid[]) AS id,
			       unnest($2::int[]) AS new_position
		) AS data
		WHERE groups.id = data.id
		  AND groups.user_id = $3
	`

	result, err := tx.ExecContext(ctx, query, pq.Array(groupIDs), pq.Array(positionValues), userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if int(rowsAffected) != len(positions) {
		return fmt.Errorf("expected to update %d groups, but updated %d (group not found or access denied)", len(positions), rowsAffected)
	}

	return tx.Commit()
}

// loopUpdatePositions uses individual UPDATE statements (SQLite compatible)
func (r *GroupRepository) loopUpdatePositions(ctx context.Context, userID string, positions map[string]int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `UPDATE groups SET position = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2 AND user_id = $3`

	for groupID, position := range positions {
		result, err := tx.ExecContext(ctx, query, position, groupID, userID)
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
	}

	return tx.Commit()
}
