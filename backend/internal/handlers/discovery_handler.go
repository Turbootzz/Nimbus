package handlers

import (
	"errors"
	"net/url"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/services"
)

// DiscoveryHandler offers Docker containers as services. Admins only, like
// the Docker integration it reads: the container list shows what runs on
// the host.
type DiscoveryHandler struct {
	service *services.DiscoveryService
}

func NewDiscoveryHandler(service *services.DiscoveryService) *DiscoveryHandler {
	return &DiscoveryHandler{service: service}
}

// requireAdmin returns the admin's user ID, or an error to return as is
func requireAdmin(c *fiber.Ctx) (string, error) {
	userID, err := RequireUserID(c)
	if err != nil {
		return "", err
	}
	if !IsAdmin(c) {
		return "", fiber.NewError(fiber.StatusForbidden, "Admin access required")
	}
	return userID, nil
}

// requestHost is the host name the request was sent to, "" when it is not
// a plain host
func requestHost(c *fiber.Ctx) string {
	parsed, err := url.Parse("http://" + c.Hostname())
	if err != nil {
		return ""
	}
	return parsed.Hostname()
}

// DiscoverDocker lists the containers of the admin's Docker integrations
// that are not a service yet
func (h *DiscoveryHandler) DiscoverDocker(c *fiber.Ctx) error {
	userID, err := requireAdmin(c)
	if err != nil {
		return err
	}
	// UserContext: outgoing requests must not hold on to fasthttp's recycled RequestCtx
	result, err := h.service.Discover(c.UserContext(), userID, requestHost(c))
	if err != nil {
		return discoveryError(c, err, "discover containers")
	}
	return c.JSON(result)
}

// AddDiscovered creates a service from a discovered container
func (h *DiscoveryHandler) AddDiscovered(c *fiber.Ctx) error {
	userID, err := requireAdmin(c)
	if err != nil {
		return err
	}
	var req models.DiscoveryAddRequest
	if err := c.BodyParser(&req); err != nil {
		return BadRequest(c, "Invalid request body")
	}
	service, err := h.service.Add(c.Context(), userID, &req)
	if err != nil {
		return discoveryError(c, err, "add container")
	}
	return Created(c, service.ToResponse())
}

// IgnoreDiscovered hides a container from the tray
func (h *DiscoveryHandler) IgnoreDiscovered(c *fiber.Ctx) error {
	userID, err := requireAdmin(c)
	if err != nil {
		return err
	}
	var req models.DiscoveryKeyRequest
	if err := c.BodyParser(&req); err != nil {
		return BadRequest(c, "Invalid request body")
	}
	if err := h.service.Ignore(c.Context(), userID, req.Key); err != nil {
		return discoveryError(c, err, "ignore container")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// UnignoreDiscovered offers an ignored container again (?key=)
func (h *DiscoveryHandler) UnignoreDiscovered(c *fiber.Ctx) error {
	userID, err := requireAdmin(c)
	if err != nil {
		return err
	}
	if err := h.service.Unignore(c.Context(), userID, c.Query("key")); err != nil {
		return discoveryError(c, err, "un-ignore container")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func discoveryError(c *fiber.Ctx, err error, action string) error {
	if errors.Is(err, services.ErrDiscoveryHandled) {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "This container was added or ignored already"})
	}
	return serviceError(c, err, action, nil)
}
