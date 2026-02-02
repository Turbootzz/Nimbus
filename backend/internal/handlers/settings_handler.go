package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
)

type SettingsHandler struct {
	settingsRepo *repository.SettingsRepository
}

func NewSettingsHandler(settingsRepo *repository.SettingsRepository) *SettingsHandler {
	return &SettingsHandler{
		settingsRepo: settingsRepo,
	}
}

// GetSettings returns all system settings (admin only)
func (h *SettingsHandler) GetSettings(c *fiber.Ctx) error {
	settings, err := h.settingsRepo.GetAll(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve settings",
		})
	}

	return c.JSON(fiber.Map{
		"settings": settings,
	})
}

// GetSetting returns a specific setting by key (admin only)
func (h *SettingsHandler) GetSetting(c *fiber.Ctx) error {
	key := c.Params("key")
	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Setting key is required",
		})
	}

	setting, err := h.settingsRepo.Get(c.Context(), key)
	if err != nil {
		if errors.Is(err, repository.ErrSettingNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Setting not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve setting",
		})
	}

	return c.JSON(setting)
}

// GetPublicRegistrationStatus returns whether public registration is enabled (public endpoint)
func (h *SettingsHandler) GetPublicRegistrationStatus(c *fiber.Ctx) error {
	enabled, err := h.settingsRepo.IsPublicRegistrationEnabled(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check registration status",
		})
	}

	return c.JSON(fiber.Map{
		"enabled": enabled,
	})
}

// UpdateSetting updates a setting value (admin only)
func (h *SettingsHandler) UpdateSetting(c *fiber.Ctx) error {
	key := c.Params("key")
	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Setting key is required",
		})
	}

	// Parse request body
	var req models.UpdateSettingRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate value for known settings
	if key == "public_registration_enabled" {
		if req.Value != "true" && req.Value != "false" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Value must be 'true' or 'false'",
			})
		}
	}

	// Get current user ID for audit trail
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}

	// Update setting
	err = h.settingsRepo.Update(c.Context(), key, req.Value, &userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update setting",
		})
	}

	// Return updated setting
	setting, err := h.settingsRepo.Get(c.Context(), key)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Setting updated but failed to retrieve",
		})
	}

	return c.JSON(setting)
}
