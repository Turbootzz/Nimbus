package handlers

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/services"
)

type OAuthHandler struct {
	oauthService *services.OAuthService
	authService  *services.AuthService
	userRepo     *repository.UserRepository
}

func NewOAuthHandler(
	oauthService *services.OAuthService,
	authService *services.AuthService,
	userRepo *repository.UserRepository,
) *OAuthHandler {
	return &OAuthHandler{
		oauthService: oauthService,
		authService:  authService,
		userRepo:     userRepo,
	}
}

// InitiateOAuth starts the OAuth flow by redirecting to the provider
// GET /api/v1/auth/oauth/:provider
func (h *OAuthHandler) InitiateOAuth(c *fiber.Ctx) error {
	providerStr := c.Params("provider")
	provider := models.OAuthProvider(providerStr)

	// Validate provider
	if !provider.IsValid() || provider == models.ProviderLocal {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid OAuth provider",
		})
	}

	// Check if provider is configured
	if !h.oauthService.IsProviderConfigured(provider) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("OAuth provider '%s' is not configured", provider),
		})
	}

	// Get redirect URL (where to go after OAuth completes)
	redirectTo := c.Query("redirect", "/dashboard")

	// Generate authorization URL
	authURL, err := h.oauthService.GetAuthURL(provider, redirectTo)
	if err != nil {
		log.Printf("Failed to generate OAuth URL: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to initiate OAuth flow",
		})
	}

	// Redirect to OAuth provider
	return c.Redirect(authURL, fiber.StatusTemporaryRedirect)
}

// HandleCallback processes the OAuth callback from the provider
// GET /api/v1/auth/oauth/:provider/callback
func (h *OAuthHandler) HandleCallback(c *fiber.Ctx) error {
	providerStr := c.Params("provider")
	provider := models.OAuthProvider(providerStr)

	// Validate provider
	if !provider.IsValid() || provider == models.ProviderLocal {
		return h.redirectWithError(c, "Invalid OAuth provider")
	}

	// Get code and state from query params
	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		return h.redirectWithError(c, "Missing OAuth parameters")
	}

	// Exchange code for user info
	userInfo, err := h.oauthService.ExchangeCode(c.Context(), provider, code, state)
	if err != nil {
		log.Printf("OAuth exchange failed: %v", err)
		if errors.Is(err, services.ErrInvalidState) {
			return h.redirectWithError(c, "Invalid OAuth state")
		}
		if errors.Is(err, services.ErrExpiredState) {
			return h.redirectWithError(c, "OAuth session expired")
		}
		return h.redirectWithError(c, "OAuth authentication failed")
	}

	// Check if user already exists with this OAuth provider
	existingUser, err := h.userRepo.GetByProviderID(string(provider), userInfo.ProviderID)
	if err == nil {
		// User exists - log them in
		return h.loginUser(c, existingUser)
	}

	// Check if user exists with this email (different provider)
	existingUser, err = h.userRepo.GetByEmail(userInfo.Email)
	if err == nil {
		// Email exists - link this provider to the existing account
		err = h.userRepo.LinkOAuthProvider(
			existingUser.ID,
			string(provider),
			userInfo.ProviderID,
			&userInfo.AvatarURL,
		)
		if err != nil {
			log.Printf("Failed to link OAuth provider: %v", err)
			return h.redirectWithError(c, "Failed to link account")
		}

		// Refresh user data
		existingUser, err = h.userRepo.GetByID(existingUser.ID)
		if err != nil {
			log.Printf("Failed to fetch user: %v", err)
			return h.redirectWithError(c, "Failed to fetch user")
		}

		return h.loginUser(c, existingUser)
	}

	// User doesn't exist - create new account
	newUser := &models.User{
		Email:         userInfo.Email,
		Name:          userInfo.Name,
		Provider:      string(provider),
		ProviderID:    &userInfo.ProviderID,
		AvatarURL:     &userInfo.AvatarURL,
		EmailVerified: userInfo.EmailVerified,
		Role:          "user", // Default role
		Password:      nil,    // OAuth users don't have passwords
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	err = h.userRepo.Create(newUser)
	if err != nil {
		log.Printf("Failed to create OAuth user: %v", err)
		return h.redirectWithError(c, "Failed to create account")
	}

	return h.loginUser(c, newUser)
}

// LinkProvider links an OAuth provider to the currently logged-in user
// POST /api/v1/auth/oauth/link/:provider
func (h *OAuthHandler) LinkProvider(c *fiber.Ctx) error {
	// Get current user from context (set by auth middleware)
	_ = c.Locals("user_id").(string) // userID for future implementation

	providerStr := c.Params("provider")
	provider := models.OAuthProvider(providerStr)

	// Validate provider
	if !provider.IsValid() || provider == models.ProviderLocal {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid OAuth provider",
		})
	}

	// Check if provider is configured
	if !h.oauthService.IsProviderConfigured(provider) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("OAuth provider '%s' is not configured", provider),
		})
	}

	// TODO: Implement OAuth linking flow
	// This would need to store the userID in the state token
	// Then in the callback, link the provider instead of creating a new user

	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
		"error": "Provider linking not yet implemented",
	})
}

// UnlinkProvider removes an OAuth provider link from the current user
// DELETE /api/v1/auth/oauth/unlink/:provider
func (h *OAuthHandler) UnlinkProvider(c *fiber.Ctx) error {
	// Get current user from context (set by auth middleware)
	userID := c.Locals("user_id").(string)

	providerStr := c.Params("provider")
	provider := models.OAuthProvider(providerStr)

	// Validate provider
	if !provider.IsValid() || provider == models.ProviderLocal {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid OAuth provider",
		})
	}

	// Unlink the provider
	err := h.userRepo.UnlinkOAuthProvider(userID, string(provider))
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "User not found",
			})
		}
		if errors.Is(err, repository.ErrProviderNotLinked) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Provider not linked to this account",
			})
		}
		log.Printf("Failed to unlink OAuth provider: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Get updated user info
	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		log.Printf("Failed to fetch user: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch user",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Provider unlinked successfully",
		"user":    user.ToResponse(),
	})
}

// GetProviderStatus returns the OAuth providers configuration status
// GET /api/v1/auth/oauth/providers
func (h *OAuthHandler) GetProviderStatus(c *fiber.Ctx) error {
	providers := []fiber.Map{
		{
			"name":      "google",
			"enabled":   h.oauthService.IsProviderConfigured(models.ProviderGoogle),
			"configure": h.oauthService.IsProviderConfigured(models.ProviderGoogle),
		},
		{
			"name":      "github",
			"enabled":   h.oauthService.IsProviderConfigured(models.ProviderGitHub),
			"configure": h.oauthService.IsProviderConfigured(models.ProviderGitHub),
		},
		{
			"name":      "discord",
			"enabled":   h.oauthService.IsProviderConfigured(models.ProviderDiscord),
			"configure": h.oauthService.IsProviderConfigured(models.ProviderDiscord),
		},
	}

	return c.JSON(fiber.Map{
		"providers": providers,
	})
}

// Helper: loginUser creates a JWT token and sets the auth cookie
func (h *OAuthHandler) loginUser(c *fiber.Ctx, user *models.User) error {
	// Generate JWT token
	token, err := h.authService.GenerateToken(user.ID, user.Email, user.Role)
	if err != nil {
		log.Printf("Failed to generate token: %v", err)
		return h.redirectWithError(c, "Failed to generate auth token")
	}

	// Set httpOnly cookie
	secure := os.Getenv("COOKIE_SECURE") == "true"
	domain := os.Getenv("COOKIE_DOMAIN")

	c.Cookie(&fiber.Cookie{
		Name:     "auth_token",
		Value:    token,
		Expires:  time.Now().Add(24 * time.Hour),
		HTTPOnly: true,
		Secure:   secure,
		SameSite: "Lax",
		Domain:   domain,
	})

	// Update last activity
	_ = h.userRepo.UpdateLastActivity(user.ID)

	// Redirect to frontend dashboard
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}

	return c.Redirect(fmt.Sprintf("%s/dashboard", frontendURL), fiber.StatusTemporaryRedirect)
}

// Helper: redirectWithError redirects to the login page with an error message
func (h *OAuthHandler) redirectWithError(c *fiber.Ctx, message string) error {
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}

	return c.Redirect(
		fmt.Sprintf("%s/login?error=%s", frontendURL, message),
		fiber.StatusTemporaryRedirect,
	)
}
