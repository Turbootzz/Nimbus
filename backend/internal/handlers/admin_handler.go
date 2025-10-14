package handlers

import (
	"errors"
	"log"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/repository"
)

type AdminHandler struct {
	userRepo *repository.UserRepository
	// TODO: Add these repositories when implementing invitation system
	// activityRepo   *repository.ActivityLogRepository
	// invitationRepo *repository.InvitationRepository
	// settingsRepo   *repository.SettingsRepository
}

func NewAdminHandler(userRepo *repository.UserRepository) *AdminHandler {
	return &AdminHandler{
		userRepo: userRepo,
	}
}

// GetAllUsers returns all users with optional filtering and pagination (admin only)
func (h *AdminHandler) GetAllUsers(c *fiber.Ctx) error {
	// Parse query parameters
	search := c.Query("search", "")
	role := c.Query("role", "")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	// Validate and set defaults
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Normalize search (trim and lowercase)
	search = strings.TrimSpace(strings.ToLower(search))

	// Build filter
	filter := repository.UserFilter{
		Search: search,
		Role:   role,
		Limit:  limit,
		Offset: offset,
	}

	// Get filtered users
	result, err := h.userRepo.GetFiltered(filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve users",
		})
	}

	// Convert to response format (without passwords)
	userResponses := make([]interface{}, len(result.Users))
	for i, user := range result.Users {
		userResponses[i] = user.ToResponse()
	}

	// Return with pagination metadata
	return c.JSON(fiber.Map{
		"users":       userResponses,
		"total":       result.Total,
		"page":        result.Page,
		"total_pages": result.TotalPages,
		"limit":       limit,
	})
}

// GetUserStats returns user statistics (admin only)
func (h *AdminHandler) GetUserStats(c *fiber.Ctx) error {
	stats, err := h.userRepo.GetStats()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve user statistics",
		})
	}

	return c.JSON(stats)
}

// UpdateUserRole updates a user's role (admin only)
func (h *AdminHandler) UpdateUserRole(c *fiber.Ctx) error {
	userID := c.Params("id")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "User ID is required",
		})
	}

	// Parse request body
	var req struct {
		Role string `json:"role"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate role
	if req.Role != "admin" && req.Role != "user" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Role must be 'admin' or 'user'",
		})
	}

	// Prevent admin from demoting themselves
	currentUserID := c.Locals("user_id").(string)
	if currentUserID == userID && req.Role != "admin" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot change your own role",
		})
	}

	// Update role
	err := h.userRepo.UpdateRole(userID, req.Role)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update user role",
		})
	}

	// Get updated user
	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve updated user",
		})
	}

	return c.JSON(user.ToResponse())
}

// DeleteUser deletes a user (admin only)
func (h *AdminHandler) DeleteUser(c *fiber.Ctx) error {
	userID := c.Params("id")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "User ID is required",
		})
	}

	// Prevent admin from deleting themselves
	currentUserID := c.Locals("user_id").(string)
	if currentUserID == userID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot delete your own account",
		})
	}

	// Delete user
	err := h.userRepo.Delete(userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "User not found",
			})
		}
		log.Printf("Failed to delete user %s: %v", userID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete user",
		})
	}

	return c.JSON(fiber.Map{
		"message": "User deleted successfully",
	})
}
