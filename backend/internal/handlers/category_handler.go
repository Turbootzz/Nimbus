package handlers

import (
	"database/sql"
	"regexp"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
)

type CategoryHandler struct {
	categoryRepo *repository.CategoryRepository
}

func NewCategoryHandler(categoryRepo *repository.CategoryRepository) *CategoryHandler {
	return &CategoryHandler{
		categoryRepo: categoryRepo,
	}
}

// Hex color regex for validation
var hexColorRegex = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// CreateCategory handles category creation
func (h *CategoryHandler) CreateCategory(c *fiber.Ctx) error {
	// Get user ID from context (set by auth middleware)
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized: user ID not found",
		})
	}

	var req models.CategoryCreateRequest

	// Parse request body
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Name is required",
		})
	}

	// Validate name length
	if len(req.Name) > models.MaxCategoryNameLen {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Category name is too long (max 100 characters)",
		})
	}

	// Set default color if not provided
	if req.Color == "" {
		req.Color = models.DefaultCategoryColor
	}

	// Validate hex color format
	if !hexColorRegex.MatchString(req.Color) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid color format. Use hex format like #6366f1",
		})
	}

	// Create category
	category := &models.Category{
		UserID:    userID,
		Name:      req.Name,
		Color:     req.Color,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.categoryRepo.Create(c.Context(), category); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create category",
		})
	}

	// Return created category
	return c.Status(fiber.StatusCreated).JSON(category.ToResponse())
}

// GetCategories retrieves all categories for the authenticated user
func (h *CategoryHandler) GetCategories(c *fiber.Ctx) error {
	// Get user ID from context
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized: user ID not found",
		})
	}

	// Get all categories for user
	categories, err := h.categoryRepo.GetAllByUserID(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch categories",
		})
	}

	// Convert to response format
	responses := make([]models.CategoryResponse, len(categories))
	for i, category := range categories {
		responses[i] = category.ToResponse()
	}

	return c.JSON(responses)
}

// GetCategory retrieves a single category by ID
func (h *CategoryHandler) GetCategory(c *fiber.Ctx) error {
	// Get user ID from context
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized: user ID not found",
		})
	}

	// Get category ID from URL params
	categoryID := c.Params("id")
	if categoryID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Category ID is required",
		})
	}

	// Get category from database
	category, err := h.categoryRepo.GetByID(c.Context(), categoryID)
	if err == sql.ErrNoRows {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Category not found",
		})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch category",
		})
	}

	// Verify ownership
	if category.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	return c.JSON(category.ToResponse())
}

// UpdateCategory handles category updates
func (h *CategoryHandler) UpdateCategory(c *fiber.Ctx) error {
	// Get user ID from context
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized: user ID not found",
		})
	}

	// Get category ID from URL params
	categoryID := c.Params("id")
	if categoryID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Category ID is required",
		})
	}

	var req models.CategoryUpdateRequest

	// Parse request body
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Name is required",
		})
	}

	// Validate name length
	if len(req.Name) > models.MaxCategoryNameLen {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Category name is too long (max 100 characters)",
		})
	}

	// Validate hex color format
	if req.Color != "" && !hexColorRegex.MatchString(req.Color) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid color format. Use hex format like #6366f1",
		})
	}

	// Get existing category to verify ownership
	existing, err := h.categoryRepo.GetByID(c.Context(), categoryID)
	if err == sql.ErrNoRows {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Category not found",
		})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch category",
		})
	}

	// Verify ownership
	if existing.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	// Update category fields
	existing.Name = req.Name
	if req.Color != "" {
		existing.Color = req.Color
	}
	existing.UpdatedAt = time.Now()

	// Save updates
	if err := h.categoryRepo.Update(c.Context(), existing); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update category",
		})
	}

	return c.JSON(existing.ToResponse())
}

// DeleteCategory handles category deletion
func (h *CategoryHandler) DeleteCategory(c *fiber.Ctx) error {
	// Get user ID from context
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized: user ID not found",
		})
	}

	// Get category ID from URL params
	categoryID := c.Params("id")
	if categoryID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Category ID is required",
		})
	}

	// Verify ownership before deleting
	category, err := h.categoryRepo.GetByID(c.Context(), categoryID)
	if err == sql.ErrNoRows {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Category not found",
		})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch category",
		})
	}

	if category.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	// Delete category (services will have category_id set to NULL automatically)
	if err := h.categoryRepo.Delete(c.Context(), categoryID, userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete category",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ReorderCategories handles bulk position updates
func (h *CategoryHandler) ReorderCategories(c *fiber.Ctx) error {
	// Get user ID from context
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized: user ID not found",
		})
	}

	var req models.CategoryReorderRequest

	// Parse request body
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if len(req.Categories) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Categories array is required",
		})
	}

	// Convert to map for repository
	positions := make(map[string]int)
	for _, cp := range req.Categories {
		if cp.ID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "All categories must have an ID",
			})
		}
		if cp.Position < 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Position must be non-negative",
			})
		}
		positions[cp.ID] = cp.Position
	}

	// Update positions
	if err := h.categoryRepo.UpdatePositions(c.Context(), userID, positions); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update category positions",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
