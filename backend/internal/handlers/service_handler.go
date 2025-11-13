package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/services"
	"github.com/nimbus/backend/internal/utils"
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

	// Validate and set icon fields
	iconType := req.IconType
	if iconType == "" {
		iconType = models.IconTypeEmoji
	}

	// Validate icon_type
	if iconType != models.IconTypeEmoji && iconType != models.IconTypeImageUpload && iconType != models.IconTypeImageURL {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid icon_type. Must be 'emoji', 'image_upload', or 'image_url'",
		})
	}

	// Set default emoji if emoji type and no icon provided
	icon := req.Icon
	if iconType == models.IconTypeEmoji && icon == "" {
		icon = models.DefaultIcon
	}

	// Validate icon_image_path for image types
	iconImagePath := strings.TrimSpace(req.IconImagePath)
	if iconType == models.IconTypeImageUpload {
		if iconImagePath == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "icon_image_path is required for image_upload type",
			})
		}
		// Prevent path traversal attacks - only allow filename, no path separators
		if strings.Contains(iconImagePath, "..") || strings.Contains(iconImagePath, "/") || strings.Contains(iconImagePath, "\\") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "icon_image_path must be a filename only, no path separators allowed",
			})
		}
	}
	if iconType == models.IconTypeImageURL {
		if iconImagePath == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "icon_image_path (URL) is required for image_url type",
			})
		}
		// Validate URL format and security (prevent SSRF attacks)
		if err := utils.ValidateExternalImageURL(iconImagePath); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("Invalid or unsafe image URL: %s", err.Error()),
			})
		}
	}

	// Create service
	service := &models.Service{
		UserID:        userID,
		Name:          req.Name,
		URL:           req.URL,
		Icon:          icon,
		IconType:      iconType,
		IconImagePath: iconImagePath,
		Description:   req.Description,
		Status:        models.StatusUnknown, // Initial status
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
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

	// Validate and set icon fields - preserve existing values if not provided
	iconType := req.IconType
	if iconType == "" {
		iconType = existingService.IconType // Preserve existing icon_type instead of defaulting to emoji
	}

	// Validate icon_type
	if iconType != models.IconTypeEmoji && iconType != models.IconTypeImageUpload && iconType != models.IconTypeImageURL {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid icon_type. Must be 'emoji', 'image_upload', or 'image_url'",
		})
	}

	// Determine effective icon (use incoming value or preserve existing)
	icon := req.Icon
	if icon == "" {
		icon = existingService.Icon // Preserve existing icon if not provided
		// Only default to DefaultIcon if both req.Icon and existingService.Icon are empty
		if icon == "" && iconType == models.IconTypeEmoji {
			icon = models.DefaultIcon
		}
	}

	// Determine effective icon_image_path (use incoming value or preserve existing)
	iconImagePath := strings.TrimSpace(req.IconImagePath)
	if iconImagePath == "" {
		iconImagePath = existingService.IconImagePath // Preserve existing path if not provided
	}

	// Validate icon_image_path for image types (only when effective path is being used)
	if iconType == models.IconTypeImageUpload {
		if iconImagePath == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "icon_image_path is required for image_upload type",
			})
		}
		// Prevent path traversal attacks - only allow filename, no path separators
		if strings.Contains(iconImagePath, "..") || strings.Contains(iconImagePath, "/") || strings.Contains(iconImagePath, "\\") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "icon_image_path must be a filename only, no path separators allowed",
			})
		}
	}
	if iconType == models.IconTypeImageURL {
		if iconImagePath == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "icon_image_path (URL) is required for image_url type",
			})
		}
		// Validate URL format and security (prevent SSRF attacks) only if non-empty
		if iconImagePath != "" {
			if err := utils.ValidateExternalImageURL(iconImagePath); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": fmt.Sprintf("Invalid or unsafe image URL: %s", err.Error()),
				})
			}
		}
	}

	// Delete old uploaded image if switching away from image_upload
	if existingService.IconType == models.IconTypeImageUpload && iconType != models.IconTypeImageUpload && existingService.IconImagePath != "" {
		// Sanitize filename to prevent path traversal
		safeFilename := filepath.Base(existingService.IconImagePath)
		// Validate sanitized filename before deletion
		if safeFilename != "" && safeFilename != "." && safeFilename != ".." &&
			!strings.Contains(safeFilename, "/") && !strings.Contains(safeFilename, "\\") {
			oldFilePath := filepath.Join(UploadDir, safeFilename)
			os.Remove(oldFilePath) // Ignore error, file may already be deleted
		}
	}

	// Update service
	existingService.Name = req.Name
	existingService.URL = req.URL
	existingService.Icon = icon
	existingService.IconType = iconType
	existingService.IconImagePath = iconImagePath
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

	// Get existing service to check for uploaded images to clean up
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

	// Verify service belongs to user
	if existingService.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
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

	// Clean up uploaded image file if exists
	if existingService.IconType == models.IconTypeImageUpload && existingService.IconImagePath != "" {
		// Sanitize filename to prevent path traversal
		safeFilename := filepath.Base(existingService.IconImagePath)
		// Validate sanitized filename before deletion
		if safeFilename != "" && safeFilename != "." && safeFilename != ".." &&
			!strings.Contains(safeFilename, "/") && !strings.Contains(safeFilename, "\\") {
			filePath := filepath.Join(UploadDir, safeFilename)
			os.Remove(filePath) // Ignore error, file may already be deleted
		}
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

// ReorderServices handles bulk position updates for services
func (h *ServiceHandler) ReorderServices(c *fiber.Ctx) error {
	// Get user ID from context
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized: user ID not found",
		})
	}

	var req models.ServiceReorderRequest

	// Parse request body
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if len(req.Services) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "At least one service position is required",
		})
	}

	// Convert to map for repository method
	positions := make(map[string]int)
	for _, sp := range req.Services {
		if sp.ID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Service ID cannot be empty",
			})
		}
		if sp.Position < 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Position must be non-negative",
			})
		}
		positions[sp.ID] = sp.Position
	}

	// Update positions in database (validates ownership)
	if err := h.serviceRepo.UpdatePositions(c.Context(), userID, positions); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "One or more services not found or access denied",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update service positions",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Service positions updated successfully",
	})
}
