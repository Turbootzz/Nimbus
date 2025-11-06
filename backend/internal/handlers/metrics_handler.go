package handlers

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/services"
)

type MetricsHandler struct {
	metricsService *services.MetricsService
	serviceRepo    repository.ServiceRepositoryInterface
}

func NewMetricsHandler(metricsService *services.MetricsService, serviceRepo repository.ServiceRepositoryInterface) *MetricsHandler {
	return &MetricsHandler{
		metricsService: metricsService,
		serviceRepo:    serviceRepo,
	}
}

// GetServiceMetrics retrieves metrics for a specific service
// GET /api/v1/metrics/:serviceID
func (h *MetricsHandler) GetServiceMetrics(c *fiber.Ctx) error {
	serviceID := c.Params("id")
	if serviceID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Service ID is required",
		})
	}

	// Get authenticated user
	userID, ok := c.Locals("user_id").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Verify service belongs to user
	service, err := h.serviceRepo.GetByID(c.Context(), serviceID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Service not found",
		})
	}

	if service.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	// Parse query parameters
	timeRange := c.Query("range", "24h")    // 1h, 6h, 24h, 7d, 30d
	intervalStr := c.Query("interval", "5") // interval in minutes

	interval, err := strconv.Atoi(intervalStr)
	if err != nil || interval < 1 {
		interval = 5
	}

	// Calculate start and end times based on range
	endTime := time.Now()
	var startTime time.Time

	switch timeRange {
	case "1h":
		startTime = endTime.Add(-1 * time.Hour)
	case "6h":
		startTime = endTime.Add(-6 * time.Hour)
	case "24h":
		startTime = endTime.Add(-24 * time.Hour)
	case "7d":
		startTime = endTime.AddDate(0, 0, -7)
	case "30d":
		startTime = endTime.AddDate(0, 0, -30)
	default:
		startTime = endTime.Add(-24 * time.Hour)
	}

	// Get metrics
	metrics, err := h.metricsService.GetServiceMetrics(c.Context(), serviceID, startTime, endTime, interval)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve metrics",
		})
	}

	return c.JSON(metrics)
}

// GetRecentStatusLogs retrieves recent status logs for a service
// GET /api/v1/services/:id/status-logs
func (h *MetricsHandler) GetRecentStatusLogs(c *fiber.Ctx) error {
	serviceID := c.Params("id")
	if serviceID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Service ID is required",
		})
	}

	// Get authenticated user
	userID, ok := c.Locals("user_id").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Verify service belongs to user
	service, err := h.serviceRepo.GetByID(c.Context(), serviceID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Service not found",
		})
	}

	if service.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	// Parse limit from query
	limitStr := c.Query("limit", "100")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 1000 {
		limit = 100
	}

	// Get status logs
	logs, err := h.metricsService.GetRecentStatusLogs(c.Context(), serviceID, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve status logs",
		})
	}

	// Convert to response format
	responses := make([]interface{}, len(logs))
	for i, log := range logs {
		responses[i] = log.ToResponse()
	}

	return c.JSON(fiber.Map{
		"logs":  responses,
		"count": len(responses),
	})
}

// GetPrometheusMetrics exports metrics in Prometheus format
// GET /metrics
func (h *MetricsHandler) GetPrometheusMetrics(c *fiber.Ctx) error {
	metrics, err := h.metricsService.GetPrometheusMetrics(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("# Error retrieving metrics\n")
	}

	// Format as Prometheus text
	output := services.FormatPrometheusMetrics(metrics)

	// Set content type for Prometheus
	c.Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	return c.SendString(output)
}
