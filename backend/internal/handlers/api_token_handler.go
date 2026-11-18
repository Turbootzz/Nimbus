package handlers

import (
	"errors"
	"fmt"
	"log"
	"net"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/utils"
)

const maxAPITokensPerUser = 10

type APITokenHandler struct {
	tokenRepo   *repository.APITokenRepository
	activityLog *repository.ActivityLogRepository
}

func NewAPITokenHandler(tokenRepo *repository.APITokenRepository) *APITokenHandler {
	return &APITokenHandler{tokenRepo: tokenRepo}
}

// SetActivityLog makes token creation and revocation show up in the activity log
func (h *APITokenHandler) SetActivityLog(activityLog *repository.ActivityLogRepository) {
	h.activityLog = activityLog
}

// logActivity records a token event; a failure never fails the request
func (h *APITokenHandler) logActivity(c *fiber.Ctx, userID, action string, details map[string]interface{}) {
	if h.activityLog == nil {
		return
	}
	entry := &models.UserActivityLog{UserID: &userID, ActorID: &userID, Action: action, Details: details}
	if ip := c.IP(); net.ParseIP(ip) != nil {
		entry.IPAddress = &ip
	}
	if err := h.activityLog.Create(c.Context(), entry); err != nil {
		log.Printf("Failed to log %s: %v", action, err)
	}
}

// requestedScopes validates the scopes of a create request and returns them
// in display order, or a message for the client. Without scopes the legacy
// read_only flag decides, and tokens default to read access.
func requestedScopes(req *models.APITokenCreateRequest) (scopes []string, problem string) {
	if req.Scopes == nil {
		return models.LegacyScopes(req.ReadOnly == nil || *req.ReadOnly), ""
	}
	if len(req.Scopes) == 0 {
		return nil, "At least one scope is required"
	}
	for _, scope := range req.Scopes {
		if !slices.Contains(models.APITokenScopes, scope) {
			return nil, fmt.Sprintf("Unknown scope %q", scope)
		}
	}
	for _, scope := range models.APITokenScopes {
		if slices.Contains(req.Scopes, scope) {
			scopes = append(scopes, scope)
		}
	}
	return scopes, ""
}

// CreateToken generates a new personal access token. The plaintext token is
// returned only in this response; only its hash is stored.
func (h *APITokenHandler) CreateToken(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}

	var req models.APITokenCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Name is required",
		})
	}
	if utf8.RuneCountInString(req.Name) > 100 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Name must be 100 characters or less",
		})
	}

	scopes, problem := requestedScopes(&req)
	if problem != "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": problem,
		})
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		if !req.ExpiresAt.After(time.Now()) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Expiry must be in the future",
			})
		}
		// The column has no time zone, so store UTC
		utc := req.ExpiresAt.UTC()
		expiresAt = &utc
	}

	plaintext, hash, prefix, err := utils.GenerateAPIToken()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate token",
		})
	}

	apiToken := &models.APIToken{
		UserID:      userID,
		Name:        req.Name,
		TokenHash:   hash,
		TokenPrefix: prefix,
		Scopes:      scopes,
		ExpiresAt:   expiresAt,
	}

	if err := h.tokenRepo.Create(c.Context(), apiToken, maxAPITokensPerUser); err != nil {
		if errors.Is(err, repository.ErrAPITokenLimitReached) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("Maximum token limit reached (%d)", maxAPITokensPerUser),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create token",
		})
	}

	h.logActivity(c, userID, models.ActionAPITokenCreated, map[string]interface{}{
		"token_id": apiToken.ID,
		"name":     apiToken.Name,
		"scopes":   apiToken.Scopes,
	})

	return c.Status(fiber.StatusCreated).JSON(models.APITokenCreateResponse{
		Token:    plaintext,
		APIToken: *apiToken,
	})
}

// ListTokens returns all tokens for the authenticated user (never the plaintext)
func (h *APITokenHandler) ListTokens(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}

	tokens, err := h.tokenRepo.ListByUserID(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list tokens",
		})
	}

	return c.JSON(tokens)
}

// DeleteToken revokes a token owned by the authenticated user
func (h *APITokenHandler) DeleteToken(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}

	id := c.Params("id")

	if err := h.tokenRepo.Delete(c.Context(), id, userID); err != nil {
		if errors.Is(err, repository.ErrAPITokenNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Token not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete token",
		})
	}

	h.logActivity(c, userID, models.ActionAPITokenRevoked, map[string]interface{}{"token_id": id})

	return c.SendStatus(fiber.StatusNoContent)
}
