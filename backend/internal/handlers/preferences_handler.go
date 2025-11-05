package handlers

import (
	"database/sql"
	"fmt"
	"net/url"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
)

type PreferencesHandler struct {
	preferencesRepo *repository.PreferencesRepository
	validator       *validator.Validate
}

func NewPreferencesHandler(preferencesRepo *repository.PreferencesRepository) *PreferencesHandler {
	v := validator.New()

	// Register custom validator for HTTP(S) URLs only
	v.RegisterValidation("httpurl", func(fl validator.FieldLevel) bool {
		urlStr := fl.Field().String()
		if urlStr == "" {
			return true // Empty is valid (omitempty will handle required check)
		}

		parsedURL, err := url.Parse(urlStr)
		if err != nil {
			return false
		}

		// Only allow http and https schemes to prevent XSS
		return parsedURL.Scheme == "http" || parsedURL.Scheme == "https"
	})

	return &PreferencesHandler{
		preferencesRepo: preferencesRepo,
		validator:       v,
	}
}

// GetPreferences retrieves the current user's preferences
func (h *PreferencesHandler) GetPreferences(c *fiber.Ctx) error {
	// Get user ID from context (set by auth middleware)
	userID, ok := c.Locals("user_id").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	preferences, err := h.preferencesRepo.GetByUserID(c.Context(), userID)
	if err == sql.ErrNoRows {
		// Return default preferences if user hasn't set any yet
		return c.JSON(models.PreferencesResponse{
			ThemeMode:        "light",
			ThemeBackground:  nil,
			ThemeAccentColor: nil,
			UpdatedAt:        time.Time{}, // Zero value for time
		})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve preferences",
		})
	}

	return c.JSON(preferences.ToResponse())
}

// UpdatePreferences updates the current user's preferences
func (h *PreferencesHandler) UpdatePreferences(c *fiber.Ctx) error {
	// Get user ID from context (set by auth middleware)
	userID, ok := c.Locals("user_id").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Parse request body
	var req models.PreferencesUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Log the incoming request for debugging
	fmt.Printf("[PreferencesHandler] UserID: %s, Update: ThemeMode=%v, ThemeBackground=%v, ThemeAccentColor=%v, OpenInNewTab=%v\n",
		userID, req.ThemeMode, req.ThemeBackground, req.ThemeAccentColor, req.OpenInNewTab)

	// Validate request using struct tags
	if err := h.validator.Struct(req); err != nil {
		// Use comma-ok to safely type assert validation errors
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			// Parse validation errors and return field-specific messages
			errorMessages := make(map[string]string)

			for _, fieldError := range validationErrors {
				field := fieldError.Field()
				switch fieldError.Tag() {
				case "required":
					errorMessages[field] = fmt.Sprintf("%s is required", field)
				case "oneof":
					errorMessages[field] = fmt.Sprintf("%s must be one of: %s", field, fieldError.Param())
				case "hexcolor":
					errorMessages[field] = fmt.Sprintf("%s must be a valid hex color (e.g., #3B82F6)", field)
				case "httpurl":
					errorMessages[field] = fmt.Sprintf("%s must be a valid HTTP or HTTPS URL", field)
				default:
					errorMessages[field] = fmt.Sprintf("%s is invalid", field)
				}
			}

			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":  "Validation failed",
				"fields": errorMessages,
			})
		}

		// If not ValidationErrors, treat as generic validation error
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Validation error: %s", err.Error()),
		})
	}

	// Upsert preferences (create if doesn't exist, update if exists)
	if err := h.preferencesRepo.Upsert(c.Context(), userID, &req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update preferences",
		})
	}

	// Retrieve and return updated preferences
	preferences, err := h.preferencesRepo.GetByUserID(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve updated preferences",
		})
	}

	return c.JSON(preferences.ToResponse())
}
