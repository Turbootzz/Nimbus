package handlers

import (
	"log"
	"net"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/services"
)

// Requests one visitor may make to the public status page per
// rateLimitWindow (15 minutes); an open page refreshes once a minute
const statusPageRateLimit = 300

type StatusPageHandler struct {
	service *services.StatusPageService
}

func NewStatusPageHandler(service *services.StatusPageService) *StatusPageHandler {
	return &StatusPageHandler{service: service}
}

// GetPublic returns the status page to anyone. While the page is off it
// answers exactly like a route that does not exist.
func (h *StatusPageHandler) GetPublic(c *fiber.Ctx) error {
	page, err := h.service.Public(c.Context())
	if err != nil {
		log.Printf("Failed to build status page: %v", err)
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Status is temporarily unavailable"})
	}
	if page == nil {
		return NotFound(c, "Not found")
	}
	// Turning the page off must not leave copies in caches
	c.Set(fiber.HeaderCacheControl, "no-store")
	return c.JSON(page)
}

// RateLimit limits the public page per visitor with the limiter that guards
// forgot-password. It steps aside while the page is off, so a 429 cannot
// reveal that the feature exists.
func (h *StatusPageHandler) RateLimit() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if page, err := h.service.Public(c.Context()); err == nil && page == nil {
			return c.Next()
		}
		if isRateLimited("status-page:"+visitorIP(c), statusPageRateLimit) {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "Too many requests"})
		}
		return c.Next()
	}
}

// visitorIP is the address to rate limit on. Requests arrive through the
// bundled nginx or the frontend, so the peer is a loopback or private
// address and the visitor is in the forwarding headers: the last public hop
// of X-Forwarded-For (earlier ones can be made up by the visitor), else
// X-Real-IP. From a public peer the headers are ignored.
func visitorIP(c *fiber.Ctx) string {
	if !isInternalIP(net.ParseIP(c.IP())) {
		return c.IP()
	}
	hops := strings.Split(c.Get(fiber.HeaderXForwardedFor), ",")
	for i := len(hops) - 1; i >= 0; i-- {
		if hop := net.ParseIP(strings.TrimSpace(hops[i])); hop != nil && !isInternalIP(hop) {
			return hop.String()
		}
	}
	if realIP := net.ParseIP(c.Get("X-Real-IP")); realIP != nil {
		return realIP.String()
	}
	return c.IP()
}

func isInternalIP(ip net.IP) bool {
	return ip != nil && (ip.IsLoopback() || ip.IsPrivate())
}

// GetSettings returns the status page configuration (admin only)
func (h *StatusPageHandler) GetSettings(c *fiber.Ctx) error {
	settings, err := h.service.Settings(c.Context())
	if err != nil {
		return serviceError(c, err, "load status page settings", nil)
	}
	return c.JSON(settings)
}

// UpdateSettings saves the status page configuration (admin only)
func (h *StatusPageHandler) UpdateSettings(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}

	var req models.StatusPageSettings
	if err := c.BodyParser(&req); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	settings, err := h.service.Save(c.Context(), userID, req)
	if err != nil {
		return serviceError(c, err, "save status page settings", nil)
	}
	return c.JSON(settings)
}
