package handlers

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
)

type PreferencesHandler struct {
	preferencesRepo *repository.PreferencesRepository
}

func NewPreferencesHandler(preferencesRepo *repository.PreferencesRepository) *PreferencesHandler {
	return &PreferencesHandler{
		preferencesRepo: preferencesRepo,
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
		return c.JSON(fiber.Map{
			"theme_mode": "light",
			"updated_at": nil,
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

	// Validate theme_mode
	if req.ThemeMode != "light" && req.ThemeMode != "dark" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "theme_mode must be 'light' or 'dark'",
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
