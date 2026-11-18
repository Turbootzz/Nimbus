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
//
// scope is the read scope (models.ScopeRead*) of the routes behind it: a token
// needs it for safe requests and also write:services for all others. Without
// a scope the routes are closed to tokens, so new routes are session-only
// until someone decides otherwise.
func AuthMiddleware(authService *services.AuthService, userRepo *repository.UserRepository, apiTokenRepo *repository.APITokenRepository, scope ...string) fiber.Handler {
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
			return authenticateAPIToken(c, token, userRepo, apiTokenRepo, scope)
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

// rateCountedKey marks a request that was counted against the rate limit
const rateCountedKey = "api_token_rate_counted"

// isSafeMethod reports whether a request method only reads
func isSafeMethod(method string) bool {
	return method == fiber.MethodGet || method == fiber.MethodHead || method == fiber.MethodOptions
}

// authorizeAPIToken validates a personal access token against the route's
// scope and the rate limit, then stores the owner's identity in the request
// context. scope is the read scope of the route group; without one the
// routes are closed to tokens.
func authorizeAPIToken(c *fiber.Ctx, token string, userRepo *repository.UserRepository, apiTokenRepo *repository.APITokenRepository, scope []string) error {
	user, apiToken, err := ResolveAPIToken(c.Context(), token, userRepo, apiTokenRepo)
	if err != nil {
		return err
	}

	readScope := ""
	if len(scope) > 0 {
		readScope = scope[0]
	}
	if !apiToken.Allows(readScope, !isSafeMethod(c.Method())) {
		return errInsufficientScope
	}
	// Nested route groups run this twice; a request counts once
	if c.Locals(rateCountedKey) == nil {
		if !AllowAPIToken(apiToken.ID) {
			return ErrAPITokenRateLimited
		}
		c.Locals(rateCountedKey, true)
	}

	c.Locals("user_id", user.ID)
	c.Locals("email", user.Email)
	c.Locals("role", user.Role)
	c.Locals("auth_method", AuthMethodAPIToken)
	return nil
}

// rateLimited answers a request that went over the token's rate limit
func rateLimited(c *fiber.Ctx) error {
	c.Set(fiber.HeaderRetryAfter, "60")
	return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
		"error": "API token rate limit exceeded",
	})
}

// authenticateAPIToken authorizes a personal access token or answers with
// the matching error
func authenticateAPIToken(c *fiber.Ctx, token string, userRepo *repository.UserRepository, apiTokenRepo *repository.APITokenRepository, scope []string) error {
	err := authorizeAPIToken(c, token, userRepo, apiTokenRepo, scope)
	switch {
	case err == nil:
		return c.Next()
	case errors.Is(err, repository.ErrAPITokenNotFound):
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid API token",
		})
	case errors.Is(err, ErrAPITokenExpired):
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "API token expired",
		})
	case errors.Is(err, errInsufficientScope):
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "API token lacks the required scope",
		})
	case errors.Is(err, ErrAPITokenRateLimited):
		return rateLimited(c)
	case errors.Is(err, repository.ErrUserNotFound):
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User not found - invalid token",
		})
	default:
		c.Context().Logger().Printf("API token auth DB error: %v", err)
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Authentication service unavailable",
		})
	}
}

// OptionalAuthMiddleware tries to authenticate but doesn't fail if no token
// Used for endpoints that support multiple auth methods (e.g., JWT or API key).
// scope works as in AuthMiddleware.
func OptionalAuthMiddleware(authService *services.AuthService, userRepo *repository.UserRepository, apiTokenRepo *repository.APITokenRepository, scope ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := extractToken(c)

		// If no token, just continue without setting context
		if token == "" {
			return c.Next()
		}

		// Personal access token: authenticate against the database, but keep
		// optional semantics — any failure but the rate limit falls through
		// unauthenticated
		if utils.IsAPIToken(token) {
			if err := authorizeAPIToken(c, token, userRepo, apiTokenRepo, scope); errors.Is(err, ErrAPITokenRateLimited) {
				return rateLimited(c)
			}
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

// RequireSessionAuth restricts a route to session (JWT) authentication.
// Used on sensitive routes (token management, account management, admin) so a
// leaked API token can never reach them. Denies by default when no auth
// middleware ran at all.
func RequireSessionAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Locals("auth_method") != AuthMethodSession {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Session authentication required",
			})
		}
		return c.Next()
	}
}
