package repository

import (
	"cmp"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/widgets"
)

// DashboardRows is what a dashboard import writes. All IDs are set by the
// caller, so rows can point at each other before they exist.
type DashboardRows struct {
	// Boards are the boards the document lists: created when the user has
	// no board with that ID yet. Groups and tiles without a BoardID go to
	// the user's default board.
	Boards       []*models.Board
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

// DashboardState is what a user's dashboard holds: the groups, services and
// widgets on the boards an import works on (the default board, and the boards
// its document lists), and all the user's integrations
type DashboardState struct {
	Groups       int
	Services     int
	Widgets      int
	Integrations int
	// DefaultGroupID is the default group of the default board, "" when the
	// user has none yet
	DefaultGroupID string
}

// DashboardScope is a user's dashboard as an import of one document sees it
type DashboardScope struct {
	State DashboardState
	// Kept is what a replace leaves: the widgets on the user's other
	// boards, and the integrations tiles there use
	Kept DashboardState
	// DefaultBoardID is the user's default board
	DefaultBoardID string
	// DefaultGroups is the default group of every board that has one
	DefaultGroups map[string]string
}

// DashboardApplied is the result of DashboardRepository.Apply
type DashboardApplied struct {
	// Before is the dashboard as it was
	Before DashboardState
	// Scope is Before with what Apply left alone
	Scope *DashboardScope
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

// querier is rowQuerier plus what a scope needs to read and create
type querier interface {
	execQuerier
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// State counts the user's dashboard on the default board and the given
// boards, for an import's dry run
func (r *DashboardRepository) State(ctx context.Context, userID string, boards ...string) (DashboardState, error) {
	scope, err := r.Scope(ctx, userID, boards)
	if err != nil {
		return DashboardState{}, err
	}
	return scope.State, nil
}

// Scope describes the user's dashboard for an import that works on the
// default board and the given boards
func (r *DashboardRepository) Scope(ctx context.Context, userID string, boards []string) (*DashboardScope, error) {
	return dashboardScope(ctx, r.db, userID, boards)
}

// inBoards returns "board_id IN ($from, ...)" and its arguments
func inBoards(from int, boards []string) (string, []any) {
	marks, args := make([]string, len(boards)), make([]any, len(boards))
	for i, board := range boards {
		marks[i], args[i] = "$"+strconv.Itoa(from+i), board
	}
	return "board_id IN (" + strings.Join(marks, ", ") + ")", args
}

// splitIntegrations sorts the user's integrations into the ones tiles
// outside the boards in onBoards (see inBoards) use, by integration_id or
// in a widget's config (a calendar's sources), which a replace must keep,
// and the others
func splitIntegrations(ctx context.Context, db querier, userID, onBoards string, boardArgs []any) (kept, others []string, err error) {
	rows, err := db.QueryContext(ctx, `
		SELECT CAST(integration_id AS TEXT), CAST('' AS TEXT), CAST('' AS TEXT) FROM services
		WHERE user_id = $1 AND integration_id IS NOT NULL AND NOT `+onBoards+`
		UNION ALL
		SELECT COALESCE(CAST(integration_id AS TEXT), ''), type, CAST(config AS TEXT) FROM widgets
		WHERE user_id = $1 AND NOT `+onBoards,
		append([]any{userID}, boardArgs...)...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list tiles on other boards: %w", err)
	}
	used := map[string]bool{}
	for rows.Next() {
		var id, typ, config string
		if err := rows.Scan(&id, &typ, &config); err != nil {
			rows.Close()
			return nil, nil, fmt.Errorf("failed to scan tile: %w", err)
		}
		used[id] = true
		for _, linked := range widgets.ConfigIntegrationIDs(typ, json.RawMessage(config)) {
			used[linked] = true
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("failed to iterate tiles: %w", err)
	}

	rows, err = db.QueryContext(ctx, `SELECT CAST(id AS TEXT) FROM integrations WHERE user_id = $1`, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list integrations: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, nil, fmt.Errorf("failed to scan integration: %w", err)
		}
		if used[id] {
			kept = append(kept, id)
		} else {
			others = append(others, id)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("failed to iterate integrations: %w", err)
	}
	return kept, others, nil
}

func dashboardScope(ctx context.Context, db querier, userID string, boards []string) (*DashboardScope, error) {
	defaultBoard, err := defaultBoardID(ctx, db, userID)
	if err != nil {
		return nil, err
	}
	scope := &DashboardScope{DefaultBoardID: defaultBoard, DefaultGroups: map[string]string{}}
	onBoards, boardArgs := inBoards(2, append([]string{defaultBoard}, boards...))
	query := `
		SELECT
			(SELECT COUNT(*) FROM groups WHERE user_id = $1 AND ` + onBoards + `),
			(SELECT COUNT(*) FROM services WHERE user_id = $1 AND ` + onBoards + `),
			(SELECT COUNT(*) FROM widgets WHERE user_id = $1 AND ` + onBoards + `),
			(SELECT COUNT(*) FROM integrations WHERE user_id = $1),
			(SELECT COUNT(*) FROM widgets WHERE user_id = $1 AND NOT ` + onBoards + `)
	`
	state := &scope.State
	err = db.QueryRowContext(ctx, query, append([]any{userID}, boardArgs...)...).Scan(
		&state.Groups, &state.Services, &state.Widgets, &state.Integrations, &scope.Kept.Widgets,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to read dashboard state: %w", err)
	}
	kept, _, err := splitIntegrations(ctx, db, userID, onBoards, boardArgs)
	if err != nil {
		return nil, err
	}
	scope.Kept.Integrations = len(kept)

	rows, err := db.QueryContext(ctx,
		`SELECT CAST(board_id AS TEXT), CAST(id AS TEXT) FROM groups WHERE user_id = $1 AND is_default = TRUE`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list default groups: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var board, group string
		if err := rows.Scan(&board, &group); err != nil {
			return nil, fmt.Errorf("failed to scan default group: %w", err)
		}
		scope.DefaultGroups[board] = group
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate default groups: %w", err)
	}
	state.DefaultGroupID = scope.DefaultGroups[defaultBoard]
	return scope, nil
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
// It works on the user's default board and the boards in rows.Boards; their
// other boards, and the rows of other users, are never touched. With
// replace, the groups, services and widgets on those boards are deleted
// first, and the integrations no tile elsewhere uses. Without it, the rows
// are added: groups go after the existing ones, tile positions are moved
// behind the user's last tile, and a default group in rows becomes the
// default group its board already has. Apply changes rows accordingly.
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
	boards := make([]string, len(rows.Boards))
	for i, board := range rows.Boards {
		boards[i] = board.ID
	}
	scope, err := dashboardScope(ctx, tx, userID, boards)
	if err != nil {
		return nil, err
	}
	applied := &DashboardApplied{Before: scope.State, Scope: scope}

	kept := scope.State
	if replace {
		if applied.RemovedIcons, err = wipeDashboard(ctx, tx, userID, append(boards, scope.DefaultBoardID)); err != nil {
			return nil, err
		}
		kept = scope.Kept
	} else {
		kept.Widgets += scope.Kept.Widgets
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
	}
	rows.own(userID, scope.DefaultBoardID)
	if !replace {
		rows.useDefaultGroups(scope.DefaultGroups)
	}

	// A replace makes the boards what the document says; a merge only adds
	if err := upsertBoards(ctx, tx, rows, replace); err != nil {
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

// upsertBoards creates the boards of rows the user doesn't have yet, each
// with a default group unless rows brings one. With update, the boards the
// user has get the name and shared flag of rows; the default board is never
// shared. Call it after own, in the transaction that holds the user lock.
func upsertBoards(ctx context.Context, tx *sql.Tx, rows *DashboardRows, update bool) error {
	for _, board := range rows.Boards {
		var exists int
		err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM boards WHERE id = $1 AND user_id = $2`, board.ID, board.UserID).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to look up board: %w", err)
		}
		if exists > 0 {
			if !update {
				continue
			}
			_, err := tx.ExecContext(ctx, `
				UPDATE boards SET name = $1, is_shared = ($2 AND NOT is_default), updated_at = CURRENT_TIMESTAMP
				WHERE id = $3 AND user_id = $4`,
				board.Name, board.IsShared, board.ID, board.UserID)
			if err != nil {
				return fmt.Errorf("failed to update board: %w", err)
			}
			continue
		}
		if err := insertBoard(ctx, tx, board, false); err != nil {
			return err
		}
		hasDefault := slices.ContainsFunc(rows.Groups, func(g *models.Group) bool { return g.IsDefault && g.BoardID == board.ID })
		if !hasDefault {
			if err := insertDefaultGroup(ctx, tx, board.UserID, board.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

// own makes the user the owner of every row, and puts groups and tiles
// without a board on the user's default board
func (rows *DashboardRows) own(userID, defaultBoard string) {
	for _, board := range rows.Boards {
		board.UserID = userID
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
}

// useDefaultGroups drops the default groups from rows whose board already
// has one (existing, by board ID), and points their tiles at that one: a
// board has exactly one default group. Call it after own.
func (rows *DashboardRows) useDefaultGroups(existing map[string]string) {
	kept := make([]*models.Group, 0, len(rows.Groups))
	for _, group := range rows.Groups {
		existingID := existing[group.BoardID]
		if !group.IsDefault || existingID == "" {
			kept = append(kept, group)
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
	}
	rows.Groups = kept
}

// wipeDashboard deletes the user's tiles and groups on the given boards and
// the integrations no tile elsewhere uses, and returns the uploaded icon
// files the deleted services used that no service left uses
func wipeDashboard(ctx context.Context, tx *sql.Tx, userID string, boards []string) ([]string, error) {
	onBoards, boardArgs := inBoards(2, boards)
	args := append([]any{userID}, boardArgs...)
	_, unused, err := splitIntegrations(ctx, tx, userID, onBoards, boardArgs)
	if err != nil {
		return nil, err
	}
	rows, err := tx.QueryContext(ctx,
		`SELECT icon_image_path FROM services WHERE user_id = $1 AND `+onBoards+` AND icon_type = '`+models.IconTypeImageUpload+`' AND icon_image_path <> ''`,
		args...)
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
	for _, table := range []string{"widgets", "services"} {
		if _, err := tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE user_id = $1 AND `+onBoards, args...); err != nil {
			return nil, fmt.Errorf("failed to delete %s: %w", table, err)
		}
	}
	for _, id := range unused {
		if _, err := tx.ExecContext(ctx, `DELETE FROM integrations WHERE id = $1 AND user_id = $2`, id, userID); err != nil {
			return nil, fmt.Errorf("failed to delete integration: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM groups WHERE user_id = $1 AND `+onBoards, args...); err != nil {
		return nil, fmt.Errorf("failed to delete groups: %w", err)
	}
	return unusedIcons(ctx, tx, icons)
}

// unusedIcons leaves out the icon files a service still uses: services can
// share a file, also with other users
func unusedIcons(ctx context.Context, tx *sql.Tx, icons []string) ([]string, error) {
	var unused []string
	for _, icon := range icons {
		var inUse bool
		err := tx.QueryRowContext(ctx,
			`SELECT EXISTS (SELECT 1 FROM services WHERE icon_type = $1 AND icon_image_path = $2)`,
			models.IconTypeImageUpload, icon).Scan(&inUse)
		if err != nil {
			return nil, fmt.Errorf("failed to check service icon: %w", err)
		}
		if !inUse {
			unused = append(unused, icon)
		}
	}
	return unused, nil
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
