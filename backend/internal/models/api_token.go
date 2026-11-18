package models

import (
	"slices"
	"time"
)

// API token scopes. Safe requests (GET, HEAD) need the read scope of the
// area they touch; every other request needs that scope plus write:services.
const (
	ScopeReadServices  = "read:services"  // services, groups, webhooks
	ScopeReadMetrics   = "read:metrics"   // uptime metrics, status logs, Prometheus export
	ScopeReadWidgets   = "read:widgets"   // widgets, dashboard data, preferences
	ScopeWriteServices = "write:services" // any change; implies read:services
	ScopeMCP           = "mcp"            // the /mcp endpoint
)

// APITokenScopes lists every valid scope, in display order
var APITokenScopes = []string{ScopeReadServices, ScopeReadMetrics, ScopeReadWidgets, ScopeWriteServices, ScopeMCP}

// ReadScopes is what a token gets when no scopes are requested
func ReadScopes() []string {
	return []string{ScopeReadServices, ScopeReadMetrics, ScopeReadWidgets}
}

// LegacyScopes maps the old read-only flag to scopes. Migration 000038
// applies the same mapping to existing tokens.
func LegacyScopes(readOnly bool) []string {
	if readOnly {
		return ReadScopes()
	}
	return append(ReadScopes(), ScopeWriteServices)
}

// APIToken is a personal access token for programmatic API access.
// TokenHash is never serialized; the plaintext token is only returned at creation.
type APIToken struct {
	ID          string   `json:"id" db:"id"`
	UserID      string   `json:"user_id" db:"user_id"`
	Name        string   `json:"name" db:"name"`
	TokenHash   string   `json:"-" db:"token_hash"`
	TokenPrefix string   `json:"token_prefix" db:"token_prefix"`
	Scopes      []string `json:"scopes" db:"scopes"`
	// ReadOnly is derived from Scopes: true without write:services
	ReadOnly   bool       `json:"read_only" db:"read_only"`
	LastUsedAt *time.Time `json:"last_used_at" db:"last_used_at"`
	ExpiresAt  *time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
}

// HasScope reports whether the token grants a scope
func (t *APIToken) HasScope(scope string) bool {
	if scope == ScopeReadServices && slices.Contains(t.Scopes, ScopeWriteServices) {
		return true
	}
	return slices.Contains(t.Scopes, scope)
}

// Allows reports whether the token may make a request in the area guarded by
// readScope. An empty readScope means the area is closed to tokens.
func (t *APIToken) Allows(readScope string, write bool) bool {
	if readScope == "" || !t.HasScope(readScope) {
		return false
	}
	return !write || t.HasScope(ScopeWriteServices)
}

// Expired reports whether the token's expiry has passed
func (t *APIToken) Expired(now time.Time) bool {
	return t.ExpiresAt != nil && !now.Before(*t.ExpiresAt)
}

// APITokenCreateRequest is the payload for creating a token. Without Scopes
// the legacy ReadOnly flag decides (default: read scopes only).
type APITokenCreateRequest struct {
	Name      string     `json:"name"`
	Scopes    []string   `json:"scopes"`
	ReadOnly  *bool      `json:"read_only"`
	ExpiresAt *time.Time `json:"expires_at"`
}

// APITokenCreateResponse carries the plaintext token exactly once
type APITokenCreateResponse struct {
	Token    string   `json:"token"`
	APIToken APIToken `json:"api_token"`
}
