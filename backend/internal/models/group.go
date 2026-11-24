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
	TileStyleCards    = "cards"
	TileStyleChips    = "chips" // one-line chips, for quick-launch groups
)

// IsValidTileStyle checks if the given tile style exists
func IsValidTileStyle(style string) bool {
	return style == TileStyleCards || style == TileStyleChips
}

// hexColorRegex validates hex color format #RRGGBB
var hexColorRegex = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// IsValidHexColor checks if the given color is a valid hex color
func IsValidHexColor(color string) bool {
	return hexColorRegex.MatchString(color)
}

// Group represents a service group for organizing services
type Group struct {
	ID                string    `json:"id" db:"id"`
	UserID            string    `json:"user_id" db:"user_id"`
	BoardID           string    `json:"board_id" db:"board_id"` // "" on create means the default board
	Name              string    `json:"name" db:"name"`
	Color             string    `json:"color" db:"color"`
	Position          int       `json:"position" db:"position"`
	IsDefault         bool      `json:"is_default" db:"is_default"`
	MonitoringEnabled bool      `json:"monitoring_enabled" db:"monitoring_enabled"`
	TileStyle         string    `json:"tile_style" db:"tile_style"` // "cards" or "chips"
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

// GroupCreateRequest represents the data needed to create a new group
type GroupCreateRequest struct {
	Name              string `json:"name" validate:"required,max=35"`
	Color             string `json:"color"`              // Optional, defaults to DefaultGroupColor
	MonitoringEnabled *bool  `json:"monitoring_enabled"` // Optional, defaults to true
	TileStyle         string `json:"tile_style"`         // Optional, defaults to cards
	BoardID           string `json:"board_id"`           // Optional board; the default board when omitted
}

// GroupUpdateRequest represents the data needed to update a group
type GroupUpdateRequest struct {
	Name              string `json:"name" validate:"omitempty,max=35"`
	Color             string `json:"color"`              // Optional, preserves existing if empty
	MonitoringEnabled *bool  `json:"monitoring_enabled"` // Optional, preserves existing if nil
	TileStyle         string `json:"tile_style"`         // Optional, preserves existing if empty
}

// GroupResponse is the safe group data to return to clients
type GroupResponse struct {
	ID                string    `json:"id"`
	BoardID           string    `json:"board_id"`
	Name              string    `json:"name"`
	Color             string    `json:"color"`
	Position          int       `json:"position"`
	IsDefault         bool      `json:"is_default"`
	MonitoringEnabled bool      `json:"monitoring_enabled"`
	TileStyle         string    `json:"tile_style"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// ToResponse converts Group to GroupResponse
func (g *Group) ToResponse() GroupResponse {
	return GroupResponse{
		ID:                g.ID,
		BoardID:           g.BoardID,
		Name:              g.Name,
		Color:             g.Color,
		Position:          g.Position,
		IsDefault:         g.IsDefault,
		MonitoringEnabled: g.MonitoringEnabled,
		TileStyle:         g.TileStyle,
		CreatedAt:         g.CreatedAt,
		UpdatedAt:         g.UpdatedAt,
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
