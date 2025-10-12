package workers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/services"
)

// HealthMonitor runs periodic health checks on all services
type HealthMonitor struct {
	healthCheckService *services.HealthCheckService
	serviceRepo        *repository.ServiceRepository
	interval           time.Duration
	ctx                context.Context
	cancel             context.CancelFunc
	wg                 sync.WaitGroup
}

// NewHealthMonitor creates a new health monitor worker
func NewHealthMonitor(
	healthCheckService *services.HealthCheckService,
	serviceRepo *repository.ServiceRepository,
	interval time.Duration,
) *HealthMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	return &HealthMonitor{
		healthCheckService: healthCheckService,
		serviceRepo:        serviceRepo,
		interval:           interval,
		ctx:                ctx,
		cancel:             cancel,
	}
}

// Start begins the health monitoring loop
func (h *HealthMonitor) Start() {
	h.wg.Add(1)
	go h.run()
	fmt.Printf("Health monitor started (interval: %v)\n", h.interval)
}

// Stop gracefully stops the health monitor
func (h *HealthMonitor) Stop() {
	fmt.Println("Stopping health monitor...")
	h.cancel()
	h.wg.Wait()
	fmt.Println("Health monitor stopped")
}

// run is the main monitoring loop
func (h *HealthMonitor) run() {
	defer h.wg.Done()

	// Run first check immediately
	h.checkAllServices()

	// Create ticker for periodic checks
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.checkAllServices()
		case <-h.ctx.Done():
			return
		}
	}
}

// checkAllServices fetches all services and checks them concurrently
func (h *HealthMonitor) checkAllServices() {
	startTime := time.Now()
	fmt.Println("Starting health check cycle...")

	// Get all services (across all users)
	// Note: We're using a new context for each check cycle, not the worker's context
	// This allows individual checks to complete even during shutdown
	ctx, cancel := context.WithTimeout(context.Background(), h.interval)
	defer cancel()

	services, err := h.getAllServices(ctx)
	if err != nil {
		fmt.Printf("Failed to fetch services for health check: %v\n", err)
		return
	}

	if len(services) == 0 {
		fmt.Println("No services to check")
		return
	}

	// Check services concurrently with a limit
	h.checkServicesConcurrently(ctx, services)

	duration := time.Since(startTime)
	fmt.Printf("Health check cycle completed in %v (checked %d services)\n", duration, len(services))
}

// getAllServices retrieves all services from all users
func (h *HealthMonitor) getAllServices(ctx context.Context) ([]*models.Service, error) {
	return h.serviceRepo.GetAll(ctx)
}

// checkServicesConcurrently checks multiple services concurrently with a worker pool
func (h *HealthMonitor) checkServicesConcurrently(ctx context.Context, services []*models.Service) {
	const maxWorkers = 10 // Limit concurrent checks to avoid overwhelming the system

	// Create a semaphore to limit concurrency
	sem := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup

	for _, service := range services {
		// Check if context is cancelled before launching goroutine
		select {
		case <-ctx.Done():
			// Context cancelled - stop launching new goroutines
			wg.Wait() // Wait for already-launched goroutines to finish
			return
		default:
		}

		wg.Add(1)
		go func(svc *models.Service) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			// Check the service
			if err := h.healthCheckService.CheckService(ctx, svc); err != nil {
				fmt.Printf("Error checking service %s: %v\n", svc.Name, err)
			}
		}(service)
	}

	// Wait for all checks to complete
	wg.Wait()
}
