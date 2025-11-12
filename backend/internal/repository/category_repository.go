package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"
	"github.com/nimbus/backend/internal/models"
)

type CategoryRepository struct {
	db           *sql.DB
	isPostgreSQL bool
}

func NewCategoryRepository(db *sql.DB) *CategoryRepository {
	// Detect if we're using PostgreSQL
	isPostgreSQL := false
	if db != nil {
		var version string
		err := db.QueryRow("SELECT version()").Scan(&version)
		if err == nil {
			isPostgreSQL = len(version) > 10 && version[:10] == "PostgreSQL"
		}
	}

	return &CategoryRepository{
		db:           db,
		isPostgreSQL: isPostgreSQL,
	}
}

// placeholder returns the correct placeholder for the database type
func (r *CategoryRepository) placeholder(n int) string {
	if r.isPostgreSQL {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}

// Create creates a new category
func (r *CategoryRepository) Create(ctx context.Context, category *models.Category) error {
	// Use a transaction with row-level locking to prevent concurrent position conflicts
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get the max position for this user
	var maxPos sql.NullInt64
	posQuery := fmt.Sprintf(`
		SELECT position
		FROM categories
		WHERE user_id = %s
		ORDER BY position DESC
		LIMIT 1
	`, r.placeholder(1))

	if r.isPostgreSQL {
		posQuery += " FOR UPDATE"
	}

	err = tx.QueryRowContext(ctx, posQuery, category.UserID).Scan(&maxPos)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if maxPos.Valid {
		category.Position = int(maxPos.Int64) + 1
	} else {
		category.Position = 0
	}

	if r.isPostgreSQL {
		// PostgreSQL with RETURNING
		query := fmt.Sprintf(`
			INSERT INTO categories (user_id, name, color, position, created_at, updated_at)
			VALUES (%s, %s, %s, %s, %s, %s)
			RETURNING id
		`, r.placeholder(1), r.placeholder(2), r.placeholder(3), r.placeholder(4), r.placeholder(5), r.placeholder(6))

		err = tx.QueryRowContext(
			ctx,
			query,
			category.UserID,
			category.Name,
			category.Color,
			category.Position,
			category.CreatedAt,
			category.UpdatedAt,
		).Scan(&category.ID)
	} else {
		// SQLite - use provided ID or generate one
		if category.ID == "" {
			// Generate a unique ID for SQLite testing using name hash and timestamp
			category.ID = fmt.Sprintf("cat-%s-%d", category.Name, category.CreatedAt.UnixNano())
		}

		query := fmt.Sprintf(`
			INSERT INTO categories (id, user_id, name, color, position, created_at, updated_at)
			VALUES (%s, %s, %s, %s, %s, %s, %s)
		`, r.placeholder(1), r.placeholder(2), r.placeholder(3), r.placeholder(4), r.placeholder(5), r.placeholder(6), r.placeholder(7))

		_, err = tx.ExecContext(
			ctx,
			query,
			category.ID,
			category.UserID,
			category.Name,
			category.Color,
			category.Position,
			category.CreatedAt,
			category.UpdatedAt,
		)
	}

	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetByID retrieves a category by ID
func (r *CategoryRepository) GetByID(ctx context.Context, id string) (*models.Category, error) {
	category := &models.Category{}
	query := fmt.Sprintf(`
		SELECT id, user_id, name, color, position, created_at, updated_at
		FROM categories
		WHERE id = %s
	`, r.placeholder(1))

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&category.ID,
		&category.UserID,
		&category.Name,
		&category.Color,
		&category.Position,
		&category.CreatedAt,
		&category.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}

	return category, err
}

// GetAllByUserID retrieves all categories for a specific user
func (r *CategoryRepository) GetAllByUserID(ctx context.Context, userID string) ([]*models.Category, error) {
	query := fmt.Sprintf(`
		SELECT id, user_id, name, color, position, created_at, updated_at
		FROM categories
		WHERE user_id = %s
		ORDER BY position ASC, created_at DESC
	`, r.placeholder(1))

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []*models.Category
	for rows.Next() {
		category := &models.Category{}
		err := rows.Scan(
			&category.ID,
			&category.UserID,
			&category.Name,
			&category.Color,
			&category.Position,
			&category.CreatedAt,
			&category.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}

	return categories, rows.Err()
}

// Update updates an existing category
func (r *CategoryRepository) Update(ctx context.Context, category *models.Category) error {
	query := fmt.Sprintf(`
		UPDATE categories
		SET name = %s, color = %s, updated_at = %s
		WHERE id = %s AND user_id = %s
	`, r.placeholder(1), r.placeholder(2), r.placeholder(3), r.placeholder(4), r.placeholder(5))

	result, err := r.db.ExecContext(
		ctx,
		query,
		category.Name,
		category.Color,
		category.UpdatedAt,
		category.ID,
		category.UserID,
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

// Delete deletes a category by ID (services will have category_id set to NULL)
func (r *CategoryRepository) Delete(ctx context.Context, id, userID string) error {
	query := fmt.Sprintf(`DELETE FROM categories WHERE id = %s AND user_id = %s`,
		r.placeholder(1), r.placeholder(2))

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

// UpdatePositions updates positions for multiple categories in a transaction
func (r *CategoryRepository) UpdatePositions(ctx context.Context, userID string, positions map[string]int) error {
	if r.isPostgreSQL {
		return r.bulkUpdatePositionsPostgreSQL(ctx, userID, positions)
	}
	return r.loopUpdatePositions(ctx, userID, positions)
}

// bulkUpdatePositionsPostgreSQL uses PostgreSQL array operations for optimal performance
func (r *CategoryRepository) bulkUpdatePositionsPostgreSQL(ctx context.Context, userID string, positions map[string]int) error {
	if len(positions) == 0 {
		return nil
	}

	// Convert map to arrays for PostgreSQL
	categoryIDs := make([]string, 0, len(positions))
	positionValues := make([]int, 0, len(positions))

	for id, pos := range positions {
		categoryIDs = append(categoryIDs, id)
		positionValues = append(positionValues, pos)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Single bulk UPDATE using PostgreSQL array operations
	query := `
		UPDATE categories
		SET position = data.new_position,
		    updated_at = CURRENT_TIMESTAMP
		FROM (
			SELECT unnest($1::uuid[]) AS id,
			       unnest($2::int[]) AS new_position
		) AS data
		WHERE categories.id = data.id
		  AND categories.user_id = $3
	`

	result, err := tx.ExecContext(ctx, query, pq.Array(categoryIDs), pq.Array(positionValues), userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	// Verify all categories were updated
	if int(rowsAffected) != len(positions) {
		return fmt.Errorf("expected to update %d categories, but updated %d (category not found or access denied)", len(positions), rowsAffected)
	}

	return tx.Commit()
}

// loopUpdatePositions uses individual UPDATE statements (SQLite compatible)
func (r *CategoryRepository) loopUpdatePositions(ctx context.Context, userID string, positions map[string]int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Verify all categories belong to the user and update positions
	query := fmt.Sprintf(`UPDATE categories SET position = %s, updated_at = CURRENT_TIMESTAMP WHERE id = %s AND user_id = %s`,
		r.placeholder(1), r.placeholder(2), r.placeholder(3))

	for categoryID, position := range positions {
		result, err := tx.ExecContext(ctx, query, position, categoryID, userID)
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
