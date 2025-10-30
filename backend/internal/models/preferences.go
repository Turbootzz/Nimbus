package models

import "time"

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
	ThemeMode        string  `json:"theme_mode" validate:"required,oneof=light dark"`
	ThemeBackground  *string `json:"theme_background" validate:"omitempty,httpurl"` // Custom validator - only allows http(s)
	ThemeAccentColor *string `json:"theme_accent_color" validate:"omitempty,hexcolor"`
	OpenInNewTab     *bool   `json:"open_in_new_tab"` // Optional, defaults to true if not provided
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
