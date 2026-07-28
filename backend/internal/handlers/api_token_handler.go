package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/utils"
)

const maxAPITokensPerUser = 10

type APITokenHandler struct {
	tokenRepo *repository.APITokenRepository
}

func NewAPITokenHandler(tokenRepo *repository.APITokenRepository) *APITokenHandler {
	return &APITokenHandler{tokenRepo: tokenRepo}
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

	count, err := h.tokenRepo.CountByUserID(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check token limit",
		})
	}
	if count >= maxAPITokensPerUser {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Maximum token limit reached (10)",
		})
	}

	plaintext, hash, prefix, err := utils.GenerateAPIToken()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate token",
		})
	}

	// Tokens default to read-only; write access must be requested explicitly
	readOnly := true
	if req.ReadOnly != nil {
		readOnly = *req.ReadOnly
	}

	apiToken := &models.APIToken{
		UserID:      userID,
		Name:        req.Name,
		TokenHash:   hash,
		TokenPrefix: prefix,
		ReadOnly:    readOnly,
	}

	if err := h.tokenRepo.Create(c.Context(), apiToken); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create token",
		})
	}

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

	return c.SendStatus(fiber.StatusNoContent)
}
