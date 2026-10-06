package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/nimbus/backend/internal/models"
)

type PreferencesRepository struct {
	db *sql.DB
}

func NewPreferencesRepository(db *sql.DB) *PreferencesRepository {
	return &PreferencesRepository{db: db}
}

// GetByUserID retrieves preferences for a specific user
func (r *PreferencesRepository) GetByUserID(ctx context.Context, userID string) (*models.UserPreferences, error) {
	preferences := &models.UserPreferences{}
	query := `
		SELECT id, user_id, theme_mode, theme_background, theme_accent_color, open_in_new_tab, enable_card_resizing, enable_service_grouping, card_scale, view_mode,
			wallpaper_blur, wallpaper_dim, card_opacity, card_blur, layout_mode, status_strip, created_at, updated_at
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
		&preferences.EnableCardResizing,
		&preferences.EnableServiceGrouping,
		&preferences.CardScale,
		&preferences.ViewMode,
		&preferences.WallpaperBlur,
		&preferences.WallpaperDim,
		&preferences.CardOpacity,
		&preferences.CardBlur,
		&preferences.LayoutMode,
		&preferences.StatusStrip,
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
		INSERT INTO user_preferences (user_id, theme_mode, theme_background, theme_accent_color, open_in_new_tab, enable_card_resizing, enable_service_grouping, card_scale, view_mode, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
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
		preferences.EnableCardResizing,
		preferences.EnableServiceGrouping,
		preferences.CardScale,
		preferences.ViewMode,
		preferences.CreatedAt,
		preferences.UpdatedAt,
	).Scan(&preferences.ID)

	return err
}

// Upsert creates or updates preferences. Only the fields in the request are
// written: a new row gets the column defaults for the rest, an existing row
// keeps them. One INSERT ... ON CONFLICT, so concurrent first saves can't race.
func (r *PreferencesRepository) Upsert(ctx context.Context, userID string, p *models.PreferencesUpdateRequest) error {
	columns := []string{"user_id"}
	args := []any{userID}
	// add writes a column when the request has it; pointers become NULL or
	// their value
	add := func(column string, value any, given bool) {
		if given {
			columns = append(columns, column)
			args = append(args, value)
		}
	}
	add("theme_mode", p.ThemeMode, p.ThemeMode != nil)
	// Nullable fields: an explicit null clears the value, omitted keeps it
	add("theme_background", p.ThemeBackground.GetValue(), p.ThemeBackground.IsSet())
	add("theme_accent_color", p.ThemeAccentColor.GetValue(), p.ThemeAccentColor.IsSet())
	add("open_in_new_tab", p.OpenInNewTab, p.OpenInNewTab != nil)
	add("enable_card_resizing", p.EnableCardResizing, p.EnableCardResizing != nil)
	add("enable_service_grouping", p.EnableServiceGrouping, p.EnableServiceGrouping != nil)
	add("card_scale", p.CardScale, p.CardScale != nil)
	add("view_mode", p.ViewMode, p.ViewMode != nil)
	add("wallpaper_blur", p.WallpaperBlur, p.WallpaperBlur != nil)
	add("wallpaper_dim", p.WallpaperDim, p.WallpaperDim != nil)
	add("card_opacity", p.CardOpacity, p.CardOpacity != nil)
	add("card_blur", p.CardBlur, p.CardBlur != nil)
	add("layout_mode", p.LayoutMode, p.LayoutMode != nil)
	if p.StatusStrip != nil {
		add("status_strip", *p.StatusStrip, true)
	}

	placeholders := make([]string, len(columns))
	updates := []string{"updated_at = CURRENT_TIMESTAMP"}
	for i, column := range columns {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		if column != "user_id" {
			updates = append(updates, column+" = EXCLUDED."+column)
		}
	}
	query := fmt.Sprintf(`INSERT INTO user_preferences (%s, created_at, updated_at)
		VALUES (%s, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (user_id) DO UPDATE SET %s`,
		strings.Join(columns, ", "), strings.Join(placeholders, ", "), strings.Join(updates, ", "))

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// WallpapersInUse returns the uploaded wallpapers that preferences point at
func (r *PreferencesRepository) WallpapersInUse(ctx context.Context) (map[string]bool, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT theme_background FROM user_preferences WHERE theme_background LIKE '/uploads/wallpapers/%'`)
	if err != nil {
		return nil, fmt.Errorf("failed to list wallpapers: %w", err)
	}
	defer rows.Close()
	inUse := map[string]bool{}
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, fmt.Errorf("failed to scan wallpaper: %w", err)
		}
		inUse[path] = true
	}
	return inUse, rows.Err()
}

// StripIntegrations returns the integrations that enabled status strips
// show, as integration ID to the IDs of the users whose strips show it.
// Only the owner's strip counts, so the caller checks which one that is.
func (r *PreferencesRepository) StripIntegrations(ctx context.Context) (map[string][]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT user_id, status_strip FROM user_preferences`)
	if err != nil {
		return nil, fmt.Errorf("failed to list status strips: %w", err)
	}
	defer rows.Close()
	used := map[string][]string{}
	for rows.Next() {
		var userID string
		var strip models.StatusStrip
		if err := rows.Scan(&userID, &strip); err != nil {
			return nil, fmt.Errorf("failed to scan status strip: %w", err)
		}
		for _, chip := range strip.Chips {
			if strip.Enabled && chip.Source == "integration" {
				used[chip.ID] = append(used[chip.ID], userID)
			}
		}
	}
	return used, rows.Err()
}
