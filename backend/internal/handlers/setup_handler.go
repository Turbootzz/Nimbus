package handlers

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/services"
	"github.com/nimbus/backend/internal/utils"
)

type SetupHandler struct {
	userRepo    *repository.UserRepository
	authService *services.AuthService
}

func NewSetupHandler(userRepo *repository.UserRepository, authService *services.AuthService) *SetupHandler {
	return &SetupHandler{
		userRepo:    userRepo,
		authService: authService,
	}
}

// GetSetupStatus checks if initial setup is needed (no users exist)
func (h *SetupHandler) GetSetupStatus(c *fiber.Ctx) error {
	count, err := h.userRepo.Count()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check setup status",
		})
	}

	return c.JSON(fiber.Map{
		"needs_setup": count == 0,
	})
}

// CreateInitialAdmin creates the first admin user (only works when no users exist)
func (h *SetupHandler) CreateInitialAdmin(c *fiber.Ctx) error {
	// Parse request body
	var req models.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.Name == "" || req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Name, email, and password are required",
		})
	}

	// Validate password length
	if len(req.Password) < 8 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Password must be at least 8 characters",
		})
	}

	// Hash password
	hashedPassword, err := h.authService.HashPassword(req.Password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process password",
		})
	}

	// Create admin user
	user := &models.User{
		Email:         req.Email,
		Name:          req.Name,
		Password:      &hashedPassword,
		Role:          "admin", // First user is always admin
		Provider:      "local",
		EmailVerified: false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Atomically check no users exist and create admin
	if err := h.userRepo.CreateAdminIfNone(user); err != nil {
		if errors.Is(err, repository.ErrUsersAlreadyExist) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Setup already completed. Users already exist.",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create admin user",
		})
	}

	// Generate JWT token
	token, err := h.authService.GenerateToken(user.ID, user.Email, user.Role)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate auth token",
		})
	}

	// Set auth cookie
	c.Cookie(utils.NewAuthCookie(token, 0, utils.GetCookieConfig()))

	return c.Status(fiber.StatusCreated).JSON(models.AuthResponse{
		Token: "", // Empty - using httpOnly cookie
		User:  user.ToResponse(),
	})
}
