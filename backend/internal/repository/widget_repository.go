package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/nimbus/backend/internal/models"
)

// MaxTilePosition is the highest position a reorder may set. It is far below
// the 32-bit INTEGER limit, so MAX(position)+1 always fits.
const MaxTilePosition = 1_000_000

// Sentinel errors for widget repository
var (
	ErrWidgetNotFound     = errors.New("widget not found")
	ErrWidgetLimitReached = errors.New("widget limit reached")
	ErrTileNotFound       = errors.New("tile not found")
)

const widgetColumns = `
	id, user_id, type, title, group_id, integration_id, config, card_size,
	position, refresh_seconds, enabled, created_at, updated_at
`

// tileReorderQueries maps a tile kind to its position update
var tileReorderQueries = map[string]string{
	models.TileKindService: `UPDATE services SET position = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2 AND user_id = $3`,
	models.TileKindWidget:  `UPDATE widgets SET position = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2 AND user_id = $3`,
}

type WidgetRepository struct {
	db *sql.DB
}

func NewWidgetRepository(db *sql.DB) *WidgetRepository {
	return &WidgetRepository{db: db}
}

func scanWidget(row rowScanner) (*models.Widget, error) {
	var widget models.Widget
	var config []byte
	err := row.Scan(
		&widget.ID,
		&widget.UserID,
		&widget.Type,
		&widget.Title,
		&widget.GroupID,
		&widget.IntegrationID,
		&config,
		&widget.CardSize,
		&widget.Position,
		&widget.RefreshSeconds,
		&widget.Enabled,
		&widget.CreatedAt,
		&widget.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	widget.Config = json.RawMessage(config)
	return &widget, nil
}

// lockUser serialises writes to one user's tiles. Same row lock as
// IntegrationRepository.Create.
func lockUser(ctx context.Context, tx *sql.Tx, userID string) error {
	if _, err := tx.ExecContext(ctx, `UPDATE users SET id = id WHERE id = $1`, userID); err != nil {
		return fmt.Errorf("failed to lock user: %w", err)
	}
	return nil
}

// nextTilePosition returns the position after the user's last service or
// widget. Services and widgets share one position space. Call it after
// lockUser, in the same transaction.
func nextTilePosition(ctx context.Context, tx *sql.Tx, userID string) (int, error) {
	maxPos, err := maxTilePosition(ctx, tx, userID)
	if err != nil {
		return 0, err
	}
	if maxPos < MaxTilePosition {
		return maxPos + 1, nil
	}
	// Older reorders accepted any position; renumber so the next one fits
	return compactTiles(ctx, tx, userID)
}

// maxTilePosition returns the user's highest tile position, or -1 without tiles
func maxTilePosition(ctx context.Context, tx *sql.Tx, userID string) (int, error) {
	query := `
		SELECT MAX(position) FROM (
			SELECT position FROM services WHERE user_id = $1
			UNION ALL
			SELECT position FROM widgets WHERE user_id = $1
		) AS tiles
	`
	var maxPos sql.NullInt64
	if err := tx.QueryRowContext(ctx, query, userID).Scan(&maxPos); err != nil {
		return 0, fmt.Errorf("failed to get max tile position: %w", err)
	}
	if !maxPos.Valid {
		return -1, nil
	}
	return int(maxPos.Int64), nil
}

// compactTiles renumbers the user's tiles 0..n-1 in their current order and
// returns n, the next free position
func compactTiles(ctx context.Context, tx *sql.Tx, userID string) (int, error) {
	query := `
		SELECT kind, id FROM (
			SELECT 'service' AS kind, id, position, created_at FROM services WHERE user_id = $1
			UNION ALL
			SELECT 'widget' AS kind, id, position, created_at FROM widgets WHERE user_id = $1
		) AS tiles
		ORDER BY position, kind, created_at
	`
	rows, err := tx.QueryContext(ctx, query, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to list tiles: %w", err)
	}
	var tiles []models.TilePosition
	for rows.Next() {
		var tile models.TilePosition
		if err := rows.Scan(&tile.Kind, &tile.ID); err != nil {
			rows.Close()
			return 0, fmt.Errorf("failed to scan tile: %w", err)
		}
		tile.Position = len(tiles)
		tiles = append(tiles, tile)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("failed to iterate tiles: %w", err)
	}

	if err := setTilePositions(ctx, tx, userID, tiles); err != nil {
		return 0, err
	}
	return len(tiles), nil
}

// setTilePositions updates positions within tx, failing with ErrTileNotFound
// when a tile does not exist or belongs to another user
func setTilePositions(ctx context.Context, tx *sql.Tx, userID string, tiles []models.TilePosition) error {
	for _, tile := range tiles {
		query, ok := tileReorderQueries[tile.Kind]
		if !ok {
			return fmt.Errorf("unknown tile kind %q", tile.Kind)
		}
		result, err := tx.ExecContext(ctx, query, tile.Position, tile.ID, userID)
		if err != nil {
			return fmt.Errorf("failed to update tile position: %w", err)
		}
		if err := requireAffected(result, ErrTileNotFound); err != nil {
			return err
		}
	}
	return nil
}

// Create stores a new widget at the end of the user's grid if the user is
// under maxPerUser.
func (r *WidgetRepository) Create(ctx context.Context, widget *models.Widget, maxPerUser int) error {
	if widget.ID == "" {
		widget.ID = uuid.New().String()
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := lockUser(ctx, tx, widget.UserID); err != nil {
		return err
	}

	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM widgets WHERE user_id = $1`, widget.UserID).Scan(&count); err != nil {
		return fmt.Errorf("failed to count widgets: %w", err)
	}
	if count >= maxPerUser {
		return ErrWidgetLimitReached
	}

	if widget.Position, err = nextTilePosition(ctx, tx, widget.UserID); err != nil {
		return err
	}

	insert := `
		INSERT INTO widgets (id, user_id, type, title, group_id, integration_id,
			config, card_size, position, refresh_seconds, enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING created_at, updated_at
	`
	err = tx.QueryRowContext(ctx, insert,
		widget.ID,
		widget.UserID,
		widget.Type,
		widget.Title,
		widget.GroupID,
		widget.IntegrationID,
		string(widget.Config),
		widget.CardSize,
		widget.Position,
		widget.RefreshSeconds,
		widget.Enabled,
	).Scan(&widget.CreatedAt, &widget.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create widget: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit widget: %w", err)
	}
	return nil
}

// GetByID returns a widget scoped to its owner
func (r *WidgetRepository) GetByID(ctx context.Context, id, userID string) (*models.Widget, error) {
	query := `SELECT ` + widgetColumns + ` FROM widgets WHERE id = $1 AND user_id = $2`

	widget, err := scanWidget(r.db.QueryRowContext(ctx, query, id, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrWidgetNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get widget: %w", err)
	}
	return widget, nil
}

// ListByUserID returns all widgets for a user in grid order
func (r *WidgetRepository) ListByUserID(ctx context.Context, userID string) ([]models.Widget, error) {
	query := `SELECT ` + widgetColumns + ` FROM widgets WHERE user_id = $1 ORDER BY position, created_at`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list widgets: %w", err)
	}
	defer rows.Close()

	widgets := []models.Widget{}
	for rows.Next() {
		widget, err := scanWidget(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan widget: %w", err)
		}
		widgets = append(widgets, *widget)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate widgets: %w", err)
	}
	return widgets, nil
}

// Update saves all editable fields. Type and position are never changed here.
func (r *WidgetRepository) Update(ctx context.Context, widget *models.Widget) error {
	query := `
		UPDATE widgets
		SET title = $1, group_id = $2, integration_id = $3, config = $4,
			card_size = $5, refresh_seconds = $6, enabled = $7,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $8 AND user_id = $9
		RETURNING updated_at
	`
	err := r.db.QueryRowContext(ctx, query,
		widget.Title,
		widget.GroupID,
		widget.IntegrationID,
		string(widget.Config),
		widget.CardSize,
		widget.RefreshSeconds,
		widget.Enabled,
		widget.ID,
		widget.UserID,
	).Scan(&widget.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrWidgetNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to update widget: %w", err)
	}
	return nil
}

// Delete removes a widget scoped to its owner
func (r *WidgetRepository) Delete(ctx context.Context, id, userID string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM widgets WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete widget: %w", err)
	}
	return requireAffected(result, ErrWidgetNotFound)
}

// ReorderTiles sets the positions of services and widgets in one
// transaction. It fails with ErrTileNotFound, and changes nothing, when a
// tile does not exist or belongs to another user.
func (r *WidgetRepository) ReorderTiles(ctx context.Context, userID string, tiles []models.TilePosition) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := lockUser(ctx, tx, userID); err != nil {
		return err
	}
	if err := setTilePositions(ctx, tx, userID, tiles); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit tile positions: %w", err)
	}
	return nil
}
