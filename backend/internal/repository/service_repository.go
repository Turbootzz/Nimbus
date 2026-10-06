package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"
	"github.com/nimbus/backend/internal/db"
	"github.com/nimbus/backend/internal/models"
)

type ServiceRepository struct {
	db           *sql.DB
	isPostgreSQL bool
}

func NewServiceRepository(sqlDB *sql.DB) *ServiceRepository {
	return &ServiceRepository{
		db:           sqlDB,
		isPostgreSQL: db.IsPostgreSQL(sqlDB),
	}
}

// Create creates a new service
func (r *ServiceRepository) Create(ctx context.Context, service *models.Service) error {
	// Use a transaction with row-level locking to prevent concurrent position conflicts
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Append after the last service or widget. The user row lock keeps
	// concurrent creates from getting the same position.
	if err := lockUser(ctx, tx, service.UserID); err != nil {
		return err
	}
	if service.Position, err = nextTilePosition(ctx, tx, service.UserID); err != nil {
		return err
	}

	query := `
		INSERT INTO services (user_id, name, url, icon, icon_type, icon_image_path, description, status, position, card_size, group_id, integration_id, monitoring_enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id
	`

	err = tx.QueryRowContext(
		ctx,
		query,
		service.UserID,
		service.Name,
		service.URL,
		service.Icon,
		service.IconType,
		service.IconImagePath,
		service.Description,
		service.Status,
		service.Position,
		service.CardSize,
		service.GroupID,
		service.IntegrationID,
		service.MonitoringEnabled,
		service.CreatedAt,
		service.UpdatedAt,
	).Scan(&service.ID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// serviceColumns is the column list every service query scans with scanService
const serviceColumns = `
	id, user_id, name, url, icon, icon_type, icon_image_path, description, status,
	response_time, position, card_size, group_id, integration_id, monitoring_enabled,
	created_at, updated_at
`

// monitoredServiceColumns is serviceColumns for queries that join groups as g
const monitoredServiceColumns = `
	s.id, s.user_id, s.name, s.url, s.icon, s.icon_type, s.icon_image_path, s.description, s.status,
	s.response_time, s.position, s.card_size, s.group_id, s.integration_id, s.monitoring_enabled,
	s.created_at, s.updated_at
`

func scanService(row rowScanner) (*models.Service, error) {
	service := &models.Service{}
	err := row.Scan(
		&service.ID,
		&service.UserID,
		&service.Name,
		&service.URL,
		&service.Icon,
		&service.IconType,
		&service.IconImagePath,
		&service.Description,
		&service.Status,
		&service.ResponseTime,
		&service.Position,
		&service.CardSize,
		&service.GroupID,
		&service.IntegrationID,
		&service.MonitoringEnabled,
		&service.CreatedAt,
		&service.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return service, nil
}

// queryServices runs a query selecting serviceColumns and scans every row
func (r *ServiceRepository) queryServices(ctx context.Context, query string, args ...any) ([]*models.Service, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []*models.Service
	for rows.Next() {
		service, err := scanService(rows)
		if err != nil {
			return nil, err
		}
		services = append(services, service)
	}
	return services, rows.Err()
}

// GetByID retrieves a service by ID
func (r *ServiceRepository) GetByID(ctx context.Context, id string) (*models.Service, error) {
	query := `SELECT ` + serviceColumns + ` FROM services WHERE id = $1`
	return scanService(r.db.QueryRowContext(ctx, query, id))
}

// GetAllByUserID retrieves all services for a specific user
func (r *ServiceRepository) GetAllByUserID(ctx context.Context, userID string) ([]*models.Service, error) {
	query := `SELECT ` + serviceColumns + ` FROM services WHERE user_id = $1 ORDER BY position ASC, created_at DESC`
	return r.queryServices(ctx, query, userID)
}

// GetAll retrieves all services across all users
func (r *ServiceRepository) GetAll(ctx context.Context) ([]*models.Service, error) {
	query := `SELECT ` + serviceColumns + ` FROM services ORDER BY created_at DESC`
	return r.queryServices(ctx, query)
}

// GetAllForMonitoring retrieves all services where monitoring is enabled (used by health check worker)
// A service is included only if:
// 1. The service has monitoring_enabled = TRUE
// 2. AND either the service has no group (group_id IS NULL) OR its group has monitoring_enabled = TRUE
func (r *ServiceRepository) GetAllForMonitoring(ctx context.Context) ([]*models.Service, error) {
	query := `
		SELECT ` + monitoredServiceColumns + `
		FROM services s
		LEFT JOIN groups g ON s.group_id = g.id
		WHERE s.monitoring_enabled = TRUE
		  AND (s.group_id IS NULL OR g.monitoring_enabled = TRUE)
		ORDER BY s.created_at DESC
	`
	return r.queryServices(ctx, query)
}

// GetAllForMonitoringByUserID retrieves all services for a user where monitoring is enabled
// A service is included only if:
// 1. The service belongs to the specified user
// 2. The service has monitoring_enabled = TRUE
// 3. AND either the service has no group (group_id IS NULL) OR its group has monitoring_enabled = TRUE
func (r *ServiceRepository) GetAllForMonitoringByUserID(ctx context.Context, userID string) ([]*models.Service, error) {
	query := `
		SELECT ` + monitoredServiceColumns + `
		FROM services s
		LEFT JOIN groups g ON s.group_id = g.id
		WHERE s.user_id = $1
		  AND s.monitoring_enabled = TRUE
		  AND (s.group_id IS NULL OR g.monitoring_enabled = TRUE)
		ORDER BY s.position ASC, s.created_at DESC
	`
	return r.queryServices(ctx, query, userID)
}

// Update updates an existing service
func (r *ServiceRepository) Update(ctx context.Context, service *models.Service) error {
	query := `
		UPDATE services
		SET name = $1, url = $2, icon = $3, icon_type = $4, icon_image_path = $5, description = $6, card_size = $7, group_id = $8, integration_id = $9, monitoring_enabled = $10, updated_at = $11
		WHERE id = $12 AND user_id = $13
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		service.Name,
		service.URL,
		service.Icon,
		service.IconType,
		service.IconImagePath,
		service.Description,
		service.CardSize,
		service.GroupID,
		service.IntegrationID,
		service.MonitoringEnabled,
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
	if r.isPostgreSQL {
		return r.bulkUpdatePositionsPostgreSQL(ctx, userID, positions)
	}
	return r.loopUpdatePositions(ctx, userID, positions)
}

// bulkUpdatePositionsPostgreSQL uses PostgreSQL array operations for optimal performance
func (r *ServiceRepository) bulkUpdatePositionsPostgreSQL(ctx context.Context, userID string, positions map[string]int) error {
	if len(positions) == 0 {
		return nil
	}

	// Convert map to arrays for PostgreSQL
	serviceIDs := make([]string, 0, len(positions))
	positionValues := make([]int, 0, len(positions))

	for id, pos := range positions {
		serviceIDs = append(serviceIDs, id)
		positionValues = append(positionValues, pos)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Single bulk UPDATE using PostgreSQL array operations
	query := `
		UPDATE services
		SET position = data.new_position,
		    updated_at = CURRENT_TIMESTAMP
		FROM (
			SELECT unnest($1::uuid[]) AS id,
			       unnest($2::int[]) AS new_position
		) AS data
		WHERE services.id = data.id
		  AND services.user_id = $3
	`

	result, err := tx.ExecContext(ctx, query, pq.Array(serviceIDs), pq.Array(positionValues), userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	// Verify all services were updated
	if int(rowsAffected) != len(positions) {
		return fmt.Errorf("expected to update %d services, but updated %d (service not found or access denied)", len(positions), rowsAffected)
	}

	return tx.Commit()
}

// loopUpdatePositions uses individual UPDATE statements (SQLite compatible)
func (r *ServiceRepository) loopUpdatePositions(ctx context.Context, userID string, positions map[string]int) error {
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
