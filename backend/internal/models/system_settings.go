package models

import "time"

// SystemSetting represents a system-wide configuration setting
type SystemSetting struct {
	Key       string    `json:"key" db:"key"`
	Value     string    `json:"value" db:"value"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	UpdatedBy *string   `json:"updated_by" db:"updated_by"`
}

// UpdateSettingRequest represents a request to update a setting
type UpdateSettingRequest struct {
	Value string `json:"value" validate:"required"`
}
