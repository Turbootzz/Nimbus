package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/services"
	"github.com/nimbus/backend/internal/widgets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	widgetOwnerID     = integrationOwnerID
	widgetOtherID     = integrationOtherID
	ownerGroupID      = "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeee1"
	otherGroupID      = "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeee2"
	ownerServiceID    = "ffffffff-ffff-ffff-ffff-fffffffffff1"
	otherServiceID    = "ffffffff-ffff-ffff-ffff-fffffffffff2"
	ownerIntegration  = "abababab-abab-abab-abab-ababababab01"
	otherIntegration  = "abababab-abab-abab-abab-ababababab02"
	ownerWrongKindInt = "abababab-abab-abab-abab-ababababab03"
)

// handlerQueueWidget needs a handler-fake integration, like a Sonarr queue
type handlerQueueWidget struct{}

func (handlerQueueWidget) Type() string { return "handler-queue" }
func (handlerQueueWidget) Meta() widgets.Meta {
	return widgets.Meta{
		Name:              "Queue",
		Category:          widgets.CategoryGeneral,
		DefaultSize:       models.CardSize2x1,
		AllowedSizes:      []string{models.CardSize2x1},
		IntegrationKinds:  []string{"handler-fake"},
		MinRefreshSeconds: 60,
	}
}
func (handlerQueueWidget) Validate(c json.RawMessage) (json.RawMessage, error) { return c, nil }
func (handlerQueueWidget) Fetch(context.Context, *widgets.FetchRequest) (any, error) {
	return map[string]int{"queued": 3}, nil
}

func init() {
	widgets.Register(handlerQueueWidget{})
}

// setupWidgetTestDB extends the integration schema with the tables widgets
// touch; widgets mirrors migration 000027 for SQLite
func setupWidgetTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db := setupIntegrationTestDB(t)
	schema := `
		CREATE TABLE groups (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			name TEXT NOT NULL,
			color TEXT DEFAULT '#0ea5e9',
			position INTEGER DEFAULT 0,
			is_default INTEGER DEFAULT 0,
			monitoring_enabled INTEGER DEFAULT 1,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE services (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			position INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP
		);
		CREATE TABLE widgets (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			type TEXT NOT NULL,
			title TEXT NOT NULL DEFAULT '',
			group_id TEXT REFERENCES groups(id) ON DELETE SET NULL,
			integration_id TEXT REFERENCES integrations(id) ON DELETE SET NULL,
			config TEXT NOT NULL DEFAULT '{}',
			card_size TEXT NOT NULL DEFAULT '2x1',
			position INTEGER NOT NULL DEFAULT 0,
			refresh_seconds INTEGER NOT NULL DEFAULT 300 CHECK (refresh_seconds >= 10),
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		INSERT INTO groups (id, user_id, name) VALUES
			('eeeeeeee-eeee-eeee-eeee-eeeeeeeeeee1', 'dddddddd-dddd-dddd-dddd-ddddddddddd1', 'Media'),
			('eeeeeeee-eeee-eeee-eeee-eeeeeeeeeee2', 'dddddddd-dddd-dddd-dddd-ddddddddddd2', 'Other');
		INSERT INTO services (id, user_id, position) VALUES
			('ffffffff-ffff-ffff-ffff-fffffffffff1', 'dddddddd-dddd-dddd-dddd-ddddddddddd1', 3),
			('ffffffff-ffff-ffff-ffff-fffffffffff2', 'dddddddd-dddd-dddd-dddd-ddddddddddd2', 0);
		INSERT INTO integrations (id, user_id, kind, name, base_url) VALUES
			('abababab-abab-abab-abab-ababababab01', 'dddddddd-dddd-dddd-dddd-ddddddddddd1', 'handler-fake', 'Sonarr', 'http://sonarr.lan'),
			('abababab-abab-abab-abab-ababababab02', 'dddddddd-dddd-dddd-dddd-ddddddddddd2', 'handler-fake', 'Sonarr', 'http://sonarr.lan'),
			('abababab-abab-abab-abab-ababababab03', 'dddddddd-dddd-dddd-dddd-ddddddddddd1', 'other-kind', 'Proxmox', 'http://pve.lan');
	`
	_, err := db.Exec(schema)
	require.NoError(t, err)
	return db
}

// setupWidgetTestApp mirrors the routes in main.go with a stub auth
// middleware that sets the given user ID
func setupWidgetTestApp(db *sql.DB, userID string) *fiber.App {
	return setupWidgetTestAppAs(db, userID, "user")
}

func setupWidgetTestAppAs(db *sql.DB, userID, role string) *fiber.App {
	service := services.NewWidgetService(
		repository.NewWidgetRepository(db),
		repository.NewGroupRepository(db),
		repository.NewIntegrationRepository(db),
	)
	handler := NewWidgetHandler(service)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", userID)
		c.Locals("role", role)
		return c.Next()
	})
	app.Get("/widgets/types", handler.ListTypes)
	app.Post("/widgets", handler.CreateWidget)
	app.Get("/widgets", handler.ListWidgets)
	app.Get("/widgets/:id", handler.GetWidget)
	app.Put("/widgets/:id", handler.UpdateWidget)
	app.Delete("/widgets/:id", handler.DeleteWidget)
	app.Put("/dashboard/reorder", handler.ReorderTiles)
	return app
}

func doWidgetRequest(t *testing.T, app *fiber.App, method, path, body string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, string(raw)
}

func createWidget(t *testing.T, app *fiber.App, body string) models.Widget {
	t.Helper()
	status, resp := doWidgetRequest(t, app, http.MethodPost, "/widgets", body)
	require.Equal(t, fiber.StatusCreated, status, resp)
	var widget models.Widget
	require.NoError(t, json.Unmarshal([]byte(resp), &widget))
	return widget
}

func TestWidgetHandler_ListTypes(t *testing.T) {
	app := setupWidgetTestApp(setupWidgetTestDB(t), widgetOwnerID)

	status, body := doWidgetRequest(t, app, http.MethodGet, "/widgets/types", "")
	require.Equal(t, fiber.StatusOK, status)

	var types []widgets.Meta
	require.NoError(t, json.Unmarshal([]byte(body), &types))
	byType := map[string]widgets.Meta{}
	for _, meta := range types {
		byType[meta.Type] = meta
	}
	for _, typ := range []string{"clock", "markdown", "bookmarks", "iframe"} {
		assert.True(t, byType[typ].Static, typ)
	}
	assert.Equal(t, []string{"handler-fake"}, byType["handler-queue"].IntegrationKinds)
}

func TestWidgetHandler_CRUD(t *testing.T) {
	app := setupWidgetTestApp(setupWidgetTestDB(t), widgetOwnerID)

	created := createWidget(t, app, `{"type":"markdown","title":" Notes ","group_id":"`+ownerGroupID+`","config":{"content":"# Hi","junk":true}}`)
	assert.Equal(t, "markdown", created.Type)
	assert.Equal(t, "Notes", created.Title)
	assert.Equal(t, models.CardSize2x2, created.CardSize, "type default size")
	assert.Equal(t, 300, created.RefreshSeconds)
	assert.True(t, created.Enabled)
	assert.Equal(t, 4, created.Position, "after the owner's service at 3")
	assert.Equal(t, ownerGroupID, *created.GroupID)
	assert.JSONEq(t, `{"content":"# Hi"}`, string(created.Config), "config is normalised")

	status, body := doWidgetRequest(t, app, http.MethodGet, "/widgets/"+created.ID, "")
	require.Equal(t, fiber.StatusOK, status)
	assert.Contains(t, body, `"title":"Notes"`)

	// Partial update keeps everything that is left out
	status, body = doWidgetRequest(t, app, http.MethodPut, "/widgets/"+created.ID, `{"card_size":"1x1","group_id":""}`)
	require.Equal(t, fiber.StatusOK, status, body)
	var updated models.Widget
	require.NoError(t, json.Unmarshal([]byte(body), &updated))
	assert.Equal(t, models.CardSize1x1, updated.CardSize)
	assert.Nil(t, updated.GroupID, "empty group_id clears the group")
	assert.Equal(t, "Notes", updated.Title)
	assert.JSONEq(t, `{"content":"# Hi"}`, string(updated.Config))

	status, body = doWidgetRequest(t, app, http.MethodPut, "/widgets/"+created.ID, `{"config":{"content":"new"},"enabled":false}`)
	require.Equal(t, fiber.StatusOK, status, body)
	assert.Contains(t, body, `"content":"new"`)
	assert.Contains(t, body, `"enabled":false`)

	status, body = doWidgetRequest(t, app, http.MethodPut, "/widgets/"+created.ID, `{"type":"clock"}`)
	assert.Equal(t, fiber.StatusBadRequest, status)
	assert.Contains(t, body, "Type cannot be changed")

	status, body = doWidgetRequest(t, app, http.MethodGet, "/widgets", "")
	require.Equal(t, fiber.StatusOK, status)
	var list []models.Widget
	require.NoError(t, json.Unmarshal([]byte(body), &list))
	assert.Len(t, list, 1)

	status, _ = doWidgetRequest(t, app, http.MethodDelete, "/widgets/"+created.ID, "")
	assert.Equal(t, fiber.StatusNoContent, status)
	status, _ = doWidgetRequest(t, app, http.MethodGet, "/widgets/"+created.ID, "")
	assert.Equal(t, fiber.StatusNotFound, status)
}

func TestWidgetHandler_EmptyListIsArray(t *testing.T) {
	app := setupWidgetTestApp(setupWidgetTestDB(t), widgetOwnerID)
	status, body := doWidgetRequest(t, app, http.MethodGet, "/widgets", "")
	require.Equal(t, fiber.StatusOK, status)
	assert.Equal(t, "[]", body)
}

func TestWidgetHandler_CreateValidation(t *testing.T) {
	app := setupWidgetTestApp(setupWidgetTestDB(t), widgetOwnerID)

	cases := map[string]struct {
		body    string
		wantErr string
	}{
		"invalid json":            {`{`, "Invalid request body"},
		"missing type":            {`{}`, "Unknown widget type"},
		"unknown type":            {`{"type":"doom"}`, "Unknown widget type"},
		"config not an object":    {`{"type":"clock","config":[1]}`, "config must be a JSON object"},
		"type rejects config":     {`{"type":"clock","config":{"timezone":"Mars/Olympus"}}`, "unknown timezone"},
		"required field missing":  {`{"type":"iframe"}`, "Invalid config: invalid URL"},
		"title too long":          {`{"type":"clock","title":"` + strings.Repeat("a", 101) + `"}`, "Title must be 100"},
		"unknown card size":       {`{"type":"clock","card_size":"3x3"}`, "not available for Clock"},
		"clock is not tall":       {`{"type":"clock","card_size":"1x2"}`, "not available for Clock"},
		"embed one row high":      {`{"type":"iframe","card_size":"1x1","config":{"url":"https://a.test"}}`, "not available for Embed"},
		"size not allowed":        {`{"type":"handler-queue","card_size":"2x2","integration_id":"` + ownerIntegration + `"}`, "not available for Queue"},
		"refresh below minimum":   {`{"type":"clock","refresh_seconds":5}`, "between 10 and"},
		"refresh below type min":  {`{"type":"handler-queue","refresh_seconds":30,"integration_id":"` + ownerIntegration + `"}`, "between 60 and"},
		"group of other user":     {`{"type":"clock","group_id":"` + otherGroupID + `"}`, "Group not found"},
		"group not a uuid":        {`{"type":"clock","group_id":"media"}`, "Group not found"},
		"integration on static":   {`{"type":"clock","integration_id":"` + ownerIntegration + `"}`, "don't use an integration"},
		"integration missing":     {`{"type":"handler-queue"}`, "need an integration"},
		"integration other user":  {`{"type":"handler-queue","integration_id":"` + otherIntegration + `"}`, "Integration not found"},
		"integration not a uuid":  {`{"type":"handler-queue","integration_id":"x"}`, "Integration not found"},
		"integration wrong kind":  {`{"type":"handler-queue","integration_id":"` + ownerWrongKindInt + `"}`, "can't use a other-kind integration"},
		"config too large":        {`{"type":"markdown","config":{"content":"` + strings.Repeat("a", 70*1024) + `"}}`, "Config is too large"},
		"bookmark javascript url": {`{"type":"bookmarks","config":{"items":[{"name":"x","url":"javascript:alert(1)"}]}}`, "invalid URL"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			status, body := doWidgetRequest(t, app, http.MethodPost, "/widgets", tc.body)
			assert.Equal(t, fiber.StatusBadRequest, status, body)
			assert.Contains(t, body, tc.wantErr)
		})
	}

	status, body := doWidgetRequest(t, app, http.MethodGet, "/widgets", "")
	require.Equal(t, fiber.StatusOK, status)
	assert.Equal(t, "[]", body, "nothing was stored")
}

func TestWidgetHandler_IntegrationWidget(t *testing.T) {
	db := setupWidgetTestDB(t)
	app := setupWidgetTestApp(db, widgetOwnerID)

	created := createWidget(t, app, `{"type":"handler-queue","integration_id":"`+ownerIntegration+`"}`)
	assert.Equal(t, ownerIntegration, *created.IntegrationID)
	assert.Equal(t, 300, created.RefreshSeconds)

	// Deleting the integration leaves the widget without one; it can still
	// be resized, but not saved with a missing integration explicitly
	_, err := db.Exec(`DELETE FROM integrations WHERE id = ?`, ownerIntegration)
	require.NoError(t, err)
	status, body := doWidgetRequest(t, app, http.MethodPut, "/widgets/"+created.ID, `{"title":"Downloads"}`)
	assert.Equal(t, fiber.StatusOK, status, body)
	status, body = doWidgetRequest(t, app, http.MethodPut, "/widgets/"+created.ID, `{"integration_id":""}`)
	assert.Equal(t, fiber.StatusBadRequest, status)
	assert.Contains(t, body, "need an integration")
}

func TestWidgetHandler_OtherUsersWidget(t *testing.T) {
	db := setupWidgetTestDB(t)
	owner := setupWidgetTestApp(db, widgetOwnerID)
	other := setupWidgetTestApp(db, widgetOtherID)

	created := createWidget(t, owner, `{"type":"clock"}`)
	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		status, body := doWidgetRequest(t, other, method, "/widgets/"+created.ID, `{"title":"x"}`)
		assert.Equal(t, fiber.StatusNotFound, status, method)
		assert.Contains(t, body, "Widget not found", method)
	}

	status, body := doWidgetRequest(t, other, http.MethodGet, "/widgets", "")
	require.Equal(t, fiber.StatusOK, status)
	assert.Equal(t, "[]", body)
}

func TestWidgetHandler_ReorderTiles(t *testing.T) {
	db := setupWidgetTestDB(t)
	app := setupWidgetTestApp(db, widgetOwnerID)
	widget := createWidget(t, app, `{"type":"clock"}`)

	status, body := doWidgetRequest(t, app, http.MethodPut, "/dashboard/reorder",
		`{"tiles":[{"id":"`+widget.ID+`","kind":"widget","position":0},{"id":"`+ownerServiceID+`","kind":"service","position":1}]}`)
	require.Equal(t, fiber.StatusOK, status, body)

	var servicePos int
	require.NoError(t, db.QueryRow(`SELECT position FROM services WHERE id = ?`, ownerServiceID).Scan(&servicePos))
	assert.Equal(t, 1, servicePos)
	status, body = doWidgetRequest(t, app, http.MethodGet, "/widgets/"+widget.ID, "")
	require.Equal(t, fiber.StatusOK, status)
	assert.Contains(t, body, `"position":0`)

	tile := func(id, kind string, pos int) string {
		b, _ := json.Marshal(models.TilePosition{ID: id, Kind: kind, Position: pos})
		return string(b)
	}
	cases := map[string]struct {
		body       string
		wantStatus int
		wantErr    string
	}{
		"invalid json":  {`{`, fiber.StatusBadRequest, "Invalid request body"},
		"no tiles":      {`{"tiles":[]}`, fiber.StatusBadRequest, "At least one tile"},
		"unknown kind":  {`{"tiles":[` + tile(widget.ID, "group", 0) + `]}`, fiber.StatusBadRequest, "service or widget"},
		"not a uuid":    {`{"tiles":[` + tile("abc", "widget", 0) + `]}`, fiber.StatusBadRequest, "must be a UUID"},
		"negative":      {`{"tiles":[` + tile(widget.ID, "widget", -1) + `]}`, fiber.StatusBadRequest, "between 0 and"},
		"too large":     {`{"tiles":[` + tile(widget.ID, "widget", 2147483647) + `]}`, fiber.StatusBadRequest, "between 0 and"},
		"duplicate":     {`{"tiles":[` + tile(widget.ID, "widget", 0) + `,` + tile(strings.ToUpper(widget.ID), "widget", 1) + `]}`, fiber.StatusBadRequest, "only appear once"},
		"other's tile":  {`{"tiles":[` + tile(widget.ID, "widget", 5) + `,` + tile(otherServiceID, "service", 6) + `]}`, fiber.StatusNotFound, "not found"},
		"wrong kind id": {`{"tiles":[` + tile(widget.ID, "service", 5) + `]}`, fiber.StatusNotFound, "not found"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			status, body := doWidgetRequest(t, app, http.MethodPut, "/dashboard/reorder", tc.body)
			assert.Equal(t, tc.wantStatus, status, body)
			assert.Contains(t, body, tc.wantErr)
		})
	}

	status, body = doWidgetRequest(t, app, http.MethodGet, "/widgets/"+widget.ID, "")
	require.Equal(t, fiber.StatusOK, status)
	assert.Contains(t, body, `"position":0`, "failed reorders change nothing")
}

func TestWidgetHandler_TallSize(t *testing.T) {
	app := setupWidgetTestApp(setupWidgetTestDB(t), widgetOwnerID)

	note := createWidget(t, app, `{"type":"markdown","card_size":"1x2","config":{"content":"a"}}`)
	assert.Equal(t, "1x2", note.CardSize)
	embed := createWidget(t, app, `{"type":"iframe","config":{"url":"https://a.test"}}`)
	assert.Equal(t, "2x2", embed.CardSize, "embeds default to 2x2")
	status, body := doWidgetRequest(t, app, http.MethodPut, "/widgets/"+embed.ID, `{"card_size":"1x2"}`)
	assert.Equal(t, fiber.StatusOK, status, body)
}

func TestWidgetHandler_AdminOnlyType(t *testing.T) {
	db := setupWidgetTestDB(t)
	user := setupWidgetTestAppAs(db, widgetOwnerID, "user")
	admin := setupWidgetTestAppAs(db, widgetOwnerID, "admin")
	config := `{"url":"http://10.0.0.2/api","headers":[{"name":"X-Api-Key","value":"k1"}],"fields":[{"label":"A","path":"a"}]}`

	typesOf := func(app *fiber.App) map[string]widgets.Meta {
		_, body := doWidgetRequest(t, app, http.MethodGet, "/widgets/types", "")
		var types []widgets.Meta
		require.NoError(t, json.Unmarshal([]byte(body), &types))
		byType := map[string]widgets.Meta{}
		for _, meta := range types {
			byType[meta.Type] = meta
		}
		return byType
	}
	assert.NotContains(t, typesOf(user), "custom_api")
	assert.True(t, typesOf(admin)["custom_api"].AdminOnly)

	status, body := doWidgetRequest(t, user, http.MethodPost, "/widgets", `{"type":"custom_api","config":`+config+`}`)
	assert.Equal(t, fiber.StatusBadRequest, status)
	assert.Contains(t, body, "Only admins")

	created := createWidget(t, admin, `{"type":"custom_api","config":`+config+`}`)

	// A user (e.g. a demoted admin) can still move it, not change what it fetches
	status, body = doWidgetRequest(t, user, http.MethodPut, "/widgets/"+created.ID, `{"title":"Stats"}`)
	assert.Equal(t, fiber.StatusOK, status, body)
	status, _ = doWidgetRequest(t, user, http.MethodPut, "/widgets/"+created.ID, `{"config":`+config+`}`)
	assert.Equal(t, fiber.StatusBadRequest, status)
}

func TestWidgetHandler_SecretsAreMasked(t *testing.T) {
	db := setupWidgetTestDB(t)
	app := setupWidgetTestAppAs(db, widgetOwnerID, "admin")
	storedConfig := func(id string) string {
		var config string
		require.NoError(t, db.QueryRow(`SELECT config FROM widgets WHERE id = ?`, id).Scan(&config))
		return config
	}

	created := createWidget(t, app, `{"type":"custom_api","config":{"url":"http://10.0.0.2/api","headers":[{"name":"X-Api-Key","value":"k1"}],"fields":[{"label":"A","path":"a"}]}}`)
	assert.NotContains(t, string(created.Config), "k1")
	assert.Contains(t, string(created.Config), `"value":"********"`)
	assert.Contains(t, storedConfig(created.ID), "k1")

	for _, path := range []string{"/widgets", "/widgets/" + created.ID} {
		_, body := doWidgetRequest(t, app, http.MethodGet, path, "")
		assert.NotContains(t, body, "k1", path)
	}

	// Saving the masked config with another label keeps the stored key
	status, body := doWidgetRequest(t, app, http.MethodPut, "/widgets/"+created.ID,
		`{"config":{"url":"http://10.0.0.2/api","headers":[{"name":"X-Api-Key","value":"********"}],"fields":[{"label":"B","path":"a"}]}}`)
	require.Equal(t, fiber.StatusOK, status, body)
	assert.NotContains(t, body, "k1")
	assert.Contains(t, storedConfig(created.ID), "k1")
	assert.Contains(t, storedConfig(created.ID), `"label":"B"`)
}

func TestWidgetHandler_CalendarIntegrations(t *testing.T) {
	db := setupWidgetTestDB(t)
	_, err := db.Exec(`INSERT INTO integrations (id, user_id, kind, name, base_url) VALUES
		('abababab-abab-abab-abab-ababababab04', ?, 'sonarr', 'Sonarr', 'http://sonarr.lan')`, widgetOwnerID)
	require.NoError(t, err)
	app := setupWidgetTestApp(db, widgetOwnerID)
	create := func(id string) (int, string) {
		return doWidgetRequest(t, app, http.MethodPost, "/widgets", `{"type":"calendar","config":{"integrations":["`+id+`"]}}`)
	}

	status, body := create("abababab-abab-abab-abab-ababababab04")
	assert.Equal(t, fiber.StatusCreated, status, body)

	status, body = create(otherIntegration) // another user's
	assert.Equal(t, fiber.StatusBadRequest, status)
	assert.Contains(t, body, "Integration not found")

	status, body = create(ownerWrongKindInt) // not Sonarr or Radarr
	assert.Equal(t, fiber.StatusBadRequest, status)
	assert.Contains(t, body, "can't use a other-kind integration")
}
