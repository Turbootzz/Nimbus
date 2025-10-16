package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/services"
)

// AuthMiddleware protects routes by requiring a valid JWT token from httpOnly cookie
// SECURITY: Uses httpOnly cookies instead of Authorization header to prevent XSS attacks
func AuthMiddleware(authService *services.AuthService, userRepo *repository.UserRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var token string

		// First, try to get token from httpOnly cookie (preferred method)
		token = c.Cookies("auth_token")

		// Fallback: Check Authorization header for backward compatibility or API clients
		if token == "" {
			authHeader := c.Get("Authorization")
			if authHeader != "" {
				parts := strings.Split(authHeader, " ")
				if len(parts) == 2 && parts[0] == "Bearer" {
					token = parts[1]
				}
			}
		}

		// If no token found, return unauthorized
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing authentication token",
			})
		}

		// Validate token
		claims, err := authService.ValidateToken(token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired token",
			})
		}

		// Extract user ID from claims
		userID, err := authService.GetUserIDFromToken(claims)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid token claims",
			})
		}

		// Verify user exists in database
		// This prevents tokens for deleted/non-existent users from being valid
		_, err = userRepo.GetByID(userID)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "User not found - invalid session",
			})
		}

		// Store user info in context
		c.Locals("user_id", userID)
		c.Locals("email", (*claims)["email"])
		c.Locals("role", (*claims)["role"])

		// Continue to next handler
		return c.Next()
	}
}

// AdminOnly middleware ensures the user has admin role
func AdminOnly() fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, ok := c.Locals("role").(string)
		if !ok || role != "admin" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Admin access required",
			})
		}
		return c.Next()
	}
}
