package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/nimbus/backend/internal/models"
)

// Sentinel errors for user repository
var (
	ErrUserNotFound = errors.New("user not found")
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user in the database
func (r *UserRepository) Create(user *models.User) error {
	query := `
		INSERT INTO users (id, email, name, password, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`

	user.ID = uuid.New().String()

	err := r.db.QueryRow(
		query,
		user.ID,
		user.Email,
		user.Name,
		user.Password,
		user.Role,
		user.CreatedAt,
		user.UpdatedAt,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(email string) (*models.User, error) {
	query := `
		SELECT id, email, name, password, role, last_activity_at, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	user := &models.User{}
	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.Password,
		&user.Role,
		&user.LastActivityAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(id string) (*models.User, error) {
	query := `
		SELECT id, email, name, password, role, last_activity_at, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	user := &models.User{}
	err := r.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.Password,
		&user.Role,
		&user.LastActivityAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// EmailExists checks if an email is already registered
func (r *UserRepository) EmailExists(email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`

	var exists bool
	err := r.db.QueryRow(query, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check email existence: %w", err)
	}

	return exists, nil
}

// UserFilter represents search and filter options
type UserFilter struct {
	Search string // Search in name and email
	Role   string // Filter by role (admin, user, or empty for all)
	Limit  int    // Pagination: items per page
	Offset int    // Pagination: offset
}

// UserListResult represents paginated user list with metadata
type UserListResult struct {
	Users      []*models.User
	Total      int
	Page       int
	TotalPages int
}

// GetAll retrieves all users (for admin purposes)
func (r *UserRepository) GetAll() ([]*models.User, error) {
	query := `
		SELECT id, email, name, password, role, last_activity_at, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.Name,
			&user.Password,
			&user.Role,
			&user.LastActivityAt,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}

// GetFiltered retrieves users with search, filter, and pagination
func (r *UserRepository) GetFiltered(filter UserFilter) (*UserListResult, error) {
	// Build WHERE clause dynamically
	where := "WHERE 1=1"
	args := []interface{}{}
	argCount := 0

	// Add search filter (lowercase the search term for case-insensitive comparison)
	if filter.Search != "" {
		argCount++
		where += fmt.Sprintf(" AND (LOWER(name) LIKE $%d OR LOWER(email) LIKE $%d)", argCount, argCount)
		args = append(args, "%"+strings.ToLower(filter.Search)+"%")
	}

	// Add role filter
	if filter.Role != "" {
		argCount++
		where += fmt.Sprintf(" AND role = $%d", argCount)
		args = append(args, filter.Role)
	}

	// Get total count for pagination
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM users %s", where)
	var total int
	if err := r.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to count users: %w", err)
	}

	// Set default pagination
	if filter.Limit == 0 {
		filter.Limit = 20 // Default page size
	}

	// Build main query with pagination
	query := fmt.Sprintf(`
		SELECT id, email, name, password, role, last_activity_at, created_at, updated_at
		FROM users
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argCount+1, argCount+2)

	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get filtered users: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		if err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.Name,
			&user.Password,
			&user.Role,
			&user.LastActivityAt,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	// Calculate pagination metadata
	page := (filter.Offset / filter.Limit) + 1
	totalPages := (total + filter.Limit - 1) / filter.Limit // Ceiling division

	return &UserListResult{
		Users:      users,
		Total:      total,
		Page:       page,
		TotalPages: totalPages,
	}, nil
}

// UpdateRole updates a user's role (admin operation)
func (r *UserRepository) UpdateRole(userID string, newRole string) error {
	query := `
		UPDATE users
		SET role = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`

	result, err := r.db.Exec(query, newRole, userID)
	if err != nil {
		return fmt.Errorf("failed to update user role: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// Delete deletes a user (admin operation)
func (r *UserRepository) Delete(userID string) error {
	query := `DELETE FROM users WHERE id = $1`

	result, err := r.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// GetStats returns user statistics (admin operation)
func (r *UserRepository) GetStats() (map[string]int, error) {
	// Use CASE WHEN for SQLite compatibility (FILTER is PostgreSQL-specific)
	// Use COALESCE to handle NULL when table is empty
	query := `
		SELECT
			COUNT(*) as total,
			COALESCE(SUM(CASE WHEN role = 'admin' THEN 1 ELSE 0 END), 0) as admins,
			COALESCE(SUM(CASE WHEN role = 'user' THEN 1 ELSE 0 END), 0) as users
		FROM users
	`

	var total, admins, users int
	err := r.db.QueryRow(query).Scan(&total, &admins, &users)
	if err != nil {
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}

	return map[string]int{
		"total":  total,
		"admins": admins,
		"users":  users,
	}, nil
}

// BulkUpdateRole updates multiple users' roles at once
func (r *UserRepository) BulkUpdateRole(userIDs []string, newRole string) (int, error) {
	if len(userIDs) == 0 {
		return 0, nil
	}

	// Build placeholders for IN clause
	placeholders := ""
	args := []interface{}{newRole}
	for i, id := range userIDs {
		if i > 0 {
			placeholders += ", "
		}
		placeholders += fmt.Sprintf("$%d", i+2)
		args = append(args, id)
	}

	query := fmt.Sprintf(`
		UPDATE users
		SET role = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id IN (%s)
	`, placeholders)

	result, err := r.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to bulk update roles: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rowsAffected), nil
}

// BulkDelete deletes multiple users at once
func (r *UserRepository) BulkDelete(userIDs []string) (int, error) {
	if len(userIDs) == 0 {
		return 0, nil
	}

	// Build placeholders for IN clause
	placeholders := ""
	args := []interface{}{}
	for i, id := range userIDs {
		if i > 0 {
			placeholders += ", "
		}
		placeholders += fmt.Sprintf("$%d", i+1)
		args = append(args, id)
	}

	query := fmt.Sprintf(`DELETE FROM users WHERE id IN (%s)`, placeholders)

	result, err := r.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to bulk delete users: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rowsAffected), nil
}

// UpdateLastActivity updates a user's last activity timestamp
func (r *UserRepository) UpdateLastActivity(userID string) error {
	query := `
		UPDATE users
		SET last_activity_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	_, err := r.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("failed to update last activity: %w", err)
	}

	return nil
}
