package models

import "time"

// Service represents a service/link in the homelab dashboard
type Service struct {
	ID          string    `json:"id" db:"id"`
	UserID      string    `json:"user_id" db:"user_id"`
	Name        string    `json:"name" db:"name"`
	URL         string    `json:"url" db:"url"`
	Icon        string    `json:"icon" db:"icon"` // Emoji or icon identifier
	Description string    `json:"description" db:"description"`
	Status      string    `json:"status" db:"status"` // "online", "offline", "unknown"
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// ServiceCreateRequest represents the data needed to create a new service
type ServiceCreateRequest struct {
	Name        string `json:"name" validate:"required"`
	URL         string `json:"url" validate:"required,url"`
	Icon        string `json:"icon"`
	Description string `json:"description"`
}

// ServiceUpdateRequest represents the data needed to update a service
type ServiceUpdateRequest struct {
	Name        string `json:"name" validate:"required"`
	URL         string `json:"url" validate:"required,url"`
	Icon        string `json:"icon"`
	Description string `json:"description"`
}

// ServiceResponse is the safe service data to return to clients
type ServiceResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	URL         string    `json:"url"`
	Icon        string    `json:"icon"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ToResponse converts Service to ServiceResponse
func (s *Service) ToResponse() ServiceResponse {
	return ServiceResponse{
		ID:          s.ID,
		Name:        s.Name,
		URL:         s.URL,
		Icon:        s.Icon,
		Description: s.Description,
		Status:      s.Status,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}
}
