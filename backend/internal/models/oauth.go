package models

import "time"

// OAuthProvider represents the supported OAuth providers
type OAuthProvider string

const (
	ProviderLocal   OAuthProvider = "local"
	ProviderGoogle  OAuthProvider = "google"
	ProviderGitHub  OAuthProvider = "github"
	ProviderDiscord OAuthProvider = "discord"
)

// IsValid checks if the provider is supported
func (p OAuthProvider) IsValid() bool {
	switch p {
	case ProviderLocal, ProviderGoogle, ProviderGitHub, ProviderDiscord:
		return true
	default:
		return false
	}
}

// OAuthConfig holds configuration for an OAuth provider
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

// OAuthUserInfo represents user information from OAuth providers
// This is a normalized structure that all providers map to
type OAuthUserInfo struct {
	ProviderID    string // Unique user ID from the provider
	Email         string
	Name          string
	AvatarURL     string
	EmailVerified bool
}

// OAuthStateToken represents the state token for OAuth flow (CSRF protection)
type OAuthStateToken struct {
	State      string    `json:"state"`       // Random state value
	Provider   string    `json:"provider"`    // OAuth provider
	RedirectTo string    `json:"redirect_to"` // Where to redirect after OAuth
	CreatedAt  time.Time `json:"created_at"`  // Token creation time
}

// OAuthCallbackRequest represents the OAuth callback request
type OAuthCallbackRequest struct {
	Code  string `query:"code" validate:"required"`
	State string `query:"state" validate:"required"`
}

// LinkProviderRequest represents a request to link an OAuth provider to existing account
type LinkProviderRequest struct {
	Provider string `json:"provider" validate:"required"`
}
