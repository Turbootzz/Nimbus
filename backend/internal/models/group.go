package models

import (
	"regexp"
	"time"
)

// Group constants
const (
	DefaultGroupColor = "#0ea5e9" // Nimbus default blue
	MaxGroupNameLen   = 35
	DefaultGroupName  = "Services"
)

// hexColorRegex validates hex color format #RRGGBB
var hexColorRegex = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// IsValidHexColor checks if the given color is a valid hex color
func IsValidHexColor(color string) bool {
	return hexColorRegex.MatchString(color)
}

// Group represents a service group for organizing services
type Group struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	Name      string    `json:"name" db:"name"`
	Color     string    `json:"color" db:"color"`
	Position  int       `json:"position" db:"position"`
	IsDefault bool      `json:"is_default" db:"is_default"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// GroupCreateRequest represents the data needed to create a new group
type GroupCreateRequest struct {
	Name  string `json:"name" validate:"required,max=35"`
	Color string `json:"color"` // Optional, defaults to DefaultGroupColor
}

// GroupUpdateRequest represents the data needed to update a group
type GroupUpdateRequest struct {
	Name  string `json:"name" validate:"omitempty,max=35"`
	Color string `json:"color"` // Optional, preserves existing if empty
}

// GroupResponse is the safe group data to return to clients
type GroupResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	Position  int       `json:"position"`
	IsDefault bool      `json:"is_default"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ToResponse converts Group to GroupResponse
func (g *Group) ToResponse() GroupResponse {
	return GroupResponse{
		ID:        g.ID,
		Name:      g.Name,
		Color:     g.Color,
		Position:  g.Position,
		IsDefault: g.IsDefault,
		CreatedAt: g.CreatedAt,
		UpdatedAt: g.UpdatedAt,
	}
}

// GroupPosition represents a group ID and its new position
type GroupPosition struct {
	ID       string `json:"id" validate:"required"`
	Position int    `json:"position" validate:"min=0"`
}

// GroupReorderRequest represents bulk position updates
type GroupReorderRequest struct {
	Groups []GroupPosition `json:"groups" validate:"required,dive"`
}
