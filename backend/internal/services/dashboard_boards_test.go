package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"testing"

	"github.com/nimbus/backend/internal/dashdoc"
	"github.com/nimbus/backend/internal/db"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newBoardTransferEnv is a transfer service that knows boards, as in main.go
func newBoardTransferEnv(t *testing.T) (*transferEnv, *repository.BoardRepository) {
	t.Helper()
	env := newTransferEnv(t)
	boards := repository.NewBoardRepository(env.db)
	env.svc.Boards = boards
	return env, boards
}

// addBoard gives the user a board with one group, service and widget on it
func (e *transferEnv) addBoard(t *testing.T, boards *repository.BoardRepository, userID, name string, shared bool) *models.Board {
	t.Helper()
	board := &models.Board{UserID: userID, Name: name, Slug: models.BoardSlug(name), IsShared: shared}
	require.NoError(t, boards.Create(context.Background(), board, false))
	group := e.addGroup(t, userID, name+" apps", 5, false)
	_, err := e.db.Exec(`UPDATE groups SET board_id = ? WHERE id = ?`, board.ID, group)
	require.NoError(t, err)
	e.addService(t, userID, models.Service{Name: name + " service", URL: "https://" + board.Slug + ".example.com", BoardID: board.ID, GroupID: &group})
	e.addService(t, userID, models.Service{Name: "Plex", URL: "https://plex.example.com", BoardID: board.ID})
	widget := e.addWidget(t, userID, models.WidgetRequest{Type: "markdown", Title: new(name + " notes"), Config: json.RawMessage(`{"content":"hi"}`)})
	_, err = e.db.Exec(`UPDATE widgets SET board_id = ? WHERE id = ?`, board.ID, widget.ID)
	require.NoError(t, err)
	return board
}

func (e *transferEnv) countOn(t *testing.T, table, boardID string) int {
	t.Helper()
	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM `+table+` WHERE board_id = ?`, boardID).Scan(&n))
	return n
}

func TestExportOfOneBoardNamesNoBoards(t *testing.T) {
	env, _ := newBoardTransferEnv(t)
	env.seedDashboard(t)
	withBoards := toJSON(t, env.export(t, transferAdmin))
	assert.NotContains(t, withBoards, "board", "a dashboard with one board exports as it did before boards existed")

	env.svc.Boards = nil
	assert.Equal(t, toJSON(t, env.export(t, transferAdmin)), withBoards)
}

func TestRoundTripWithTwoBoards(t *testing.T) {
	env, boards := newBoardTransferEnv(t)
	ctx := context.Background()
	env.seedDashboard(t)
	family := env.addBoard(t, boards, transferAdmin, "Family", true)
	lab := env.addBoard(t, boards, transferAdmin, "Lab", false)

	first := env.export(t, transferAdmin)
	require.Equal(t, []dashdoc.Board{
		{Ref: "family", Name: "Family", Slug: "family", Shared: true},
		{Ref: "lab", Name: "Lab", Slug: "lab"},
	}, first.Boards)
	refs := map[string]string{}
	for _, group := range first.Groups {
		refs["group "+group.Name] = group.BoardRef
	}
	for _, service := range first.Services {
		refs["service "+service.Name+" "+service.URL] = service.BoardRef
	}
	for _, widget := range first.Widgets {
		refs["widget "+widget.Title] = widget.BoardRef
	}
	assert.Equal(t, "family", refs["group Family apps"])
	assert.Equal(t, "family", refs["service Family service https://family.example.com"])
	assert.Equal(t, "lab", refs["service Plex https://plex.example.com"], "the last one listed: Plex is on three boards")
	assert.Equal(t, "lab", refs["widget Lab notes"])
	assert.Equal(t, "", refs["group Media"])
	assert.Equal(t, "", refs["widget Notes"])

	// Into an empty account: the same dashboard, boards included
	summary, err := env.svc.Apply(ctx, transferFresh, reparse(t, first), dashdoc.ModeReplace)
	require.NoError(t, err)
	assert.Equal(t, 7, summary.Services.Created)
	assert.Equal(t, toJSON(t, first), toJSON(t, env.export(t, transferFresh)))
	assert.Equal(t, toJSON(t, first), toJSON(t, env.export(t, transferAdmin)), "the source account is untouched")
	fresh, err := boards.ListByUserID(ctx, transferFresh)
	require.NoError(t, err)
	require.Len(t, fresh, 3)
	assert.True(t, fresh[0].IsDefault && !fresh[0].IsShared)
	assert.True(t, fresh[1].IsShared, "an admin's import shares the board")
	assert.NotEqual(t, family.ID, fresh[1].ID)
	assert.Equal(t, 2, env.countOn(t, "services", fresh[1].ID))
	assert.Equal(t, 2, env.countOn(t, "groups", fresh[1].ID), "its own default group and the imported one")

	// Onto itself: replaced board by board, nothing doubled
	_, err = env.svc.Apply(ctx, transferAdmin, reparse(t, first), dashdoc.ModeReplace)
	require.NoError(t, err)
	assert.Equal(t, toJSON(t, first), toJSON(t, env.export(t, transferAdmin)))
	own, err := boards.ListByUserID(ctx, transferAdmin)
	require.NoError(t, err)
	require.Len(t, own, 3)
	assert.Equal(t, []string{family.ID, lab.ID}, []string{own[1].ID, own[2].ID}, "the boards the user has are found by their slug")

	// Someone who is not an admin gets the boards, none of them shared
	_, err = env.svc.Apply(ctx, transferUser, withoutAdminOnly(reparse(t, first)), dashdoc.ModeMerge)
	require.NoError(t, err)
	theirs, err := boards.ListByUserID(ctx, transferUser)
	require.NoError(t, err)
	require.Len(t, theirs, 3)
	for _, board := range theirs {
		assert.False(t, board.IsShared, board.Name)
	}
}

// withoutAdminOnly drops what only an admin may import
func withoutAdminOnly(doc *dashdoc.Document) *dashdoc.Document {
	kept := doc.Widgets[:0]
	for _, widget := range doc.Widgets {
		if widget.Type != "custom_api" {
			kept = append(kept, widget)
		}
	}
	doc.Widgets = kept
	doc.Preferences = nil
	return doc
}

func TestImportLeavesBoardsTheDocumentDoesNotListAlone(t *testing.T) {
	env, boards := newBoardTransferEnv(t)
	ctx := context.Background()
	env.seedDashboard(t)
	family := env.addBoard(t, boards, transferAdmin, "Family", true)
	lab := env.addBoard(t, boards, transferAdmin, "Lab", false)
	// The lab board shows the Sonarr integration
	sonarr := env.id(t, "integrations", "name", "Sonarr")
	_, err := env.db.Exec(`UPDATE services SET integration_id = ? WHERE board_id = ? AND name = 'Lab service'`, sonarr, lab.ID)
	require.NoError(t, err)

	// A document from before boards existed, and one that lists Family only
	old := &dashdoc.Document{Version: 1, Services: []dashdoc.Service{{Name: "Router", URL: "http://192.168.1.1"}}}
	preview, err := env.svc.Preview(ctx, transferAdmin, reparse(t, old), dashdoc.ModeReplace)
	require.NoError(t, err)
	assert.Equal(t, dashdoc.Count{Created: 1, Removed: 3}, preview.Services, "only the default board's services go")
	assert.Equal(t, 1, preview.Integrations.Removed, "Sonarr is still shown on the lab board")
	summary, err := env.svc.Apply(ctx, transferAdmin, reparse(t, old), dashdoc.ModeReplace)
	require.NoError(t, err)
	assert.Equal(t, preview.Services, summary.Services)
	assert.Equal(t, preview.Integrations, summary.Integrations)
	assert.Equal(t, 2, env.countOn(t, "services", family.ID))
	assert.Equal(t, 2, env.countOn(t, "services", lab.ID))
	assert.Equal(t, 1, env.countOn(t, "widgets", lab.ID))
	assert.Equal(t, sonarr, env.id(t, "integrations", "name", "Sonarr"), "an integration another board uses stays")
	defaultBoard, err := boards.EnsureDefault(ctx, transferAdmin)
	require.NoError(t, err)
	assert.Equal(t, 1, env.countOn(t, "services", defaultBoard.ID))

	listed := &dashdoc.Document{
		Version:  1,
		Boards:   []dashdoc.Board{{Ref: "f", Name: "Family night", Slug: "family"}},
		Groups:   []dashdoc.Group{{Ref: "g", Name: "Games", BoardRef: "f"}},
		Services: []dashdoc.Service{{Name: "Steam", URL: "http://192.168.1.5", BoardRef: "f", GroupRef: "g"}},
	}
	_, err = env.svc.Apply(ctx, transferAdmin, reparse(t, listed), dashdoc.ModeReplace)
	require.NoError(t, err)
	assert.Equal(t, 1, env.countOn(t, "services", family.ID))
	assert.Equal(t, 0, env.countOn(t, "services", defaultBoard.ID), "the default board is always part of a document")
	assert.Equal(t, 2, env.countOn(t, "services", lab.ID))
	renamed, err := boards.GetByID(ctx, family.ID)
	require.NoError(t, err)
	assert.Equal(t, "Family night", renamed.Name)
	assert.False(t, renamed.IsShared, "a replace makes the board what the document says")

	// A merge adds to the board and leaves its name alone
	listed.Boards[0].Name = "Something else"
	_, err = env.svc.Apply(ctx, transferAdmin, reparse(t, listed), dashdoc.ModeMerge)
	require.NoError(t, err)
	assert.Equal(t, 2, env.countOn(t, "services", family.ID))
	renamed, err = boards.GetByID(ctx, family.ID)
	require.NoError(t, err)
	assert.Equal(t, "Family night", renamed.Name)
}

func TestDocumentBoardsAreValidated(t *testing.T) {
	env, _ := newBoardTransferEnv(t)
	ctx := context.Background()
	for name, c := range map[string]struct {
		doc  dashdoc.Document
		path string
	}{
		"unknown board":      {dashdoc.Document{Services: []dashdoc.Service{{Name: "A", URL: "http://a.test", BoardRef: "nope"}}}, "services[0].board_ref"},
		"board without ref":  {dashdoc.Document{Boards: []dashdoc.Board{{Name: "A"}}}, "boards[0].ref"},
		"board without name": {dashdoc.Document{Boards: []dashdoc.Board{{Ref: "a"}}}, "boards[0].name"},
		"bad slug":           {dashdoc.Document{Boards: []dashdoc.Board{{Ref: "a", Name: "A", Slug: "A b"}}}, "boards[0].slug"},
		"same slug twice":    {dashdoc.Document{Boards: []dashdoc.Board{{Ref: "a", Name: "Media"}, {Ref: "b", Name: "media"}}}, "boards[1].slug"},
		"tile off its group's board": {dashdoc.Document{
			Boards:   []dashdoc.Board{{Ref: "a", Name: "A"}},
			Groups:   []dashdoc.Group{{Ref: "g", Name: "G", BoardRef: "a"}},
			Services: []dashdoc.Service{{Name: "A", URL: "http://a.test", GroupRef: "g"}},
		}, "services[0].board_ref"},
		"two default groups on a board": {dashdoc.Document{
			Boards: []dashdoc.Board{{Ref: "a", Name: "A"}},
			Groups: []dashdoc.Group{{Name: "One", BoardRef: "a", Default: true}, {Name: "Two", BoardRef: "a", Default: true}},
		}, "groups[1].default"},
	} {
		c.doc.Version = 1
		_, err := env.svc.Preview(ctx, transferAdmin, &c.doc, dashdoc.ModeMerge)
		require.Error(t, err, name)
		assert.Contains(t, issuePaths(t, err), c.path, name)
	}

	// One default group per board is fine
	ok := &dashdoc.Document{
		Version: 1,
		Boards:  []dashdoc.Board{{Ref: "a", Name: "A"}},
		Groups:  []dashdoc.Group{{Name: "One", Default: true}, {Name: "Two", BoardRef: "a", Default: true}},
	}
	_, err := env.svc.Preview(ctx, transferAdmin, ok, dashdoc.ModeMerge)
	assert.NoError(t, err)
}

func TestSyncOnlyWorksOnTheBoardsOfTheDocument(t *testing.T) {
	env, boards := newBoardTransferEnv(t)
	ctx := context.Background()
	env.seedDashboard(t)
	family := env.addBoard(t, boards, transferAdmin, "Family", false)
	lab := env.addBoard(t, boards, transferAdmin, "Lab", false)
	doc := env.export(t, transferAdmin)

	// The export describes the dashboard: syncing it twice writes nothing
	env.sync(t, transferAdmin, reparse(t, doc), true)
	before := env.dump(t, transferAdmin)
	assertNothingChanged(t, env.sync(t, transferAdmin, reparse(t, doc), true))
	assert.Equal(t, before, env.dump(t, transferAdmin), "not a single column may change")

	// A file without boards manages the default board; with prune it empties
	// that one and nothing else
	labBefore := env.countOn(t, "services", lab.ID)
	plain := &dashdoc.Document{Version: 1, Groups: []dashdoc.Group{}, Services: []dashdoc.Service{{Name: "Plex", URL: "https://plex.example.com"}}, Widgets: []dashdoc.Widget{}, Integrations: []dashdoc.Integration{}}
	summary := env.sync(t, transferAdmin, plain, true)
	assert.Equal(t, dashdoc.Count{Removed: 2, Updated: 1}, summary.Services, "Plex on the default board is the one that is matched")
	assert.Equal(t, 2, env.countOn(t, "services", family.ID))
	assert.Equal(t, labBefore, env.countOn(t, "services", lab.ID))
	assert.Equal(t, 1, env.countOn(t, "widgets", family.ID))
	assert.Equal(t, 2, env.countOn(t, "groups", lab.ID))

	// A file that lists a board manages that one too: same name, other board, other row
	withFamily := &dashdoc.Document{
		Version: 1,
		Boards:  []dashdoc.Board{{Ref: "fam", Name: "Family", Shared: true}, {Ref: "new", Name: "Wall Screen"}},
		Groups:  []dashdoc.Group{},
		Services: []dashdoc.Service{
			{Name: "Plex", URL: "https://plex.example.com"},
			{Name: "Plex", URL: "https://plex.example.com/web", BoardRef: "fam", Position: 1},
			{Name: "Clock", URL: "http://192.168.1.9", BoardRef: "new", Position: 2},
		},
		Widgets:      []dashdoc.Widget{},
		Integrations: []dashdoc.Integration{},
	}
	familyPlex := ""
	require.NoError(t, env.db.QueryRow(`SELECT id FROM services WHERE board_id = ? AND name = 'Plex'`, family.ID).Scan(&familyPlex))
	summary = env.sync(t, transferAdmin, withFamily, true)
	assert.Equal(t, dashdoc.Count{Created: 1, Updated: 1, Removed: 1}, summary.Services)
	var url string
	require.NoError(t, env.db.QueryRow(`SELECT url FROM services WHERE id = ?`, familyPlex).Scan(&url))
	assert.Equal(t, "https://plex.example.com/web", url, "the service on the family board kept its row")
	assert.Equal(t, 1, env.countOn(t, "services", family.ID))
	assert.Equal(t, 0, env.countOn(t, "widgets", family.ID))
	assert.Equal(t, labBefore, env.countOn(t, "services", lab.ID), "the lab board is not in the file")
	own, err := boards.ListByUserID(ctx, transferAdmin)
	require.NoError(t, err)
	require.Len(t, own, 4)
	assert.True(t, own[1].IsShared, "the file shares the family board")
	assert.Equal(t, "wall-screen", own[3].Slug)
	assert.Equal(t, 1, env.countOn(t, "services", own[3].ID))
	assert.Equal(t, 1, env.countOn(t, "groups", own[3].ID), "a new board gets its default group")

	before = env.dump(t, transferAdmin)
	assertNothingChanged(t, env.sync(t, transferAdmin, withFamily, true))
	assert.Equal(t, before, env.dump(t, transferAdmin))
	// Only the shared flag changes: that alone is written
	withFamily.Boards[0].Shared = false
	assertNothingChanged(t, env.sync(t, transferAdmin, withFamily, true))
	unshared, err := boards.GetByID(ctx, family.ID)
	require.NoError(t, err)
	assert.False(t, unshared.IsShared)
}

// TestBoardsTransferOnPostgres syncs, exports and imports a dashboard with
// boards on a real PostgreSQL: the SQL with board lists, the per-board
// default group index and the foreign keys. See TestSyncOnPostgres for how
// to run it (NIMBUS_TEST_POSTGRES_DSN, an EMPTY throwaway database).
func TestBoardsTransferOnPostgres(t *testing.T) {
	dsn := os.Getenv("NIMBUS_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("NIMBUS_TEST_POSTGRES_DSN is not set")
	}
	database, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { database.Close() })
	t.Chdir("../..")
	require.NoError(t, db.RunMigrations(database))
	_, err = database.Exec(`INSERT INTO users (id, email, name, password, role) VALUES ($1, 'boards-admin@example.com', 'Admin', 'x', 'admin')`, transferFresh)
	require.NoError(t, err)

	env := newTransferEnvOn(t, database)
	boards := repository.NewBoardRepository(database)
	env.svc.Boards = boards
	ctx := context.Background()
	count := func(query string, args ...any) int {
		t.Helper()
		var n int
		require.NoError(t, database.QueryRow(query, args...).Scan(&n))
		return n
	}

	doc := &dashdoc.Document{
		Version: 1,
		Boards:  []dashdoc.Board{{Ref: "fam", Name: "Family", Shared: true}, {Ref: "lab", Name: "Lab"}},
		Groups: []dashdoc.Group{
			{Ref: "main", Name: "Services", Default: true},
			{Ref: "fam-main", Name: "Everyone", Default: true, BoardRef: "fam"},
			{Ref: "fam-media", Name: "Media", BoardRef: "fam", TileStyle: "chips"},
		},
		Integrations: []dashdoc.Integration{{Ref: "sonarr", Kind: "sonarr", Name: "Sonarr", BaseURL: "http://192.168.1.10:8989"}},
		Services: []dashdoc.Service{
			{Name: "Router", URL: "http://192.168.1.1", GroupRef: "main"},
			{Name: "Sonarr", URL: "http://192.168.1.10:8989", BoardRef: "fam", GroupRef: "fam-media", IntegrationRef: "sonarr", Position: 1},
			{Name: "Proxmox", URL: "https://192.168.1.20:8006", BoardRef: "lab", Position: 2},
		},
		Widgets: []dashdoc.Widget{{Type: "clock", Title: "Clock", BoardRef: "fam", Position: 3}},
	}
	first, err := env.svc.Sync(ctx, transferFresh, reparse(t, doc), true)
	require.NoError(t, err)
	assert.Equal(t, 3, first.Services.Created)
	own, err := boards.ListByUserID(ctx, transferFresh)
	require.NoError(t, err)
	require.Len(t, own, 3)
	family, lab := own[1], own[2]
	assert.True(t, family.IsShared)
	assert.Equal(t, 2, count(`SELECT COUNT(*) FROM groups WHERE board_id = $1`, family.ID))
	assert.Equal(t, 1, count(`SELECT COUNT(*) FROM groups WHERE board_id = $1 AND is_default`, lab.ID), "a board the file gives no default group gets one")
	assert.Equal(t, 1, count(`SELECT COUNT(*) FROM widgets WHERE board_id = $1`, family.ID))

	second, err := env.svc.Sync(ctx, transferFresh, reparse(t, doc), true)
	require.NoError(t, err)
	assertNothingChanged(t, second)

	// The lab board leaves the file: it stays as it is, and so does what it shows
	withoutLab := reparse(t, doc)
	withoutLab.Boards, withoutLab.Services = withoutLab.Boards[:1], withoutLab.Services[:2]
	_, err = env.svc.Sync(ctx, transferFresh, withoutLab, true)
	require.NoError(t, err)
	assert.Equal(t, 1, count(`SELECT COUNT(*) FROM services WHERE board_id = $1`, lab.ID))

	// Export, replace with it, export again: the same document
	exported := env.export(t, transferFresh)
	require.Len(t, exported.Boards, 2)
	summary, err := env.svc.Apply(ctx, transferFresh, reparse(t, exported), dashdoc.ModeReplace)
	require.NoError(t, err)
	assert.Equal(t, dashdoc.Count{Created: 3, Removed: 3}, summary.Services)
	assert.Equal(t, toJSON(t, exported), toJSON(t, env.export(t, transferFresh)))
	assert.Equal(t, 3, count(`SELECT COUNT(*) FROM boards WHERE user_id = $1`, transferFresh))

	// A document from before boards replaces the default board only
	plain := &dashdoc.Document{Version: 1, Services: []dashdoc.Service{{Name: "NAS", URL: "http://192.168.1.30"}}}
	summary, err = env.svc.Apply(ctx, transferFresh, plain, dashdoc.ModeReplace)
	require.NoError(t, err)
	assert.Equal(t, dashdoc.Count{Created: 1, Removed: 1}, summary.Services)
	assert.Zero(t, summary.Integrations.Removed, "Sonarr is shown on the family board")
	assert.Equal(t, 1, count(`SELECT COUNT(*) FROM services WHERE board_id = $1 AND integration_id IS NOT NULL`, family.ID))
	_, err = env.svc.Apply(ctx, transferFresh, reparse(t, exported), dashdoc.ModeMerge)
	require.NoError(t, err)
	assert.Equal(t, 2, count(`SELECT COUNT(*) FROM services WHERE board_id = $1`, family.ID))
	assert.Equal(t, 1, count(`SELECT COUNT(*) FROM groups WHERE board_id = $1 AND is_default`, family.ID))
}

func TestReplaceMovesTheStatusPageOnlyOnItsBoard(t *testing.T) {
	env, boards := newBoardTransferEnv(t)
	ctx := context.Background()
	env.seedDashboard(t)
	defaultOnly := env.export(t, transferAdmin)
	lab := env.addBoard(t, boards, transferAdmin, "Lab", false)
	var labGroup string
	require.NoError(t, env.db.QueryRow(`SELECT id FROM groups WHERE board_id = ? AND name = 'Lab apps'`, lab.ID).Scan(&labGroup))
	page := &fakeStatusPage{models.StatusPageSettings{Enabled: true, GroupID: labGroup}}
	env.svc.SetStatusPage(page)

	// A document without the Lab board leaves its groups, and the page, alone
	summary, err := env.svc.Apply(ctx, transferAdmin, reparse(t, defaultOnly), dashdoc.ModeReplace)
	require.NoError(t, err)
	assert.Nil(t, summary.StatusPage)
	assert.Equal(t, labGroup, page.GroupID)

	// One that lists it re-creates the group there, and the page follows,
	// also with a group of that name on the default board ahead of it
	env.addGroup(t, transferAdmin, "Lab apps", -1, false)
	summary, err = env.svc.Apply(ctx, transferAdmin, reparse(t, env.export(t, transferAdmin)), dashdoc.ModeReplace)
	require.NoError(t, err)
	assert.Equal(t, &dashdoc.StatusPageImpact{Group: "Lab apps", Kept: true}, summary.StatusPage)
	var recreated string
	require.NoError(t, env.db.QueryRow(`SELECT id FROM groups WHERE board_id = ? AND name = 'Lab apps'`, lab.ID).Scan(&recreated))
	assert.NotEqual(t, labGroup, recreated)
	assert.Equal(t, recreated, page.GroupID)
}
