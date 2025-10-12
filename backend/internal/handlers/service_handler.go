package handlers

import (
	"database/sql"
	"errors"
	"net/url"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/services"
)

type ServiceHandler struct {
	serviceRepo        *repository.ServiceRepository
	healthCheckService *services.HealthCheckService
}

func NewServiceHandler(serviceRepo *repository.ServiceRepository, healthCheckService *services.HealthCheckService) *ServiceHandler {
	return &ServiceHandler{
		serviceRepo:        serviceRepo,
		healthCheckService: healthCheckService,
	}
}

// CreateService handles service creation
func (h *ServiceHandler) CreateService(c *fiber.Ctx) error {
	// Get user ID from context (set by auth middleware)
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized: user ID not found",
		})
	}

	var req models.ServiceCreateRequest

	// Parse request body
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.Name == "" || req.URL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Name and URL are required",
		})
	}

	// Validate URL format
	parsedURL, err := url.ParseRequestURI(req.URL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid URL format. URL must include scheme (http/https) and host",
		})
	}

	// Set default icon if not provided
	if req.Icon == "" {
		req.Icon = models.DefaultIcon
	}

	// Create service
	service := &models.Service{
		UserID:      userID,
		Name:        req.Name,
		URL:         req.URL,
		Icon:        req.Icon,
		Description: req.Description,
		Status:      models.StatusUnknown, // Initial status
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := h.serviceRepo.Create(c.Context(), service); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create service",
		})
	}

	// Return created service
	return c.Status(fiber.StatusCreated).JSON(service.ToResponse())
}

// GetServices retrieves all services for the authenticated user
func (h *ServiceHandler) GetServices(c *fiber.Ctx) error {
	// Get user ID from context
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized: user ID not found",
		})
	}

	// Get all services for user
	services, err := h.serviceRepo.GetAllByUserID(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve services",
		})
	}

	// Convert to response format
	response := make([]models.ServiceResponse, 0, len(services))
	for _, service := range services {
		response = append(response, service.ToResponse())
	}

	return c.JSON(response)
}

// GetService retrieves a single service by ID
func (h *ServiceHandler) GetService(c *fiber.Ctx) error {
	// Get user ID from context
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized: user ID not found",
		})
	}

	// Get service ID from URL params
	serviceID := c.Params("id")
	if serviceID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Service ID is required",
		})
	}

	// Get service from database
	service, err := h.serviceRepo.GetByID(c.Context(), serviceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Service not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve service",
		})
	}

	if service == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Service not found",
		})
	}

	// Verify service belongs to user
	if service.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	return c.JSON(service.ToResponse())
}

// UpdateService handles service updates
func (h *ServiceHandler) UpdateService(c *fiber.Ctx) error {
	// Get user ID from context
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized: user ID not found",
		})
	}

	// Get service ID from URL params
	serviceID := c.Params("id")
	if serviceID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Service ID is required",
		})
	}

	var req models.ServiceUpdateRequest

	// Parse request body
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.Name == "" || req.URL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Name and URL are required",
		})
	}

	// Validate URL format
	parsedURL, err := url.ParseRequestURI(req.URL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid URL format. URL must include scheme (http/https) and host",
		})
	}

	// Get existing service to verify ownership
	existingService, err := h.serviceRepo.GetByID(c.Context(), serviceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Service not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve service",
		})
	}

	if existingService == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Service not found",
		})
	}

	// Verify service belongs to user
	if existingService.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	// Set default icon if not provided (mirrors create flow)
	icon := req.Icon
	if icon == "" {
		icon = models.DefaultIcon
	}

	// Update service
	existingService.Name = req.Name
	existingService.URL = req.URL
	existingService.Icon = icon
	existingService.Description = req.Description
	existingService.UpdatedAt = time.Now()

	if err := h.serviceRepo.Update(c.Context(), existingService); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update service",
		})
	}

	return c.JSON(existingService.ToResponse())
}

// DeleteService handles service deletion
func (h *ServiceHandler) DeleteService(c *fiber.Ctx) error {
	// Get user ID from context
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized: user ID not found",
		})
	}

	// Get service ID from URL params
	serviceID := c.Params("id")
	if serviceID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Service ID is required",
		})
	}

	// Delete service (repository checks ownership)
	if err := h.serviceRepo.Delete(c.Context(), serviceID, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Service not found or access denied",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete service",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Service deleted successfully",
	})
}

// CheckService manually triggers a health check for a specific service
func (h *ServiceHandler) CheckService(c *fiber.Ctx) error {
	// Get user ID from context
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized: user ID not found",
		})
	}

	// Get service ID from URL params
	serviceID := c.Params("id")
	if serviceID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Service ID is required",
		})
	}

	// Get service from database
	service, err := h.serviceRepo.GetByID(c.Context(), serviceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Service not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve service",
		})
	}

	if service == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Service not found",
		})
	}

	// Verify service belongs to user
	if service.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	// Perform health check
	if err := h.healthCheckService.CheckService(c.Context(), service); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to perform health check",
		})
	}

	// Fetch updated service to return new status and response time
	updatedService, err := h.serviceRepo.GetByID(c.Context(), serviceID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Health check completed but failed to fetch updated service",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Health check completed",
		"service": updatedService.ToResponse(),
	})
}
