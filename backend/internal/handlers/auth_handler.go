package handlers

import (
	"errors"
	"log"
	"net/mail"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/services"
	"github.com/nimbus/backend/internal/utils"
)

type AuthHandler struct {
	userRepo     *repository.UserRepository
	authService  *services.AuthService
	settingsRepo *repository.SettingsRepository
	cookieConfig utils.CookieConfig
}

func NewAuthHandler(userRepo *repository.UserRepository, authService *services.AuthService, settingsRepo *repository.SettingsRepository) *AuthHandler {
	return &AuthHandler{
		userRepo:     userRepo,
		authService:  authService,
		settingsRepo: settingsRepo,
		cookieConfig: utils.GetCookieConfig(),
	}
}

// Register handles user registration
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	// Check if public registration is enabled
	isEnabled, err := h.settingsRepo.IsPublicRegistrationEnabled(c.Context())
	if err != nil {
		log.Printf("Failed to check registration setting: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check registration status",
		})
	}
	if !isEnabled {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Public registration is disabled",
		})
	}

	var req models.RegisterRequest

	// Parse request body
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.Email == "" || req.Password == "" || req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Email, password, and name are required",
		})
	}

	// Validate email format
	if _, err := mail.ParseAddress(req.Email); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid email format",
		})
	}

	// Check if email already exists
	exists, err := h.userRepo.EmailExists(req.Email)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check email",
		})
	}
	if exists {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Email already registered",
		})
	}

	// Hash password
	hashedPassword, err := h.authService.HashPassword(req.Password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process password",
		})
	}

	// Create user
	user := &models.User{
		Email:         req.Email,
		Name:          req.Name,
		Password:      &hashedPassword,
		Role:          "user", // Default role
		Provider:      "local",
		EmailVerified: false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := h.userRepo.Create(user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create user",
		})
	}

	// Generate token
	token, err := h.authService.GenerateToken(user.ID, user.Email, user.Role)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate token",
		})
	}

	// Set httpOnly cookie (session cookie, cleared when browser closes)
	c.Cookie(utils.NewAuthCookie(token, 0, h.cookieConfig))

	// Return response without token in body (cookie handles authentication)
	return c.Status(fiber.StatusCreated).JSON(models.AuthResponse{
		Token: "", // Empty for security - using httpOnly cookie instead
		User:  user.ToResponse(),
	})
}

// Login handles user login
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req models.LoginRequest

	// Parse request body
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Email and password are required",
		})
	}

	// Get user by email
	user, err := h.userRepo.GetByEmail(req.Email)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid email or password",
		})
	}

	// Check if user has a password (OAuth users won't have one)
	if user.Password == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "This account uses OAuth login. Please use the OAuth provider button.",
		})
	}

	// Compare password
	if err := h.authService.ComparePassword(*user.Password, req.Password); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid email or password",
		})
	}

	// Update last activity timestamp
	if err := h.userRepo.UpdateLastActivity(user.ID); err != nil {
		// Log error but don't fail login
		log.Printf("Failed to update last activity for user %s: %v", user.Email, err)
	}

	// Generate token with appropriate expiration
	var token string
	maxAge := 0 // Session cookie by default
	if req.RememberMe {
		// Remember me: 30 days for both token and cookie
		maxAge = 30 * 24 * 60 * 60 // 30 days in seconds
		token, err = h.authService.GenerateTokenWithExpiration(user.ID, user.Email, user.Role, 30*24*time.Hour)
	} else {
		// Session: 24 hours for token, session cookie (cleared on browser close)
		token, err = h.authService.GenerateToken(user.ID, user.Email, user.Role)
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate token",
		})
	}

	// Set httpOnly cookie
	c.Cookie(utils.NewAuthCookie(token, maxAge, h.cookieConfig))

	// Return response without token in body (cookie handles authentication)
	return c.JSON(models.AuthResponse{
		Token: "",
		User:  user.ToResponse(),
	})
}

// Logout handles user logout by clearing the httpOnly cookie
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	c.Cookie(utils.ClearAuthCookie(h.cookieConfig))
	return c.JSON(fiber.Map{
		"message": "Logged out successfully",
	})
}

// GetMe returns the current authenticated user
func (h *AuthHandler) GetMe(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}

	// Get user from database
	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		// User in token doesn't exist in DB - token is invalid/stale
		// Return 401 to trigger frontend logout
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User not found - invalid session",
		})
	}

	// Update last activity if more than 1 minute has passed
	// This tracks actual website usage without overwhelming the database
	shouldUpdate := user.LastActivityAt == nil || time.Since(*user.LastActivityAt) > time.Minute

	if shouldUpdate {
		if err := h.userRepo.UpdateLastActivity(user.ID); err != nil {
			// Log error but don't fail the request
			log.Printf("Failed to update last activity for user %s: %v", user.Email, err)
		} else {
			// Re-fetch user to get the updated timestamp from database
			updatedUser, err := h.userRepo.GetByID(userID)
			if err == nil {
				user = updatedUser
			}
		}
	}

	return c.JSON(user.ToResponse())
}

// DeleteAccount allows the authenticated user to delete their own account.
// DB-level FK CASCADE wipes related rows (services, preferences, activity
// logs, groups, webhooks, password reset tokens, invitations).
// Refuses if the caller is the last remaining admin — a server with no
// admins is unrecoverable through normal flows.
func (h *AuthHandler) DeleteAccount(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}

	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "User not found",
			})
		}
		log.Printf("Failed to load user %s while deleting account: %v", userID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete account",
		})
	}
	if user.Role == "admin" {
		// Tiny TOCTOU window between this count and Delete: two of the last
		// two admins self-deleting at the same millisecond could both pass.
		// Not worth atomic locking for a self-serve homelab flow.
		stats, err := h.userRepo.GetStats()
		if err != nil {
			log.Printf("Failed to get user stats while deleting account %s: %v", userID, err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to delete account",
			})
		}
		if stats["admins"] <= 1 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Cannot delete the only admin account. Promote another user to admin first.",
			})
		}
	}

	if err := h.userRepo.Delete(userID); err != nil {
		// Race: another request could have deleted this user since GetByID.
		if errors.Is(err, repository.ErrUserNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "User not found",
			})
		}
		log.Printf("Failed to delete account %s: %v", userID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete account",
		})
	}

	c.Cookie(utils.ClearAuthCookie(h.cookieConfig))
	return c.JSON(fiber.Map{
		"message": "Account deleted successfully",
	})
}
