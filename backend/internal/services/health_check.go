package services

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
)

// HealthCheckService handles health checking of services
type HealthCheckService struct {
	serviceRepo repository.ServiceRepositoryInterface
	httpClient  *http.Client
}

// NewHealthCheckService creates a new health check service
func NewHealthCheckService(serviceRepo repository.ServiceRepositoryInterface, timeout time.Duration) *HealthCheckService {
	return &HealthCheckService{
		serviceRepo: serviceRepo,
		httpClient: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Don't follow redirects - consider them successful
				return http.ErrUseLastResponse
			},
		},
	}
}

// CheckService performs a health check on a single service
func (h *HealthCheckService) CheckService(ctx context.Context, service *models.Service) error {
	start := time.Now()

	// Create request with context for cancellation
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, service.URL, nil)
	if err != nil {
		// Invalid URL - mark as offline
		return h.updateStatus(ctx, service.ID, models.StatusOffline, nil)
	}

	// Set user agent
	req.Header.Set("User-Agent", "Nimbus-HealthCheck/1.0")

	// Perform the request
	resp, err := h.httpClient.Do(req)
	responseTime := int(time.Since(start).Milliseconds())

	if err != nil {
		// Request failed - service is offline
		return h.updateStatus(ctx, service.ID, models.StatusOffline, &responseTime)
	}
	defer resp.Body.Close()

	// Consider 2xx and 3xx status codes as "online"
	// 4xx and 5xx are considered "offline" (service is responding but not healthy)
	var status string
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		status = models.StatusOnline
	} else {
		status = models.StatusOffline
	}

	return h.updateStatus(ctx, service.ID, status, &responseTime)
}

// CheckAllServices checks all services for a specific user
func (h *HealthCheckService) CheckAllServices(ctx context.Context, userID string) error {
	services, err := h.serviceRepo.GetAllByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to fetch services: %w", err)
	}

	// Check each service sequentially
	for _, service := range services {
		// Check if context was cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := h.CheckService(ctx, service); err != nil {
			// Log error but continue checking other services
			fmt.Printf("Failed to check service %s (%s): %v\n", service.Name, service.ID, err)
		}
	}

	return nil
}

// CheckAllServicesForAllUsers checks all services across all users
func (h *HealthCheckService) CheckAllServicesForAllUsers(ctx context.Context) error {
	// This will be implemented when we need to check services for all users
	// For now, services are checked per-user
	// TODO: Add method to get all services across all users
	return fmt.Errorf("not implemented yet - check services per user")
}

// updateStatus is a helper to update service status and response time
// Uses a background context to ensure status updates persist even if the check request is cancelled
func (h *HealthCheckService) updateStatus(ctx context.Context, serviceID, status string, responseTime *int) error {
	// Create independent context with timeout for DB update
	// This ensures status is saved even if the HTTP check context is cancelled
	updateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return h.serviceRepo.UpdateStatusWithResponseTime(updateCtx, serviceID, status, responseTime)
}
