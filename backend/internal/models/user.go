package models

import "time"

// User represents a user in the system
type User struct {
	ID             string     `json:"id" db:"id"`
	Email          string     `json:"email" db:"email"`
	Name           string     `json:"name" db:"name"`
	Password       *string    `json:"-" db:"password"`                        // Never expose password in JSON, nullable for OAuth users
	Role           string     `json:"role" db:"role"`                         // "admin" or "user"
	Provider       string     `json:"provider" db:"provider"`                 // Auth provider: "local", "google", "github", "discord"
	ProviderID     *string    `json:"provider_id,omitempty" db:"provider_id"` // External provider user ID
	AvatarURL      *string    `json:"avatar_url,omitempty" db:"avatar_url"`   // Profile picture URL from OAuth provider
	EmailVerified  bool       `json:"email_verified" db:"email_verified"`     // Whether email is verified
	LastActivityAt *time.Time `json:"last_activity_at" db:"last_activity_at"` // Last login/activity
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// UserResponse is the safe user data to return to clients
type UserResponse struct {
	ID             string     `json:"id"`
	Email          string     `json:"email"`
	Name           string     `json:"name"`
	Role           string     `json:"role"`
	Provider       string     `json:"provider"`
	AvatarURL      *string    `json:"avatar_url,omitempty"`
	EmailVerified  bool       `json:"email_verified"`
	LastActivityAt *time.Time `json:"last_activity_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// ToResponse converts User to UserResponse (without password)
func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:             u.ID,
		Email:          u.Email,
		Name:           u.Name,
		Role:           u.Role,
		Provider:       u.Provider,
		AvatarURL:      u.AvatarURL,
		EmailVerified:  u.EmailVerified,
		LastActivityAt: u.LastActivityAt,
		CreatedAt:      u.CreatedAt,
		UpdatedAt:      u.UpdatedAt,
	}
}

// LoginRequest represents login credentials
type LoginRequest struct {
	Email      string `json:"email" validate:"required,email"`
	Password   string `json:"password" validate:"required,min=8"`
	RememberMe bool   `json:"remember_me"`
}

// RegisterRequest represents registration data
type RegisterRequest struct {
	Name     string `json:"name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

// AuthResponse represents authentication response with token
type AuthResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}
