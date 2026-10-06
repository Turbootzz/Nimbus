package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// NullableString represents a string that can be explicitly set to null
type NullableString struct {
	Value *string
	Set   bool // true if the field was present in JSON (even if null)
}

// UnmarshalJSON implements json.Unmarshaler to track presence
func (ns *NullableString) UnmarshalJSON(data []byte) error {
	ns.Set = true
	if string(data) == "null" {
		ns.Value = nil
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	ns.Value = &s
	return nil
}

// MarshalJSON implements json.Marshaler
func (ns NullableString) MarshalJSON() ([]byte, error) {
	if ns.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(*ns.Value)
}

// IsSet returns true if the field was present in the JSON payload
func (ns NullableString) IsSet() bool {
	return ns.Set
}

// GetValue returns the string value (can be nil)
func (ns NullableString) GetValue() *string {
	return ns.Value
}

// StatusStrip is the row of KPI chips at the top of the dashboard
type StatusStrip struct {
	Enabled bool         `json:"enabled"`
	Chips   []StatusChip `json:"chips" validate:"max=6,unique,dive"`
}

// StatusChip shows one KPI of an integration, or one value of a custom API
// widget (its kpi is the field label)
type StatusChip struct {
	Source string `json:"source" validate:"oneof=integration widget"`
	ID     string `json:"id" validate:"uuid"`
	KPI    string `json:"kpi" validate:"required,max=40"`
}

// Value stores the strip as JSON; an empty chip list is [] not null
func (s StatusStrip) Value() (driver.Value, error) {
	if s.Chips == nil {
		s.Chips = []StatusChip{}
	}
	raw, err := json.Marshal(s)
	return string(raw), err
}

// Scan reads the strip from its JSON column
func (s *StatusStrip) Scan(src any) error {
	var raw []byte
	switch v := src.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	case nil:
		*s = StatusStrip{Chips: []StatusChip{}}
		return nil
	default:
		return fmt.Errorf("unexpected status_strip type %T", src)
	}
	if err := json.Unmarshal(raw, s); err != nil {
		return err
	}
	if s.Chips == nil {
		s.Chips = []StatusChip{}
	}
	return nil
}

// UserPreferences represents a user's theme and UI preferences
type UserPreferences struct {
	ID                    string      `json:"id" db:"id"`
	UserID                string      `json:"user_id" db:"user_id"`
	ThemeMode             string      `json:"theme_mode" db:"theme_mode"`                 // "light", "dark", or "auto"
	ThemeBackground       *string     `json:"theme_background" db:"theme_background"`     // Background image URL or color
	ThemeAccentColor      *string     `json:"theme_accent_color" db:"theme_accent_color"` // Hex color like #3B82F6
	OpenInNewTab          bool        `json:"open_in_new_tab" db:"open_in_new_tab"`       // Whether to open services in new tab
	EnableCardResizing    bool        `json:"enable_card_resizing" db:"enable_card_resizing"`
	EnableServiceGrouping bool        `json:"enable_service_grouping" db:"enable_service_grouping"`
	CardScale             string      `json:"card_scale" db:"card_scale"`         // "small", "medium", or "large"
	ViewMode              string      `json:"view_mode" db:"view_mode"`           // "grid" or "list"
	WallpaperBlur         int         `json:"wallpaper_blur" db:"wallpaper_blur"` // px, 0-20
	WallpaperDim          int         `json:"wallpaper_dim" db:"wallpaper_dim"`   // %, 0-80
	CardOpacity           int         `json:"card_opacity" db:"card_opacity"`     // %, 0-100
	CardBlur              int         `json:"card_blur" db:"card_blur"`           // px, 0-40
	LayoutMode            string      `json:"layout_mode" db:"layout_mode"`       // "classic" or "canvas"
	StatusStrip           StatusStrip `json:"status_strip" db:"status_strip"`
	CreatedAt             time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time   `json:"updated_at" db:"updated_at"`
}

// PreferencesUpdateRequest represents the data needed to update preferences
type PreferencesUpdateRequest struct {
	ThemeMode             *string        `json:"theme_mode" validate:"omitempty,oneof=light dark auto"`
	ThemeBackground       NullableString `json:"theme_background"`        // Tracks presence separately from value
	ThemeAccentColor      NullableString `json:"theme_accent_color"`      // Tracks presence separately from value
	OpenInNewTab          *bool          `json:"open_in_new_tab"`         // Optional, defaults to true if not provided
	EnableCardResizing    *bool          `json:"enable_card_resizing"`    // Optional, defaults to true if not provided
	EnableServiceGrouping *bool          `json:"enable_service_grouping"` // Optional, defaults to true if not provided
	CardScale             *string        `json:"card_scale" validate:"omitempty,oneof=small medium large"`
	ViewMode              *string        `json:"view_mode" validate:"omitempty,oneof=grid list"`
	WallpaperBlur         *int           `json:"wallpaper_blur" validate:"omitempty,min=0,max=20"`
	WallpaperDim          *int           `json:"wallpaper_dim" validate:"omitempty,min=0,max=80"`
	CardOpacity           *int           `json:"card_opacity" validate:"omitempty,min=0,max=100"`
	CardBlur              *int           `json:"card_blur" validate:"omitempty,min=0,max=40"`
	LayoutMode            *string        `json:"layout_mode" validate:"omitempty,oneof=classic canvas"`
	StatusStrip           *StatusStrip   `json:"status_strip"`
}

// PreferencesResponse is the safe preferences data to return to clients
type PreferencesResponse struct {
	ThemeMode             string      `json:"theme_mode"`
	ThemeBackground       *string     `json:"theme_background,omitempty"`
	ThemeAccentColor      *string     `json:"theme_accent_color,omitempty"`
	OpenInNewTab          bool        `json:"open_in_new_tab"`
	EnableCardResizing    bool        `json:"enable_card_resizing"`
	EnableServiceGrouping bool        `json:"enable_service_grouping"`
	CardScale             string      `json:"card_scale"`
	ViewMode              string      `json:"view_mode"`
	WallpaperBlur         int         `json:"wallpaper_blur"`
	WallpaperDim          int         `json:"wallpaper_dim"`
	CardOpacity           int         `json:"card_opacity"`
	CardBlur              int         `json:"card_blur"`
	LayoutMode            string      `json:"layout_mode"`
	StatusStrip           StatusStrip `json:"status_strip"`
	UpdatedAt             time.Time   `json:"updated_at"`
}

// ToResponse converts UserPreferences to PreferencesResponse
func (p *UserPreferences) ToResponse() PreferencesResponse {
	return PreferencesResponse{
		ThemeMode:             p.ThemeMode,
		ThemeBackground:       p.ThemeBackground,
		ThemeAccentColor:      p.ThemeAccentColor,
		OpenInNewTab:          p.OpenInNewTab,
		EnableCardResizing:    p.EnableCardResizing,
		EnableServiceGrouping: p.EnableServiceGrouping,
		CardScale:             p.CardScale,
		ViewMode:              p.ViewMode,
		WallpaperBlur:         p.WallpaperBlur,
		WallpaperDim:          p.WallpaperDim,
		CardOpacity:           p.CardOpacity,
		CardBlur:              p.CardBlur,
		LayoutMode:            p.LayoutMode,
		StatusStrip:           p.StatusStrip,
		UpdatedAt:             p.UpdatedAt,
	}
}
