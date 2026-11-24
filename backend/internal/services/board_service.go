package services

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/nimbus/backend/internal/integrations"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
)

// BoardAccess is a board a user may see. Board is nil when no board service
// is wired (a handler under test): the user then has one dashboard, theirs.
type BoardAccess struct {
	Board *models.Board
	// CanEdit is true for the owner. Everyone else only reads: a shared
	// board is edited by the admin who owns it.
	CanEdit bool
}

// ID is the board's ID, or "" without a board service
func (a *BoardAccess) ID() string {
	if a.Board == nil {
		return ""
	}
	return a.Board.ID
}

// BoardService manages boards and decides who may see one. Every handler
// and tool that takes a board asks Resolve, so the rule lives here only.
type BoardService struct {
	repo *repository.BoardRepository
}

func NewBoardService(repo *repository.BoardRepository) *BoardService {
	return &BoardService{repo: repo}
}

// Resolve returns the board a request means: boardID, or the user's default
// board for "". The user must own it, or it must be shared by an admin;
// anything else is repository.ErrBoardNotFound, also a board that exists.
func (s *BoardService) Resolve(ctx context.Context, userID, boardID string) (*BoardAccess, error) {
	boardID = strings.TrimSpace(boardID)
	if s == nil {
		if boardID != "" {
			return nil, repository.ErrBoardNotFound
		}
		return &BoardAccess{CanEdit: true}, nil
	}
	if boardID == "" {
		board, err := s.repo.EnsureDefault(ctx, userID)
		if err != nil {
			return nil, err
		}
		return &BoardAccess{Board: board, CanEdit: true}, nil
	}
	parsed, err := uuid.Parse(boardID)
	if err != nil {
		return nil, repository.ErrBoardNotFound
	}
	board, err := s.repo.GetVisible(ctx, parsed.String(), userID)
	if err != nil {
		return nil, err
	}
	return &BoardAccess{Board: board, CanEdit: board.UserID == userID}, nil
}

// ResolveOwned is Resolve for a write: only the user's own boards
func (s *BoardService) ResolveOwned(ctx context.Context, userID, boardID string) (*BoardAccess, error) {
	access, err := s.Resolve(ctx, userID, boardID)
	if err != nil {
		return nil, err
	}
	if !access.CanEdit {
		return nil, repository.ErrBoardNotFound
	}
	return access, nil
}

// ResolveRef is Resolve by ID, slug or name, for callers a person types a
// board to (the MCP tools). The user's own boards are looked at first.
func (s *BoardService) ResolveRef(ctx context.Context, userID, ref string) (*BoardAccess, error) {
	ref = strings.TrimSpace(ref)
	if s == nil || ref == "" {
		return s.Resolve(ctx, userID, ref)
	}
	if _, err := uuid.Parse(ref); err == nil {
		return s.Resolve(ctx, userID, ref)
	}
	boards, err := s.repo.ListVisible(ctx, userID) // own boards come first
	if err != nil {
		return nil, err
	}
	for _, matches := range []func(*models.Board) bool{
		func(b *models.Board) bool { return b.Slug == strings.ToLower(ref) },
		func(b *models.Board) bool { return strings.EqualFold(b.Name, ref) },
	} {
		for _, board := range boards {
			if matches(board) {
				return &BoardAccess{Board: board, CanEdit: board.UserID == userID}, nil
			}
		}
	}
	return nil, repository.ErrBoardNotFound
}

// CanRead reports whether the user may read a tile of ownerID on boardID:
// their own, or one on a board they can see
func (s *BoardService) CanRead(ctx context.Context, userID, ownerID, boardID string) bool {
	if ownerID == userID {
		return true
	}
	if s == nil || boardID == "" {
		return false
	}
	// The tile must also be the board owner's, not only on a visible board
	access, err := s.Resolve(ctx, userID, boardID)
	return err == nil && access.Board.UserID == ownerID
}

// List returns the boards the user sees: their own, then shared ones
func (s *BoardService) List(ctx context.Context, userID string) ([]*models.Board, error) {
	if _, err := s.repo.EnsureDefault(ctx, userID); err != nil {
		return nil, err
	}
	return s.repo.ListVisible(ctx, userID)
}

// ViewerKeys lists what a viewer of the board may follow live
func (s *BoardService) ViewerKeys(ctx context.Context, boardID string) (map[string]bool, error) {
	return s.repo.ViewerKeys(ctx, boardID)
}

// BoardIntegration adds the KPIs of its kind: a viewer may not be allowed to
// set up that kind (admin only), so /integrations/kinds lacks its labels
type BoardIntegration struct {
	models.BoardIntegration
	KPIs []integrations.KPI `json:"kpis,omitempty"`
}

// Integrations describes the integrations the board's services show
func (s *BoardService) Integrations(ctx context.Context, boardID string) ([]BoardIntegration, error) {
	list, err := s.repo.Integrations(ctx, boardID)
	if err != nil {
		return nil, err
	}
	out := make([]BoardIntegration, len(list))
	for i, item := range list {
		out[i].BoardIntegration = item
		if kind, ok := integrations.Get(item.Kind); ok {
			out[i].KPIs = kind.Meta().KPIs
		}
	}
	return out, nil
}

// Create adds a board for the user. Only an admin can share one.
func (s *BoardService) Create(ctx context.Context, userID string, isAdmin bool, req *models.BoardRequest) (*models.Board, error) {
	board := &models.Board{UserID: userID}
	if err := applyBoardRequest(board, req, isAdmin); err != nil {
		return nil, err
	}
	autoSlug := board.Slug == ""
	if autoSlug {
		board.Slug = models.BoardSlug(board.Name)
	}
	if err := s.repo.Create(ctx, board, autoSlug); err != nil {
		return nil, boardWriteError(err)
	}
	return board, nil
}

// Update changes the name, slug or shared flag of the user's board
func (s *BoardService) Update(ctx context.Context, userID string, isAdmin bool, boardID string, req *models.BoardRequest) (*models.Board, error) {
	access, err := s.ResolveOwned(ctx, userID, boardID)
	if err != nil {
		return nil, err
	}
	board, err := s.repo.Update(ctx, access.Board.ID, userID, func(board *models.Board) error {
		return applyBoardRequest(board, req, isAdmin)
	})
	if err != nil {
		return nil, boardWriteError(err)
	}
	return board, nil
}

// applyBoardRequest validates req on top of board; omitted fields stay
func applyBoardRequest(board *models.Board, req *models.BoardRequest, isAdmin bool) error {
	if req.Name != nil {
		board.Name = strings.TrimSpace(*req.Name)
	}
	if board.Name == "" {
		return invalid("Name is required")
	}
	if utf8.RuneCountInString(board.Name) > models.MaxBoardNameLen {
		return invalid("Name must be %d characters or less", models.MaxBoardNameLen)
	}
	if req.Slug != nil && strings.TrimSpace(*req.Slug) != "" {
		board.Slug = strings.TrimSpace(*req.Slug)
		if !models.IsValidBoardSlug(board.Slug) {
			return invalid("URL name must be lowercase letters, digits and dashes (at most %d), and not a reserved word", models.MaxBoardSlugLen)
		}
	}
	if req.IsShared != nil && *req.IsShared != board.IsShared {
		// Unsharing is always fine; a demoted admin can take a board back
		if *req.IsShared && !isAdmin {
			return invalid("Only admins can share a board")
		}
		// Imports, discovery and API clients without a board write to the
		// default board; nothing should become public that way
		if *req.IsShared && board.IsDefault {
			return invalid("The default board can't be shared. Create a board to share.")
		}
		board.IsShared = *req.IsShared
	}
	return nil
}

func boardWriteError(err error) error {
	switch err {
	case repository.ErrBoardSlugTaken:
		return invalid("You already have a board with this URL name")
	case repository.ErrBoardLimitReached:
		return invalid("Maximum board limit reached (%d)", models.MaxBoardsPerUser)
	}
	return err
}

// Reorder puts the user's own boards in the given order
func (s *BoardService) Reorder(ctx context.Context, userID string, ids []string) error {
	if len(ids) == 0 || len(ids) > models.MaxBoardsPerUser {
		return invalid("Between 1 and %d boards are required", models.MaxBoardsPerUser)
	}
	seen := map[string]bool{}
	for i, id := range ids {
		parsed, err := uuid.Parse(id)
		if err != nil || seen[parsed.String()] {
			return invalid("Each board must be listed once, by ID")
		}
		ids[i] = parsed.String()
		seen[ids[i]] = true
	}
	return s.repo.Reorder(ctx, userID, ids)
}

// Delete removes the user's board. With moveTo its tiles and groups go to
// that board of the user first; without it they are deleted too.
func (s *BoardService) Delete(ctx context.Context, userID, boardID, moveTo string) error {
	if strings.TrimSpace(moveTo) != "" {
		target, err := s.ResolveOwned(ctx, userID, moveTo)
		if err != nil {
			return invalid("The board to move the tiles to was not found")
		}
		if moveTo = target.Board.ID; moveTo == boardID {
			return invalid("The tiles can't move to the board that is deleted")
		}
	}
	err := s.repo.Delete(ctx, boardID, userID, strings.TrimSpace(moveTo))
	if err == repository.ErrCannotDeleteDefaultBoard {
		return invalid("The default board can't be deleted")
	}
	return err
}

// MoveTile puts the user's service or widget (models.TileKind*) on another
// of their boards, ungrouped and after the last tile. It returns the tile's
// new position.
func (s *BoardService) MoveTile(ctx context.Context, userID, kind, id, boardID string) (int, error) {
	access, err := s.ResolveOwned(ctx, userID, boardID)
	if err != nil {
		return 0, err
	}
	if access.Board == nil {
		return 0, repository.ErrBoardNotFound
	}
	return s.repo.MoveTile(ctx, userID, kind, id, access.Board.ID)
}
