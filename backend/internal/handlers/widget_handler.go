package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/services"
)

type WidgetHandler struct {
	service *services.WidgetService
}

func NewWidgetHandler(service *services.WidgetService) *WidgetHandler {
	return &WidgetHandler{service: service}
}

var widgetNotFound = map[error]string{repository.ErrWidgetNotFound: "Widget not found"}

// ListTypes returns the widget types the server supports
func (h *WidgetHandler) ListTypes(c *fiber.Ctx) error {
	if _, err := RequireUserID(c); err != nil {
		return err
	}
	return c.JSON(h.service.Types())
}

// ListWidgets returns the user's widgets in grid order
func (h *WidgetHandler) ListWidgets(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}
	list, err := h.service.List(c.Context(), userID)
	if err != nil {
		return serviceError(c, err, "list widgets", widgetNotFound)
	}
	return c.JSON(list)
}

// GetWidget returns one widget owned by the user
func (h *WidgetHandler) GetWidget(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}
	widget, err := h.service.Get(c.Context(), c.Params("id"), userID)
	if err != nil {
		return serviceError(c, err, "get widget", widgetNotFound)
	}
	return c.JSON(widget)
}

// CreateWidget adds a widget at the end of the user's grid
func (h *WidgetHandler) CreateWidget(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}
	var req models.WidgetRequest
	if err := c.BodyParser(&req); err != nil {
		return BadRequest(c, "Invalid request body")
	}
	widget, err := h.service.Create(c.Context(), userID, &req)
	if err != nil {
		return serviceError(c, err, "create widget", widgetNotFound)
	}
	return Created(c, widget)
}

// UpdateWidget changes a widget; omitted fields are kept
func (h *WidgetHandler) UpdateWidget(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}
	var req models.WidgetRequest
	if err := c.BodyParser(&req); err != nil {
		return BadRequest(c, "Invalid request body")
	}
	widget, err := h.service.Update(c.Context(), c.Params("id"), userID, &req)
	if err != nil {
		return serviceError(c, err, "update widget", widgetNotFound)
	}
	return c.JSON(widget)
}

// DeleteWidget removes a widget owned by the user
func (h *WidgetHandler) DeleteWidget(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}
	if err := h.service.Delete(c.Context(), c.Params("id"), userID); err != nil {
		return serviceError(c, err, "delete widget", widgetNotFound)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// RefreshWidget fetches a widget's data now instead of on its next turn
func (h *WidgetHandler) RefreshWidget(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}
	if err := h.service.Refresh(c.Context(), c.Params("id"), userID); err != nil {
		return serviceError(c, err, "refresh widget", widgetNotFound)
	}
	return c.SendStatus(fiber.StatusAccepted)
}

// ReorderTiles saves the order of services and widgets on the dashboard
func (h *WidgetHandler) ReorderTiles(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}
	var req models.TileReorderRequest
	if err := c.BodyParser(&req); err != nil {
		return BadRequest(c, "Invalid request body")
	}
	notFound := map[error]string{repository.ErrTileNotFound: "One or more tiles not found"}
	if err := h.service.ReorderTiles(c.Context(), userID, &req); err != nil {
		return serviceError(c, err, "reorder tiles", notFound)
	}
	return c.JSON(fiber.Map{"message": "Tile positions updated successfully"})
}
