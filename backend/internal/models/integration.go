package models

import (
	"encoding/json"
	"time"
)

// Integration auth types (mirrors the CHECK constraint in migration 000026)
const (
	IntegrationAuthNone   = "none"
	IntegrationAuthAPIKey = "api_key"
	IntegrationAuthBasic  = "basic"
	IntegrationAuthToken  = "token"
)

// IsValidIntegrationAuthType checks if the given auth type is known
func IsValidIntegrationAuthType(authType string) bool {
	switch authType {
	case IntegrationAuthNone, IntegrationAuthAPIKey, IntegrationAuthBasic, IntegrationAuthToken:
		return true
	}
	return false
}

// Integration is a user's connection to an external app. It never carries
// credentials (only HasCredentials), so it is always safe to return to clients.
type Integration struct {
	ID             string          `json:"id"`
	UserID         string          `json:"user_id"`
	Kind           string          `json:"kind"`
	Name           string          `json:"name"`
	BaseURL        string          `json:"base_url"`
	AuthType       string          `json:"auth_type"`
	HasCredentials bool            `json:"has_credentials"`
	VerifyTLS      bool            `json:"verify_tls"`
	Options        json.RawMessage `json:"options"`
	RefreshSeconds int             `json:"refresh_seconds"`
	LastTestAt     *time.Time      `json:"last_test_at"`
	LastTestOK     *bool           `json:"last_test_ok"`
	LastError      *string         `json:"last_error"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// IntegrationCredentials are the decrypted secrets. Which fields are used
// depends on the auth type. Never return this to clients.
type IntegrationCredentials struct {
	APIKey   string `json:"api_key,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Token    string `json:"token,omitempty"`
}

// IntegrationRequest is the body for create, update and the unsaved test.
// On update, empty or omitted fields keep their stored value; that includes
// Credentials, so the UI never needs to send secrets back.
type IntegrationRequest struct {
	Kind           string                  `json:"kind"`
	Name           string                  `json:"name"`
	BaseURL        string                  `json:"base_url"`
	AuthType       string                  `json:"auth_type"`
	Credentials    *IntegrationCredentials `json:"credentials"`
	VerifyTLS      *bool                   `json:"verify_tls"`
	Options        json.RawMessage         `json:"options"`
	RefreshSeconds *int                    `json:"refresh_seconds"`
}

// IntegrationTestResult is the outcome of a connection test
type IntegrationTestResult struct {
	OK        bool   `json:"ok"`
	Error     string `json:"error,omitempty"`
	LatencyMs int64  `json:"latency_ms"`
}
