package models

import (
	"encoding/json"
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

// UserPreferences represents a user's theme and UI preferences
type UserPreferences struct {
	ID               string    `json:"id" db:"id"`
	UserID           string    `json:"user_id" db:"user_id"`
	ThemeMode        string    `json:"theme_mode" db:"theme_mode"`                 // "light" or "dark"
	ThemeBackground  *string   `json:"theme_background" db:"theme_background"`     // Background image URL or color
	ThemeAccentColor *string   `json:"theme_accent_color" db:"theme_accent_color"` // Hex color like #3B82F6
	OpenInNewTab     bool      `json:"open_in_new_tab" db:"open_in_new_tab"`       // Whether to open services in new tab
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

// PreferencesUpdateRequest represents the data needed to update preferences
type PreferencesUpdateRequest struct {
	ThemeMode        *string        `json:"theme_mode" validate:"omitempty,oneof=light dark"`
	ThemeBackground  NullableString `json:"theme_background"`   // Tracks presence separately from value
	ThemeAccentColor NullableString `json:"theme_accent_color"` // Tracks presence separately from value
	OpenInNewTab     *bool          `json:"open_in_new_tab"`    // Optional, defaults to true if not provided
}

// PreferencesResponse is the safe preferences data to return to clients
type PreferencesResponse struct {
	ThemeMode        string    `json:"theme_mode"`
	ThemeBackground  *string   `json:"theme_background,omitempty"`
	ThemeAccentColor *string   `json:"theme_accent_color,omitempty"`
	OpenInNewTab     bool      `json:"open_in_new_tab"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// ToResponse converts UserPreferences to PreferencesResponse
func (p *UserPreferences) ToResponse() PreferencesResponse {
	return PreferencesResponse{
		ThemeMode:        p.ThemeMode,
		ThemeBackground:  p.ThemeBackground,
		ThemeAccentColor: p.ThemeAccentColor,
		OpenInNewTab:     p.OpenInNewTab,
		UpdatedAt:        p.UpdatedAt,
	}
}
