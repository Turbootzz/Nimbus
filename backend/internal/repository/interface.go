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
	GetAllForMonitoring(ctx context.Context) ([]*models.Service, error)
	GetAllForMonitoringByUserID(ctx context.Context, userID string) ([]*models.Service, error)
	Update(ctx context.Context, service *models.Service) error
	Delete(ctx context.Context, id, userID string) error
	UpdateStatus(ctx context.Context, id, status string) error
	UpdateStatusWithResponseTime(ctx context.Context, id, status string, responseTime *int) error
}

// Ensure ServiceRepository implements the interface
var _ ServiceRepositoryInterface = (*ServiceRepository)(nil)

// IntegrationRepositoryInterface defines the interface for integration repository operations
type IntegrationRepositoryInterface interface {
	Create(ctx context.Context, integration *models.Integration, credentialsEnc []byte, maxPerUser int) error
	GetByID(ctx context.Context, id, userID string) (*models.Integration, error)
	GetCredentials(ctx context.Context, id, userID string) ([]byte, error)
	ListByUserID(ctx context.Context, userID string) ([]models.Integration, error)
	Update(ctx context.Context, integration *models.Integration, credentialsEnc []byte) error
	UpdateTestResult(ctx context.Context, id, userID string, ok bool, lastError *string) error
	Delete(ctx context.Context, id, userID string) error
}

// Ensure IntegrationRepository implements the interface
var _ IntegrationRepositoryInterface = (*IntegrationRepository)(nil)

// WidgetRepositoryInterface defines the interface for widget repository operations
type WidgetRepositoryInterface interface {
	Create(ctx context.Context, widget *models.Widget, maxPerUser int) error
	GetByID(ctx context.Context, id, userID string) (*models.Widget, error)
	ListByUserID(ctx context.Context, userID string) ([]models.Widget, error)
	Update(ctx context.Context, widget *models.Widget) error
	Delete(ctx context.Context, id, userID string) error
	ReorderTiles(ctx context.Context, userID string, tiles []models.TilePosition) error
}

// Ensure WidgetRepository implements the interface
var _ WidgetRepositoryInterface = (*WidgetRepository)(nil)
