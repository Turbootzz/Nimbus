package handlers

import (
	"database/sql"
	"errors"
	"sort"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/services"
	"github.com/nimbus/backend/internal/utils"
)

const maxWebhooksPerUser = 10

type WebhookHandler struct {
	webhookRepo         *repository.WebhookRepository
	notificationService *services.NotificationService
}

func NewWebhookHandler(webhookRepo *repository.WebhookRepository, notificationService *services.NotificationService) *WebhookHandler {
	return &WebhookHandler{
		webhookRepo:         webhookRepo,
		notificationService: notificationService,
	}
}

// invalidFormatError lists the accepted formats
const invalidFormatError = "Invalid format. Must be one of: generic, discord, slack, ntfy, gotify, telegram, pushover"

// webhookFieldErrors answers 400 with per-field messages for the form. The
// first message (by field name, so it is stable) doubles as the summary.
func webhookFieldErrors(c *fiber.Ctx, fields map[string]string) error {
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"error":  fields[names[0]],
		"fields": fields,
	})
}

// CreateWebhook handles webhook creation
func (h *WebhookHandler) CreateWebhook(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}

	var req models.WebhookCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Name is required",
		})
	}
	if len(req.Name) > 100 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Name must be 100 characters or less",
		})
	}

	// Validate format
	format := req.Format
	if format == "" {
		format = models.WebhookFormatGeneric
	}
	if !models.IsValidWebhookFormat(format) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": invalidFormatError,
		})
	}

	// Telegram and Pushover post to a fixed endpoint; the rest need a URL
	if models.WebhookFixedURL(format) == "" {
		if req.URL == "" {
			return webhookFieldErrors(c, map[string]string{"url": "URL is required"})
		}

		// Validate URL security
		if err := utils.ValidateWebhookURL(req.URL); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":  "Invalid webhook URL",
				"detail": err.Error(),
				"fields": map[string]string{"url": "Invalid webhook URL: " + err.Error()},
			})
		}
	}

	// Check webhook limit
	count, err := h.webhookRepo.CountByUserID(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check webhook limit",
		})
	}
	if count >= maxWebhooksPerUser {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Maximum webhook limit reached (10)",
		})
	}

	// Set defaults
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	triggers := models.WebhookTriggers{OnOffline: true, OnOnline: false}
	if req.Triggers != nil {
		triggers = *req.Triggers
	}

	retryCount := 0
	if req.RetryCount != nil {
		if *req.RetryCount < 0 || *req.RetryCount > 5 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Retry count must be between 0 and 5",
			})
		}
		retryCount = *req.RetryCount
	}

	retryDelaySeconds := 30
	if req.RetryDelaySeconds != nil {
		if *req.RetryDelaySeconds < 10 || *req.RetryDelaySeconds > 300 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Retry delay must be between 10 and 300 seconds",
			})
		}
		retryDelaySeconds = *req.RetryDelaySeconds
	}

	now := time.Now()
	webhook := &models.Webhook{
		ID:                uuid.New().String(), // credentials are bound to it
		UserID:            userID,
		Name:              req.Name,
		URL:               req.URL,
		Enabled:           enabled,
		Triggers:          triggers,
		Format:            format,
		VerifyTLS:         req.VerifyTLS == nil || *req.VerifyTLS,
		RetryCount:        retryCount,
		RetryDelaySeconds: retryDelaySeconds,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if req.Options != nil {
		webhook.Options = *req.Options
	}

	creds := models.WebhookCredentials{}
	if req.Credentials != nil {
		creds = *req.Credentials
	}
	if fields := services.PrepareWebhookTarget(webhook, &creds); len(fields) > 0 {
		return webhookFieldErrors(c, fields)
	}
	webhook.CredentialsEnc, err = h.notificationService.EncryptWebhookCredentials(creds, webhook.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to store webhook credentials",
		})
	}

	if err := h.webhookRepo.Create(c.Context(), webhook); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create webhook",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(webhook.ToResponse())
}

// GetWebhooks retrieves all webhooks for the authenticated user
func (h *WebhookHandler) GetWebhooks(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}

	webhooks, err := h.webhookRepo.GetAllByUserID(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve webhooks",
		})
	}

	response := make([]models.WebhookResponse, 0, len(webhooks))
	for _, webhook := range webhooks {
		response = append(response, webhook.ToResponse())
	}

	return c.JSON(response)
}

// GetWebhook retrieves a single webhook by ID
func (h *WebhookHandler) GetWebhook(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}

	webhookID, err := RequireUUIDParam(c, "id")
	if err != nil {
		return err
	}

	webhook, err := h.webhookRepo.GetByID(c.Context(), webhookID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Webhook not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve webhook",
		})
	}

	return c.JSON(webhook.ToResponse())
}

// UpdateWebhook handles webhook updates
func (h *WebhookHandler) UpdateWebhook(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}

	webhookID, err := RequireUUIDParam(c, "id")
	if err != nil {
		return err
	}

	var req models.WebhookUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Get existing webhook
	webhook, err := h.webhookRepo.GetByID(c.Context(), webhookID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Webhook not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve webhook",
		})
	}

	// Apply updates
	if req.Name != nil {
		if *req.Name == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Name cannot be empty",
			})
		}
		if len(*req.Name) > 100 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Name must be 100 characters or less",
			})
		}
		webhook.Name = *req.Name
	}

	// A new format starts without the old format's settings and secrets
	oldFormat := webhook.Format
	if req.Format != nil {
		if !models.IsValidWebhookFormat(*req.Format) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": invalidFormatError,
			})
		}
		webhook.Format = *req.Format
	}
	formatChanged := webhook.Format != oldFormat

	if models.WebhookFixedURL(webhook.Format) == "" {
		if req.URL != nil {
			if err := utils.ValidateWebhookURL(*req.URL); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error":  "Invalid webhook URL",
					"detail": err.Error(),
					"fields": map[string]string{"url": "Invalid webhook URL: " + err.Error()},
				})
			}
			webhook.URL = *req.URL
		} else if models.WebhookFixedURL(oldFormat) != "" {
			// The stored URL is the old format's API endpoint, not the user's
			return webhookFieldErrors(c, map[string]string{"url": "URL is required"})
		}
	}

	if req.Options != nil {
		webhook.Options = *req.Options
	} else if formatChanged {
		webhook.Options = models.WebhookOptions{}
	}

	if req.VerifyTLS != nil {
		webhook.VerifyTLS = *req.VerifyTLS
	}

	// Omitted credentials keep the stored blob untouched
	var creds *models.WebhookCredentials
	if req.Credentials != nil {
		creds = req.Credentials
	} else if formatChanged {
		creds = &models.WebhookCredentials{}
	}
	if fields := services.PrepareWebhookTarget(webhook, creds); len(fields) > 0 {
		return webhookFieldErrors(c, fields)
	}
	if creds != nil {
		webhook.CredentialsEnc, err = h.notificationService.EncryptWebhookCredentials(*creds, webhook.ID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to store webhook credentials",
			})
		}
	}

	if req.Enabled != nil {
		webhook.Enabled = *req.Enabled
	}

	if req.Triggers != nil {
		webhook.Triggers = *req.Triggers
	}

	if req.RetryCount != nil {
		if *req.RetryCount < 0 || *req.RetryCount > 5 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Retry count must be between 0 and 5",
			})
		}
		webhook.RetryCount = *req.RetryCount
	}

	if req.RetryDelaySeconds != nil {
		if *req.RetryDelaySeconds < 10 || *req.RetryDelaySeconds > 300 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Retry delay must be between 10 and 300 seconds",
			})
		}
		webhook.RetryDelaySeconds = *req.RetryDelaySeconds
	}

	webhook.UpdatedAt = time.Now()

	if err := h.webhookRepo.Update(c.Context(), webhook); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update webhook",
		})
	}

	return c.JSON(webhook.ToResponse())
}

// DeleteWebhook handles webhook deletion
func (h *WebhookHandler) DeleteWebhook(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}

	webhookID, err := RequireUUIDParam(c, "id")
	if err != nil {
		return err
	}

	if err := h.webhookRepo.Delete(c.Context(), webhookID, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Webhook not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete webhook",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// TestWebhook sends a test notification to a webhook
func (h *WebhookHandler) TestWebhook(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}

	webhookID, err := RequireUUIDParam(c, "id")
	if err != nil {
		return err
	}

	// Verify webhook exists and belongs to user
	_, err = h.webhookRepo.GetByID(c.Context(), webhookID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Webhook not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve webhook",
		})
	}

	result, err := h.notificationService.TestWebhook(c.Context(), webhookID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to test webhook",
		})
	}

	response := models.WebhookTestResult{
		Success: result.Success,
	}

	if result.Success {
		response.Message = "Test notification sent successfully"
	} else {
		response.Error = result.Error
	}

	if result.StatusCode > 0 {
		response.StatusCode = &result.StatusCode
	}
	if result.ResponseTimeMs > 0 {
		response.ResponseTimeMs = &result.ResponseTimeMs
	}

	if result.Success {
		return c.JSON(response)
	}
	return c.Status(fiber.StatusBadGateway).JSON(response)
}

// GetWebhookLogs retrieves delivery logs for a webhook
func (h *WebhookHandler) GetWebhookLogs(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}

	webhookID, err := RequireUUIDParam(c, "id")
	if err != nil {
		return err
	}

	// Verify webhook exists and belongs to user
	_, err = h.webhookRepo.GetByID(c.Context(), webhookID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Webhook not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve webhook",
		})
	}

	// Parse limit from query params
	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	logs, err := h.webhookRepo.GetLogsByWebhookID(c.Context(), webhookID, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve webhook logs",
		})
	}

	response := make([]models.WebhookLogResponse, 0, len(logs))
	for _, log := range logs {
		response = append(response, log.ToResponse())
	}

	return c.JSON(response)
}
