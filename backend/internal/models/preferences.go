package models

import "time"

// UserPreferences represents a user's theme and UI preferences
type UserPreferences struct {
	ID               string    `json:"id" db:"id"`
	UserID           string    `json:"user_id" db:"user_id"`
	ThemeMode        string    `json:"theme_mode" db:"theme_mode"`                 // "light" or "dark"
	ThemeBackground  *string   `json:"theme_background" db:"theme_background"`     // Background image URL or color
	ThemeAccentColor *string   `json:"theme_accent_color" db:"theme_accent_color"` // Hex color like #3B82F6
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

// PreferencesUpdateRequest represents the data needed to update preferences
type PreferencesUpdateRequest struct {
	ThemeMode        string  `json:"theme_mode" validate:"required,oneof=light dark"`
	ThemeBackground  *string `json:"theme_background"`
	ThemeAccentColor *string `json:"theme_accent_color" validate:"omitempty,hexcolor"`
}

// PreferencesResponse is the safe preferences data to return to clients
type PreferencesResponse struct {
	ThemeMode        string    `json:"theme_mode"`
	ThemeBackground  *string   `json:"theme_background,omitempty"`
	ThemeAccentColor *string   `json:"theme_accent_color,omitempty"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// ToResponse converts UserPreferences to PreferencesResponse
func (p *UserPreferences) ToResponse() PreferencesResponse {
	return PreferencesResponse{
		ThemeMode:        p.ThemeMode,
		ThemeBackground:  p.ThemeBackground,
		ThemeAccentColor: p.ThemeAccentColor,
		UpdatedAt:        p.UpdatedAt,
	}
}
