package handlers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/services"
	"github.com/nimbus/backend/internal/utils"
)

type OAuthHandler struct {
	oauthService *services.OAuthService
	authService  *services.AuthService
	userRepo     *repository.UserRepository
	settingsRepo *repository.SettingsRepository
	cookieConfig utils.CookieConfig
}

func NewOAuthHandler(
	oauthService *services.OAuthService,
	authService *services.AuthService,
	userRepo *repository.UserRepository,
	settingsRepo *repository.SettingsRepository,
) *OAuthHandler {
	return &OAuthHandler{
		oauthService: oauthService,
		authService:  authService,
		userRepo:     userRepo,
		settingsRepo: settingsRepo,
		cookieConfig: utils.GetCookieConfig(),
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

	// Get remember_me preference
	rememberMe := c.Query("remember_me", "false") == "true"

	// Generate authorization URL (includes rememberMe in state token)
	authURL, err := h.oauthService.GetAuthURL(provider, redirectTo, rememberMe)
	if err != nil {
		log.Printf("Failed to generate OAuth URL: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to initiate OAuth flow",
		})
	}

	// Redirect to OAuth provider
	return c.Redirect(authURL, fiber.StatusTemporaryRedirect)
}

// Sentinel errors returned by resolveOAuthUser. HandleCallback maps each to a
// user-facing redirect message; the underlying error (if any) is logged at the
// failure site.
var (
	errEmailNotVerified     = errors.New("oauth email not verified")
	errRegistrationDisabled = errors.New("oauth registration disabled")
	errLinkFailed           = errors.New("oauth link failed")
	errFetchFailed          = errors.New("oauth fetch user failed")
	errCreateFailed         = errors.New("oauth create user failed")
	errRegistrationCheck    = errors.New("oauth registration check failed")
)

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

	// Extract remember_me preference from state token before validation
	rememberMe := h.oauthService.GetRememberMeFromState(state)

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

	user, err := h.resolveOAuthUser(c.Context(), provider, userInfo)
	if err != nil {
		switch {
		case errors.Is(err, errEmailNotVerified):
			return h.redirectWithError(c, "Email not verified with provider; cannot link to an existing account")
		case errors.Is(err, errRegistrationDisabled):
			return h.redirectWithError(c, "Public registration is disabled")
		case errors.Is(err, errLinkFailed):
			return h.redirectWithError(c, "Failed to link account")
		case errors.Is(err, errFetchFailed):
			return h.redirectWithError(c, "Failed to fetch user")
		case errors.Is(err, errRegistrationCheck):
			return h.redirectWithError(c, "Failed to check registration status")
		default: // errCreateFailed and anything unexpected
			return h.redirectWithError(c, "Failed to create account")
		}
	}

	return h.loginUser(c, user, rememberMe)
}

// resolveOAuthUser maps a verified provider identity to a Nimbus user, creating
// one if appropriate. Performs NO HTTP/provider calls, so it is unit-testable
// against real in-memory repositories. Returns a sentinel error that
// HandleCallback maps to a user-facing redirect.
func (h *OAuthHandler) resolveOAuthUser(
	ctx context.Context,
	provider models.OAuthProvider,
	userInfo *models.OAuthUserInfo,
) (*models.User, error) {
	// 1. Already linked by provider id -> log in. Refresh the cached avatar URL
	//    (providers, Discord especially, rotate it) but preserve local uploads.
	existingUser, err := h.userRepo.GetByProviderID(string(provider), userInfo.ProviderID)
	if err == nil {
		if shouldRefreshAvatar(existingUser.AvatarURL, userInfo.AvatarURL) {
			if updateErr := h.userRepo.UpdateAvatar(existingUser.ID, &userInfo.AvatarURL); updateErr != nil {
				log.Printf("Failed to refresh avatar for user %s: %v", existingUser.ID, updateErr)
			}
		}
		return existingUser, nil
	}

	// 2. Email match -> link, but ONLY if the provider verified a non-empty email.
	//    Without this gate, an attacker who registers at an IdP using a victim's
	//    email would be linked into the victim's account on first login (#167).
	existingUser, err = h.userRepo.GetByEmail(userInfo.Email)
	if err == nil {
		if userInfo.Email == "" || !userInfo.EmailVerified {
			return nil, errEmailNotVerified
		}
		// Preserve a locally-uploaded avatar; otherwise adopt the provider's.
		avatarToStore := &userInfo.AvatarURL
		if isLocalAvatarUpload(existingUser.AvatarURL) {
			avatarToStore = existingUser.AvatarURL
		}
		if err := h.userRepo.LinkOAuthProvider(existingUser.ID, string(provider), userInfo.ProviderID, avatarToStore); err != nil {
			log.Printf("Failed to link OAuth provider: %v", err)
			return nil, errLinkFailed
		}
		refreshed, err := h.userRepo.GetByID(existingUser.ID)
		if err != nil {
			log.Printf("Failed to fetch user: %v", err)
			return nil, errFetchFailed
		}
		return refreshed, nil
	}

	// 3. Brand-new user -> gated only by public registration. Unverified email is
	//    fine here: no pre-existing row exists, so there is nothing to take over.
	isEnabled, err := h.settingsRepo.IsPublicRegistrationEnabled(ctx)
	if err != nil {
		log.Printf("Failed to check registration setting: %v", err)
		return nil, errRegistrationCheck
	}
	if !isEnabled {
		return nil, errRegistrationDisabled
	}

	newUser := &models.User{
		Email:         userInfo.Email,
		Name:          userInfo.Name,
		Provider:      string(provider),
		ProviderID:    &userInfo.ProviderID,
		AvatarURL:     &userInfo.AvatarURL,
		EmailVerified: userInfo.EmailVerified, // persist honestly; never forced true
		Role:          "user",                 // Default role
		Password:      nil,                    // OAuth users don't have passwords
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if err := h.userRepo.Create(newUser); err != nil {
		log.Printf("Failed to create OAuth user: %v", err)
		return nil, errCreateFailed
	}
	return newUser, nil
}

// isLocalAvatarUpload reports whether the avatar is a locally-uploaded file
// (served from /uploads/) that must never be clobbered by a provider URL.
func isLocalAvatarUpload(avatarURL *string) bool {
	return avatarURL != nil && strings.HasPrefix(*avatarURL, "/uploads/")
}

// shouldRefreshAvatar reports whether the cached avatar should be replaced with
// the provider's URL. Local uploads are preserved, and an unchanged URL is left
// alone to avoid a redundant write on every login.
func shouldRefreshAvatar(current *string, providerURL string) bool {
	if isLocalAvatarUpload(current) {
		return false
	}
	return current == nil || *current != providerURL
}

// LinkProvider links an OAuth provider to the currently logged-in user
// POST /api/v1/auth/oauth/link/:provider
func (h *OAuthHandler) LinkProvider(c *fiber.Ctx) error {
	// Note: userID will be needed when linking is fully implemented
	// to store in the state token and link provider in callback
	if _, err := RequireUserID(c); err != nil {
		return err
	}

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
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}

	providerStr := c.Params("provider")
	provider := models.OAuthProvider(providerStr)

	// Validate provider
	if !provider.IsValid() || provider == models.ProviderLocal {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid OAuth provider",
		})
	}

	// Unlink the provider
	err = h.userRepo.UnlinkOAuthProvider(userID, string(provider))
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
		{
			"name":      "oidc",
			"enabled":   h.oauthService.IsProviderConfigured(models.ProviderOIDC),
			"configure": h.oauthService.IsProviderConfigured(models.ProviderOIDC),
		},
	}

	return c.JSON(fiber.Map{
		"providers": providers,
	})
}

// Helper: loginUser creates a JWT token and sets the auth cookie
func (h *OAuthHandler) loginUser(c *fiber.Ctx, user *models.User, rememberMe bool) error {
	// Generate token with appropriate expiration (same logic as regular login)
	var token string
	var err error
	maxAge := 0 // Session cookie by default

	if rememberMe {
		// Remember me: 30 days for both token and cookie
		maxAge = 30 * 24 * 60 * 60 // 30 days in seconds
		token, err = h.authService.GenerateTokenWithExpiration(user.ID, user.Email, user.Role, 30*24*time.Hour)
	} else {
		// Session: 24 hours for token, session cookie (cleared on browser close)
		token, err = h.authService.GenerateToken(user.ID, user.Email, user.Role)
	}

	if err != nil {
		log.Printf("Failed to generate token: %v", err)
		return h.redirectWithError(c, "Failed to generate auth token")
	}

	// Set httpOnly cookie
	c.Cookie(utils.NewAuthCookie(token, maxAge, h.cookieConfig))

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
		fmt.Sprintf("%s/login?error=%s", frontendURL, url.QueryEscape(message)),
		fiber.StatusTemporaryRedirect,
	)
}
