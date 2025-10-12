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
	query := `
		INSERT INTO services (user_id, name, url, icon, description, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
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
		service.CreatedAt,
		service.UpdatedAt,
	).Scan(&service.ID)

	return err
}

// GetByID retrieves a service by ID
func (r *ServiceRepository) GetByID(ctx context.Context, id string) (*models.Service, error) {
	service := &models.Service{}
	query := `
		SELECT id, user_id, name, url, icon, description, status, response_time, created_at, updated_at
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
		&service.CreatedAt,
		&service.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return service, err
}

// GetAllByUserID retrieves all services for a specific user
func (r *ServiceRepository) GetAllByUserID(ctx context.Context, userID string) ([]*models.Service, error) {
	query := `
		SELECT id, user_id, name, url, icon, description, status, response_time, created_at, updated_at
		FROM services
		WHERE user_id = $1
		ORDER BY created_at DESC
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

	_, err := r.db.ExecContext(ctx, query, status, id)
	return err
}

// UpdateStatusWithResponseTime updates both status and response time (used by health check system)
func (r *ServiceRepository) UpdateStatusWithResponseTime(ctx context.Context, id, status string, responseTime *int) error {
	query := `UPDATE services SET status = $1, response_time = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $3`

	_, err := r.db.ExecContext(ctx, query, status, responseTime, id)
	return err
}
