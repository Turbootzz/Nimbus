package services

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nimbus/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// discoveryServer spins up a test server that serves an OIDC discovery
// document at /.well-known/openid-configuration. The body is rendered by the
// caller-provided function so each test can control issuer/endpoints; the
// {{ISSUER}} placeholder is replaced with the server's own URL.
func discoveryServer(t *testing.T, status int, body func(issuer string) string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(nil)
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body(srv.URL)))
	})
	srv.Config.Handler = mux
	return srv
}

func TestDiscoverOIDCEndpoints_Success(t *testing.T) {
	srv := discoveryServer(t, http.StatusOK, func(issuer string) string {
		return `{
			"issuer": "` + issuer + `",
			"authorization_endpoint": "` + issuer + `/auth",
			"token_endpoint": "` + issuer + `/token",
			"userinfo_endpoint": "` + issuer + `/userinfo"
		}`
	})
	defer srv.Close()

	endpoint, userInfoURL, err := discoverOIDCEndpoints(srv.URL)
	require.NoError(t, err)
	assert.Equal(t, srv.URL+"/auth", endpoint.AuthURL)
	assert.Equal(t, srv.URL+"/token", endpoint.TokenURL)
	assert.Equal(t, srv.URL+"/userinfo", userInfoURL)
}

func TestDiscoverOIDCEndpoints_TrailingSlashIssuerStillMatches(t *testing.T) {
	srv := discoveryServer(t, http.StatusOK, func(issuer string) string {
		// Provider advertises the issuer with a trailing slash; our config has none.
		return `{
			"issuer": "` + issuer + `/",
			"authorization_endpoint": "` + issuer + `/auth",
			"token_endpoint": "` + issuer + `/token",
			"userinfo_endpoint": "` + issuer + `/userinfo"
		}`
	})
	defer srv.Close()

	_, _, err := discoverOIDCEndpoints(srv.URL)
	assert.NoError(t, err)
}

func TestDiscoverOIDCEndpoints_IssuerMismatch(t *testing.T) {
	srv := discoveryServer(t, http.StatusOK, func(issuer string) string {
		return `{
			"issuer": "https://evil.example.com",
			"authorization_endpoint": "` + issuer + `/auth",
			"token_endpoint": "` + issuer + `/token",
			"userinfo_endpoint": "` + issuer + `/userinfo"
		}`
	})
	defer srv.Close()

	_, _, err := discoverOIDCEndpoints(srv.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "issuer mismatch")
}

func TestDiscoverOIDCEndpoints_MissingEndpoints(t *testing.T) {
	tests := []struct {
		name string
		doc  func(issuer string) string
	}{
		{
			name: "missing authorization_endpoint",
			doc: func(issuer string) string {
				return `{"issuer":"` + issuer + `","token_endpoint":"` + issuer + `/token","userinfo_endpoint":"` + issuer + `/userinfo"}`
			},
		},
		{
			name: "missing token_endpoint",
			doc: func(issuer string) string {
				return `{"issuer":"` + issuer + `","authorization_endpoint":"` + issuer + `/auth","userinfo_endpoint":"` + issuer + `/userinfo"}`
			},
		},
		{
			name: "missing userinfo_endpoint",
			doc: func(issuer string) string {
				return `{"issuer":"` + issuer + `","authorization_endpoint":"` + issuer + `/auth","token_endpoint":"` + issuer + `/token"}`
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := discoveryServer(t, http.StatusOK, tt.doc)
			defer srv.Close()

			_, _, err := discoverOIDCEndpoints(srv.URL)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "missing required endpoints")
		})
	}
}

func TestDiscoverOIDCEndpoints_Non200(t *testing.T) {
	srv := discoveryServer(t, http.StatusNotFound, func(issuer string) string { return `{}` })
	defer srv.Close()

	_, _, err := discoverOIDCEndpoints(srv.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 404")
}

func TestDiscoverOIDCEndpoints_InvalidJSON(t *testing.T) {
	srv := discoveryServer(t, http.StatusOK, func(issuer string) string { return `not json` })
	defer srv.Close()

	_, _, err := discoverOIDCEndpoints(srv.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse discovery document")
}

func TestDiscoverOIDCEndpoints_UnreachableIssuer(t *testing.T) {
	// Port 0 on the discard host is never listening, so the request fails.
	_, _, err := discoverOIDCEndpoints("http://127.0.0.1:0")
	assert.Error(t, err)
}

// oidcService builds an OAuthService wired to a userinfo endpoint for testing
// fetchOIDCUserInfo without going through discovery.
func oidcService(userInfoURL string) *OAuthService {
	return &OAuthService{
		configs: map[models.OAuthProvider]*oauth2.Config{
			models.ProviderOIDC: {},
		},
		oidcUserInfoURL: userInfoURL,
		stateSecret:     "test-secret-key-for-testing-32ch",
	}
}

func userInfoServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
}

func TestFetchOIDCUserInfo_Success(t *testing.T) {
	srv := userInfoServer(t, http.StatusOK, `{
		"sub": "user-123",
		"name": "Ada Lovelace",
		"email": "ada@example.com",
		"picture": "https://example.com/ada.png",
		"email_verified": true
	}`)
	defer srv.Close()

	info, err := oidcService(srv.URL).fetchOIDCUserInfo(t.Context(), &oauth2.Token{AccessToken: "x"})
	require.NoError(t, err)
	assert.Equal(t, "user-123", info.ProviderID)
	assert.Equal(t, "ada@example.com", info.Email)
	assert.Equal(t, "Ada Lovelace", info.Name)
	assert.Equal(t, "https://example.com/ada.png", info.AvatarURL)
	assert.True(t, info.EmailVerified)
}

func TestFetchOIDCUserInfo_NameFallbacks(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantName string
	}{
		{
			name:     "falls back to given_name + family_name when name is absent",
			body:     `{"sub":"s","email":"u@example.com","given_name":"Grace","family_name":"Hopper"}`,
			wantName: "Grace Hopper",
		},
		{
			name:     "falls back to email when no name claims are present",
			body:     `{"sub":"s","email":"u@example.com"}`,
			wantName: "u@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := userInfoServer(t, http.StatusOK, tt.body)
			defer srv.Close()

			info, err := oidcService(srv.URL).fetchOIDCUserInfo(t.Context(), &oauth2.Token{AccessToken: "x"})
			require.NoError(t, err)
			assert.Equal(t, tt.wantName, info.Name)
		})
	}
}

func TestFetchOIDCUserInfo_EmailVerifiedFormats(t *testing.T) {
	tests := []struct {
		name string
		body string
		want bool
	}{
		{name: "bool true", body: `{"sub":"s","email":"u@e.com","email_verified":true}`, want: true},
		{name: "bool false", body: `{"sub":"s","email":"u@e.com","email_verified":false}`, want: false},
		{name: "string true (Keycloak/Authentik)", body: `{"sub":"s","email":"u@e.com","email_verified":"true"}`, want: true},
		{name: "string false", body: `{"sub":"s","email":"u@e.com","email_verified":"false"}`, want: false},
		{name: "absent", body: `{"sub":"s","email":"u@e.com"}`, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := userInfoServer(t, http.StatusOK, tt.body)
			defer srv.Close()

			info, err := oidcService(srv.URL).fetchOIDCUserInfo(t.Context(), &oauth2.Token{AccessToken: "x"})
			require.NoError(t, err)
			assert.Equal(t, tt.want, info.EmailVerified)
		})
	}
}

func TestFetchOIDCUserInfo_MissingRequiredClaims(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr string
	}{
		{
			name:    "missing sub",
			body:    `{"email":"u@example.com"}`,
			wantErr: "missing subject claim",
		},
		{
			name:    "missing email",
			body:    `{"sub":"user-123"}`,
			wantErr: "missing email claim",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := userInfoServer(t, http.StatusOK, tt.body)
			defer srv.Close()

			_, err := oidcService(srv.URL).fetchOIDCUserInfo(t.Context(), &oauth2.Token{AccessToken: "x"})
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrFetchUserInfo)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestFetchOIDCUserInfo_Non200(t *testing.T) {
	srv := userInfoServer(t, http.StatusUnauthorized, `{"error":"invalid_token"}`)
	defer srv.Close()

	_, err := oidcService(srv.URL).fetchOIDCUserInfo(t.Context(), &oauth2.Token{AccessToken: "x"})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrFetchUserInfo)
	assert.Contains(t, err.Error(), "status 401")
}

func TestFetchOIDCUserInfo_InvalidJSON(t *testing.T) {
	srv := userInfoServer(t, http.StatusOK, `not json`)
	defer srv.Close()

	_, err := oidcService(srv.URL).fetchOIDCUserInfo(t.Context(), &oauth2.Token{AccessToken: "x"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrFetchUserInfo))
}
