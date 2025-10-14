package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/nimbus/backend/internal/models"
)

type SettingsRepository struct {
	db *sql.DB
}

func NewSettingsRepository(db *sql.DB) *SettingsRepository {
	return &SettingsRepository{db: db}
}

// Get retrieves a setting by key
func (r *SettingsRepository) Get(ctx context.Context, key string) (*models.SystemSetting, error) {
	query := `
		SELECT key, value, updated_at, updated_by
		FROM system_settings
		WHERE key = $1
	`

	setting := &models.SystemSetting{}
	err := r.db.QueryRowContext(ctx, query, key).Scan(
		&setting.Key,
		&setting.Value,
		&setting.UpdatedAt,
		&setting.UpdatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("setting not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get setting: %w", err)
	}

	return setting, nil
}

// GetAll retrieves all settings
func (r *SettingsRepository) GetAll(ctx context.Context) ([]*models.SystemSetting, error) {
	query := `
		SELECT key, value, updated_at, updated_by
		FROM system_settings
		ORDER BY key
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}
	defer rows.Close()

	var settings []*models.SystemSetting
	for rows.Next() {
		setting := &models.SystemSetting{}
		err := rows.Scan(
			&setting.Key,
			&setting.Value,
			&setting.UpdatedAt,
			&setting.UpdatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan setting: %w", err)
		}
		settings = append(settings, setting)
	}

	return settings, nil
}

// Update updates or creates a setting
func (r *SettingsRepository) Update(ctx context.Context, key, value string, updatedBy *string) error {
	query := `
		INSERT INTO system_settings (key, value, updated_at, updated_by)
		VALUES ($1, $2, CURRENT_TIMESTAMP, $3)
		ON CONFLICT (key)
		DO UPDATE SET
			value = EXCLUDED.value,
			updated_at = CURRENT_TIMESTAMP,
			updated_by = EXCLUDED.updated_by
	`

	_, err := r.db.ExecContext(ctx, query, key, value, updatedBy)
	if err != nil {
		return fmt.Errorf("failed to update setting: %w", err)
	}

	return nil
}

// IsPublicRegistrationEnabled checks if public registration is enabled
func (r *SettingsRepository) IsPublicRegistrationEnabled(ctx context.Context) (bool, error) {
	setting, err := r.Get(ctx, "public_registration_enabled")
	if err != nil {
		// Default to false if setting doesn't exist
		return false, nil
	}

	return setting.Value == "true", nil
}
