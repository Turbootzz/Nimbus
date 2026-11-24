package services

import (
	"context"

	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/widgets"
)

// BoardContent is what one board shows to one user: for a viewer of a
// shared board, widgets and snapshots are the viewer's versions
type BoardContent struct {
	Board     *models.Board
	CanEdit   bool
	Services  []*models.Service
	Groups    []*models.Group
	Widgets   []models.Widget
	Snapshots []models.Snapshot
}

// BoardReader reads a whole board for a user, through BoardService.Resolve.
// The MCP tools use it; the HTTP handlers read the same pieces one by one.
type BoardReader struct {
	Boards   *BoardService
	Services *repository.ServiceRepository
	Groups   *repository.GroupRepository
	Widgets  *WidgetService
	Live     *LiveDataService
}

// ServicesForViewer strips the links of services someone else's shared board
// shows: http(s) only, as on the public status page, and without the
// credentials and query (https://user:password@host/?api_key=...) that keys
// live in, as for the widgets there
func ServicesForViewer(list []*models.Service) {
	strip := func(raw string) string { return widgets.StripURLSecrets(publicURL(raw)).(string) }
	for _, service := range list {
		service.URL = strip(service.URL)
		if service.IconType == models.IconTypeImageURL {
			service.IconImagePath = strip(service.IconImagePath)
		}
	}
}

// List returns the boards the user sees
func (r *BoardReader) List(ctx context.Context, userID string) ([]*models.Board, error) {
	return r.Boards.List(ctx, userID)
}

// Read returns the board ref means (ID, slug or name; "" is the user's
// default board), or repository.ErrBoardNotFound
func (r *BoardReader) Read(ctx context.Context, userID, ref string) (*BoardContent, error) {
	access, err := r.Boards.ResolveRef(ctx, userID, ref)
	if err != nil {
		return nil, err
	}
	content := &BoardContent{Board: access.Board, CanEdit: access.CanEdit}
	if content.Services, err = r.Services.GetAllByBoardID(ctx, access.Board.ID); err != nil {
		return nil, err
	}
	if !access.CanEdit {
		ServicesForViewer(content.Services)
	}
	if content.Groups, err = r.Groups.GetAllByBoardID(ctx, access.Board.ID); err != nil {
		return nil, err
	}
	if content.Widgets, err = r.Widgets.ListBoard(ctx, userID, access); err != nil {
		return nil, err
	}
	if access.CanEdit {
		content.Snapshots = r.Live.Snapshots(userID)
		return content, nil
	}
	keys, err := r.Boards.ViewerKeys(ctx, access.Board.ID)
	if err != nil {
		return nil, err
	}
	content.Snapshots = r.Live.ViewerSnapshots(access.Board.UserID, keys)
	return content, nil
}
