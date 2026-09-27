package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/integrations"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/services"
	"github.com/nimbus/backend/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

const (
	integrationOwnerID = "dddddddd-dddd-dddd-dddd-ddddddddddd1"
	integrationOtherID = "dddddddd-dddd-dddd-dddd-ddddddddddd2"
	integrationSecret  = "handler-s3cret/key+value"
)

// handlerFakeKind pings BaseURL/ping with the API key in the query string
type handlerFakeKind struct{}

func (handlerFakeKind) Kind() string { return "handler-fake" }
func (handlerFakeKind) Meta() integrations.Meta {
	return integrations.Meta{
		Name:      "Handler Fake",
		AuthTypes: []string{models.IntegrationAuthAPIKey, models.IntegrationAuthNone},
		KPIs:      []integrations.KPI{{Key: "queued", Label: "Queued"}},
	}
}
func (handlerFakeKind) Test(ctx context.Context, conn *integrations.Conn) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, conn.BaseURL+"/ping?apikey="+url.QueryEscape(conn.Creds.APIKey), nil)
	if err != nil {
		return err
	}
	resp, err := conn.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}
func (handlerFakeKind) Fetch(context.Context, *integrations.Conn) (*integrations.Payload, error) {
	return &integrations.Payload{}, nil
}

func init() {
	integrations.Register(handlerFakeKind{})
}

func setupIntegrationTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	// Mirrors migration 000026 for SQLite
	schema := `
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY
		);

		CREATE TABLE IF NOT EXISTS integrations (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			kind TEXT NOT NULL,
			name TEXT NOT NULL,
			base_url TEXT NOT NULL,
			auth_type TEXT NOT NULL DEFAULT 'none'
				CHECK (auth_type IN ('none', 'api_key', 'basic', 'token')),
			credentials_enc BLOB,
			verify_tls INTEGER NOT NULL DEFAULT 1,
			options TEXT NOT NULL DEFAULT '{}',
			refresh_seconds INTEGER NOT NULL DEFAULT 60 CHECK (refresh_seconds >= 10),
			last_test_at TIMESTAMP,
			last_test_ok INTEGER,
			last_error TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		INSERT INTO users (id) VALUES
			('dddddddd-dddd-dddd-dddd-ddddddddddd1'),
			('dddddddd-dddd-dddd-dddd-ddddddddddd2');
	`
	_, err = db.Exec(schema)
	require.NoError(t, err)
	return db
}

// setupIntegrationTestApp mirrors the routes in main.go with a stub auth
// middleware that sets the given user ID
func setupIntegrationTestApp(t *testing.T, db *sql.DB, userID string) *fiber.App {
	t.Helper()
	cipher, err := utils.NewCipher(bytes.Repeat([]byte{5}, 32))
	require.NoError(t, err)
	handler := NewIntegrationHandler(services.NewIntegrationService(repository.NewIntegrationRepository(db), cipher))

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", userID)
		return c.Next()
	})
	app.Get("/integrations/kinds", handler.ListKinds)
	app.Post("/integrations/test", handler.TestUnsavedIntegration)
	app.Post("/integrations", handler.CreateIntegration)
	app.Get("/integrations", handler.ListIntegrations)
	app.Get("/integrations/:id", handler.GetIntegration)
	app.Put("/integrations/:id", handler.UpdateIntegration)
	app.Delete("/integrations/:id", handler.DeleteIntegration)
	app.Post("/integrations/:id/test", handler.TestIntegration)
	return app
}

// doIntegrationRequest sends a request and returns status and body. Every
// response is checked for the secret, whatever the test is about.
func doIntegrationRequest(t *testing.T, app *fiber.App, method, path, body string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	text := string(raw)
	assert.NotContains(t, text, integrationSecret, "%s %s leaked the secret", method, path)
	assert.NotContains(t, text, url.QueryEscape(integrationSecret), "%s %s leaked the secret", method, path)
	assert.NotContains(t, text, `"credentials"`, "%s %s returned a credentials object", method, path)
	assert.NotContains(t, text, "credentials_enc", "%s %s returned the encrypted blob", method, path)
	return resp.StatusCode, text
}

// pingTestServer accepts /ping only with the right API key
func pingTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ping" && r.URL.Query().Get("apikey") == integrationSecret {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(server.Close)
	return server
}

func integrationBody(baseURL string) string {
	return fmt.Sprintf(`{"kind":"handler-fake","name":"Sonarr","base_url":%q,"auth_type":"api_key","credentials":{"api_key":%q}}`,
		baseURL, integrationSecret)
}

func createIntegration(t *testing.T, app *fiber.App, baseURL string) models.Integration {
	t.Helper()
	status, body := doIntegrationRequest(t, app, http.MethodPost, "/integrations", integrationBody(baseURL))
	require.Equal(t, fiber.StatusCreated, status, body)
	var integration models.Integration
	require.NoError(t, json.Unmarshal([]byte(body), &integration))
	return integration
}

func TestIntegrationHandler_ListKinds(t *testing.T) {
	app := setupIntegrationTestApp(t, setupIntegrationTestDB(t), integrationOwnerID)

	status, body := doIntegrationRequest(t, app, http.MethodGet, "/integrations/kinds", "")
	require.Equal(t, fiber.StatusOK, status)

	var kinds []integrations.Meta
	require.NoError(t, json.Unmarshal([]byte(body), &kinds))
	byKind := map[string]integrations.Meta{}
	for _, kind := range kinds {
		byKind[kind.Kind] = kind
	}
	fake := byKind["handler-fake"]
	assert.Equal(t, []string{"api_key", "none"}, fake.AuthTypes)
	assert.Equal(t, "Queued", fake.KPIs[0].Label)
	assert.Greater(t, len(kinds), 1, "built-in kinds are listed too")
}

func TestIntegrationHandler_CRUDNeverExposesSecrets(t *testing.T) {
	db := setupIntegrationTestDB(t)
	app := setupIntegrationTestApp(t, db, integrationOwnerID)

	created := createIntegration(t, app, "http://192.168.1.10:8989")
	assert.True(t, created.HasCredentials)
	assert.Equal(t, "api_key", created.AuthType)

	status, body := doIntegrationRequest(t, app, http.MethodGet, "/integrations", "")
	assert.Equal(t, fiber.StatusOK, status)
	assert.Contains(t, body, `"has_credentials":true`)

	status, _ = doIntegrationRequest(t, app, http.MethodGet, "/integrations/"+created.ID, "")
	assert.Equal(t, fiber.StatusOK, status)

	status, body = doIntegrationRequest(t, app, http.MethodPut, "/integrations/"+created.ID, `{"name":"Sonarr 4K"}`)
	assert.Equal(t, fiber.StatusOK, status)
	assert.Contains(t, body, `"name":"Sonarr 4K"`)
	assert.Contains(t, body, `"has_credentials":true`, "omitted credentials are kept")

	// At rest, the column holds ciphertext only
	var blob []byte
	require.NoError(t, db.QueryRow(`SELECT credentials_enc FROM integrations WHERE id = ?`, created.ID).Scan(&blob))
	assert.NotEmpty(t, blob)
	assert.NotContains(t, string(blob), integrationSecret)

	status, _ = doIntegrationRequest(t, app, http.MethodDelete, "/integrations/"+created.ID, "")
	assert.Equal(t, fiber.StatusNoContent, status)
	status, _ = doIntegrationRequest(t, app, http.MethodGet, "/integrations/"+created.ID, "")
	assert.Equal(t, fiber.StatusNotFound, status)
}

func TestIntegrationHandler_TestUnsaved(t *testing.T) {
	app := setupIntegrationTestApp(t, setupIntegrationTestDB(t), integrationOwnerID)
	server := pingTestServer(t)

	status, body := doIntegrationRequest(t, app, http.MethodPost, "/integrations/test", integrationBody(server.URL))
	require.Equal(t, fiber.StatusOK, status)
	assert.Contains(t, body, `"ok":true`)

	status, body = doIntegrationRequest(t, app, http.MethodGet, "/integrations", "")
	assert.Equal(t, fiber.StatusOK, status)
	assert.Equal(t, "[]", body, "unsaved test must not store anything")
}

func TestIntegrationHandler_TestFailureIsRedacted(t *testing.T) {
	app := setupIntegrationTestApp(t, setupIntegrationTestDB(t), integrationOwnerID)
	server := pingTestServer(t)
	deadURL := server.URL
	server.Close()

	// doIntegrationRequest asserts the secret is absent from the error
	status, body := doIntegrationRequest(t, app, http.MethodPost, "/integrations/test", integrationBody(deadURL))
	require.Equal(t, fiber.StatusOK, status)
	assert.Contains(t, body, `"ok":false`)
	assert.Contains(t, body, "[redacted]")

	created := createIntegration(t, app, deadURL)
	status, body = doIntegrationRequest(t, app, http.MethodPost, "/integrations/"+created.ID+"/test", "")
	require.Equal(t, fiber.StatusOK, status)
	assert.Contains(t, body, `"ok":false`)

	// The stored last_error is redacted too
	status, body = doIntegrationRequest(t, app, http.MethodGet, "/integrations/"+created.ID, "")
	require.Equal(t, fiber.StatusOK, status)
	assert.Contains(t, body, `"last_test_ok":false`)
	assert.Contains(t, body, "[redacted]")
}

func TestIntegrationHandler_TestSaved(t *testing.T) {
	app := setupIntegrationTestApp(t, setupIntegrationTestDB(t), integrationOwnerID)
	server := pingTestServer(t)
	created := createIntegration(t, app, server.URL)

	status, body := doIntegrationRequest(t, app, http.MethodPost, "/integrations/"+created.ID+"/test", "")
	require.Equal(t, fiber.StatusOK, status)
	assert.Contains(t, body, `"ok":true`)

	_, body = doIntegrationRequest(t, app, http.MethodGet, "/integrations/"+created.ID, "")
	assert.Contains(t, body, `"last_test_ok":true`)
}

func TestIntegrationHandler_OwnershipIsEnforced(t *testing.T) {
	db := setupIntegrationTestDB(t)
	owner := setupIntegrationTestApp(t, db, integrationOwnerID)
	other := setupIntegrationTestApp(t, db, integrationOtherID)
	created := createIntegration(t, owner, "http://192.168.1.10:8989")
	path := "/integrations/" + created.ID

	for _, tc := range []struct{ method, path, body string }{
		{http.MethodGet, path, ""},
		{http.MethodPut, path, `{"name":"Hijacked"}`},
		{http.MethodPost, path + "/test", ""},
		{http.MethodDelete, path, ""},
	} {
		status, _ := doIntegrationRequest(t, other, tc.method, tc.path, tc.body)
		assert.Equal(t, fiber.StatusNotFound, status, "%s %s by another user", tc.method, tc.path)
	}

	_, body := doIntegrationRequest(t, other, http.MethodGet, "/integrations", "")
	assert.Equal(t, "[]", body)

	_, body = doIntegrationRequest(t, owner, http.MethodGet, path, "")
	assert.Contains(t, body, `"name":"Sonarr"`, "owner's integration is untouched")
}

func TestIntegrationHandler_BadInput(t *testing.T) {
	app := setupIntegrationTestApp(t, setupIntegrationTestDB(t), integrationOwnerID)

	cases := map[string]struct{ path, body, want string }{
		"invalid json":   {"/integrations", `{`, "Invalid request body"},
		"unknown kind":   {"/integrations", `{"kind":"nope","name":"x","base_url":"http://nas.lan"}`, "Unknown integration kind"},
		"url with creds": {"/integrations", `{"kind":"handler-fake","name":"x","base_url":"http://a:b@nas.lan","auth_type":"none"}`, "must not contain credentials"},
		"missing creds":  {"/integrations", `{"kind":"handler-fake","name":"x","base_url":"http://nas.lan"}`, "Credentials are required"},
		"test bad input": {"/integrations/test", `{"kind":"handler-fake","name":"x","base_url":"ftp://nas.lan","auth_type":"none"}`, "Invalid base URL"},
	}
	for name, tc := range cases {
		status, body := doIntegrationRequest(t, app, http.MethodPost, tc.path, tc.body)
		assert.Equal(t, fiber.StatusBadRequest, status, name)
		assert.Contains(t, body, tc.want, name)
	}
}
