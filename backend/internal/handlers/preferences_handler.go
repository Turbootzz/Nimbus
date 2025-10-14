package handlers

import (
	"database/sql"
	"fmt"
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
	return &PreferencesHandler{
		preferencesRepo: preferencesRepo,
		validator:       validator.New(),
	}
}

// GetPreferences retrieves the current user's preferences
func (h *PreferencesHandler) GetPreferences(c *fiber.Ctx) error {
	// Get user ID from context (set by auth middleware)
	userID, ok := c.Locals("userID").(string)
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
	userID, ok := c.Locals("userID").(string)
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

	// Validate request using struct tags
	if err := h.validator.Struct(req); err != nil {
		// Parse validation errors and return field-specific messages
		validationErrors := err.(validator.ValidationErrors)
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
			default:
				errorMessages[field] = fmt.Sprintf("%s is invalid", field)
			}
		}

		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Validation failed",
			"fields": errorMessages,
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
