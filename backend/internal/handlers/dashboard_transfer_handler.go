package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/dashdoc"
	"github.com/nimbus/backend/internal/services"
)

// MaxImportSize is the largest dashboard document an import takes: the
// server's body limit, which the bundled reverse proxy has too
const MaxImportSize = MaxRequestBodySize

type DashboardTransferHandler struct {
	service *services.DashboardTransferService
}

func NewDashboardTransferHandler(service *services.DashboardTransferService) *DashboardTransferHandler {
	return &DashboardTransferHandler{service: service}
}

// Export returns the user's dashboard as a JSON document to download
func (h *DashboardTransferHandler) Export(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}
	doc, err := h.service.Export(c.Context(), userID)
	if err != nil {
		return serviceError(c, err, "export dashboard", nil)
	}
	c.Set(fiber.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="nimbus-dashboard-%s.json"`, time.Now().Format(time.DateOnly)))
	c.Set(fiber.HeaderCacheControl, "no-store")
	return c.JSON(doc)
}

// Import applies a dashboard document: ?mode=merge adds to the dashboard,
// ?mode=replace replaces it. With ?dry_run=1 nothing is written and the
// response tells what would happen.
func (h *DashboardTransferHandler) Import(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}
	mode, ok := dashdoc.ParseMode(c.Query("mode"))
	if !ok {
		return BadRequest(c, "mode must be merge or replace")
	}
	// Like every other write: a form posted from another site is not JSON
	if !c.Is("json") {
		return c.Status(fiber.StatusUnsupportedMediaType).JSON(fiber.Map{"error": "Content-Type must be application/json"})
	}
	if len(c.Body()) > MaxImportSize {
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
			"error": fmt.Sprintf("The document exceeds the maximum size of %d MB", MaxImportSize/(1024*1024)),
		})
	}

	// Unknown fields are refused: a typo would otherwise be dropped silently
	var doc dashdoc.Document
	decoder := json.NewDecoder(bytes.NewReader(c.Body()))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&doc); err != nil {
		return BadRequest(c, "Invalid dashboard document: "+err.Error())
	}
	if decoder.More() {
		return BadRequest(c, "Invalid dashboard document: unexpected data after the document")
	}

	run := h.service.Apply
	if c.QueryBool("dry_run") {
		run = h.service.Preview
	}
	summary, err := run(c.Context(), userID, &doc, mode)
	if err != nil {
		var invalid *dashdoc.ValidationError
		if errors.As(err, &invalid) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "The dashboard document is not valid",
				"details": invalid.Issues,
			})
		}
		return serviceError(c, err, "import dashboard", nil)
	}
	return c.JSON(summary)
}
