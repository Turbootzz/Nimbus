package handlers

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/middleware"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/services"
	"github.com/nimbus/backend/internal/testutil"
	"github.com/nimbus/backend/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The people of the board tests: the admin who owns the boards, a second
// admin, a normal user, and (no X-User header) someone who is not logged in
const (
	boardAdminID      = "a0000000-0000-0000-0000-000000000001"
	boardOtherAdminID = "a0000000-0000-0000-0000-000000000002"
	boardUserID       = "a0000000-0000-0000-0000-000000000003"

	// The admin's boards: a shared one and a private one, each with a group,
	// a service (linked to an integration) and a widget
	sharedBoardID  = "b0000000-0000-0000-0000-000000000001"
	privateBoardID = "b0000000-0000-0000-0000-000000000002"

	sharedGroupID        = "c0000000-0000-0000-0000-000000000001"
	privateGroupID       = "c0000000-0000-0000-0000-000000000002"
	sharedServiceID      = "d0000000-0000-0000-0000-000000000001"
	privateServiceID     = "d0000000-0000-0000-0000-000000000002"
	sharedWidgetID       = "e0000000-0000-0000-0000-000000000001"
	privateWidgetID      = "e0000000-0000-0000-0000-000000000002"
	sharedIntegrationID  = "f0000000-0000-0000-0000-000000000001"
	privateIntegrationID = "f0000000-0000-0000-0000-000000000002"
)

var boardTestRoles = map[string]string{boardAdminID: "admin", boardOtherAdminID: "admin", boardUserID: "user"}

type boardFixture struct {
	db   *sql.DB
	app  *fiber.App
	live *services.LiveDataService
	hub  *services.SSEHub
}

// newBoardFixture wires every route that takes a board the way main.go
// does, behind a stub auth middleware: X-User names the user, no header
// means not logged in
func newBoardFixture(t *testing.T, pingInterval time.Duration) *boardFixture {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	// One connection: every connection to :memory: is a database of its own,
	// and the stream reads from another goroutine
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`
		CREATE TABLE users (id TEXT PRIMARY KEY, role TEXT NOT NULL DEFAULT 'user');
		CREATE TABLE groups (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))), user_id TEXT NOT NULL, name TEXT NOT NULL,
			color TEXT DEFAULT '#0ea5e9', position INTEGER DEFAULT 0, is_default INTEGER DEFAULT 0,
			monitoring_enabled INTEGER DEFAULT 1, tile_style TEXT NOT NULL DEFAULT 'cards',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE integrations (
			id TEXT PRIMARY KEY, user_id TEXT NOT NULL, kind TEXT NOT NULL, name TEXT NOT NULL, base_url TEXT NOT NULL,
			auth_type TEXT NOT NULL DEFAULT 'none', credentials_enc BLOB, verify_tls INTEGER NOT NULL DEFAULT 1,
			options TEXT NOT NULL DEFAULT '{}', refresh_seconds INTEGER NOT NULL DEFAULT 60,
			last_test_at TIMESTAMP, last_test_ok INTEGER, last_error TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE services (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))), user_id TEXT NOT NULL, name TEXT NOT NULL,
			url TEXT NOT NULL, icon TEXT DEFAULT '', icon_type TEXT DEFAULT 'emoji', icon_image_path TEXT DEFAULT '',
			description TEXT DEFAULT '', status TEXT NOT NULL DEFAULT 'unknown', response_time INTEGER,
			position INTEGER DEFAULT 0, card_size TEXT DEFAULT '2x1', group_id TEXT, integration_id TEXT,
			monitoring_enabled INTEGER NOT NULL DEFAULT 1,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE widgets (
			id TEXT PRIMARY KEY, user_id TEXT NOT NULL, type TEXT NOT NULL, title TEXT NOT NULL DEFAULT '',
			group_id TEXT, integration_id TEXT, config TEXT NOT NULL DEFAULT '{}', card_size TEXT NOT NULL DEFAULT '2x1',
			position INTEGER NOT NULL DEFAULT 0, refresh_seconds INTEGER NOT NULL DEFAULT 300, enabled INTEGER NOT NULL DEFAULT 1,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE widget_snapshots (
			source_kind TEXT NOT NULL, source_id TEXT NOT NULL, user_id TEXT NOT NULL,
			payload TEXT, error TEXT, fetched_at TIMESTAMP NOT NULL, PRIMARY KEY (source_kind, source_id)
		);
		CREATE TABLE service_status_logs (
			id TEXT PRIMARY KEY, service_id TEXT NOT NULL, status TEXT NOT NULL, response_time INTEGER,
			error_message TEXT, checked_at TIMESTAMP NOT NULL
		);
	`)
	require.NoError(t, err)
	testutil.AddBoards(t, db)
	_, err = db.Exec(`
		INSERT INTO users (id, role) VALUES ('` + boardAdminID + `', 'admin'), ('` + boardOtherAdminID + `', 'admin'), ('` + boardUserID + `', 'user');
		INSERT INTO boards (id, user_id, name, slug, position, is_shared) VALUES
			('` + sharedBoardID + `', '` + boardAdminID + `', 'Family', 'family', 1, 1),
			('` + privateBoardID + `', '` + boardAdminID + `', 'Lab', 'lab', 2, 0);
		INSERT INTO groups (id, user_id, board_id, name, is_default) VALUES
			('` + sharedGroupID + `', '` + boardAdminID + `', '` + sharedBoardID + `', 'Media', 1),
			('` + privateGroupID + `', '` + boardAdminID + `', '` + privateBoardID + `', 'Servers', 1);
		INSERT INTO integrations (id, user_id, kind, name, base_url, options) VALUES
			('` + sharedIntegrationID + `', '` + boardAdminID + `', 'handler-fake', 'Sonarr', 'http://sonarr.lan:8989', '{"library":"tv","empty":""}'),
			('` + privateIntegrationID + `', '` + boardAdminID + `', 'handler-fake', 'Proxmox', 'http://pve.lan:8006', '{}');
		INSERT INTO services (id, user_id, board_id, name, url, integration_id, position) VALUES
			('` + sharedServiceID + `', '` + boardAdminID + `', '` + sharedBoardID + `', 'Jellyfin', 'http://jellyfin.lan', '` + sharedIntegrationID + `', 1),
			('` + privateServiceID + `', '` + boardAdminID + `', '` + privateBoardID + `', 'Proxmox', 'http://pve.lan', '` + privateIntegrationID + `', 2);
		INSERT INTO widgets (id, user_id, board_id, type, title, config, position) VALUES
			('` + sharedWidgetID + `', '` + boardAdminID + `', '` + sharedBoardID + `', 'rss', 'News', '{"url":"https://feeds.test/private?key=s3cret"}', 3),
			('` + privateWidgetID + `', '` + boardAdminID + `', '` + privateBoardID + `', 'markdown', 'Passwords', '{"content":"root / hunter2"}', 4);
	`)
	require.NoError(t, err)

	boards := services.NewBoardService(repository.NewBoardRepository(db))
	serviceRepo, groupRepo := repository.NewServiceRepository(db), repository.NewGroupRepository(db)
	integrationRepo, widgetRepo := repository.NewIntegrationRepository(db), repository.NewWidgetRepository(db)

	serviceHandler := NewServiceHandler(serviceRepo, groupRepo, nil, integrationRepo)
	serviceHandler.SetBoards(boards)
	groupHandler := NewGroupHandler(groupRepo)
	groupHandler.SetBoards(boards)
	widgetService := services.NewWidgetService(widgetRepo, groupRepo, integrationRepo)
	widgetService.SetBoards(boards)
	widgetHandler := NewWidgetHandler(widgetService)
	widgetHandler.SetBoards(boards)
	metricsHandler := NewMetricsHandler(services.NewMetricsService(repository.NewStatusLogRepository(db), serviceRepo), serviceRepo)
	metricsHandler.SetBoards(boards)

	cipher, err := utils.NewCipher(bytes.Repeat([]byte{5}, 32))
	require.NoError(t, err)
	hub := services.NewSSEHub()
	t.Cleanup(hub.Close)
	live := services.NewLiveDataService(widgetRepo, integrationRepo,
		services.NewIntegrationService(integrationRepo, cipher, ""), repository.NewSnapshotRepository(db), hub)
	dashboardHandler := NewDashboardHandler(live, hub)
	dashboardHandler.SetBoards(boards)
	dashboardHandler.pingInterval = pingInterval
	boardHandler := NewBoardHandler(boards)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		if userID := c.Get("X-User"); userID != "" {
			c.Locals("user_id", userID)
			c.Locals("role", boardTestRoles[userID])
		}
		return c.Next()
	})
	app.Get("/boards", boardHandler.ListBoards)
	app.Post("/boards", boardHandler.CreateBoard)
	app.Put("/boards/reorder", boardHandler.ReorderBoards)
	app.Put("/boards/:id", boardHandler.UpdateBoard)
	app.Delete("/boards/:id", boardHandler.DeleteBoard)
	app.Get("/boards/:id/integrations", boardHandler.ListBoardIntegrations)
	app.Get("/services", serviceHandler.GetServices)
	app.Post("/services", serviceHandler.CreateService)
	app.Get("/services/:id/status-logs", metricsHandler.GetRecentStatusLogs)
	app.Get("/services/:id", serviceHandler.GetService)
	app.Put("/services/:id", serviceHandler.UpdateService)
	app.Delete("/services/:id", serviceHandler.DeleteService)
	app.Post("/services/:id/check", serviceHandler.CheckService)
	app.Get("/groups", groupHandler.GetGroups)
	app.Post("/groups", groupHandler.CreateGroup)
	app.Get("/groups/:id", groupHandler.GetGroup)
	app.Put("/groups/:id", groupHandler.UpdateGroup)
	app.Delete("/groups/:id", groupHandler.DeleteGroup)
	app.Get("/widgets", widgetHandler.ListWidgets)
	app.Post("/widgets", widgetHandler.CreateWidget)
	app.Get("/widgets/:id", widgetHandler.GetWidget)
	app.Put("/widgets/:id", widgetHandler.UpdateWidget)
	app.Delete("/widgets/:id", widgetHandler.DeleteWidget)
	app.Post("/widgets/:id/refresh", widgetHandler.RefreshWidget)
	app.Put("/dashboard/reorder", widgetHandler.ReorderTiles)
	app.Get("/dashboard/data", dashboardHandler.Data)
	app.Get("/dashboard/stream", dashboardHandler.Stream)
	app.Get("/metrics/:id", metricsHandler.GetServiceMetrics)
	return &boardFixture{db: db, app: app, live: live, hub: hub}
}

// do sends a request as userID ("" is not logged in)
func (f *boardFixture) do(t *testing.T, userID, method, path, body string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if userID != "" {
		req.Header.Set("X-User", userID)
	}
	resp, err := f.app.Test(req, 5000)
	require.NoError(t, err)
	defer resp.Body.Close()
	var raw bytes.Buffer
	_, err = raw.ReadFrom(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, raw.String()
}

func (f *boardFixture) count(t *testing.T, query string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, f.db.QueryRow(query, args...).Scan(&n))
	return n
}

func TestBoards_ReadMatrix(t *testing.T) {
	f := newBoardFixture(t, time.Minute)
	reads := []string{
		"/services?board=%s",
		"/groups?board=%s",
		"/widgets?board=%s",
		"/dashboard/data?board=%s",
		"/boards/%s/integrations",
	}
	people := []struct {
		name            string
		userID          string
		shared, private int
	}{
		{"owner", boardAdminID, 200, 200},
		{"other admin", boardOtherAdminID, 200, 404},
		{"other user", boardUserID, 200, 404},
		{"anonymous", "", 401, 401},
	}
	for _, who := range people {
		for _, read := range reads {
			status, body := f.do(t, who.userID, http.MethodGet, strings.Replace(read, "%s", sharedBoardID, 1), "")
			assert.Equal(t, who.shared, status, "%s reads %s of the shared board: %s", who.name, read, body)
			status, body = f.do(t, who.userID, http.MethodGet, strings.Replace(read, "%s", privateBoardID, 1), "")
			assert.Equal(t, who.private, status, "%s reads %s of the private board: %s", who.name, read, body)
			if who.private == 404 {
				assert.NotContains(t, body, "Proxmox")
				assert.NotContains(t, body, "hunter2")
			}
		}
		// The stream of a board that may not be seen never opens (an allowed
		// one is followed in TestBoards_ViewerStream)
		if who.private != 200 {
			status, _ := f.do(t, who.userID, http.MethodGet, "/dashboard/stream?board="+privateBoardID, "")
			assert.Equal(t, who.private, status, "%s opens the stream of the private board", who.name)
		}

		// Metrics follow the board the service is on. The metrics query is
		// Postgres only: on SQLite a 500 from it means access was granted.
		for path, granted := range map[string]int{"/metrics/%s": 500, "/services/%s/status-logs": 200} {
			want := func(access int) int {
				return map[int]int{200: granted, 404: 403, 401: 401}[access]
			}
			status, body := f.do(t, who.userID, http.MethodGet, strings.Replace(path, "%s", sharedServiceID, 1), "")
			assert.Equal(t, want(who.shared), status, "%s reads %s of the shared service: %s", who.name, path, body)
			status, body = f.do(t, who.userID, http.MethodGet, strings.Replace(path, "%s", privateServiceID, 1), "")
			assert.Equal(t, want(who.private), status, "%s reads %s of the private service: %s", who.name, path, body)
		}
	}

	// A board that doesn't exist, or an ID that isn't one, answers the same
	for _, board := range []string{"b0000000-0000-0000-0000-00000000dead", "family", "'; DROP TABLE boards; --"} {
		status, body := f.do(t, boardUserID, http.MethodGet, "/services?board="+url.QueryEscape(board), "")
		assert.Equal(t, 404, status, "%s: %s", board, body)
	}
}

func TestBoards_WriteMatrix(t *testing.T) {
	serviceBody := `{"name":"Jellyfin","url":"http://jellyfin.lan"}`
	writes := []struct {
		name, method, path, body string
		// owner is the status the owner gets; everyone else must be refused
		owner int
	}{
		{"create service on the board", "POST", "/services", `{"name":"New","url":"http://new.lan","board_id":"` + sharedBoardID + `"}`, 201},
		{"update service", "PUT", "/services/" + sharedServiceID, serviceBody, 200},
		{"move service to the viewer's side", "PUT", "/services/" + sharedServiceID, `{"name":"Jellyfin","url":"http://jellyfin.lan","board_id":"` + privateBoardID + `"}`, 200},
		{"delete service", "DELETE", "/services/" + sharedServiceID, "", 200},
		{"check service", "POST", "/services/" + sharedServiceID + "/check", "", 0},
		{"create group on the board", "POST", "/groups", `{"name":"New","board_id":"` + sharedBoardID + `"}`, 201},
		{"update group", "PUT", "/groups/" + sharedGroupID, `{"name":"Renamed"}`, 200},
		{"delete group", "DELETE", "/groups/" + sharedGroupID, "", 403}, // a default group
		{"create widget on the board", "POST", "/widgets", `{"type":"clock","board_id":"` + sharedBoardID + `"}`, 201},
		{"update widget", "PUT", "/widgets/" + sharedWidgetID, `{"title":"Renamed"}`, 200},
		{"refresh widget", "POST", "/widgets/" + sharedWidgetID + "/refresh", "", 202},
		{"delete widget", "DELETE", "/widgets/" + sharedWidgetID, "", 204},
		{"reorder tiles", "PUT", "/dashboard/reorder", `{"tiles":[{"id":"` + sharedServiceID + `","kind":"service","position":9}]}`, 200},
		{"rename board", "PUT", "/boards/" + sharedBoardID, `{"name":"Mine now"}`, 200},
		{"unshare board", "PUT", "/boards/" + sharedBoardID, `{"is_shared":false}`, 200},
		{"reorder boards", "PUT", "/boards/reorder", `{"boards":["` + sharedBoardID + `"]}`, 200},
		{"delete board", "DELETE", "/boards/" + sharedBoardID, "", 204},
		{"delete board, keeping its tiles", "DELETE", "/boards/" + sharedBoardID + "?move_to=" + privateBoardID, "", 204},
	}
	for _, write := range writes {
		for _, who := range []struct{ name, userID string }{{"other admin", boardOtherAdminID}, {"other user", boardUserID}, {"anonymous", ""}} {
			f := newBoardFixture(t, time.Minute)
			before := snapshotBoards(t, f)
			status, body := f.do(t, who.userID, write.method, write.path, write.body)
			if who.userID == "" {
				assert.Equal(t, 401, status, "%s: %s as %s: %s", write.name, write.path, who.name, body)
			} else {
				assert.Contains(t, []int{400, 403, 404}, status, "%s as %s: %s", write.name, who.name, body)
			}
			assert.Equal(t, before, snapshotBoards(t, f), "%s as %s changed the owner's boards", write.name, who.name)
		}
		if write.owner != 0 {
			f := newBoardFixture(t, time.Minute)
			status, body := f.do(t, boardAdminID, write.method, write.path, write.body)
			assert.Equal(t, write.owner, status, "%s as owner: %s", write.name, body)
		}
	}
}

// snapshotBoards describes everything the admin owns, to see that a refused
// write changed nothing
func snapshotBoards(t *testing.T, f *boardFixture) string {
	t.Helper()
	var out []string
	for _, query := range []string{
		`SELECT id || '|' || name || '|' || slug || '|' || position || '|' || is_shared FROM boards WHERE user_id = ? ORDER BY id`,
		`SELECT id || '|' || name || '|' || board_id FROM groups WHERE user_id = ? ORDER BY id`,
		`SELECT id || '|' || name || '|' || board_id || '|' || position || '|' || COALESCE(group_id, '') FROM services WHERE user_id = ? ORDER BY id`,
		`SELECT id || '|' || title || '|' || board_id || '|' || position FROM widgets WHERE user_id = ? ORDER BY id`,
	} {
		rows, err := f.db.Query(query, boardAdminID)
		require.NoError(t, err)
		for rows.Next() {
			var line string
			require.NoError(t, rows.Scan(&line))
			out = append(out, line)
		}
		require.NoError(t, rows.Err())
		rows.Close()
	}
	return strings.Join(out, "\n")
}

func TestBoards_ViewerSeesOnlyWhatTheBoardShows(t *testing.T) {
	f := newBoardFixture(t, time.Minute)
	f.live.Record(services.LiveSource{Kind: models.SnapshotSourceWidget, ID: sharedWidgetID, UserID: boardAdminID}, nil,
		assert.AnError) // an error that would quote the admin's network
	f.live.Record(services.LiveSource{Kind: models.SnapshotSourceIntegration, ID: sharedIntegrationID, UserID: boardAdminID}, map[string]int{"queued": 2}, nil)
	f.live.Record(services.LiveSource{Kind: models.SnapshotSourceWidget, ID: privateWidgetID, UserID: boardAdminID}, map[string]int{"n": 1}, nil)
	f.live.Record(services.LiveSource{Kind: models.SnapshotSourceIntegration, ID: privateIntegrationID, UserID: boardAdminID}, map[string]int{"vms": 9}, nil)

	status, body := f.do(t, boardUserID, http.MethodGet, "/boards", "")
	require.Equal(t, 200, status, body)
	var boards []models.BoardResponse
	require.NoError(t, json.Unmarshal([]byte(body), &boards))
	require.Len(t, boards, 2, "the viewer's own default board, created on the way, and the shared board")
	assert.True(t, boards[0].IsDefault && boards[0].Owned && boards[0].CanEdit)
	assert.Equal(t, models.BoardResponse{ID: sharedBoardID, Name: "Family", Slug: "family", Position: 1, IsShared: true, Visibility: "all"}, boards[1])
	assert.NotContains(t, body, boardAdminID, "the owner is not named")
	assert.NotContains(t, body, "Lab")

	status, body = f.do(t, boardUserID, http.MethodGet, "/services?board="+sharedBoardID, "")
	require.Equal(t, 200, status)
	assert.Contains(t, body, "Jellyfin")
	assert.NotContains(t, body, "Proxmox")
	assert.NotContains(t, body, boardAdminID)

	status, body = f.do(t, boardUserID, http.MethodGet, "/widgets?board="+sharedBoardID, "")
	require.Equal(t, 200, status)
	assert.Contains(t, body, `"title":"News"`)
	assert.Contains(t, body, `https://feeds.test/private`)
	assert.NotContains(t, body, "s3cret", "keys in the URL of a feed the server fetches are not for viewers")
	assert.NotContains(t, body, "Passwords")
	assert.NotContains(t, body, boardAdminID)
	// The owner still gets the config they saved
	_, body = f.do(t, boardAdminID, http.MethodGet, "/widgets?board="+sharedBoardID, "")
	assert.Contains(t, body, "s3cret")

	status, body = f.do(t, boardUserID, http.MethodGet, "/dashboard/data?board="+sharedBoardID, "")
	require.Equal(t, 200, status)
	var snaps []models.Snapshot
	require.NoError(t, json.Unmarshal([]byte(body), &snaps))
	require.Len(t, snaps, 2, body)
	assert.Equal(t, sharedIntegrationID, snaps[0].SourceID)
	assert.Equal(t, sharedWidgetID, snaps[1].SourceID)
	assert.Equal(t, services.ViewerErrorMessage, snaps[1].Error)
	assert.NotContains(t, body, assert.AnError.Error())
	// The viewer's own data, without a board, is their own (nothing here)
	_, body = f.do(t, boardUserID, http.MethodGet, "/dashboard/data", "")
	assert.JSONEq(t, `[]`, body)

	status, body = f.do(t, boardUserID, http.MethodGet, "/boards/"+sharedBoardID+"/integrations", "")
	require.Equal(t, 200, status)
	assert.JSONEq(t, `[{"id":"`+sharedIntegrationID+`","kind":"handler-fake","options":{"library":true}}]`, body)

	// By ID, the admin's tiles stay the admin's
	for _, path := range []string{"/services/" + sharedServiceID, "/groups/" + sharedGroupID, "/widgets/" + sharedWidgetID} {
		status, _ := f.do(t, boardUserID, http.MethodGet, path, "")
		assert.Contains(t, []int{403, 404}, status, path)
	}

	// A viewer can't put their own tile on the shared board, nor use its group
	status, body = f.do(t, boardUserID, http.MethodPost, "/services", `{"name":"Mine","url":"http://mine.lan","group_id":"`+sharedGroupID+`"}`)
	assert.Equal(t, 403, status, body)
	assert.Equal(t, 0, f.count(t, `SELECT COUNT(*) FROM services WHERE user_id = ?`, boardUserID))
}

func TestBoards_SharedNeedsAnAdminOwner(t *testing.T) {
	f := newBoardFixture(t, time.Minute)
	// Demoted: the board is private again without anyone touching it
	_, err := f.db.Exec(`UPDATE users SET role = 'user' WHERE id = ?`, boardAdminID)
	require.NoError(t, err)

	status, _ := f.do(t, boardUserID, http.MethodGet, "/services?board="+sharedBoardID, "")
	assert.Equal(t, 404, status)
	status, _ = f.do(t, boardUserID, http.MethodGet, "/metrics/"+sharedServiceID, "")
	assert.Equal(t, 403, status)
	_, body := f.do(t, boardUserID, http.MethodGet, "/boards", "")
	assert.NotContains(t, body, "Family")
	// The owner keeps it
	status, _ = f.do(t, boardAdminID, http.MethodGet, "/services?board="+sharedBoardID, "")
	assert.Equal(t, 200, status)
}

func TestBoards_ManageOwnBoards(t *testing.T) {
	f := newBoardFixture(t, time.Minute)

	// A user creates boards; the slug follows the name and stays unique
	status, body := f.do(t, boardUserID, http.MethodPost, "/boards", `{"name":"Media Room"}`)
	require.Equal(t, 201, status, body)
	var media models.BoardResponse
	require.NoError(t, json.Unmarshal([]byte(body), &media))
	assert.Equal(t, "media-room", media.Slug)
	assert.True(t, media.Owned && media.CanEdit && !media.IsShared && !media.IsDefault)
	assert.Equal(t, 1, f.count(t, `SELECT COUNT(*) FROM groups WHERE board_id = ? AND is_default = 1`, media.ID), "a board starts with its default group")

	status, body = f.do(t, boardUserID, http.MethodPost, "/boards", `{"name":"Media room!"}`)
	require.Equal(t, 201, status, body)
	var second models.BoardResponse
	require.NoError(t, json.Unmarshal([]byte(body), &second))
	assert.Equal(t, "media-room-2", second.Slug)

	// The same slug as the admin's shared board is fine: slugs are per owner
	status, body = f.do(t, boardUserID, http.MethodPost, "/boards", `{"name":"Family","slug":"family"}`)
	require.Equal(t, 201, status, body)

	for name, bad := range map[string]string{
		"empty name":      `{"name":" "}`,
		"long name":       `{"name":"` + strings.Repeat("x", 51) + `"}`,
		"taken slug":      `{"name":"Again","slug":"media-room"}`,
		"uppercase slug":  `{"name":"A","slug":"Media"}`,
		"slug with slash": `{"name":"A","slug":"a/b"}`,
		"reserved slug":   `{"name":"A","slug":"new"}`,
		"slug like an ID": `{"name":"A","slug":"` + sharedBoardID + `"}`,
		"shared by user":  `{"name":"A","is_shared":true}`,
	} {
		status, body := f.do(t, boardUserID, http.MethodPost, "/boards", bad)
		assert.Equal(t, 400, status, "%s: %s", name, body)
	}

	// Tiles: created on a board, listed per board, moved between boards
	status, body = f.do(t, boardUserID, http.MethodPost, "/services", `{"name":"Plex","url":"http://plex.lan","board_id":"`+media.ID+`"}`)
	require.Equal(t, 201, status, body)
	var plex models.ServiceResponse
	require.NoError(t, json.Unmarshal([]byte(body), &plex))
	assert.Equal(t, media.ID, plex.BoardID)
	status, body = f.do(t, boardUserID, http.MethodPost, "/services", `{"name":"Router","url":"http://router.lan"}`)
	require.Equal(t, 201, status, body)

	_, body = f.do(t, boardUserID, http.MethodGet, "/services", "")
	assert.Contains(t, body, "Router")
	assert.NotContains(t, body, "Plex", "without a board the list is the default board, as before boards existed")
	_, body = f.do(t, boardUserID, http.MethodGet, "/services?board="+media.ID, "")
	assert.Contains(t, body, "Plex")
	assert.NotContains(t, body, "Router")

	// A group of another board is refused; a move drops the group
	status, body = f.do(t, boardUserID, http.MethodPost, "/groups", `{"name":"Films","board_id":"`+media.ID+`"}`)
	require.Equal(t, 201, status, body)
	var films models.GroupResponse
	require.NoError(t, json.Unmarshal([]byte(body), &films))
	status, body = f.do(t, boardUserID, http.MethodPost, "/services", `{"name":"X","url":"http://x.lan","group_id":"`+films.ID+`"}`)
	assert.Equal(t, 400, status, body)
	status, body = f.do(t, boardUserID, http.MethodPut, "/services/"+plex.ID, `{"name":"Plex","url":"http://plex.lan","group_id":"`+films.ID+`"}`)
	require.Equal(t, 200, status, body)
	status, body = f.do(t, boardUserID, http.MethodPut, "/services/"+plex.ID, `{"name":"Plex","url":"http://plex.lan","board_id":"`+second.ID+`"}`)
	require.Equal(t, 200, status, body)
	require.NoError(t, json.Unmarshal([]byte(body), &plex))
	assert.Equal(t, second.ID, plex.BoardID)
	assert.Nil(t, plex.GroupID)
	assert.Equal(t, 1, f.count(t, `SELECT COUNT(*) FROM services WHERE id = ? AND board_id = ? AND group_id IS NULL AND position = ?`, plex.ID, second.ID, plex.Position))

	status, body = f.do(t, boardUserID, http.MethodPost, "/widgets", `{"type":"clock","board_id":"`+media.ID+`","group_id":"`+films.ID+`"}`)
	require.Equal(t, 201, status, body)
	var clock models.Widget
	require.NoError(t, json.Unmarshal([]byte(body), &clock))
	status, body = f.do(t, boardUserID, http.MethodPut, "/widgets/"+clock.ID, `{"board_id":"`+second.ID+`"}`)
	require.Equal(t, 200, status, body)
	require.NoError(t, json.Unmarshal([]byte(body), &clock))
	assert.Equal(t, second.ID, clock.BoardID)
	assert.Nil(t, clock.GroupID)
	assert.Greater(t, clock.Position, plex.Position, "a moved tile goes to the end")
	status, body = f.do(t, boardUserID, http.MethodPut, "/widgets/"+clock.ID, `{"group_id":"`+films.ID+`"}`)
	assert.Equal(t, 400, status, body)
	status, body = f.do(t, boardUserID, http.MethodPost, "/widgets", `{"type":"clock","board_id":"`+sharedBoardID+`"}`)
	assert.Equal(t, 400, status, body)

	// Reorder, rename, share
	_, body = f.do(t, boardUserID, http.MethodGet, "/boards", "")
	var boards []models.BoardResponse
	require.NoError(t, json.Unmarshal([]byte(body), &boards))
	require.Len(t, boards, 5)
	home := boards[0]
	status, body = f.do(t, boardUserID, http.MethodPut, "/boards/reorder", `{"boards":["`+second.ID+`","`+home.ID+`","`+media.ID+`"]}`)
	require.Equal(t, 200, status, body)
	_, body = f.do(t, boardUserID, http.MethodGet, "/boards", "")
	require.NoError(t, json.Unmarshal([]byte(body), &boards))
	assert.Equal(t, second.ID, boards[0].ID)
	assert.Equal(t, sharedBoardID, boards[4].ID, "shared boards of others come last")

	status, body = f.do(t, boardUserID, http.MethodPut, "/boards/"+media.ID, `{"name":"Cinema"}`)
	require.Equal(t, 200, status, body)
	assert.Contains(t, body, `"slug":"media-room"`, "a rename keeps the URL")
	status, body = f.do(t, boardUserID, http.MethodPut, "/boards/"+media.ID, `{"is_shared":true}`)
	assert.Equal(t, 400, status, body)
	status, body = f.do(t, boardAdminID, http.MethodPut, "/boards/"+privateBoardID, `{"is_shared":true}`)
	assert.Equal(t, 200, status, body)
	_, body = f.do(t, boardAdminID, http.MethodGet, "/boards", "")
	require.NoError(t, json.Unmarshal([]byte(body), &boards))
	status, body = f.do(t, boardAdminID, http.MethodPut, "/boards/"+boards[0].ID, `{"is_shared":true}`)
	assert.Equal(t, 400, status, "the default board is never shared: %s", body)

	// Delete: the default board stays; tiles move or go
	status, body = f.do(t, boardUserID, http.MethodDelete, "/boards/"+home.ID, "")
	assert.Equal(t, 400, status, body)
	status, body = f.do(t, boardUserID, http.MethodDelete, "/boards/"+second.ID+"?move_to="+sharedBoardID, "")
	assert.Equal(t, 400, status, "tiles can't move to someone else's board: %s", body)
	status, body = f.do(t, boardUserID, http.MethodDelete, "/boards/"+second.ID+"?move_to="+media.ID, "")
	require.Equal(t, 204, status, body)
	assert.Equal(t, 1, f.count(t, `SELECT COUNT(*) FROM services WHERE id = ? AND board_id = ?`, plex.ID, media.ID))
	assert.Equal(t, 1, f.count(t, `SELECT COUNT(*) FROM widgets WHERE id = ? AND board_id = ?`, clock.ID, media.ID))
	assert.Equal(t, 0, f.count(t, `SELECT COUNT(*) FROM groups WHERE board_id = ?`, second.ID))
	status, body = f.do(t, boardUserID, http.MethodDelete, "/boards/"+media.ID, "")
	require.Equal(t, 204, status, body)
	assert.Equal(t, 0, f.count(t, `SELECT COUNT(*) FROM services WHERE id = ?`, plex.ID))
	assert.Equal(t, 0, f.count(t, `SELECT COUNT(*) FROM groups WHERE id = ?`, films.ID))
	assert.Equal(t, 1, f.count(t, `SELECT COUNT(*) FROM services WHERE user_id = ?`, boardUserID), "the default board's service stays")
}

func TestBoards_Limit(t *testing.T) {
	f := newBoardFixture(t, time.Minute)
	for i := 1; i < models.MaxBoardsPerUser; i++ { // the default board is the first
		status, body := f.do(t, boardUserID, http.MethodPost, "/boards", `{"name":"Board"}`)
		require.Equal(t, 201, status, body)
	}
	status, body := f.do(t, boardUserID, http.MethodPost, "/boards", `{"name":"One too many"}`)
	assert.Equal(t, 400, status, body)
}

func TestBoards_ViewerStream(t *testing.T) {
	f := newBoardFixture(t, 100*time.Millisecond)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go func() { _ = f.app.Listener(ln) }()
	// Closing the listener, not app.Shutdown: Shutdown writes a field of the
	// server that request contexts handed to database/sql read unsynchronized
	t.Cleanup(func() { _ = ln.Close() })

	req, err := http.NewRequest(http.MethodGet, "http://"+ln.Addr().String()+"/dashboard/stream?board="+sharedBoardID, nil)
	require.NoError(t, err)
	req.Header.Set("X-User", boardUserID)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, 200, resp.StatusCode)
	events := &sseReader{t: t, r: bufio.NewReader(resp.Body)}
	assert.Equal(t, "event: hello\ndata: {}", events.next())

	// nextEvent skips the pings
	nextEvent := func() string {
		for {
			if event := events.next(); event != ": ping" {
				return event
			}
		}
	}
	widget := func(id string) services.LiveSource {
		return services.LiveSource{Kind: models.SnapshotSourceWidget, ID: id, UserID: boardAdminID}
	}
	ms := 7

	// Only the shared board's tiles arrive, although the admin has more
	f.live.Record(widget(privateWidgetID), map[string]int{"secret": 1}, nil)
	f.live.PublishServiceStatus(boardAdminID, privateServiceID, models.StatusOffline, &ms)
	f.live.Record(services.LiveSource{Kind: models.SnapshotSourceIntegration, ID: privateIntegrationID, UserID: boardAdminID}, map[string]int{"vms": 9}, nil)
	f.live.Record(widget(sharedWidgetID), nil, assert.AnError)
	event := nextEvent()
	assert.Contains(t, event, `"source_id":"`+sharedWidgetID+`"`)
	assert.Contains(t, event, services.ViewerErrorMessage)
	assert.NotContains(t, event, assert.AnError.Error())
	f.live.PublishServiceStatus(boardAdminID, sharedServiceID, models.StatusOnline, &ms)
	assert.Equal(t, `event: service_status`+"\n"+`data: {"id":"`+sharedServiceID+`","status":"online","response_time":7}`, nextEvent())

	// The viewer's stream counts against the viewer, not the admin
	for range 19 {
		_, err := f.hub.Subscribe(boardUserID)
		require.NoError(t, err)
	}
	_, err = f.hub.Subscribe(boardUserID)
	assert.ErrorIs(t, err, services.ErrTooManyStreams)
	_, err = f.hub.Subscribe(boardAdminID)
	assert.NoError(t, err)

	// A tile that leaves the board stops arriving after the next ping
	_, err = f.db.Exec(`UPDATE widgets SET board_id = ? WHERE id = ?`, privateBoardID, sharedWidgetID)
	require.NoError(t, err)
	time.Sleep(300 * time.Millisecond)
	f.live.Record(widget(sharedWidgetID), map[string]int{"moved": 1}, nil)
	f.live.PublishServiceStatus(boardAdminID, sharedServiceID, models.StatusOffline, &ms)
	assert.Contains(t, nextEvent(), `"status":"offline"`, "the widget's snapshot was skipped")

	// Unsharing ends the stream
	_, err = f.db.Exec(`UPDATE boards SET is_shared = 0 WHERE id = ?`, sharedBoardID)
	require.NoError(t, err)
	ended := make(chan struct{})
	go func() {
		for {
			if _, err := events.r.ReadString('\n'); err != nil {
				close(ended)
				return
			}
		}
	}()
	select {
	case <-ended:
	case <-time.After(3 * time.Second):
		t.Fatal("the stream stayed open after the board was unshared")
	}
}

// Board routes behind the real auth middleware, guarded as in main.go
func TestBoards_TokenScopes(t *testing.T) {
	_, db := setupScopeTestApp(t)
	auth := middleware.AuthMiddleware(services.NewAuthService(), repository.NewUserRepository(db), repository.NewAPITokenRepository(db), models.ScopeReadServices)
	sessionOnly := middleware.RequireSessionAuth()
	handler := NewBoardHandler(services.NewBoardService(repository.NewBoardRepository(db)))
	app := fiber.New()
	boards := app.Group("/boards", auth)
	boards.Get("/", handler.ListBoards)
	boards.Post("/", sessionOnly, handler.CreateBoard)
	boards.Put("/reorder", sessionOnly, handler.ReorderBoards)
	boards.Put("/:id", sessionOnly, handler.UpdateBoard)
	boards.Delete("/:id", sessionOnly, handler.DeleteBoard)
	boards.Get("/:id/integrations", sessionOnly, handler.ListBoardIntegrations)

	read := insertScopeTestToken(t, db, []string{models.ScopeReadServices}, nil)
	resp := scopeRequest(t, app, http.MethodGet, "/boards", read, "")
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
	var list []models.BoardResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&list))
	require.Len(t, list, 1)
	id := list[0].ID

	// Managing boards is for a logged-in person, whatever the token may write
	write := insertScopeTestToken(t, db, []string{models.ScopeWriteServices}, nil)
	for _, token := range []string{read, write} {
		assert.Equal(t, fiber.StatusForbidden, scopeRequest(t, app, http.MethodPost, "/boards", token, `{"name":"Media"}`).StatusCode)
		assert.Equal(t, fiber.StatusForbidden, scopeRequest(t, app, http.MethodPut, "/boards/reorder", token, `{"boards":["`+id+`"]}`).StatusCode)
		assert.Equal(t, fiber.StatusForbidden, scopeRequest(t, app, http.MethodPut, "/boards/"+id, token, `{"name":"Mine"}`).StatusCode)
		assert.Equal(t, fiber.StatusForbidden, scopeRequest(t, app, http.MethodDelete, "/boards/"+id, token, "").StatusCode)
		assert.Equal(t, fiber.StatusForbidden, scopeRequest(t, app, http.MethodGet, "/boards/"+id+"/integrations", token, "").StatusCode)
	}
	other := insertScopeTestToken(t, db, []string{models.ScopeReadMetrics, models.ScopeReadWidgets, models.ScopeMCP}, nil)
	assert.Equal(t, fiber.StatusForbidden, scopeRequest(t, app, http.MethodGet, "/boards", other, "").StatusCode)
	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM boards`).Scan(&count))
	assert.Equal(t, 1, count)
}

// The MCP tools read a board through services.BoardReader
func TestBoards_ReaderGivesAViewerTheViewerVersion(t *testing.T) {
	f := newBoardFixture(t, time.Minute)
	ctx := context.Background()
	f.live.Record(services.LiveSource{Kind: models.SnapshotSourceWidget, ID: sharedWidgetID, UserID: boardAdminID}, map[string]int{"n": 1}, nil)
	f.live.Record(services.LiveSource{Kind: models.SnapshotSourceWidget, ID: privateWidgetID, UserID: boardAdminID}, map[string]int{"n": 2}, nil)
	groupRepo := repository.NewGroupRepository(f.db)
	widgets := services.NewWidgetService(repository.NewWidgetRepository(f.db), groupRepo, repository.NewIntegrationRepository(f.db))
	reader := &services.BoardReader{
		Boards:   services.NewBoardService(repository.NewBoardRepository(f.db)),
		Services: repository.NewServiceRepository(f.db), Groups: groupRepo, Widgets: widgets, Live: f.live,
	}

	for _, ref := range []string{sharedBoardID, "family", "Family"} {
		content, err := reader.Read(ctx, boardUserID, ref)
		require.NoError(t, err, ref)
		assert.False(t, content.CanEdit)
		require.Len(t, content.Services, 1)
		assert.Equal(t, "Jellyfin", content.Services[0].Name)
		require.Len(t, content.Widgets, 1)
		assert.NotContains(t, string(content.Widgets[0].Config), "s3cret")
		require.Len(t, content.Snapshots, 1)
		assert.Equal(t, sharedWidgetID, content.Snapshots[0].SourceID)
	}
	for _, ref := range []string{privateBoardID, "lab", "Lab", "nope"} {
		_, err := reader.Read(ctx, boardUserID, ref)
		assert.ErrorIs(t, err, repository.ErrBoardNotFound, ref)
	}

	own, err := reader.Read(ctx, boardAdminID, "lab")
	require.NoError(t, err)
	assert.True(t, own.CanEdit)
	assert.Len(t, own.Snapshots, 2, "the owner gets all their data")
	mine, err := reader.Read(ctx, boardUserID, "")
	require.NoError(t, err)
	assert.True(t, mine.Board.IsDefault && mine.CanEdit)
	assert.Empty(t, mine.Services)
}
