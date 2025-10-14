package models

import "time"

// UserInvitation represents an invitation to join the platform
type UserInvitation struct {
	ID         string     `json:"id" db:"id"`
	Email      string     `json:"email" db:"email"`
	Token      string     `json:"token" db:"token"`
	InvitedBy  string     `json:"invited_by" db:"invited_by"`
	ExpiresAt  time.Time  `json:"expires_at" db:"expires_at"`
	AcceptedAt *time.Time `json:"accepted_at" db:"accepted_at"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
}

// InvitationResponse is the safe invitation data to return to clients
type InvitationResponse struct {
	ID         string     `json:"id"`
	Email      string     `json:"email"`
	InvitedBy  string     `json:"invited_by"`
	ExpiresAt  time.Time  `json:"expires_at"`
	AcceptedAt *time.Time `json:"accepted_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	IsExpired  bool       `json:"is_expired"`
	IsAccepted bool       `json:"is_accepted"`
}

// ToResponse converts UserInvitation to InvitationResponse
func (inv *UserInvitation) ToResponse() InvitationResponse {
	return InvitationResponse{
		ID:         inv.ID,
		Email:      inv.Email,
		InvitedBy:  inv.InvitedBy,
		ExpiresAt:  inv.ExpiresAt,
		AcceptedAt: inv.AcceptedAt,
		CreatedAt:  inv.CreatedAt,
		IsExpired:  time.Now().After(inv.ExpiresAt),
		IsAccepted: inv.AcceptedAt != nil,
	}
}

// InviteUserRequest represents a request to invite a user
type InviteUserRequest struct {
	Email string `json:"email" validate:"required,email"`
}
