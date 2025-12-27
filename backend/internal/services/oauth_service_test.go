package services

import (
	"testing"

	"github.com/nimbus/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

func TestOAuthService_StateToken_RememberMe(t *testing.T) {
	// Create a minimal OAuth service with just the state secret
	service := &OAuthService{
		configs:     make(map[models.OAuthProvider]*oauth2.Config),
		stateSecret: "test-secret-key-for-testing-32ch",
	}

	tests := []struct {
		name       string
		rememberMe bool
	}{
		{
			name:       "State token with remember_me=true",
			rememberMe: true,
		},
		{
			name:       "State token with remember_me=false",
			rememberMe: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate state token
			token, err := service.generateStateToken(models.ProviderGoogle, "/dashboard", tt.rememberMe)
			assert.NoError(t, err)
			assert.NotEmpty(t, token)

			// Extract remember_me from token
			extracted := service.GetRememberMeFromState(token)
			assert.Equal(t, tt.rememberMe, extracted)
		})
	}
}

func TestOAuthService_GetRememberMeFromState_InvalidToken(t *testing.T) {
	service := &OAuthService{
		configs:     make(map[models.OAuthProvider]*oauth2.Config),
		stateSecret: "test-secret-key-for-testing-32ch",
	}

	tests := []struct {
		name     string
		token    string
		expected bool
	}{
		{
			name:     "Empty token",
			token:    "",
			expected: false,
		},
		{
			name:     "Invalid token format",
			token:    "not-a-valid-jwt-token",
			expected: false,
		},
		{
			name:     "Token with wrong secret",
			token:    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJyZW1lbWJlcl9tZSI6dHJ1ZX0.invalid",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.GetRememberMeFromState(tt.token)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestOAuthService_GetRememberMeFromState_DifferentSecret(t *testing.T) {
	// Create two services with different secrets
	service1 := &OAuthService{
		configs:     make(map[models.OAuthProvider]*oauth2.Config),
		stateSecret: "secret-one-for-testing-32chars!",
	}
	service2 := &OAuthService{
		configs:     make(map[models.OAuthProvider]*oauth2.Config),
		stateSecret: "secret-two-for-testing-32chars!",
	}

	// Generate token with service1
	token, err := service1.generateStateToken(models.ProviderGoogle, "/dashboard", true)
	assert.NoError(t, err)

	// Verify service1 can read it
	assert.True(t, service1.GetRememberMeFromState(token))

	// Verify service2 cannot read it (wrong secret)
	assert.False(t, service2.GetRememberMeFromState(token))
}

func TestOAuthService_StateToken_PreservesOtherClaims(t *testing.T) {
	service := &OAuthService{
		configs:     make(map[models.OAuthProvider]*oauth2.Config),
		stateSecret: "test-secret-key-for-testing-32ch",
	}

	// Generate token and validate it still works for state validation
	token, err := service.generateStateToken(models.ProviderGitHub, "/settings", true)
	assert.NoError(t, err)

	// State validation should still pass
	err = service.validateStateToken(token, models.ProviderGitHub)
	assert.NoError(t, err)

	// Wrong provider should fail
	err = service.validateStateToken(token, models.ProviderGoogle)
	assert.Error(t, err)
}
