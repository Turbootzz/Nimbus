package services

import (
	"cmp"
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/nimbus/backend/internal/dashdoc"
	"github.com/nimbus/backend/internal/models"
)

// docBoards are the boards of a document, matched to the boards its user
// has. A document works on the user's default board and the boards it lists;
// their other boards are none of its business.
//
// A nil docBoards is a transfer service without boards (DashboardTransferDeps.
// Boards unset): one dashboard per user, as before boards existed.
type docBoards struct {
	defaultID string
	// ids maps a document ref to the ID of its board: the board the user
	// has with that slug, or a new ID
	ids map[string]string
	// rows are the document's boards as they should be stored
	rows []*models.Board
	// stored are the user's boards among rows as they are now, by ID
	stored map[string]*models.Board
}

// resolveBoards matches the boards of doc to the user's own, by slug. Only
// an admin's document shares a board.
func (s *DashboardTransferService) resolveBoards(ctx context.Context, user *models.User, doc *dashdoc.Document) (*docBoards, error) {
	if s.Boards == nil {
		return nil, nil
	}
	defaultBoard, err := s.Boards.EnsureDefault(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	own, err := s.Boards.ListByUserID(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	bySlug := make(map[string]*models.Board, len(own))
	for _, board := range own {
		bySlug[board.Slug] = board
	}

	boards := &docBoards{defaultID: defaultBoard.ID, ids: map[string]string{}, stored: map[string]*models.Board{}}
	var issues dashdoc.Issues
	seen := map[string]bool{}
	for i, in := range doc.Boards {
		slug := cmp.Or(strings.TrimSpace(in.Slug), models.BoardSlug(in.Name))
		if !models.IsValidBoardSlug(slug) {
			issues.Add(dashdoc.Path("boards", i, "slug"), "must be lowercase letters, digits and dashes (at most %d), and not a reserved word", models.MaxBoardSlugLen)
			continue
		}
		if seen[slug] {
			issues.Add(dashdoc.Path("boards", i, "slug"), "the slug %q is used twice", slug)
			continue
		}
		seen[slug] = true

		row := &models.Board{ID: uuid.New().String(), UserID: user.ID, Name: strings.TrimSpace(in.Name), Slug: slug, IsShared: in.Shared && user.Role == "admin"}
		if existing := bySlug[slug]; existing != nil {
			row.ID, boards.stored[existing.ID] = existing.ID, existing
			// The slug of the default board names the default board, which is never shared
			row.IsShared = row.IsShared && !existing.IsDefault
		}
		boards.ids[in.Ref] = row.ID
		boards.rows = append(boards.rows, row)
	}
	return boards, issues.Err()
}

// id is the board a document entity is on: the board of its ref, or the
// default board. "" without boards.
func (b *docBoards) id(ref string) string {
	if b == nil {
		return ""
	}
	return cmp.Or(b.ids[ref], b.defaultID)
}

// of is the board of a stored row, for matching it to a document entity on
// the same board. "" without boards.
func (b *docBoards) of(boardID string) string {
	if b == nil {
		return ""
	}
	return boardID
}

// manages reports whether a stored row on boardID is the document's business
func (b *docBoards) manages(boardID string) bool {
	return b == nil || boardID == b.defaultID || b.stored[boardID] != nil
}

// listed returns the IDs of the document's boards
func (b *docBoards) listed() []string {
	if b == nil {
		return nil
	}
	ids := make([]string, len(b.rows))
	for i, row := range b.rows {
		ids[i] = row.ID
	}
	return ids
}

// changed returns the document's boards the user doesn't have, or has with
// another name or shared flag: what a sync has to write
func (b *docBoards) changed() []*models.Board {
	if b == nil {
		return nil
	}
	var changed []*models.Board
	for _, row := range b.rows {
		if old := b.stored[row.ID]; old == nil || old.Name != row.Name || old.IsShared != row.IsShared {
			changed = append(changed, row)
		}
	}
	return changed
}

// exportBoards lists the user's boards next to the default one and returns
// the ref of each by ID ("" for the default board, and without boards)
func (s *DashboardTransferService) exportBoards(ctx context.Context, userID string, doc *dashdoc.Document) (map[string]string, error) {
	refs := map[string]string{}
	if s.Boards == nil {
		return refs, nil
	}
	boards, err := s.Boards.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list boards: %w", err)
	}
	names := refNames{}
	for _, board := range boards {
		if board.IsDefault {
			continue
		}
		refs[board.ID] = names.next(board.Slug, "board")
		doc.Boards = append(doc.Boards, dashdoc.Board{Ref: refs[board.ID], Name: board.Name, Slug: board.Slug, Shared: board.IsShared})
	}
	return refs, nil
}
