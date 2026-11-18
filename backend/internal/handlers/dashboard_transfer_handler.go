package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/dashdoc"
	"github.com/nimbus/backend/internal/importers"
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
// response tells what would happen. With ?match_groups=1 a merge puts tiles
// into the user's group of the same name instead of adding a second one.
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

	summary, err := h.service.Import(c.Context(), userID, &doc, services.ImportOptions{
		Mode:        mode,
		DryRun:      c.QueryBool("dry_run"),
		MatchGroups: c.QueryBool("match_groups"),
	})
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

// Convert turns the config of another dashboard (?format=homepage, homarr,
// homer, dashy or compose; auto-detected when left out) into a dashboard
// document plus warnings about what was left out. It writes nothing: the
// document is imported with Import. ?host= is where docker-compose ports are
// reached.
func (h *DashboardTransferHandler) Convert(c *fiber.Ctx) error {
	if _, err := RequireUserID(c); err != nil {
		return err
	}
	if len(c.Body()) > importers.MaxSize {
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
			"error": fmt.Sprintf("The config exceeds the maximum size of %d MB", importers.MaxSize/(1024*1024)),
		})
	}
	result, err := importers.Convert(c.Query("format"), c.Body(), importers.Options{Host: c.Query("host")})
	if err != nil {
		return BadRequest(c, "Could not convert the config: "+err.Error())
	}
	return c.JSON(result)
}
