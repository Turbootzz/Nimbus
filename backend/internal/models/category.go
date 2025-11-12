package models

import "time"

const (
	DefaultCategoryColor = "#6366f1"
	MaxCategoryNameLen   = 100
)

// Category represents a user-defined category for organizing services
type Category struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	Name      string    `json:"name" db:"name"`
	Color     string    `json:"color" db:"color"`       // Hex color code (e.g., #6366f1)
	Position  int       `json:"position" db:"position"` // User-defined position for ordering
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// CategoryCreateRequest represents the data needed to create a new category
type CategoryCreateRequest struct {
	Name  string `json:"name" validate:"required,max=100"`
	Color string `json:"color" validate:"omitempty,hexcolor"`
}

// CategoryUpdateRequest represents the data needed to update a category
type CategoryUpdateRequest struct {
	Name  string `json:"name" validate:"required,max=100"`
	Color string `json:"color" validate:"omitempty,hexcolor"`
}

// CategoryResponse is the safe category data to return to clients
type CategoryResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	Position  int       `json:"position"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ToResponse converts Category to CategoryResponse
func (c *Category) ToResponse() CategoryResponse {
	return CategoryResponse{
		ID:        c.ID,
		Name:      c.Name,
		Color:     c.Color,
		Position:  c.Position,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

// CategoryPosition represents a category ID and its new position
type CategoryPosition struct {
	ID       string `json:"id" validate:"required"`
	Position int    `json:"position" validate:"min=0"`
}

// CategoryReorderRequest represents bulk position updates
type CategoryReorderRequest struct {
	Categories []CategoryPosition `json:"categories" validate:"required,dive"`
}
