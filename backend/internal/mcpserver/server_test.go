package mcpserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/nimbus/backend/internal/middleware"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

// mcpTestServer runs the handler the way main.go mounts it: on a Fiber app,
// with tokens checked against a real (SQLite) database
type mcpTestServer struct {
	url  string
	db   *sql.DB
	data *fakeData
}

func newMCPTestServer(t *testing.T) *mcpTestServer {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	// One connection: every connection to :memory: is its own database
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	_, err = db.Exec(`
		CREATE TABLE users (
			id TEXT PRIMARY KEY, email TEXT UNIQUE NOT NULL, name TEXT NOT NULL, password TEXT,
			role TEXT NOT NULL DEFAULT 'user', provider TEXT NOT NULL DEFAULT 'local', provider_id TEXT,
			avatar_url TEXT, email_verified INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL, updated_at TIMESTAMP NOT NULL, last_activity_at TIMESTAMP
		);
		CREATE TABLE api_tokens (
			id TEXT PRIMARY KEY, user_id TEXT NOT NULL, name TEXT NOT NULL,
			token_hash TEXT NOT NULL UNIQUE, token_prefix TEXT NOT NULL,
			read_only INTEGER NOT NULL DEFAULT 1, scopes TEXT NOT NULL DEFAULT '',
			expires_at TIMESTAMP, last_used_at TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		INSERT INTO users (id, email, name, created_at, updated_at) VALUES
			('user-alice', 'alice@example.com', 'Alice', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
			('user-bob', 'bob@example.com', 'Bob', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
	`)
	require.NoError(t, err)

	data := newFakeData()
	deps := data.deps()
	deps.Users = repository.NewUserRepository(db)
	deps.Tokens = repository.NewAPITokenRepository(db)

	app := fiber.New()
	app.All("/mcp", NewHandler(deps))
	server := httptest.NewServer(adaptor.FiberApp(app))
	t.Cleanup(server.Close)

	return &mcpTestServer{url: server.URL + "/mcp", db: db, data: data}
}

func (s *mcpTestServer) token(t *testing.T, userID string, expiresAt *time.Time, scopes ...string) string {
	t.Helper()
	token, hash, prefix, err := utils.GenerateAPIToken()
	require.NoError(t, err)
	_, err = s.db.Exec(`INSERT INTO api_tokens (id, user_id, name, token_hash, token_prefix, scopes, expires_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		uuid.New().String(), userID, "MCP", hash, prefix, strings.Join(scopes, " "), expiresAt)
	require.NoError(t, err)
	return token
}

// bearer adds the Authorization header, like a client configured with one
type bearer struct{ token string }

func (b bearer) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(req)
}

// connect opens a session with the SDK client and its default options
func (s *mcpTestServer) connect(t *testing.T, token string) *mcp.ClientSession {
	t.Helper()
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint:   s.url,
		HTTPClient: &http.Client{Transport: bearer{token}},
	}, nil)
	require.NoError(t, err)
	t.Cleanup(func() { session.Close() })
	return session
}

// call runs a tool and decodes its structured result into out
func call(t *testing.T, session *mcp.ClientSession, name string, args map[string]any, out any) *mcp.CallToolResult {
	t.Helper()
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	require.NoError(t, err)
	if out != nil {
		require.False(t, result.IsError, "tool error: %+v", result.Content)
		raw, err := json.Marshal(result.StructuredContent)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(raw, out))
	}
	return result
}

func errorText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	require.True(t, result.IsError)
	require.NotEmpty(t, result.Content)
	text, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	return text.Text
}

func TestMCP_SmokeWithSDKClient(t *testing.T) {
	server := newMCPTestServer(t)
	session := server.connect(t, server.token(t, alice, nil, models.ScopeMCP))
	ctx := context.Background()

	assert.Contains(t, session.InitializeResult().Instructions, "homelab dashboard")
	assert.Equal(t, "nimbus", session.InitializeResult().ServerInfo.Name)

	// Exactly the tools of the issue, read-only except recheck_service
	listed, err := session.ListTools(ctx, nil)
	require.NoError(t, err)
	var toolNames []string
	for _, tool := range listed.Tools {
		toolNames = append(toolNames, tool.Name)
		require.NotNil(t, tool.Annotations, tool.Name)
		assert.Equal(t, tool.Name != "recheck_service", tool.Annotations.ReadOnlyHint, tool.Name)
		assert.NotEmpty(t, tool.Description, tool.Name)
	}
	assert.ElementsMatch(t, []string{
		"list_services", "get_service_status", "get_uptime", "search_services",
		"get_widget_data", "list_integrations", "recheck_service",
	}, toolNames)

	// "What's down?"
	var services ServiceList
	call(t, session, "list_services", nil, &services)
	assert.Equal(t, 5, services.Total)
	var offline []string
	for _, service := range services.Services {
		if service.Status == models.StatusOffline {
			offline = append(offline, service.Name)
		}
	}
	assert.Equal(t, []string{"Sonarr"}, offline)

	var media ServiceList
	call(t, session, "list_services", map[string]any{"group": "Media"}, &media)
	assert.Equal(t, 2, media.Total)

	var status ServiceStatus
	call(t, session, "get_service_status", map[string]any{"service": "sonarr"}, &status)
	assert.Equal(t, "connection refused", status.RecentChecks[0].Error)

	var uptime Uptime
	call(t, session, "get_uptime", map[string]any{"service": "Jellyfin", "range": "7d"}, &uptime)
	assert.Equal(t, 99.5, uptime.UptimePercentage)
	assert.Equal(t, 200, uptime.TotalChecks)

	var found ServiceList
	call(t, session, "search_services", map[string]any{"q": "192.168"}, &found)
	assert.Equal(t, []string{"Router"}, names(found))

	var widget WidgetData
	call(t, session, "get_widget_data", map[string]any{"id": "wid-weather"}, &widget)
	assert.Equal(t, map[string]any{"temperature": float64(11)}, widget.Data)

	var integrations IntegrationList
	result := call(t, session, "list_integrations", nil, &integrations)
	assert.Len(t, integrations.Integrations, 3)
	raw, err := json.Marshal(result)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "internal", "no integration URLs in the response")
	assert.NotContains(t, string(raw), "hunter2")

	// Tool errors come back as results the assistant can read
	assert.Contains(t, errorText(t, call(t, session, "get_service_status", map[string]any{"service": "Plex"}, nil)), "search_services")
	assert.Contains(t, errorText(t, call(t, session, "get_uptime", map[string]any{"service": "Jellyfin", "range": "1y"}, nil)), "30d")

	// The dashboard resource
	resources, err := session.ListResources(ctx, nil)
	require.NoError(t, err)
	require.Len(t, resources.Resources, 1)
	assert.Equal(t, DashboardURI, resources.Resources[0].URI)
	read, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: DashboardURI})
	require.NoError(t, err)
	require.Len(t, read.Contents, 1)
	assert.Contains(t, read.Contents[0].Text, "5 services: 3 online, 1 offline, 1 unknown.")
	assert.Equal(t, "private", read.CacheScope)
}

func TestMCP_TokensOnlySeeTheirOwner(t *testing.T) {
	server := newMCPTestServer(t)
	aliceSession := server.connect(t, server.token(t, alice, nil, models.ScopeMCP))
	bobSession := server.connect(t, server.token(t, bob, nil, models.ScopeMCP))

	var services ServiceList
	call(t, bobSession, "list_services", nil, &services)
	assert.Equal(t, []string{"Vaultwarden"}, names(services))

	// Alice's IDs mean nothing to Bob's token, and the other way around
	assert.Contains(t, errorText(t, call(t, bobSession, "get_service_status", map[string]any{"service": "svc-jellyfin"}, nil)), "no service")
	assert.Contains(t, errorText(t, call(t, bobSession, "get_widget_data", map[string]any{"id": "wid-weather"}, nil)), "wid-bob")
	assert.Contains(t, errorText(t, call(t, aliceSession, "get_uptime", map[string]any{"service": "svc-bob"}, nil)), "no service")

	read, err := bobSession.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: DashboardURI})
	require.NoError(t, err)
	assert.NotContains(t, read.Contents[0].Text, "Jellyfin")
}

func TestMCP_RecheckNeedsWriteScope(t *testing.T) {
	server := newMCPTestServer(t)

	readOnly := server.connect(t, server.token(t, alice, nil, models.ScopeMCP, models.ScopeReadServices))
	assert.Contains(t, errorText(t, call(t, readOnly, "recheck_service", map[string]any{"id": "svc-sonarr"}, nil)), "write:services")
	assert.Empty(t, server.data.checked)

	writer := server.connect(t, server.token(t, alice, nil, models.ScopeMCP, models.ScopeWriteServices))
	var status ServiceStatus
	call(t, writer, "recheck_service", map[string]any{"id": "svc-sonarr"}, &status)
	assert.Equal(t, []string{"svc-sonarr"}, server.data.checked)
	assert.Equal(t, "online", status.Service.Status)
}

// post sends a raw initialize request, to look at HTTP-level answers
func (s *mcpTestServer) post(t *testing.T, authorization string) *http.Response {
	t.Helper()
	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"raw","version":"1"}}}`
	req, err := http.NewRequest(http.MethodPost, s.url, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func TestMCP_Authentication(t *testing.T) {
	server := newMCPTestServer(t)
	past := time.Now().Add(-time.Minute).UTC()
	future := time.Now().Add(time.Hour).UTC()

	tests := []struct {
		name          string
		authorization string
		want          int
	}{
		{"no token", "", http.StatusUnauthorized},
		{"not a bearer token", "Basic YWxpY2U6c2VjcmV0", http.StatusUnauthorized},
		{"unknown token", "Bearer nimbus_" + strings.Repeat("0", 64), http.StatusUnauthorized},
		{"not an API token", "Bearer some.session.jwt", http.StatusUnauthorized},
		{"expired token", "Bearer " + server.token(t, alice, &past, models.ScopeMCP), http.StatusUnauthorized},
		{"token of a deleted user", "Bearer " + server.token(t, "user-gone", nil, models.ScopeMCP), http.StatusUnauthorized},
		{"token without the mcp scope", "Bearer " + server.token(t, alice, nil, models.LegacyScopes(false)...), http.StatusForbidden},
		{"token with the mcp scope", "Bearer " + server.token(t, alice, nil, models.ScopeMCP), http.StatusOK},
		{"token with a future expiry", "Bearer " + server.token(t, alice, &future, models.ScopeMCP), http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, server.post(t, tt.authorization).StatusCode)
		})
	}

	// Stateless: there is no stream to open with GET
	req, err := http.NewRequest(http.MethodGet, server.url, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+server.token(t, alice, nil, models.ScopeMCP))
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)

	// The session cookie of the web UI is not accepted here
	req, err = http.NewRequest(http.MethodPost, server.url, strings.NewReader(`{}`))
	require.NoError(t, err)
	req.Header.Set("Cookie", "auth_token=some.session.jwt")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestMCP_RejectsLargeBodies(t *testing.T) {
	server := newMCPTestServer(t)
	req, err := http.NewRequest(http.MethodPost, server.url, strings.NewReader(strings.Repeat("x", maxBodyBytes+1)))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+server.token(t, alice, nil, models.ScopeMCP))
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusRequestEntityTooLarge, resp.StatusCode)
}

func TestMCP_RateLimit(t *testing.T) {
	server := newMCPTestServer(t)
	middleware.SetAPITokenRateLimit(2)
	t.Cleanup(func() { middleware.SetAPITokenRateLimit(middleware.DefaultAPITokenRateLimit) })

	token := "Bearer " + server.token(t, alice, nil, models.ScopeMCP)
	assert.Equal(t, http.StatusOK, server.post(t, token).StatusCode)
	assert.Equal(t, http.StatusOK, server.post(t, token).StatusCode)

	resp := server.post(t, token)
	assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
	assert.Equal(t, "60", resp.Header.Get("Retry-After"))

	// Requests that fail authentication do not use up anyone's budget
	other := "Bearer " + server.token(t, alice, nil, models.ScopeMCP)
	assert.Equal(t, http.StatusOK, server.post(t, other).StatusCode)
}
