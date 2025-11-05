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
// This method supports partial updates using atomic INSERT ... ON CONFLICT to avoid race conditions
func (r *PreferencesRepository) Upsert(ctx context.Context, userID string, preferences *models.PreferencesUpdateRequest) error {
	// Determine values for insert (with defaults for nil fields)
	insertThemeMode := "light"
	if preferences.ThemeMode != nil {
		insertThemeMode = *preferences.ThemeMode
	}

	// Atomic upsert using INSERT ... ON CONFLICT
	// For INSERT: use provided values or defaults
	// For UPDATE: use COALESCE to keep existing values when new value is NULL
	query := `
		INSERT INTO user_preferences (user_id, theme_mode, theme_background, theme_accent_color, open_in_new_tab, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (user_id) DO UPDATE SET
			theme_mode = COALESCE($6, user_preferences.theme_mode),
			theme_background = CASE
				WHEN $7::boolean THEN $3
				ELSE user_preferences.theme_background
			END,
			theme_accent_color = CASE
				WHEN $8::boolean THEN $4
				ELSE user_preferences.theme_accent_color
			END,
			open_in_new_tab = COALESCE($9, user_preferences.open_in_new_tab),
			updated_at = CURRENT_TIMESTAMP
	`

	// Flags to indicate if field was provided (even if NULL)
	hasBackground := preferences.ThemeBackground != nil
	hasAccentColor := preferences.ThemeAccentColor != nil

	_, err := r.db.ExecContext(
		ctx,
		query,
		userID,                       // $1
		insertThemeMode,              // $2 (for INSERT)
		preferences.ThemeBackground,  // $3 (for both INSERT and UPDATE)
		preferences.ThemeAccentColor, // $4 (for both INSERT and UPDATE)
		getOpenInNewTabValue(preferences.OpenInNewTab), // $5 (for INSERT)
		preferences.ThemeMode,                          // $6 (for UPDATE - COALESCE)
		hasBackground,                                  // $7 (flag: was background provided?)
		hasAccentColor,                                 // $8 (flag: was accent color provided?)
		preferences.OpenInNewTab,                       // $9 (for UPDATE - COALESCE)
	)

	return err
}
