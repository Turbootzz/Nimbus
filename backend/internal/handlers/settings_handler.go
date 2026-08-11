package handlers

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/services"
)

const invalidTLSModeError = "SMTP TLS mode must be 'starttls', 'tls' or 'none'"

type SettingsHandler struct {
	settingsRepo *repository.SettingsRepository
	emailService *services.EmailService
}

func NewSettingsHandler(settingsRepo *repository.SettingsRepository, emailService *services.EmailService) *SettingsHandler {
	return &SettingsHandler{
		settingsRepo: settingsRepo,
		emailService: emailService,
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
	switch key {
	case "public_registration_enabled", "smtp_enabled", "smtp_tls_skip_verify":
		if req.Value != "true" && req.Value != "false" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Value must be 'true' or 'false'",
			})
		}
		if key == "public_registration_enabled" && req.Value == "true" && os.Getenv("DISABLE_PUBLIC_REGISTRATION") == "true" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Registration is disabled by server configuration",
			})
		}
	case "smtp_port":
		port, err := strconv.Atoi(req.Value)
		if err != nil || port < 1 || port > 65535 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "SMTP port must be a valid number",
			})
		}
	case "smtp_tls_mode":
		if !services.IsValidTLSMode(req.Value) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": invalidTLSModeError,
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

// TLS fields are pointers so an omitted field leaves the stored value untouched
type smtpSettingsRequest struct {
	Host          string  `json:"smtp_host"`
	Port          string  `json:"smtp_port"`
	Username      string  `json:"smtp_username"`
	Password      string  `json:"smtp_password"`
	FromEmail     string  `json:"smtp_from_email"`
	FromName      string  `json:"smtp_from_name"`
	Enabled       string  `json:"smtp_enabled"`
	TLSMode       *string `json:"smtp_tls_mode"`
	TLSSkipVerify *string `json:"smtp_tls_skip_verify"`
}

func (r smtpSettingsRequest) tlsMode() string {
	if r.TLSMode == nil {
		return ""
	}
	return *r.TLSMode
}

func (r smtpSettingsRequest) tlsSkipVerify() string {
	if r.TLSSkipVerify == nil {
		return ""
	}
	return *r.TLSSkipVerify
}

func (r smtpSettingsRequest) usesNoTLS() bool {
	return strings.EqualFold(strings.TrimSpace(r.tlsMode()), services.TLSModeNone)
}

// UpdateSMTPSettings saves all SMTP settings atomically
func (h *SettingsHandler) UpdateSMTPSettings(c *fiber.Ctx) error {
	var req smtpSettingsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate boolean
	if req.Enabled != "true" && req.Enabled != "false" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "smtp_enabled must be 'true' or 'false'",
		})
	}

	if v := req.tlsSkipVerify(); v != "" && v != "true" && v != "false" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "smtp_tls_skip_verify must be 'true' or 'false'",
		})
	}

	if !services.IsValidTLSMode(req.tlsMode()) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": invalidTLSModeError,
		})
	}

	// Reject at save time what the send path would reject anyway
	if req.usesNoTLS() && (req.Username != "" || req.Password != "") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "SMTP username and password must be empty when encryption is disabled",
		})
	}

	// Validate port
	if req.Port != "" {
		port, err := strconv.Atoi(req.Port)
		if err != nil || port < 1 || port > 65535 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "SMTP port must be a valid number",
			})
		}
	}

	// When enabling SMTP, require essential fields
	if req.Enabled == "true" {
		if req.Host == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "SMTP host is required when SMTP is enabled",
			})
		}
		if req.FromEmail == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "From email is required when SMTP is enabled",
			})
		}
	}

	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}

	settings := map[string]string{
		"smtp_host":       req.Host,
		"smtp_port":       req.Port,
		"smtp_username":   req.Username,
		"smtp_password":   req.Password,
		"smtp_from_email": req.FromEmail,
		"smtp_from_name":  req.FromName,
		"smtp_enabled":    req.Enabled,
	}
	if req.TLSMode != nil {
		settings["smtp_tls_mode"] = *req.TLSMode
	}
	if req.TLSSkipVerify != nil {
		settings["smtp_tls_skip_verify"] = *req.TLSSkipVerify
	}

	if err := h.settingsRepo.UpdateBatch(c.Context(), settings, &userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save SMTP settings",
		})
	}

	return c.JSON(fiber.Map{
		"message": "SMTP settings saved successfully",
	})
}

// GetSMTPStatus returns SMTP configuration status (configured, source)
func (h *SettingsHandler) GetSMTPStatus(c *fiber.Ctx) error {
	status := h.emailService.GetSMTPStatus(c.Context())
	return c.JSON(status)
}

// TestSMTPConnection tests the SMTP connection.
// If a request body with SMTP settings is provided, those are used (pre-save test).
// Otherwise, the saved/env config is used.
func (h *SettingsHandler) TestSMTPConnection(c *fiber.Ctx) error {
	var testErr error

	// Try to parse inline config from request body
	var req smtpSettingsRequest

	if len(c.Body()) > 0 && c.BodyParser(&req) == nil && req.Host != "" {
		if !services.IsValidTLSMode(req.tlsMode()) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": invalidTLSModeError,
			})
		}

		// Use inline config for pre-save testing
		port := 587
		if req.Port != "" {
			if p, err := strconv.Atoi(req.Port); err == nil {
				port = p
			}
		}
		config := &services.SMTPConfig{
			Host:          req.Host,
			Port:          port,
			Username:      req.Username,
			Password:      req.Password,
			FromEmail:     req.FromEmail,
			FromName:      req.FromName,
			Enabled:       req.Enabled != "false",
			TLSMode:       req.tlsMode(),
			TLSSkipVerify: req.tlsSkipVerify() == "true",
		}
		testErr = h.emailService.TestConnectionWithConfig(config)
	} else {
		// Fall back to saved config
		testErr = h.emailService.TestConnection(c.Context())
	}

	if testErr != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": testErr.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "SMTP connection successful",
	})
}
