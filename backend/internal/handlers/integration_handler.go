package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/services"
)

type IntegrationHandler struct {
	service *services.IntegrationService
}

func NewIntegrationHandler(service *services.IntegrationService) *IntegrationHandler {
	return &IntegrationHandler{service: service}
}

// ListKinds returns the integration kinds the server supports
func (h *IntegrationHandler) ListKinds(c *fiber.Ctx) error {
	if _, err := RequireUserID(c); err != nil {
		return err
	}
	return c.JSON(h.service.Kinds(IsAdmin(c)))
}

// ListIntegrations returns the user's integrations (never their credentials)
func (h *IntegrationHandler) ListIntegrations(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}
	list, err := h.service.List(c.Context(), userID)
	if err != nil {
		return integrationError(c, err, "list integrations")
	}
	return c.JSON(list)
}

// GetIntegration returns one integration owned by the user
func (h *IntegrationHandler) GetIntegration(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}
	integration, err := h.service.Get(c.Context(), c.Params("id"), userID)
	if err != nil {
		return integrationError(c, err, "get integration")
	}
	return c.JSON(integration)
}

// CreateIntegration stores a new integration with encrypted credentials
func (h *IntegrationHandler) CreateIntegration(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}
	var req models.IntegrationRequest
	if err := c.BodyParser(&req); err != nil {
		return BadRequest(c, "Invalid request body")
	}
	integration, err := h.service.Create(c.Context(), userID, IsAdmin(c), &req)
	if err != nil {
		return integrationError(c, err, "create integration")
	}
	return Created(c, integration)
}

// UpdateIntegration changes an integration; omitted credentials are kept
func (h *IntegrationHandler) UpdateIntegration(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}
	var req models.IntegrationRequest
	if err := c.BodyParser(&req); err != nil {
		return BadRequest(c, "Invalid request body")
	}
	// UserContext: a reconnect may log out of the app over the network
	integration, err := h.service.Update(c.UserContext(), c.Params("id"), userID, IsAdmin(c), &req)
	if err != nil {
		return integrationError(c, err, "update integration")
	}
	return c.JSON(integration)
}

// DeleteIntegration removes an integration owned by the user
func (h *IntegrationHandler) DeleteIntegration(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}
	if err := h.service.Delete(c.UserContext(), c.Params("id"), userID); err != nil {
		return integrationError(c, err, "delete integration")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// TestUnsavedIntegration tests a connection from the request body without saving
func (h *IntegrationHandler) TestUnsavedIntegration(c *fiber.Ctx) error {
	if _, err := RequireUserID(c); err != nil {
		return err
	}
	var req models.IntegrationRequest
	if err := c.BodyParser(&req); err != nil {
		return BadRequest(c, "Invalid request body")
	}
	// UserContext: outgoing requests must not hold on to fasthttp's recycled RequestCtx
	result, err := h.service.TestUnsaved(c.UserContext(), IsAdmin(c), &req)
	if err != nil {
		return integrationError(c, err, "test integration")
	}
	return c.JSON(result)
}

// TestIntegration tests a saved integration and records the result
func (h *IntegrationHandler) TestIntegration(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}
	// UserContext: outgoing requests must not hold on to fasthttp's recycled RequestCtx
	result, err := h.service.TestSaved(c.UserContext(), c.Params("id"), userID)
	if err != nil {
		return integrationError(c, err, "test integration")
	}
	return c.JSON(result)
}

var integrationNotFound = map[error]string{repository.ErrIntegrationNotFound: "Integration not found"}

// integrationError maps service errors to responses without leaking internals
func integrationError(c *fiber.Ctx, err error, action string) error {
	return serviceError(c, err, action, integrationNotFound)
}
