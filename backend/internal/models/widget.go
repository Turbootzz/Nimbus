package models

import (
	"encoding/json"
	"time"
)

// Widget is a dashboard tile that is not a link, like a clock or a note.
// Config is type-specific JSON, validated by the type in internal/widgets.
type Widget struct {
	ID             string          `json:"id"`
	UserID         string          `json:"user_id"`
	Type           string          `json:"type"`
	Title          string          `json:"title"`
	GroupID        *string         `json:"group_id"`
	IntegrationID  *string         `json:"integration_id"`
	Config         json.RawMessage `json:"config"`
	CardSize       string          `json:"card_size"`
	Position       int             `json:"position"`
	RefreshSeconds int             `json:"refresh_seconds"`
	Enabled        bool            `json:"enabled"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// WidgetRequest is the body for create and update. On update, omitted fields
// keep their stored value, an empty group_id or integration_id clears it, and
// a given config replaces the stored one. Type can't change after create.
type WidgetRequest struct {
	Type           string          `json:"type"`
	Title          *string         `json:"title"`
	GroupID        *string         `json:"group_id"`
	IntegrationID  *string         `json:"integration_id"`
	Config         json.RawMessage `json:"config"`
	CardSize       *string         `json:"card_size"`
	RefreshSeconds *int            `json:"refresh_seconds"`
	Enabled        *bool           `json:"enabled"`
}

// Tile kinds share one position space per user
const (
	TileKindService = "service"
	TileKindWidget  = "widget"
)

// TilePosition is one entry of a dashboard reorder
type TilePosition struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Position int    `json:"position"`
}

// TileReorderRequest updates the positions of services and widgets at once
type TileReorderRequest struct {
	Tiles []TilePosition `json:"tiles"`
}
