package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupServiceIntegrationDB has everything creating a service touches:
// users, integrations (from setupIntegrationTestDB), groups, widgets and a
// full services table
func setupServiceIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()
	db := setupIntegrationTestDB(t)
	_, err := db.Exec(`
		CREATE TABLE groups (id TEXT PRIMARY KEY, user_id TEXT NOT NULL, name TEXT NOT NULL,
			color TEXT, position INTEGER DEFAULT 0, is_default INTEGER DEFAULT 0,
			monitoring_enabled INTEGER DEFAULT 1, created_at TIMESTAMP, updated_at TIMESTAMP);
		CREATE TABLE widgets (id TEXT PRIMARY KEY, user_id TEXT NOT NULL, position INTEGER NOT NULL DEFAULT 0);
		CREATE TABLE services (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			user_id TEXT NOT NULL,
			name TEXT NOT NULL,
			url TEXT NOT NULL,
			icon TEXT,
			icon_type TEXT DEFAULT 'emoji',
			icon_image_path TEXT DEFAULT '',
			description TEXT,
			status TEXT NOT NULL,
			response_time INTEGER,
			position INTEGER DEFAULT 0,
			card_size TEXT DEFAULT '2x1',
			group_id TEXT,
			integration_id TEXT REFERENCES integrations(id) ON DELETE SET NULL,
			monitoring_enabled INTEGER NOT NULL DEFAULT 1,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		);
		INSERT INTO integrations (id, user_id, kind, name, base_url) VALUES
			('abababab-abab-abab-abab-ababababab01', 'dddddddd-dddd-dddd-dddd-ddddddddddd1', 'sonarr', 'Sonarr', 'http://sonarr.lan'),
			('abababab-abab-abab-abab-ababababab02', 'dddddddd-dddd-dddd-dddd-ddddddddddd2', 'sonarr', 'Sonarr', 'http://sonarr.lan');
	`)
	require.NoError(t, err)
	return db
}

func setupServiceIntegrationApp(db *sql.DB, poller *recordingPoller) *fiber.App {
	handler := NewServiceHandler(repository.NewServiceRepository(db), repository.NewGroupRepository(db), nil, repository.NewIntegrationRepository(db))
	handler.SetPoller(poller)
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", integrationOwnerID)
		return c.Next()
	})
	app.Post("/services", handler.CreateService)
	app.Put("/services/:id", handler.UpdateService)
	app.Delete("/services/:id", handler.DeleteService)
	return app
}

func doServiceRequest(t *testing.T, app *fiber.App, method, path, body string) (int, models.ServiceResponse, string) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()
	var raw json.RawMessage
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&raw))
	var service models.ServiceResponse
	_ = json.Unmarshal(raw, &service)
	return resp.StatusCode, service, string(raw)
}

func TestServiceHandler_IntegrationLink(t *testing.T) {
	db := setupServiceIntegrationDB(t)
	poller := &recordingPoller{}
	app := setupServiceIntegrationApp(db, poller)

	// Create linked
	status, created, body := doServiceRequest(t, app, http.MethodPost, "/services",
		`{"name":"Sonarr","url":"http://sonarr.lan","integration_id":"`+ownerIntegration+`"}`)
	require.Equal(t, fiber.StatusCreated, status, body)
	require.NotNil(t, created.IntegrationID)
	assert.Equal(t, ownerIntegration, *created.IntegrationID)
	assert.Equal(t, 1, poller.kicks, "the poller starts fetching the integration")

	path := "/services/" + created.ID
	base := `"name":"Sonarr","url":"http://sonarr.lan"`

	// Omitted keeps the link, and doesn't kick
	status, updated, body := doServiceRequest(t, app, http.MethodPut, path, `{`+base+`,"description":"TV"}`)
	require.Equal(t, fiber.StatusOK, status, body)
	require.NotNil(t, updated.IntegrationID)
	assert.Equal(t, 1, poller.kicks)

	// Empty string unlinks
	status, updated, body = doServiceRequest(t, app, http.MethodPut, path, `{`+base+`,"integration_id":""}`)
	require.Equal(t, fiber.StatusOK, status, body)
	assert.Nil(t, updated.IntegrationID)
	assert.Equal(t, 2, poller.kicks)
	assert.NotContains(t, body, "integration_id", "omitted when unlinked")

	// Linking again with different casing
	status, updated, _ = doServiceRequest(t, app, http.MethodPut, path, `{`+base+`,"integration_id":"`+strings.ToUpper(ownerIntegration)+`"}`)
	require.Equal(t, fiber.StatusOK, status)
	require.NotNil(t, updated.IntegrationID)
	assert.Equal(t, 3, poller.kicks)

	status, _, body = doServiceRequest(t, app, http.MethodDelete, path, "")
	require.Equal(t, fiber.StatusOK, status, body)
	assert.Equal(t, 4, poller.kicks, "deleting a linked service kicks too")
}

func TestServiceHandler_IntegrationLinkValidation(t *testing.T) {
	app := setupServiceIntegrationApp(setupServiceIntegrationDB(t), &recordingPoller{})

	cases := map[string]struct {
		id      string
		wantErr string
	}{
		"not a uuid":         {"sonarr", "must be a UUID"},
		"another user's":     {otherIntegration, "integration not found"},
		"does not exist":     {"abababab-abab-abab-abab-ababababab99", "integration not found"},
		"whitespace is none": {"  ", ""},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			status, created, body := doServiceRequest(t, app, http.MethodPost, "/services",
				`{"name":"S","url":"http://s.lan","integration_id":"`+tc.id+`"}`)
			if tc.wantErr == "" {
				require.Equal(t, fiber.StatusCreated, status, body)
				assert.Nil(t, created.IntegrationID)
				return
			}
			assert.Equal(t, fiber.StatusBadRequest, status)
			assert.Contains(t, body, tc.wantErr)
		})
	}
}

func TestServiceHandler_IntegrationDeleteUnlinks(t *testing.T) {
	db := setupServiceIntegrationDB(t)
	_, err := db.Exec(`PRAGMA foreign_keys = ON`)
	require.NoError(t, err)
	app := setupServiceIntegrationApp(db, &recordingPoller{})

	status, created, body := doServiceRequest(t, app, http.MethodPost, "/services",
		`{"name":"Sonarr","url":"http://sonarr.lan","integration_id":"`+ownerIntegration+`"}`)
	require.Equal(t, fiber.StatusCreated, status, body)

	_, err = db.Exec(`DELETE FROM integrations WHERE id = ?`, ownerIntegration)
	require.NoError(t, err)
	service, err := repository.NewServiceRepository(db).GetByID(t.Context(), created.ID)
	require.NoError(t, err)
	assert.Nil(t, service.IntegrationID)
}
