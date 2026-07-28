package models

import "time"

// APIToken is a personal access token for programmatic API access.
// TokenHash is never serialized; the plaintext token is only returned at creation.
type APIToken struct {
	ID          string     `json:"id" db:"id"`
	UserID      string     `json:"user_id" db:"user_id"`
	Name        string     `json:"name" db:"name"`
	TokenHash   string     `json:"-" db:"token_hash"`
	TokenPrefix string     `json:"token_prefix" db:"token_prefix"`
	ReadOnly    bool       `json:"read_only" db:"read_only"`
	LastUsedAt  *time.Time `json:"last_used_at" db:"last_used_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}

// APITokenCreateRequest is the payload for creating a token
type APITokenCreateRequest struct {
	Name     string `json:"name"`
	ReadOnly *bool  `json:"read_only"`
}

// APITokenCreateResponse carries the plaintext token exactly once
type APITokenCreateResponse struct {
	Token    string   `json:"token"`
	APIToken APIToken `json:"api_token"`
}
