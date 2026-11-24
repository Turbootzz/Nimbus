package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/nimbus/backend/internal/models"
)

// Sentinel errors for board repository
var (
	ErrBoardNotFound            = errors.New("board not found")
	ErrBoardSlugTaken           = errors.New("board slug is taken")
	ErrBoardLimitReached        = errors.New("board limit reached")
	ErrCannotDeleteDefaultBoard = errors.New("cannot delete default board")
)

const boardColumns = `
	b.id, b.user_id, b.name, b.slug, b.position, b.is_default, b.is_shared, b.visibility,
	b.created_at, b.updated_at
`

// sharedByAdmin is true for a board (b) every user may see: shared, by an
// owner who is still an admin. A demoted owner's board is private again.
const sharedByAdmin = `(b.is_shared = TRUE AND EXISTS (SELECT 1 FROM users u WHERE u.id = b.user_id AND u.role = 'admin'))`

type BoardRepository struct {
	db *sql.DB
}

func NewBoardRepository(db *sql.DB) *BoardRepository {
	return &BoardRepository{db: db}
}

// execQuerier is satisfied by *sql.DB and *sql.Tx
type execQuerier interface {
	rowQuerier
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func scanBoard(row rowScanner) (*models.Board, error) {
	var board models.Board
	err := row.Scan(
		&board.ID, &board.UserID, &board.Name, &board.Slug, &board.Position,
		&board.IsDefault, &board.IsShared, &board.Visibility, &board.CreatedAt, &board.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrBoardNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read board: %w", err)
	}
	return &board, nil
}

// defaultBoardID returns the ID of the user's default board and creates the
// board when the user has none yet (a user registered after the migration).
// Safe to race: the unique index lets one insert win.
func defaultBoardID(ctx context.Context, q execQuerier, userID string) (string, error) {
	const find = `SELECT CAST(id AS TEXT) FROM boards WHERE user_id = $1 AND is_default = TRUE`
	var id string
	err := q.QueryRowContext(ctx, find, userID).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("failed to get default board: %w", err)
	}
	_, err = q.ExecContext(ctx, `
		INSERT INTO boards (id, user_id, name, slug, position, is_default)
		VALUES ($1, $2, $3, $4, 0, TRUE)
		ON CONFLICT DO NOTHING`,
		uuid.New().String(), userID, models.DefaultBoardName, models.DefaultBoardSlug)
	if err != nil {
		return "", fmt.Errorf("failed to create default board: %w", err)
	}
	if err := q.QueryRowContext(ctx, find, userID).Scan(&id); err != nil {
		return "", fmt.Errorf("failed to get default board: %w", err)
	}
	return id, nil
}

// boardOr returns boardID, or the user's default board when it is empty
func boardOr(ctx context.Context, q execQuerier, userID, boardID string) (string, error) {
	if boardID != "" {
		return boardID, nil
	}
	return defaultBoardID(ctx, q, userID)
}

// EnsureDefault returns the user's default board, creating it when missing
func (r *BoardRepository) EnsureDefault(ctx context.Context, userID string) (*models.Board, error) {
	id, err := defaultBoardID(ctx, r.db, userID)
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

// GetByID returns a board whoever owns it; callers check access
func (r *BoardRepository) GetByID(ctx context.Context, id string) (*models.Board, error) {
	return scanBoard(r.db.QueryRowContext(ctx, `SELECT `+boardColumns+` FROM boards b WHERE b.id = $1`, id))
}

// GetVisible returns a board the user owns or that is shared with everyone
func (r *BoardRepository) GetVisible(ctx context.Context, id, userID string) (*models.Board, error) {
	query := `SELECT ` + boardColumns + ` FROM boards b WHERE b.id = $1 AND (b.user_id = $2 OR ` + sharedByAdmin + `)`
	return scanBoard(r.db.QueryRowContext(ctx, query, id, userID))
}

func (r *BoardRepository) list(ctx context.Context, query string, args ...any) ([]*models.Board, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list boards: %w", err)
	}
	defer rows.Close()

	boards := []*models.Board{}
	for rows.Next() {
		board, err := scanBoard(rows)
		if err != nil {
			return nil, err
		}
		boards = append(boards, board)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate boards: %w", err)
	}
	return boards, nil
}

// ListByUserID returns the user's own boards in their order
func (r *BoardRepository) ListByUserID(ctx context.Context, userID string) ([]*models.Board, error) {
	query := `SELECT ` + boardColumns + ` FROM boards b WHERE b.user_id = $1 ORDER BY b.position, b.created_at, b.id`
	return r.list(ctx, query, userID)
}

// ListVisible returns the user's own boards in their order, then the boards
// others share with everyone
func (r *BoardRepository) ListVisible(ctx context.Context, userID string) ([]*models.Board, error) {
	query := `
		SELECT ` + boardColumns + ` FROM boards b
		WHERE b.user_id = $1 OR ` + sharedByAdmin + `
		ORDER BY CASE WHEN b.user_id = $1 THEN 0 ELSE 1 END, b.position, b.created_at, b.id
	`
	return r.list(ctx, query, userID)
}

// slugTaken reports whether another board of the user has the slug
func slugTaken(ctx context.Context, q rowQuerier, userID, slug, exceptID string) (bool, error) {
	var count int
	err := q.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM boards WHERE user_id = $1 AND slug = $2 AND CAST(id AS TEXT) <> $3`,
		userID, slug, exceptID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check board slug: %w", err)
	}
	return count > 0, nil
}

// Create stores a new board after the user's last one, with its default
// group. With autoSlug a taken slug gets a number (media, media-2, ...);
// without it Create fails with ErrBoardSlugTaken.
func (r *BoardRepository) Create(ctx context.Context, board *models.Board, autoSlug bool) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := lockUser(ctx, tx, board.UserID); err != nil {
		return err
	}
	if _, err := defaultBoardID(ctx, tx, board.UserID); err != nil {
		return err
	}
	if err := insertBoard(ctx, tx, board, autoSlug); err != nil {
		return err
	}
	if err := insertDefaultGroup(ctx, tx, board.UserID, board.ID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit board: %w", err)
	}
	return nil
}

// insertBoard adds a board in tx, after the user's last one. The caller
// holds the user lock. It sets the board's ID (when empty), position and,
// with autoSlug, a free slug.
func insertBoard(ctx context.Context, tx *sql.Tx, board *models.Board, autoSlug bool) error {
	var count, maxPos int
	err := tx.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(MAX(position), -1) FROM boards WHERE user_id = $1`, board.UserID).
		Scan(&count, &maxPos)
	if err != nil {
		return fmt.Errorf("failed to count boards: %w", err)
	}
	if count >= models.MaxBoardsPerUser {
		return ErrBoardLimitReached
	}

	base := board.Slug
	for n := 2; ; n++ {
		taken, err := slugTaken(ctx, tx, board.UserID, board.Slug, "")
		if err != nil {
			return err
		}
		if !taken {
			break
		}
		if !autoSlug {
			return ErrBoardSlugTaken
		}
		board.Slug = base + "-" + strconv.Itoa(n)
	}

	if board.ID == "" {
		board.ID = uuid.New().String()
	}
	board.Position = maxPos + 1
	board.Visibility = models.BoardVisibilityAll
	err = tx.QueryRowContext(ctx, `
		INSERT INTO boards (id, user_id, name, slug, position, is_default, is_shared, visibility)
		VALUES ($1, $2, $3, $4, $5, FALSE, $6, $7)
		RETURNING created_at, updated_at`,
		board.ID, board.UserID, board.Name, board.Slug, board.Position, board.IsShared, board.Visibility,
	).Scan(&board.CreatedAt, &board.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create board: %w", err)
	}
	return nil
}

// Update saves the name, slug and shared flag of the user's board
func (r *BoardRepository) Update(ctx context.Context, board *models.Board) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := lockUser(ctx, tx, board.UserID); err != nil {
		return err
	}
	taken, err := slugTaken(ctx, tx, board.UserID, board.Slug, board.ID)
	if err != nil {
		return err
	}
	if taken {
		return ErrBoardSlugTaken
	}
	result, err := tx.ExecContext(ctx, `
		UPDATE boards SET name = $1, slug = $2, is_shared = $3, updated_at = CURRENT_TIMESTAMP
		WHERE id = $4 AND user_id = $5`,
		board.Name, board.Slug, board.IsShared, board.ID, board.UserID)
	if err != nil {
		return fmt.Errorf("failed to update board: %w", err)
	}
	if err := requireAffected(result, ErrBoardNotFound); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit board: %w", err)
	}
	return nil
}

// Reorder puts the user's boards in the order of ids. It fails with
// ErrBoardNotFound, and changes nothing, when one is not the user's.
func (r *BoardRepository) Reorder(ctx context.Context, userID string, ids []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for position, id := range ids {
		result, err := tx.ExecContext(ctx,
			`UPDATE boards SET position = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2 AND user_id = $3`,
			position, id, userID)
		if err != nil {
			return fmt.Errorf("failed to update board position: %w", err)
		}
		if err := requireAffected(result, ErrBoardNotFound); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit board positions: %w", err)
	}
	return nil
}

// Delete removes the user's board; the default board can't be deleted. With
// moveTo, its services, widgets and groups go to that board of the user
// first (tiles of its default group become ungrouped there); without it
// they are deleted with the board.
func (r *BoardRepository) Delete(ctx context.Context, id, userID, moveTo string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := lockUser(ctx, tx, userID); err != nil {
		return err
	}
	owned := `SELECT ` + boardColumns + ` FROM boards b WHERE b.id = $1 AND b.user_id = $2`
	board, err := scanBoard(tx.QueryRowContext(ctx, owned, id, userID))
	if err != nil {
		return err
	}
	if board.IsDefault {
		return ErrCannotDeleteDefaultBoard
	}

	exec := func(what, query string, args ...any) error {
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("failed to %s: %w", what, err)
		}
		return nil
	}
	if moveTo != "" {
		if _, err := scanBoard(tx.QueryRowContext(ctx, owned, moveTo, userID)); err != nil {
			return err
		}
		var offset int
		err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(position), -1) + 1 FROM groups WHERE board_id = $1`, moveTo).Scan(&offset)
		if err != nil {
			return fmt.Errorf("failed to get max group position: %w", err)
		}
		// What was in the board's default group is ungrouped on the other board
		inDefaultGroup := ` WHERE board_id = $1 AND group_id IN (SELECT id FROM groups WHERE board_id = $1 AND is_default = TRUE)`
		moves := []struct {
			what, query string
			args        []any
		}{
			{"ungroup services", `UPDATE services SET group_id = NULL` + inDefaultGroup, []any{id}},
			{"ungroup widgets", `UPDATE widgets SET group_id = NULL` + inDefaultGroup, []any{id}},
			{"move groups", `UPDATE groups SET board_id = $1, position = position + $2 WHERE board_id = $3 AND is_default = FALSE`, []any{moveTo, offset, id}},
			{"move services", `UPDATE services SET board_id = $1 WHERE board_id = $2`, []any{moveTo, id}},
			{"move widgets", `UPDATE widgets SET board_id = $1 WHERE board_id = $2`, []any{moveTo, id}},
		}
		for _, move := range moves {
			if err := exec(move.what, move.query, move.args...); err != nil {
				return err
			}
		}
	}
	// Explicit, tiles first: they point at groups (and SQLite tests have no cascade)
	for _, table := range []string{"widgets", "services", "groups"} {
		if err := exec("delete "+table, `DELETE FROM `+table+` WHERE board_id = $1 AND user_id = $2`, id, userID); err != nil {
			return err
		}
	}
	if err := exec("delete board", `DELETE FROM boards WHERE id = $1 AND user_id = $2`, id, userID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit board delete: %w", err)
	}
	return nil
}

// MoveTile puts a service or widget of the user on another of their boards:
// ungrouped (groups belong to a board) and after the last tile. The caller
// checks that the board is the user's. It returns the tile's new position.
func (r *BoardRepository) MoveTile(ctx context.Context, userID, kind, id, boardID string) (int, error) {
	table, ok := map[string]string{models.TileKindService: "services", models.TileKindWidget: "widgets"}[kind]
	if !ok {
		return 0, fmt.Errorf("unknown tile kind %q", kind)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := lockUser(ctx, tx, userID); err != nil {
		return 0, err
	}
	position, err := nextTilePosition(ctx, tx, userID)
	if err != nil {
		return 0, err
	}
	result, err := tx.ExecContext(ctx,
		`UPDATE `+table+` SET board_id = $1, group_id = NULL, position = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $3 AND user_id = $4`,
		boardID, position, id, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to move tile: %w", err)
	}
	if err := requireAffected(result, ErrTileNotFound); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit tile move: %w", err)
	}
	return position, nil
}

// ViewerKeys lists what a viewer of the board may follow live: its services
// ("service:<id>"), its widgets and the integrations its services show
// (snapshot keys, "widget:<id>" and "integration:<id>")
func (r *BoardRepository) ViewerKeys(ctx context.Context, boardID string) (map[string]bool, error) {
	query := `
		SELECT 'service', CAST(id AS TEXT) FROM services WHERE board_id = $1
		UNION ALL
		SELECT 'widget', CAST(id AS TEXT) FROM widgets WHERE board_id = $1
		UNION ALL
		SELECT 'integration', CAST(integration_id AS TEXT) FROM services WHERE board_id = $1 AND integration_id IS NOT NULL
	`
	rows, err := r.db.QueryContext(ctx, query, boardID)
	if err != nil {
		return nil, fmt.Errorf("failed to list board tiles: %w", err)
	}
	defer rows.Close()

	keys := map[string]bool{}
	for rows.Next() {
		var kind, id string
		if err := rows.Scan(&kind, &id); err != nil {
			return nil, fmt.Errorf("failed to scan board tile: %w", err)
		}
		keys[kind+":"+id] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate board tiles: %w", err)
	}
	return keys, nil
}

// Integrations describes the integrations the board's services show, without
// anything that reaches them: no name, URL, credentials or option values
func (r *BoardRepository) Integrations(ctx context.Context, boardID string) ([]models.BoardIntegration, error) {
	query := `
		SELECT DISTINCT CAST(i.id AS TEXT), i.kind, CAST(i.options AS TEXT)
		FROM integrations i
		JOIN services s ON s.integration_id = i.id
		WHERE s.board_id = $1
		ORDER BY 1
	`
	rows, err := r.db.QueryContext(ctx, query, boardID)
	if err != nil {
		return nil, fmt.Errorf("failed to list board integrations: %w", err)
	}
	defer rows.Close()

	list := []models.BoardIntegration{}
	for rows.Next() {
		var options sql.NullString
		item := models.BoardIntegration{Options: map[string]bool{}}
		if err := rows.Scan(&item.ID, &item.Kind, &options); err != nil {
			return nil, fmt.Errorf("failed to scan board integration: %w", err)
		}
		var values map[string]any
		// Options that don't parse simply count as not set
		_ = json.Unmarshal([]byte(options.String), &values)
		for key, value := range values {
			if value != nil && value != "" && value != false && value != float64(0) {
				item.Options[key] = true
			}
		}
		list = append(list, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate board integrations: %w", err)
	}
	return list, nil
}

// insertDefaultGroup gives a new board its default group
func insertDefaultGroup(ctx context.Context, tx *sql.Tx, userID, boardID string) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO groups (id, user_id, board_id, name, color, position, is_default, monitoring_enabled, tile_style, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, 0, TRUE, TRUE, $6, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		uuid.New().String(), userID, boardID, models.DefaultGroupName, models.DefaultGroupColor, models.TileStyleCards)
	if err != nil {
		return fmt.Errorf("failed to create default group: %w", err)
	}
	return nil
}
