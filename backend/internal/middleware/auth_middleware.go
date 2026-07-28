package middleware

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/services"
	"github.com/nimbus/backend/internal/utils"
)

// Auth method values stored in c.Locals("auth_method")
const (
	AuthMethodSession  = "session"
	AuthMethodAPIToken = "api_token"
)

// extractToken reads the auth token from the httpOnly cookie, falling back to
// the Authorization Bearer header for API clients
func extractToken(c *fiber.Ctx) string {
	if token := c.Cookies("auth_token"); token != "" {
		return token
	}

	authHeader := c.Get("Authorization")
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && parts[0] == "Bearer" {
			return parts[1]
		}
	}

	return ""
}

// AuthMiddleware protects routes by requiring a valid JWT token from httpOnly cookie
// or a personal access token (nimbus_ prefix) via the Authorization Bearer header.
// SECURITY: Uses httpOnly cookies for browser sessions to prevent XSS attacks
func AuthMiddleware(authService *services.AuthService, userRepo *repository.UserRepository, apiTokenRepo *repository.APITokenRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := extractToken(c)

		// If no token found, return unauthorized
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing authentication token",
			})
		}

		// Personal access tokens are authenticated against the database, not as JWTs
		if utils.IsAPIToken(token) {
			return authenticateAPIToken(c, token, userRepo, apiTokenRepo)
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
			// Distinguish between user not found (401) and DB errors (503)
			if errors.Is(err, repository.ErrUserNotFound) {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "User not found - invalid session",
				})
			}
			// Database or infrastructure error - log and return service unavailable
			c.Context().Logger().Printf("Auth middleware DB error: %v", err)
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": "Authentication service unavailable",
			})
		}

		// Store user info in context
		c.Locals("user_id", userID)
		c.Locals("email", (*claims)["email"])
		c.Locals("role", (*claims)["role"])
		c.Locals("auth_method", AuthMethodSession)

		// Continue to next handler
		return c.Next()
	}
}

// authenticateAPIToken validates a personal access token, enforces its
// read-only flag, and stores the owner's identity in the request context
func authenticateAPIToken(c *fiber.Ctx, token string, userRepo *repository.UserRepository, apiTokenRepo *repository.APITokenRepository) error {
	apiToken, err := apiTokenRepo.GetByTokenHash(c.Context(), utils.HashAPIToken(token))
	if err != nil {
		if errors.Is(err, repository.ErrAPITokenNotFound) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid API token",
			})
		}
		c.Context().Logger().Printf("API token auth DB error: %v", err)
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Authentication service unavailable",
		})
	}

	// Read-only tokens can never modify anything, even if leaked
	if apiToken.ReadOnly && c.Method() != fiber.MethodGet && c.Method() != fiber.MethodHead {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Read-only token cannot modify resources",
		})
	}

	user, err := userRepo.GetByID(apiToken.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "User not found - invalid token",
			})
		}
		c.Context().Logger().Printf("API token auth DB error: %v", err)
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Authentication service unavailable",
		})
	}

	// Best-effort usage stamp; auth must not fail on this
	if err := apiTokenRepo.UpdateLastUsed(c.Context(), apiToken.ID); err != nil {
		c.Context().Logger().Printf("Failed to update api token last_used_at: %v", err)
	}

	c.Locals("user_id", user.ID)
	c.Locals("email", user.Email)
	c.Locals("role", user.Role)
	c.Locals("auth_method", AuthMethodAPIToken)

	return c.Next()
}

// OptionalAuthMiddleware tries to authenticate but doesn't fail if no token
// Used for endpoints that support multiple auth methods (e.g., JWT or API key)
func OptionalAuthMiddleware(authService *services.AuthService, userRepo *repository.UserRepository, apiTokenRepo *repository.APITokenRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := extractToken(c)

		// If no token, just continue without setting context
		if token == "" {
			return c.Next()
		}

		// Personal access token: authenticate against the database, but keep
		// optional semantics — any failure falls through unauthenticated
		if utils.IsAPIToken(token) {
			apiToken, err := apiTokenRepo.GetByTokenHash(c.Context(), utils.HashAPIToken(token))
			if err != nil {
				return c.Next()
			}
			if apiToken.ReadOnly && c.Method() != fiber.MethodGet && c.Method() != fiber.MethodHead {
				return c.Next()
			}
			user, err := userRepo.GetByID(apiToken.UserID)
			if err != nil {
				return c.Next()
			}
			if err := apiTokenRepo.UpdateLastUsed(c.Context(), apiToken.ID); err != nil {
				c.Context().Logger().Printf("Failed to update api token last_used_at: %v", err)
			}
			c.Locals("user_id", user.ID)
			c.Locals("email", user.Email)
			c.Locals("role", user.Role)
			c.Locals("auth_method", AuthMethodAPIToken)
			return c.Next()
		}

		// Validate token
		claims, err := authService.ValidateToken(token)
		if err != nil {
			// Invalid token - continue without auth context
			return c.Next()
		}

		// Extract user ID from claims
		userID, err := authService.GetUserIDFromToken(claims)
		if err != nil {
			return c.Next()
		}

		// Verify user exists
		_, err = userRepo.GetByID(userID)
		if err != nil {
			return c.Next()
		}

		// Store user info in context
		c.Locals("user_id", userID)
		c.Locals("email", (*claims)["email"])
		c.Locals("role", (*claims)["role"])
		c.Locals("auth_method", AuthMethodSession)

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

// RequireSessionAuth blocks API-token authentication. Used on token-management
// routes so a leaked token can never create or revoke tokens.
func RequireSessionAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Locals("auth_method") == AuthMethodAPIToken {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "API tokens cannot manage API tokens",
			})
		}
		return c.Next()
	}
}
