package handlers

import (
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/services"
)

type AuthHandler struct {
	userRepo    *repository.UserRepository
	authService *services.AuthService
}

func NewAuthHandler(userRepo *repository.UserRepository, authService *services.AuthService) *AuthHandler {
	return &AuthHandler{
		userRepo:    userRepo,
		authService: authService,
	}
}

// getCookieSecure returns whether cookies should be secure based on environment
// Returns true for production (HTTPS), false for local development (HTTP)
func (h *AuthHandler) getCookieSecure() bool {
	// Check COOKIE_SECURE env var (defaults to true for production safety)
	secure := os.Getenv("COOKIE_SECURE")
	return secure != "false" // Default to true unless explicitly set to "false"
}

// getCookieDomain returns the domain for cookies based on environment
func (h *AuthHandler) getCookieDomain() string {
	return os.Getenv("COOKIE_DOMAIN")
}

// Register handles user registration
func (h *AuthHandler) Register(c *fiber.Ctx) error {
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
		Email:     req.Email,
		Name:      req.Name,
		Password:  hashedPassword,
		Role:      "user", // Default role
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
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

	// Set httpOnly cookie (default: session cookie, cleared when browser closes)
	// SECURITY: httpOnly prevents XSS attacks, secure ensures HTTPS-only, sameSite prevents CSRF
	c.Cookie(&fiber.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		Domain:   h.getCookieDomain(),
		HTTPOnly: true,
		Secure:   h.getCookieSecure(), // Controlled by COOKIE_SECURE env var
		SameSite: "Lax",
		MaxAge:   0, // Session cookie (cleared when browser closes)
	})

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

	// Compare password
	if err := h.authService.ComparePassword(user.Password, req.Password); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid email or password",
		})
	}

	// Update last activity timestamp
	if err := h.userRepo.UpdateLastActivity(user.ID); err != nil {
		// Log error but don't fail login
		log.Printf("Failed to update last activity for user %s: %v", user.Email, err)
	}

	// Generate token
	token, err := h.authService.GenerateToken(user.ID, user.Email, user.Role)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate token",
		})
	}

	maxAge := 0 // Session cookie by default
	if req.RememberMe {
		maxAge = 30 * 24 * 60 * 60 // 30 days in seconds
	}

	// Set httpOnly cookie
	// SECURITY: httpOnly prevents XSS attacks, secure ensures HTTPS-only
	c.Cookie(&fiber.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		Domain:   h.getCookieDomain(),
		HTTPOnly: true,
		Secure:   h.getCookieSecure(),
		SameSite: "Lax",
		MaxAge:   maxAge,
	})

	// Return response without token in body (cookie handles authentication)
	return c.JSON(models.AuthResponse{
		Token: "",
		User:  user.ToResponse(),
	})
}

// Logout handles user logout by clearing the httpOnly cookie
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	// Clear the auth_token cookie by setting MaxAge to -1
	c.Cookie(&fiber.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		Domain:   h.getCookieDomain(),
		HTTPOnly: true,
		Secure:   h.getCookieSecure(),
		SameSite: "Lax",
		MaxAge:   -1, // Delete the cookie
	})

	return c.JSON(fiber.Map{
		"message": "Logged out successfully",
	})
}

// GetMe returns the current authenticated user
func (h *AuthHandler) GetMe(c *fiber.Ctx) error {
	// Get user ID from context (set by auth middleware)
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized: user ID not found",
		})
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
