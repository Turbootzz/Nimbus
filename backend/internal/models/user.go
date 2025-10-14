package models

import "time"

// User represents a user in the system
type User struct {
	ID             string     `json:"id" db:"id"`
	Email          string     `json:"email" db:"email"`
	Name           string     `json:"name" db:"name"`
	Password       string     `json:"-" db:"password"`                        // Never expose password in JSON
	Role           string     `json:"role" db:"role"`                         // "admin" or "user"
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
