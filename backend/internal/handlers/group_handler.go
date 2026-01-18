package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
)

type GroupHandler struct {
	groupRepo *repository.GroupRepository
}

func NewGroupHandler(groupRepo *repository.GroupRepository) *GroupHandler {
	return &GroupHandler{
		groupRepo: groupRepo,
	}
}

// CreateGroup handles group creation
func (h *GroupHandler) CreateGroup(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized: user ID not found",
		})
	}

	var req models.GroupCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate name
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Name is required",
		})
	}
	if len(req.Name) > models.MaxGroupNameLen {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Name must be %d characters or less", models.MaxGroupNameLen),
		})
	}

	// Validate color
	color := req.Color
	if color == "" {
		color = models.DefaultGroupColor
	}
	if !models.IsValidHexColor(color) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid color format. Must be hex format #RRGGBB",
		})
	}

	now := time.Now()
	group := &models.Group{
		UserID:    userID,
		Name:      req.Name,
		Color:     color,
		IsDefault: false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := h.groupRepo.Create(c.Context(), group); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create group",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(group.ToResponse())
}

// GetGroups retrieves all groups for the authenticated user
func (h *GroupHandler) GetGroups(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized: user ID not found",
		})
	}

	// Ensure default group exists
	_, err := h.groupRepo.EnsureDefaultGroup(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to ensure default group",
		})
	}

	groups, err := h.groupRepo.GetAllByUserID(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve groups",
		})
	}

	response := make([]models.GroupResponse, 0, len(groups))
	for _, group := range groups {
		response = append(response, group.ToResponse())
	}

	return c.JSON(response)
}

// GetGroup retrieves a single group by ID
func (h *GroupHandler) GetGroup(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized: user ID not found",
		})
	}

	groupID := c.Params("id")
	if groupID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Group ID is required",
		})
	}

	group, err := h.groupRepo.GetByID(c.Context(), groupID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Group not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve group",
		})
	}

	if group.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	return c.JSON(group.ToResponse())
}

// UpdateGroup handles group updates (name and color only)
func (h *GroupHandler) UpdateGroup(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized: user ID not found",
		})
	}

	groupID := c.Params("id")
	if groupID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Group ID is required",
		})
	}

	var req models.GroupUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Get existing group
	existingGroup, err := h.groupRepo.GetByID(c.Context(), groupID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Group not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve group",
		})
	}

	if existingGroup.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	// Validate and apply name if provided
	if req.Name != "" {
		if len(req.Name) > models.MaxGroupNameLen {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("Name must be %d characters or less", models.MaxGroupNameLen),
			})
		}
		existingGroup.Name = req.Name
	}

	// Validate and apply color if provided
	if req.Color != "" {
		if !models.IsValidHexColor(req.Color) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid color format. Must be hex format #RRGGBB",
			})
		}
		existingGroup.Color = req.Color
	}

	existingGroup.UpdatedAt = time.Now()

	if err := h.groupRepo.Update(c.Context(), existingGroup); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update group",
		})
	}

	return c.JSON(existingGroup.ToResponse())
}

// DeleteGroup handles group deletion (default group cannot be deleted)
// Query param: delete_services=true to also delete all services in the group
func (h *GroupHandler) DeleteGroup(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized: user ID not found",
		})
	}

	groupID := c.Params("id")
	if groupID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Group ID is required",
		})
	}

	// Check if services should be deleted with the group
	deleteServices := c.Query("delete_services") == "true"

	if err := h.groupRepo.Delete(c.Context(), groupID, userID, deleteServices); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Group not found or access denied",
			})
		}
		if errors.Is(err, repository.ErrCannotDeleteDefaultGroup) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Cannot delete default group",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete group",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Group deleted successfully",
	})
}

// ReorderGroups handles bulk position updates for groups
func (h *GroupHandler) ReorderGroups(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized: user ID not found",
		})
	}

	var req models.GroupReorderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if len(req.Groups) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "At least one group position is required",
		})
	}

	// Get group count for bounds validation
	existingGroups, err := h.groupRepo.GetAllByUserID(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch groups",
		})
	}
	groupCount := len(existingGroups)

	positions := make(map[string]int)
	for _, gp := range req.Groups {
		if gp.ID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Group ID cannot be empty",
			})
		}
		if gp.Position < 0 || gp.Position >= groupCount {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("Position must be between 0 and %d", groupCount-1),
			})
		}
		positions[gp.ID] = gp.Position
	}

	if err := h.groupRepo.UpdatePositions(c.Context(), userID, positions); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "One or more groups not found or access denied",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update group positions",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Group positions updated successfully",
	})
}
