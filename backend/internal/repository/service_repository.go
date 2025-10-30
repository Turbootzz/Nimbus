package repository

import (
	"context"
	"database/sql"

	"github.com/nimbus/backend/internal/models"
)

type ServiceRepository struct {
	db *sql.DB
}

func NewServiceRepository(db *sql.DB) *ServiceRepository {
	return &ServiceRepository{db: db}
}

// Create creates a new service
func (r *ServiceRepository) Create(ctx context.Context, service *models.Service) error {
	// Get the max position for this user and set new service to end
	var maxPos sql.NullInt64
	posQuery := `SELECT MAX(position) FROM services WHERE user_id = $1`
	_ = r.db.QueryRowContext(ctx, posQuery, service.UserID).Scan(&maxPos)

	if maxPos.Valid {
		service.Position = int(maxPos.Int64) + 1
	} else {
		service.Position = 0
	}

	query := `
		INSERT INTO services (user_id, name, url, icon, description, status, position, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		service.UserID,
		service.Name,
		service.URL,
		service.Icon,
		service.Description,
		service.Status,
		service.Position,
		service.CreatedAt,
		service.UpdatedAt,
	).Scan(&service.ID)

	return err
}

// GetByID retrieves a service by ID
func (r *ServiceRepository) GetByID(ctx context.Context, id string) (*models.Service, error) {
	service := &models.Service{}
	query := `
		SELECT id, user_id, name, url, icon, description, status, response_time, position, created_at, updated_at
		FROM services
		WHERE id = $1
	`

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&service.ID,
		&service.UserID,
		&service.Name,
		&service.URL,
		&service.Icon,
		&service.Description,
		&service.Status,
		&service.ResponseTime,
		&service.Position,
		&service.CreatedAt,
		&service.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}

	return service, err
}

// GetAllByUserID retrieves all services for a specific user
func (r *ServiceRepository) GetAllByUserID(ctx context.Context, userID string) ([]*models.Service, error) {
	query := `
		SELECT id, user_id, name, url, icon, description, status, response_time, position, created_at, updated_at
		FROM services
		WHERE user_id = $1
		ORDER BY position ASC, created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []*models.Service
	for rows.Next() {
		service := &models.Service{}
		err := rows.Scan(
			&service.ID,
			&service.UserID,
			&service.Name,
			&service.URL,
			&service.Icon,
			&service.Description,
			&service.Status,
			&service.ResponseTime,
			&service.Position,
			&service.CreatedAt,
			&service.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		services = append(services, service)
	}

	return services, rows.Err()
}

// GetAll retrieves all services across all users (used by health check monitor)
func (r *ServiceRepository) GetAll(ctx context.Context) ([]*models.Service, error) {
	query := `
		SELECT id, user_id, name, url, icon, description, status, response_time, position, created_at, updated_at
		FROM services
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []*models.Service
	for rows.Next() {
		service := &models.Service{}
		err := rows.Scan(
			&service.ID,
			&service.UserID,
			&service.Name,
			&service.URL,
			&service.Icon,
			&service.Description,
			&service.Status,
			&service.ResponseTime,
			&service.Position,
			&service.CreatedAt,
			&service.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		services = append(services, service)
	}

	return services, rows.Err()
}

// Update updates an existing service
func (r *ServiceRepository) Update(ctx context.Context, service *models.Service) error {
	query := `
		UPDATE services
		SET name = $1, url = $2, icon = $3, description = $4, updated_at = $5
		WHERE id = $6 AND user_id = $7
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		service.Name,
		service.URL,
		service.Icon,
		service.Description,
		service.UpdatedAt,
		service.ID,
		service.UserID,
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

// Delete deletes a service by ID
func (r *ServiceRepository) Delete(ctx context.Context, id, userID string) error {
	query := `DELETE FROM services WHERE id = $1 AND user_id = $2`

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

// UpdateStatus updates the status of a service (used by health check system)
func (r *ServiceRepository) UpdateStatus(ctx context.Context, id, status string) error {
	query := `UPDATE services SET status = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`

	result, err := r.db.ExecContext(ctx, query, status, id)
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

// UpdateStatusWithResponseTime updates both status and response time (used by health check system)
func (r *ServiceRepository) UpdateStatusWithResponseTime(ctx context.Context, id, status string, responseTime *int) error {
	query := `UPDATE services SET status = $1, response_time = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $3`

	result, err := r.db.ExecContext(ctx, query, status, responseTime, id)
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

// UpdatePositions updates positions for multiple services in a transaction
func (r *ServiceRepository) UpdatePositions(ctx context.Context, userID string, positions map[string]int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Verify all services belong to the user and update positions
	query := `UPDATE services SET position = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2 AND user_id = $3`

	for serviceID, position := range positions {
		result, err := tx.ExecContext(ctx, query, position, serviceID, userID)
		if err != nil {
			return err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}

		if rowsAffected == 0 {
			return sql.ErrNoRows // Service doesn't exist or doesn't belong to user
		}
	}

	return tx.Commit()
}
