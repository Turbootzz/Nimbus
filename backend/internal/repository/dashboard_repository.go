package repository

import (
	"cmp"
	"context"
	"database/sql"
	"fmt"

	"github.com/nimbus/backend/internal/models"
)

// DashboardRows is what a dashboard import writes. All IDs are set by the
// caller, so rows can point at each other before they exist.
type DashboardRows struct {
	Groups       []*models.Group
	Integrations []*models.Integration
	Services     []*models.Service
	Widgets      []*models.Widget
	// Preferences is nil to leave the user's preferences as they are
	Preferences *models.PreferencesUpdateRequest
}

// DashboardLimits are the per-user limits an import must stay under
type DashboardLimits struct {
	Integrations int
	Widgets      int
}

// DashboardState is what a user's dashboard holds
type DashboardState struct {
	Groups       int
	Services     int
	Widgets      int
	Integrations int
	// DefaultGroupID is "" when the user has no default group yet
	DefaultGroupID string
}

// DashboardApplied is the result of DashboardRepository.Apply
type DashboardApplied struct {
	// Before is the dashboard as it was
	Before DashboardState
	// RemovedIcons are the uploaded icon files of services a replace deleted
	RemovedIcons []string
}

// DashboardRepository writes a whole dashboard at once, for the import
type DashboardRepository struct {
	db *sql.DB
}

func NewDashboardRepository(db *sql.DB) *DashboardRepository {
	return &DashboardRepository{db: db}
}

// rowQuerier is satisfied by *sql.DB and *sql.Tx
type rowQuerier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// State counts the user's dashboard, for an import's dry run
func (r *DashboardRepository) State(ctx context.Context, userID string) (DashboardState, error) {
	return dashboardState(ctx, r.db, userID)
}

func dashboardState(ctx context.Context, db rowQuerier, userID string) (DashboardState, error) {
	query := `
		SELECT
			(SELECT COUNT(*) FROM groups WHERE user_id = $1),
			(SELECT COUNT(*) FROM services WHERE user_id = $1),
			(SELECT COUNT(*) FROM widgets WHERE user_id = $1),
			(SELECT COUNT(*) FROM integrations WHERE user_id = $1),
			COALESCE((SELECT CAST(g.id AS TEXT) FROM groups g JOIN boards b ON b.id = g.board_id
				WHERE g.user_id = $1 AND g.is_default = TRUE AND b.is_default = TRUE LIMIT 1), '')
	`
	var state DashboardState
	err := db.QueryRowContext(ctx, query, userID).Scan(
		&state.Groups, &state.Services, &state.Widgets, &state.Integrations, &state.DefaultGroupID,
	)
	if err != nil {
		return state, fmt.Errorf("failed to read dashboard state: %w", err)
	}
	return state, nil
}

// CheckLimits reports ErrIntegrationLimitReached or ErrWidgetLimitReached
// when the rows don't fit next to what the user keeps
func (rows *DashboardRows) CheckLimits(kept DashboardState, limits DashboardLimits) error {
	if kept.Integrations+len(rows.Integrations) > limits.Integrations {
		return ErrIntegrationLimitReached
	}
	if kept.Widgets+len(rows.Widgets) > limits.Widgets {
		return ErrWidgetLimitReached
	}
	return nil
}

// Apply writes rows for the user in one transaction: all of it, or nothing.
//
// With replace, the user's groups, services, widgets and integrations are
// deleted first; rows of other users are never touched. Without it, the rows
// are added: groups go after the existing ones, tile positions are moved
// behind the user's last tile, and a default group in rows becomes the
// default group the user already has. Apply changes rows accordingly.
// Integrations are stored without credentials.
func (r *DashboardRepository) Apply(ctx context.Context, userID string, replace bool, rows *DashboardRows, limits DashboardLimits) (*DashboardApplied, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := lockUser(ctx, tx, userID); err != nil {
		return nil, err
	}
	before, err := dashboardState(ctx, tx, userID)
	if err != nil {
		return nil, err
	}
	applied := &DashboardApplied{Before: before}

	kept := before
	if replace {
		if applied.RemovedIcons, err = wipeDashboard(ctx, tx, userID); err != nil {
			return nil, err
		}
		kept = DashboardState{}
	}
	if err := rows.CheckLimits(kept, limits); err != nil {
		return nil, err
	}

	groupOffset, tileOffset := 0, 0
	if !replace {
		if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(position), -1) + 1 FROM groups WHERE user_id = $1`, userID).Scan(&groupOffset); err != nil {
			return nil, fmt.Errorf("failed to get max group position: %w", err)
		}
		if tileOffset, err = nextTilePosition(ctx, tx, userID); err != nil {
			return nil, err
		}
		rows.useDefaultGroup(kept.DefaultGroupID)
	}

	if err := rows.own(ctx, tx, userID); err != nil {
		return nil, err
	}
	for i, group := range rows.Groups {
		group.Position = groupOffset + i
		if err := insertGroup(ctx, tx, group); err != nil {
			return nil, err
		}
	}
	for _, integration := range rows.Integrations {
		if err := insertIntegration(ctx, tx, integration); err != nil {
			return nil, err
		}
	}
	for _, service := range rows.Services {
		service.Position += tileOffset
		if err := insertService(ctx, tx, service); err != nil {
			return nil, err
		}
	}
	for _, widget := range rows.Widgets {
		widget.Position += tileOffset
		if err := insertWidget(ctx, tx, widget); err != nil {
			return nil, err
		}
	}
	if rows.Preferences != nil {
		if err := upsertPreferences(ctx, tx, userID, rows.Preferences); err != nil {
			return nil, fmt.Errorf("failed to save preferences: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit dashboard: %w", err)
	}
	return applied, nil
}

// own makes the user the owner of every row, and puts groups and tiles
// without a board on the user's default board
func (rows *DashboardRows) own(ctx context.Context, tx *sql.Tx, userID string) error {
	defaultBoard, err := defaultBoardID(ctx, tx, userID)
	if err != nil {
		return err
	}
	for _, group := range rows.Groups {
		group.UserID, group.BoardID = userID, cmp.Or(group.BoardID, defaultBoard)
	}
	for _, integration := range rows.Integrations {
		integration.UserID = userID
	}
	for _, service := range rows.Services {
		service.UserID, service.BoardID = userID, cmp.Or(service.BoardID, defaultBoard)
	}
	for _, widget := range rows.Widgets {
		widget.UserID, widget.BoardID = userID, cmp.Or(widget.BoardID, defaultBoard)
	}
	return nil
}

// useDefaultGroup drops the default group from rows when the user already
// has one (existingID), and points its tiles at that one: a user has exactly
// one default group
func (rows *DashboardRows) useDefaultGroup(existingID string) {
	if existingID == "" {
		return
	}
	for i, group := range rows.Groups {
		if !group.IsDefault {
			continue
		}
		for _, service := range rows.Services {
			if service.GroupID != nil && *service.GroupID == group.ID {
				service.GroupID = &existingID
			}
		}
		for _, widget := range rows.Widgets {
			if widget.GroupID != nil && *widget.GroupID == group.ID {
				widget.GroupID = &existingID
			}
		}
		rows.Groups = append(rows.Groups[:i:i], rows.Groups[i+1:]...)
		return
	}
}

// wipeDashboard deletes the user's tiles, integrations and groups, and
// returns the uploaded icon files the services used
func wipeDashboard(ctx context.Context, tx *sql.Tx, userID string) ([]string, error) {
	rows, err := tx.QueryContext(ctx,
		`SELECT icon_image_path FROM services WHERE user_id = $1 AND icon_type = $2 AND icon_image_path <> ''`,
		userID, models.IconTypeImageUpload)
	if err != nil {
		return nil, fmt.Errorf("failed to list service icons: %w", err)
	}
	var icons []string
	for rows.Next() {
		var icon string
		if err := rows.Scan(&icon); err != nil {
			rows.Close()
			return nil, fmt.Errorf("failed to scan service icon: %w", err)
		}
		icons = append(icons, icon)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate service icons: %w", err)
	}

	// Tiles first: they point at groups and integrations
	for _, table := range []string{"widgets", "services", "integrations", "groups"} {
		if _, err := tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE user_id = $1`, userID); err != nil {
			return nil, fmt.Errorf("failed to delete %s: %w", table, err)
		}
	}
	return icons, nil
}

func insertGroup(ctx context.Context, tx *sql.Tx, group *models.Group) error {
	query := `
		INSERT INTO groups (id, user_id, name, color, position, is_default, monitoring_enabled, tile_style, created_at, updated_at, board_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := tx.ExecContext(ctx, query,
		group.ID, group.UserID, group.Name, group.Color, group.Position,
		group.IsDefault, group.MonitoringEnabled, group.TileStyle, group.CreatedAt, group.UpdatedAt, group.BoardID,
	)
	if err != nil {
		return fmt.Errorf("failed to create group: %w", err)
	}
	return nil
}

// insertIntegration stores an integration without credentials
func insertIntegration(ctx context.Context, tx *sql.Tx, integration *models.Integration) error {
	query := `
		INSERT INTO integrations (id, user_id, kind, name, base_url, auth_type,
			verify_tls, options, refresh_seconds)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := tx.ExecContext(ctx, query,
		integration.ID, integration.UserID, integration.Kind, integration.Name,
		integration.BaseURL, integration.AuthType, integration.VerifyTLS,
		optionsJSON(integration.Options), integration.RefreshSeconds,
	)
	if err != nil {
		return fmt.Errorf("failed to create integration: %w", err)
	}
	return nil
}

func insertService(ctx context.Context, tx *sql.Tx, service *models.Service) error {
	query := `
		INSERT INTO services (id, user_id, name, url, icon, icon_type, icon_image_path, description, status,
			position, card_size, group_id, integration_id, monitoring_enabled, created_at, updated_at, board_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`
	_, err := tx.ExecContext(ctx, query,
		service.ID, service.UserID, service.Name, service.URL, service.Icon, service.IconType,
		service.IconImagePath, service.Description, service.Status, service.Position, service.CardSize,
		service.GroupID, service.IntegrationID, service.MonitoringEnabled, service.CreatedAt, service.UpdatedAt, service.BoardID,
	)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}
	return nil
}

func insertWidget(ctx context.Context, tx *sql.Tx, widget *models.Widget) error {
	query := `
		INSERT INTO widgets (id, user_id, type, title, group_id, integration_id,
			config, card_size, position, refresh_seconds, enabled, board_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := tx.ExecContext(ctx, query,
		widget.ID, widget.UserID, widget.Type, widget.Title, widget.GroupID, widget.IntegrationID,
		string(widget.Config), widget.CardSize, widget.Position, widget.RefreshSeconds, widget.Enabled, widget.BoardID,
	)
	if err != nil {
		return fmt.Errorf("failed to create widget: %w", err)
	}
	return nil
}
