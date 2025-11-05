package repository

import (
	"context"
	"database/sql"

	"github.com/nimbus/backend/internal/models"
)

type PreferencesRepository struct {
	db *sql.DB
}

func NewPreferencesRepository(db *sql.DB) *PreferencesRepository {
	return &PreferencesRepository{db: db}
}

// getOpenInNewTabValue returns the open_in_new_tab value, defaulting to true if nil
func getOpenInNewTabValue(value *bool) bool {
	if value != nil {
		return *value
	}
	return true
}

// GetByUserID retrieves preferences for a specific user
func (r *PreferencesRepository) GetByUserID(ctx context.Context, userID string) (*models.UserPreferences, error) {
	preferences := &models.UserPreferences{}
	query := `
		SELECT id, user_id, theme_mode, theme_background, theme_accent_color, open_in_new_tab, created_at, updated_at
		FROM user_preferences
		WHERE user_id = $1
	`

	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&preferences.ID,
		&preferences.UserID,
		&preferences.ThemeMode,
		&preferences.ThemeBackground,
		&preferences.ThemeAccentColor,
		&preferences.OpenInNewTab,
		&preferences.CreatedAt,
		&preferences.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}

	return preferences, err
}

// Create creates default preferences for a new user
func (r *PreferencesRepository) Create(ctx context.Context, preferences *models.UserPreferences) error {
	query := `
		INSERT INTO user_preferences (user_id, theme_mode, theme_background, theme_accent_color, open_in_new_tab, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		preferences.UserID,
		preferences.ThemeMode,
		preferences.ThemeBackground,
		preferences.ThemeAccentColor,
		preferences.OpenInNewTab,
		preferences.CreatedAt,
		preferences.UpdatedAt,
	).Scan(&preferences.ID)

	return err
}

// Update updates existing user preferences
func (r *PreferencesRepository) Update(ctx context.Context, userID string, preferences *models.PreferencesUpdateRequest) error {
	query := `
		UPDATE user_preferences
		SET theme_mode = $1, theme_background = $2, theme_accent_color = $3, open_in_new_tab = $4, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $5
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		preferences.ThemeMode,
		preferences.ThemeBackground,
		preferences.ThemeAccentColor,
		getOpenInNewTabValue(preferences.OpenInNewTab),
		userID,
	)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// Upsert creates or updates preferences (used when user might not have preferences yet)
// This method supports partial updates - only provided fields will be updated
func (r *PreferencesRepository) Upsert(ctx context.Context, userID string, preferences *models.PreferencesUpdateRequest) error {
	// First, check if preferences exist
	existing, err := r.GetByUserID(ctx, userID)

	if err == sql.ErrNoRows {
		// No existing preferences - create with defaults for missing fields
		themeMode := "light"
		if preferences.ThemeMode != nil {
			themeMode = *preferences.ThemeMode
		}

		query := `
			INSERT INTO user_preferences (user_id, theme_mode, theme_background, theme_accent_color, open_in_new_tab, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`

		_, err := r.db.ExecContext(
			ctx,
			query,
			userID,
			themeMode,
			preferences.ThemeBackground,
			preferences.ThemeAccentColor,
			getOpenInNewTabValue(preferences.OpenInNewTab),
		)

		return err
	}

	if err != nil {
		return err
	}

	// Preferences exist - update only provided fields
	themeMode := existing.ThemeMode
	if preferences.ThemeMode != nil {
		themeMode = *preferences.ThemeMode
	}

	themeBackground := existing.ThemeBackground
	if preferences.ThemeBackground != nil {
		themeBackground = preferences.ThemeBackground
	}

	themeAccentColor := existing.ThemeAccentColor
	if preferences.ThemeAccentColor != nil {
		themeAccentColor = preferences.ThemeAccentColor
	}

	openInNewTab := existing.OpenInNewTab
	if preferences.OpenInNewTab != nil {
		openInNewTab = *preferences.OpenInNewTab
	}

	query := `
		UPDATE user_preferences
		SET theme_mode = $1, theme_background = $2, theme_accent_color = $3, open_in_new_tab = $4, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $5
	`

	_, err = r.db.ExecContext(
		ctx,
		query,
		themeMode,
		themeBackground,
		themeAccentColor,
		openInNewTab,
		userID,
	)

	return err
}
