package mcpserver

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	svc "github.com/nimbus/backend/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeBoards gives alice a default board and a media board, and shows bob's
// shared family board to everyone. Read answers like services.BoardReader:
// only boards the user may see.
type fakeBoards struct {
	boards  []*models.Board
	content map[string]*svc.BoardContent // by board ID
}

func (f *fakeBoards) visible(userID string) []*models.Board {
	var boards []*models.Board
	for _, board := range f.boards {
		if board.UserID == userID || board.IsShared {
			boards = append(boards, board)
		}
	}
	return boards
}

func (f *fakeBoards) List(_ context.Context, userID string) ([]*models.Board, error) {
	return f.visible(userID), nil
}

func (f *fakeBoards) Read(_ context.Context, userID, ref string) (*svc.BoardContent, error) {
	for _, board := range f.visible(userID) {
		isDefault := ref == "" && board.IsDefault && board.UserID == userID
		if isDefault || (ref != "" && (board.ID == ref || board.Slug == ref || strings.EqualFold(board.Name, ref))) {
			content := *f.content[board.ID]
			content.Board, content.CanEdit = board, board.UserID == userID
			// A copy, like a repository returning fresh rows
			content.Services = append([]*models.Service(nil), content.Services...)
			return &content, nil
		}
	}
	return nil, repository.ErrBoardNotFound
}

func newBoardTools() (*tools, *fakeData) {
	data := newFakeData()
	deps := data.deps()
	deps.Boards = &fakeBoards{
		boards: []*models.Board{
			{ID: "brd-home", UserID: alice, Name: "Home", Slug: "home", IsDefault: true},
			{ID: "brd-media", UserID: alice, Name: "Media", Slug: "media"},
			{ID: "brd-family", UserID: bob, Name: "Family", Slug: "family", IsShared: true},
			{ID: "brd-bob", UserID: bob, Name: "Bob only", Slug: "bob-only"},
		},
		content: map[string]*svc.BoardContent{
			"brd-home":  {Services: data.services[alice][2:], Groups: data.groups[alice], Widgets: data.widgets[alice][1:]},
			"brd-media": {Services: data.services[alice][:2], Groups: data.groups[alice], Widgets: data.widgets[alice][:1], Snapshots: data.snapshots[alice]},
			"brd-family": {
				Services: data.services[bob],
				Widgets:  data.widgets[bob],
				Snapshots: []models.Snapshot{{
					SourceKind: models.SnapshotSourceWidget, SourceID: "wid-bob", Payload: json.RawMessage(`{"text":"shared"}`), FetchedAt: time.Now(),
				}},
			},
			"brd-bob": {},
		},
	}
	data.logs["svc-bob"] = []*models.StatusLog{{Status: models.StatusOffline, ErrorMessage: ptr("dial tcp 10.0.0.7:8080: refused"), CheckedAt: time.Now()}}
	return &tools{deps}, data
}

func TestBoards_ToolsReadTheDefaultBoardUnlessOneIsNamed(t *testing.T) {
	tools, _ := newBoardTools()
	ctx := context.Background()
	reader := requestAs(alice, models.ScopeMCP)

	_, list, err := tools.listServices(ctx, reader, listServicesInput{})
	require.NoError(t, err)
	assert.Equal(t, []string{"Router", "Pi-hole", "pi-hole"}, names(list))

	for _, board := range []string{"media", "Media", "brd-media"} {
		_, list, err = tools.listServices(ctx, reader, listServicesInput{Board: board})
		require.NoError(t, err, board)
		assert.Equal(t, []string{"Jellyfin", "Sonarr"}, names(list), board)
	}
	_, list, err = tools.searchServices(ctx, reader, searchInput{Q: "sonarr"})
	require.NoError(t, err)
	assert.Empty(t, list.Services, "Sonarr is not on the default board")
	_, list, err = tools.searchServices(ctx, reader, searchInput{Q: "sonarr", Board: "media"})
	require.NoError(t, err)
	assert.Equal(t, []string{"Sonarr"}, names(list))

	_, _, err = tools.getServiceStatus(ctx, reader, serviceInput{Service: "Sonarr"})
	require.Error(t, err)
	_, status, err := tools.getServiceStatus(ctx, reader, serviceInput{Service: "Sonarr", Board: "media"})
	require.NoError(t, err)
	assert.Equal(t, "connection refused", status.RecentChecks[0].Error, "the owner reads the real error")
	_, _, err = tools.getUptime(ctx, reader, uptimeInput{Service: "Jellyfin", Board: "media"})
	require.NoError(t, err)

	_, _, err = tools.getWidgetData(ctx, reader, idInput{ID: "wid-weather"})
	require.Error(t, err)
	_, widget, err := tools.getWidgetData(ctx, reader, idInput{ID: "wid-weather", Board: "media"})
	require.NoError(t, err)
	assert.NotNil(t, widget.Data)
}

func TestBoards_SharedBoardIsReadOnlyAndOthersAreInvisible(t *testing.T) {
	tools, data := newBoardTools()
	ctx := context.Background()
	writer := requestAs(alice, models.ScopeMCP, models.ScopeWriteServices)

	_, list, err := tools.listServices(ctx, writer, listServicesInput{Board: "family"})
	require.NoError(t, err)
	assert.Equal(t, []string{"Vaultwarden"}, names(list))

	_, status, err := tools.getServiceStatus(ctx, writer, serviceInput{Service: "Vaultwarden", Board: "family"})
	require.NoError(t, err)
	assert.Equal(t, svc.ViewerCheckError, status.RecentChecks[0].Error, "what the check ran into is the owner's business")

	_, widget, err := tools.getWidgetData(ctx, writer, idInput{ID: "wid-bob", Board: "family"})
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"text": "shared"}, widget.Data)

	_, _, err = tools.recheckService(ctx, writer, recheckInput{ID: "Vaultwarden", Board: "family"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "only its owner")
	assert.Empty(t, data.checked)

	// A private board of someone else is like a board that does not exist,
	// and the error only names boards the caller sees
	for _, board := range []string{"bob-only", "brd-bob", "nope"} {
		_, _, err = tools.listServices(ctx, writer, listServicesInput{Board: board})
		require.Error(t, err, board)
		assert.Contains(t, err.Error(), "no board")
		assert.Contains(t, err.Error(), "Family (family), shared with you, read-only")
		assert.NotContains(t, err.Error(), "Bob only")
	}
}

func TestBoards_DashboardResourceListsTheBoards(t *testing.T) {
	boardTools, _ := newBoardTools()
	result, err := boardTools.readDashboard(context.Background(), &mcp.ReadResourceRequest{
		Extra: &mcp.RequestExtra{TokenInfo: &auth.TokenInfo{UserID: alice}},
	})
	require.NoError(t, err)
	text := result.Contents[0].Text
	assert.Contains(t, text, "3 services:", "the default board")
	assert.Contains(t, text, "## Boards")
	assert.Contains(t, text, "- Home (home), default")
	assert.Contains(t, text, "- Media (media)\n")
	assert.Contains(t, text, "- Family (family), shared with you, read-only")
	assert.NotContains(t, text, "Bob only")

	// Without a board service a board can't be named
	plain := newFakeData()
	_, _, err = (&tools{plain.deps()}).listServices(context.Background(), requestAs(alice, models.ScopeMCP), listServicesInput{Board: "media"})
	require.Error(t, err)
}
