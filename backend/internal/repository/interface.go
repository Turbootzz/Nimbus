package repository

import (
	"context"

	"github.com/nimbus/backend/internal/models"
)

// ServiceRepositoryInterface defines the interface for service repository operations
type ServiceRepositoryInterface interface {
	Create(ctx context.Context, service *models.Service) error
	GetByID(ctx context.Context, id string) (*models.Service, error)
	GetAllByUserID(ctx context.Context, userID string) ([]*models.Service, error)
	GetAll(ctx context.Context) ([]*models.Service, error)
	Update(ctx context.Context, service *models.Service) error
	Delete(ctx context.Context, id, userID string) error
	UpdateStatus(ctx context.Context, id, status string) error
	UpdateStatusWithResponseTime(ctx context.Context, id, status string, responseTime *int) error
}

// Ensure ServiceRepository implements the interface
var _ ServiceRepositoryInterface = (*ServiceRepository)(nil)
