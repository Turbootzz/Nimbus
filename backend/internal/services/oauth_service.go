package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/nimbus/backend/internal/models"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

var (
	ErrInvalidProvider = errors.New("invalid OAuth provider")
	ErrInvalidState    = errors.New("invalid OAuth state token")
	ErrExpiredState    = errors.New("OAuth state token expired")
	ErrCodeExchange    = errors.New("failed to exchange OAuth code")
	ErrFetchUserInfo   = errors.New("failed to fetch user info from provider")
)

// OAuthService handles OAuth authentication flows
type OAuthService struct {
	configs         map[models.OAuthProvider]*oauth2.Config
	oidcUserInfoURL string // discovered at startup for the OIDC provider
	stateSecret     string
}

// NewOAuthService creates a new OAuth service
func NewOAuthService(
	googleConfig models.OAuthConfig,
	githubConfig models.OAuthConfig,
	discordConfig models.OAuthConfig,
	oidcConfig models.OAuthConfig,
	stateSecret string,
) *OAuthService {
	service := &OAuthService{
		configs:     make(map[models.OAuthProvider]*oauth2.Config),
		stateSecret: stateSecret,
	}

	// Configure Google OAuth
	if googleConfig.ClientID != "" {
		service.configs[models.ProviderGoogle] = &oauth2.Config{
			ClientID:     googleConfig.ClientID,
			ClientSecret: googleConfig.ClientSecret,
			RedirectURL:  googleConfig.RedirectURL,
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			},
			Endpoint: google.Endpoint,
		}
	}

	// Configure GitHub OAuth
	if githubConfig.ClientID != "" {
		service.configs[models.ProviderGitHub] = &oauth2.Config{
			ClientID:     githubConfig.ClientID,
			ClientSecret: githubConfig.ClientSecret,
			RedirectURL:  githubConfig.RedirectURL,
			Scopes:       []string{"user:email", "read:user"},
			Endpoint:     github.Endpoint,
		}
	}

	// Configure Discord OAuth
	if discordConfig.ClientID != "" {
		service.configs[models.ProviderDiscord] = &oauth2.Config{
			ClientID:     discordConfig.ClientID,
			ClientSecret: discordConfig.ClientSecret,
			RedirectURL:  discordConfig.RedirectURL,
			Scopes:       []string{"identify", "email"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://discord.com/api/oauth2/authorize",
				TokenURL: "https://discord.com/api/oauth2/token",
			},
		}
	}

	// Configure generic OIDC — discover endpoints from the issuer URL
	if oidcConfig.ClientID != "" && oidcConfig.IssuerURL != "" {
		if endpoint, userInfoURL, err := discoverOIDCEndpoints(oidcConfig.IssuerURL); err != nil {
			log.Printf("OIDC provider disabled: failed to discover endpoints for %s: %v", oidcConfig.IssuerURL, err)
		} else {
			service.configs[models.ProviderOIDC] = &oauth2.Config{
				ClientID:     oidcConfig.ClientID,
				ClientSecret: oidcConfig.ClientSecret,
				RedirectURL:  oidcConfig.RedirectURL,
				Scopes:       []string{"openid", "profile", "email"},
				Endpoint:     endpoint,
			}
			service.oidcUserInfoURL = userInfoURL
		}
	}

	return service
}

// discoverOIDCEndpoints fetches the provider's discovery document and returns
// the oauth2.Endpoint and userinfo URL.
func discoverOIDCEndpoints(issuerURL string) (oauth2.Endpoint, string, error) {
	discoveryURL := strings.TrimRight(issuerURL, "/") + "/.well-known/openid-configuration"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
	if err != nil {
		return oauth2.Endpoint{}, "", fmt.Errorf("build request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return oauth2.Endpoint{}, "", fmt.Errorf("fetch discovery document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return oauth2.Endpoint{}, "", fmt.Errorf("discovery document returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return oauth2.Endpoint{}, "", fmt.Errorf("read discovery document: %w", err)
	}

	var doc struct {
		AuthorizationEndpoint string `json:"authorization_endpoint"`
		TokenEndpoint         string `json:"token_endpoint"`
		UserinfoEndpoint      string `json:"userinfo_endpoint"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return oauth2.Endpoint{}, "", fmt.Errorf("parse discovery document: %w", err)
	}

	if doc.AuthorizationEndpoint == "" || doc.TokenEndpoint == "" {
		return oauth2.Endpoint{}, "", fmt.Errorf("discovery document missing required endpoints")
	}

	return oauth2.Endpoint{
		AuthURL:  doc.AuthorizationEndpoint,
		TokenURL: doc.TokenEndpoint,
	}, doc.UserinfoEndpoint, nil
}

// GetAuthURL generates the OAuth authorization URL with state token
func (s *OAuthService) GetAuthURL(provider models.OAuthProvider, redirectTo string, rememberMe bool) (string, error) {
	config, ok := s.configs[provider]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrInvalidProvider, provider)
	}

	// Generate state token for CSRF protection (includes rememberMe preference)
	stateToken, err := s.generateStateToken(provider, redirectTo, rememberMe)
	if err != nil {
		return "", fmt.Errorf("failed to generate state token: %w", err)
	}

	// Generate authorization URL
	authURL := config.AuthCodeURL(stateToken, oauth2.AccessTypeOffline)
	return authURL, nil
}

// ExchangeCode exchanges the authorization code for user information
func (s *OAuthService) ExchangeCode(ctx context.Context, provider models.OAuthProvider, code string, state string) (*models.OAuthUserInfo, error) {
	// Validate state token
	if err := s.validateStateToken(state, provider); err != nil {
		return nil, err
	}

	// Get OAuth config for provider
	config, ok := s.configs[provider]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrInvalidProvider, provider)
	}

	// Exchange code for token
	token, err := config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCodeExchange, err)
	}

	// Fetch user info from provider
	userInfo, err := s.fetchUserInfo(ctx, provider, token)
	if err != nil {
		return nil, err
	}

	return userInfo, nil
}

// generateStateToken creates a JWT state token for CSRF protection
func (s *OAuthService) generateStateToken(provider models.OAuthProvider, redirectTo string, rememberMe bool) (string, error) {
	claims := jwt.MapClaims{
		"provider":    string(provider),
		"redirect_to": redirectTo,
		"remember_me": rememberMe,
		"created_at":  time.Now().Unix(),
		"exp":         time.Now().Add(5 * time.Minute).Unix(), // 5 minute expiry
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.stateSecret))
}

// GetRememberMeFromState extracts the remember_me preference from the state token
func (s *OAuthService) GetRememberMeFromState(stateToken string) bool {
	token, err := jwt.Parse(stateToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.stateSecret), nil
	})

	if err != nil {
		return false
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return false
	}

	rememberMe, ok := claims["remember_me"].(bool)
	if !ok {
		return false
	}

	return rememberMe
}

// validateStateToken validates the state token and extracts provider
func (s *OAuthService) validateStateToken(stateToken string, expectedProvider models.OAuthProvider) error {
	token, err := jwt.Parse(stateToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.stateSecret), nil
	})

	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidState, err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return ErrInvalidState
	}

	// Check provider matches
	provider, ok := claims["provider"].(string)
	if !ok || models.OAuthProvider(provider) != expectedProvider {
		return ErrInvalidState
	}

	// Check expiration
	exp, ok := claims["exp"].(float64)
	if !ok || time.Unix(int64(exp), 0).Before(time.Now()) {
		return ErrExpiredState
	}

	return nil
}

// fetchUserInfo fetches user information from the OAuth provider
func (s *OAuthService) fetchUserInfo(ctx context.Context, provider models.OAuthProvider, token *oauth2.Token) (*models.OAuthUserInfo, error) {
	switch provider {
	case models.ProviderGoogle:
		return s.fetchGoogleUserInfo(ctx, token)
	case models.ProviderGitHub:
		return s.fetchGitHubUserInfo(ctx, token)
	case models.ProviderDiscord:
		return s.fetchDiscordUserInfo(ctx, token)
	case models.ProviderOIDC:
		return s.fetchOIDCUserInfo(ctx, token)
	default:
		return nil, fmt.Errorf("%w: %s", ErrInvalidProvider, provider)
	}
}

// fetchGoogleUserInfo fetches user info from Google
func (s *OAuthService) fetchGoogleUserInfo(ctx context.Context, token *oauth2.Token) (*models.OAuthUserInfo, error) {
	client := s.configs[models.ProviderGoogle].Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFetchUserInfo, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrFetchUserInfo, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFetchUserInfo, err)
	}

	var googleUser struct {
		ID            string `json:"id"`
		Email         string `json:"email"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
		VerifiedEmail bool   `json:"verified_email"`
	}

	if err := json.Unmarshal(body, &googleUser); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFetchUserInfo, err)
	}

	return &models.OAuthUserInfo{
		ProviderID:    googleUser.ID,
		Email:         googleUser.Email,
		Name:          googleUser.Name,
		AvatarURL:     googleUser.Picture,
		EmailVerified: googleUser.VerifiedEmail,
	}, nil
}

// fetchGitHubUserInfo fetches user info from GitHub
func (s *OAuthService) fetchGitHubUserInfo(ctx context.Context, token *oauth2.Token) (*models.OAuthUserInfo, error) {
	client := s.configs[models.ProviderGitHub].Client(ctx, token)

	// Fetch user profile
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFetchUserInfo, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrFetchUserInfo, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFetchUserInfo, err)
	}

	var githubUser struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}

	if err := json.Unmarshal(body, &githubUser); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFetchUserInfo, err)
	}

	// If email is not public, fetch from emails endpoint
	if githubUser.Email == "" {
		githubUser.Email, err = s.fetchGitHubEmail(client)
		if err != nil {
			return nil, err
		}
	}

	// Use login as name if name is empty
	name := githubUser.Name
	if name == "" {
		name = githubUser.Login
	}

	return &models.OAuthUserInfo{
		ProviderID:    fmt.Sprintf("%d", githubUser.ID),
		Email:         githubUser.Email,
		Name:          name,
		AvatarURL:     githubUser.AvatarURL,
		EmailVerified: true, // GitHub emails are considered verified
	}, nil
}

// fetchGitHubEmail fetches the primary email from GitHub (when not public)
func (s *OAuthService) fetchGitHubEmail(client *http.Client) (string, error) {
	resp, err := client.Get("https://api.github.com/user/emails")
	if err != nil {
		return "", fmt.Errorf("%w: failed to fetch emails", ErrFetchUserInfo)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: failed to fetch emails, status %d", ErrFetchUserInfo, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrFetchUserInfo, err)
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}

	if err := json.Unmarshal(body, &emails); err != nil {
		return "", fmt.Errorf("%w: %v", ErrFetchUserInfo, err)
	}

	// Find primary verified email
	for _, email := range emails {
		if email.Primary && email.Verified {
			return email.Email, nil
		}
	}

	// Fallback to first verified email
	for _, email := range emails {
		if email.Verified {
			return email.Email, nil
		}
	}

	return "", fmt.Errorf("%w: no verified email found", ErrFetchUserInfo)
}

// fetchDiscordUserInfo fetches user info from Discord
func (s *OAuthService) fetchDiscordUserInfo(ctx context.Context, token *oauth2.Token) (*models.OAuthUserInfo, error) {
	client := s.configs[models.ProviderDiscord].Client(ctx, token)
	resp, err := client.Get("https://discord.com/api/users/@me")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFetchUserInfo, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrFetchUserInfo, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFetchUserInfo, err)
	}

	var discordUser struct {
		ID            string `json:"id"`
		Username      string `json:"username"`
		GlobalName    string `json:"global_name"`
		Email         string `json:"email"`
		Avatar        string `json:"avatar"`
		Verified      bool   `json:"verified"`
		Discriminator string `json:"discriminator"`
	}

	if err := json.Unmarshal(body, &discordUser); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFetchUserInfo, err)
	}

	// Use global_name if available, otherwise username
	name := discordUser.GlobalName
	if name == "" {
		name = discordUser.Username
	}

	// Construct avatar URL if avatar hash exists
	avatarURL := ""
	if discordUser.Avatar != "" {
		avatarURL = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", discordUser.ID, discordUser.Avatar)
	}

	return &models.OAuthUserInfo{
		ProviderID:    discordUser.ID,
		Email:         discordUser.Email,
		Name:          name,
		AvatarURL:     avatarURL,
		EmailVerified: discordUser.Verified,
	}, nil
}

// fetchOIDCUserInfo fetches user info from a generic OIDC provider's userinfo endpoint.
// The endpoint URL is discovered at startup from the issuer's discovery document.
func (s *OAuthService) fetchOIDCUserInfo(ctx context.Context, token *oauth2.Token) (*models.OAuthUserInfo, error) {
	client := s.configs[models.ProviderOIDC].Client(ctx, token)
	resp, err := client.Get(s.oidcUserInfoURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFetchUserInfo, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrFetchUserInfo, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFetchUserInfo, err)
	}

	var u struct {
		Sub           string      `json:"sub"`
		Name          string      `json:"name"`
		GivenName     string      `json:"given_name"`
		FamilyName    string      `json:"family_name"`
		Email         string      `json:"email"`
		Picture       string      `json:"picture"`
		EmailVerified interface{} `json:"email_verified"` // bool or string depending on provider
	}
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFetchUserInfo, err)
	}

	name := u.Name
	if name == "" {
		name = strings.TrimSpace(u.GivenName + " " + u.FamilyName)
	}
	if name == "" {
		name = u.Email
	}

	emailVerified := false
	switch v := u.EmailVerified.(type) {
	case bool:
		emailVerified = v
	case string:
		emailVerified = v == "true"
	}

	return &models.OAuthUserInfo{
		ProviderID:    u.Sub,
		Email:         u.Email,
		Name:          name,
		AvatarURL:     u.Picture,
		EmailVerified: emailVerified,
	}, nil
}

// IsProviderConfigured checks if a provider is configured and available
func (s *OAuthService) IsProviderConfigured(provider models.OAuthProvider) bool {
	_, ok := s.configs[provider]
	return ok
}
