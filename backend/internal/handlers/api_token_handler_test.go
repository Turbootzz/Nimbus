package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"

	_ "github.com/mattn/go-sqlite3"
)

const testTokenUserID = "cccccccc-cccc-cccc-cccc-ccccccccccc1"
const testTokenOtherUserID = "cccccccc-cccc-cccc-cccc-ccccccccccc2"

// setupAPITokenTestDB creates an in-memory SQLite database with the api_tokens table
func setupAPITokenTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	schema := `
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY
		);

		CREATE TABLE IF NOT EXISTS api_tokens (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			name TEXT NOT NULL,
			token_hash TEXT NOT NULL UNIQUE,
			token_prefix TEXT NOT NULL,
			read_only INTEGER NOT NULL DEFAULT 1,
			scopes TEXT NOT NULL DEFAULT '',
			expires_at TIMESTAMP,
			last_used_at TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		INSERT INTO users (id) VALUES
			('cccccccc-cccc-cccc-cccc-ccccccccccc1'),
			('cccccccc-cccc-cccc-cccc-ccccccccccc2');
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create test tables: %v", err)
	}

	return db
}

// setupAPITokenTestApp builds a fiber app with token routes and a stub auth
// middleware that sets the given user ID
func setupAPITokenTestApp(db *sql.DB, userID string) *fiber.App {
	handler := NewAPITokenHandler(repository.NewAPITokenRepository(db))

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", userID)
		return c.Next()
	})
	app.Post("/tokens", handler.CreateToken)
	app.Get("/tokens", handler.ListTokens)
	app.Delete("/tokens/:id", handler.DeleteToken)
	return app
}

func createTokenRequest(t *testing.T, app *fiber.App, body string) *http.Response {
	t.Helper()
	req := httptest.NewRequest("POST", "/tokens", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	return resp
}

func TestAPITokenHandler_CreateToken(t *testing.T) {
	db := setupAPITokenTestDB(t)
	defer db.Close()
	app := setupAPITokenTestApp(db, testTokenUserID)

	resp := createTokenRequest(t, app, `{"name":"Hearth","read_only":true}`)
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var result models.APITokenCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !strings.HasPrefix(result.Token, "nimbus_") {
		t.Errorf("expected plaintext token with nimbus_ prefix, got %q", result.Token)
	}
	if result.APIToken.Name != "Hearth" {
		t.Errorf("expected name Hearth, got %q", result.APIToken.Name)
	}
	if !result.APIToken.ReadOnly {
		t.Error("expected read_only true")
	}
	if result.APIToken.TokenPrefix != result.Token[:12] {
		t.Errorf("token_prefix %q does not match token start", result.APIToken.TokenPrefix)
	}
}

func TestAPITokenHandler_CreateToken_DefaultsReadOnly(t *testing.T) {
	db := setupAPITokenTestDB(t)
	defer db.Close()
	app := setupAPITokenTestApp(db, testTokenUserID)

	resp := createTokenRequest(t, app, `{"name":"Defaults"}`)
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var result models.APITokenCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if !result.APIToken.ReadOnly {
		t.Error("expected read_only to default to true")
	}
}

func TestAPITokenHandler_CreateToken_Validation(t *testing.T) {
	db := setupAPITokenTestDB(t)
	defer db.Close()
	app := setupAPITokenTestApp(db, testTokenUserID)

	// Missing name
	resp := createTokenRequest(t, app, `{"read_only":true}`)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400 for missing name, got %d", resp.StatusCode)
	}

	// Whitespace-only name
	resp = createTokenRequest(t, app, `{"name":"   "}`)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400 for whitespace-only name, got %d", resp.StatusCode)
	}

	// Name too long
	longName := strings.Repeat("a", 101)
	resp = createTokenRequest(t, app, fmt.Sprintf(`{"name":"%s"}`, longName))
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400 for long name, got %d", resp.StatusCode)
	}
}

func TestAPITokenHandler_CreateToken_Limit(t *testing.T) {
	db := setupAPITokenTestDB(t)
	defer db.Close()
	app := setupAPITokenTestApp(db, testTokenUserID)

	for i := 0; i < 10; i++ {
		resp := createTokenRequest(t, app, fmt.Sprintf(`{"name":"Token %d"}`, i))
		if resp.StatusCode != fiber.StatusCreated {
			t.Fatalf("expected 201 for token %d, got %d", i, resp.StatusCode)
		}
	}

	resp := createTokenRequest(t, app, `{"name":"One too many"}`)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400 at token limit, got %d", resp.StatusCode)
	}
}

func TestAPITokenHandler_ListTokens(t *testing.T) {
	db := setupAPITokenTestDB(t)
	defer db.Close()

	// Own tokens
	ownApp := setupAPITokenTestApp(db, testTokenUserID)
	createTokenRequest(t, ownApp, `{"name":"Mine"}`)

	// Another user's token
	otherApp := setupAPITokenTestApp(db, testTokenOtherUserID)
	createTokenRequest(t, otherApp, `{"name":"Not mine"}`)

	req := httptest.NewRequest("GET", "/tokens", nil)
	resp, err := ownApp.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body := new(bytes.Buffer)
	if _, err := body.ReadFrom(resp.Body); err != nil {
		t.Fatalf("Failed to read body: %v", err)
	}

	var tokens []models.APIToken
	if err := json.Unmarshal(body.Bytes(), &tokens); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("expected 1 token, got %d", len(tokens))
	}
	if tokens[0].Name != "Mine" {
		t.Errorf("expected own token, got %+v", tokens[0])
	}

	// Plaintext and hash must never appear in list responses
	if strings.Contains(body.String(), "token_hash") {
		t.Error("list response leaks token_hash")
	}
	if strings.Contains(body.String(), "nimbus_") && !strings.Contains(body.String(), `"token_prefix":"nimbus_`) {
		t.Error("list response leaks plaintext token")
	}
}

func TestAPITokenHandler_DeleteToken(t *testing.T) {
	db := setupAPITokenTestDB(t)
	defer db.Close()
	app := setupAPITokenTestApp(db, testTokenUserID)

	resp := createTokenRequest(t, app, `{"name":"Doomed"}`)
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	var result models.APITokenCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Another user cannot delete it
	otherApp := setupAPITokenTestApp(db, testTokenOtherUserID)
	req := httptest.NewRequest("DELETE", "/tokens/"+result.APIToken.ID, nil)
	deleteResp, err := otherApp.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if deleteResp.StatusCode != fiber.StatusNotFound {
		t.Errorf("expected 404 for other user's delete, got %d", deleteResp.StatusCode)
	}

	// Owner can
	req = httptest.NewRequest("DELETE", "/tokens/"+result.APIToken.ID, nil)
	deleteResp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if deleteResp.StatusCode != fiber.StatusNoContent {
		t.Errorf("expected 204, got %d", deleteResp.StatusCode)
	}

	// Gone now
	req = httptest.NewRequest("DELETE", "/tokens/"+result.APIToken.ID, nil)
	deleteResp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if deleteResp.StatusCode != fiber.StatusNotFound {
		t.Errorf("expected 404 for deleted token, got %d", deleteResp.StatusCode)
	}
}

// decodeCreatedToken reads the token from a create response
func decodeCreatedToken(t *testing.T, resp *http.Response) models.APIToken {
	t.Helper()
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	var result models.APITokenCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	return result.APIToken
}

func TestAPITokenHandler_CreateToken_Scopes(t *testing.T) {
	db := setupAPITokenTestDB(t)
	defer db.Close()
	app := setupAPITokenTestApp(db, testTokenUserID)

	tests := []struct {
		name         string
		body         string
		wantScopes   string
		wantReadOnly bool
	}{
		{"defaults to the read scopes", `{"name":"a"}`, "read:services read:metrics read:widgets", true},
		{"legacy read_only false is full access without mcp", `{"name":"b","read_only":false}`, "read:services read:metrics read:widgets write:services", false},
		{"explicit scopes win over read_only", `{"name":"c","read_only":false,"scopes":["read:services"]}`, "read:services", true},
		{"scopes come back in display order without duplicates", `{"name":"d","scopes":["mcp","write:services","mcp","read:services"]}`, "read:services write:services mcp", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := decodeCreatedToken(t, createTokenRequest(t, app, tt.body))
			if got := strings.Join(token.Scopes, " "); got != tt.wantScopes {
				t.Errorf("scopes = %q, want %q", got, tt.wantScopes)
			}
			if token.ReadOnly != tt.wantReadOnly {
				t.Errorf("read_only = %v, want %v", token.ReadOnly, tt.wantReadOnly)
			}

			// The stored row matches, with read_only kept in sync for older versions
			var scopes string
			var readOnly bool
			if err := db.QueryRow(`SELECT scopes, read_only FROM api_tokens WHERE id = ?`, token.ID).Scan(&scopes, &readOnly); err != nil {
				t.Fatalf("Failed to read token: %v", err)
			}
			if scopes != tt.wantScopes || readOnly != tt.wantReadOnly {
				t.Errorf("stored scopes %q read_only %v, want %q %v", scopes, readOnly, tt.wantScopes, tt.wantReadOnly)
			}
		})
	}

	for name, body := range map[string]string{
		"unknown scope": `{"name":"x","scopes":["read:services","admin"]}`,
		"empty scopes":  `{"name":"x","scopes":[]}`,
	} {
		if resp := createTokenRequest(t, app, body); resp.StatusCode != fiber.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d", name, resp.StatusCode)
		}
	}
}

func TestAPITokenHandler_CreateToken_Expiry(t *testing.T) {
	db := setupAPITokenTestDB(t)
	defer db.Close()
	app := setupAPITokenTestApp(db, testTokenUserID)

	// No expiry by default
	if token := decodeCreatedToken(t, createTokenRequest(t, app, `{"name":"forever"}`)); token.ExpiresAt != nil {
		t.Errorf("expected no expiry, got %v", token.ExpiresAt)
	}

	future := time.Now().Add(24 * time.Hour).Truncate(time.Second)
	body := fmt.Sprintf(`{"name":"temp","expires_at":%q}`, future.Format(time.RFC3339))
	token := decodeCreatedToken(t, createTokenRequest(t, app, body))
	if token.ExpiresAt == nil || !token.ExpiresAt.Equal(future) {
		t.Errorf("expires_at = %v, want %v", token.ExpiresAt, future)
	}

	// The list returns it too
	tokens, err := repository.NewAPITokenRepository(db).ListByUserID(context.Background(), testTokenUserID)
	if err != nil {
		t.Fatalf("Failed to list tokens: %v", err)
	}
	found := false
	for _, listed := range tokens {
		if listed.ID == token.ID {
			found = listed.ExpiresAt != nil && listed.ExpiresAt.Equal(future)
		}
	}
	if !found {
		t.Error("expected the stored token to keep its expiry")
	}

	for name, body := range map[string]string{
		"past expiry":    fmt.Sprintf(`{"name":"x","expires_at":%q}`, time.Now().Add(-time.Hour).Format(time.RFC3339)),
		"invalid expiry": `{"name":"x","expires_at":"tomorrow"}`,
	} {
		if resp := createTokenRequest(t, app, body); resp.StatusCode != fiber.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d", name, resp.StatusCode)
		}
	}
}

func TestAPITokenHandler_ActivityLog(t *testing.T) {
	db := setupAPITokenTestDB(t)
	defer db.Close()
	if _, err := db.Exec(`
		CREATE TABLE user_activity_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT,
			actor_id TEXT,
			action TEXT NOT NULL,
			details TEXT,
			ip_address TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`); err != nil {
		t.Fatalf("Failed to create activity log table: %v", err)
	}

	handler := NewAPITokenHandler(repository.NewAPITokenRepository(db))
	handler.SetActivityLog(repository.NewActivityLogRepository(db))
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", testTokenUserID)
		return c.Next()
	})
	app.Post("/tokens", handler.CreateToken)
	app.Delete("/tokens/:id", handler.DeleteToken)

	token := decodeCreatedToken(t, createTokenRequest(t, app, `{"name":"Logged","scopes":["mcp"]}`))
	resp, err := app.Test(httptest.NewRequest("DELETE", "/tokens/"+token.ID, nil))
	if err != nil || resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("delete failed: %v, status %d", err, resp.StatusCode)
	}
	// A failed revoke is not logged
	if _, err := app.Test(httptest.NewRequest("DELETE", "/tokens/"+token.ID, nil)); err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	rows, err := db.Query(`SELECT user_id, actor_id, action, details FROM user_activity_logs ORDER BY id`)
	if err != nil {
		t.Fatalf("Failed to read activity log: %v", err)
	}
	defer rows.Close()
	var actions []string
	for rows.Next() {
		var userID, actorID, action, details string
		if err := rows.Scan(&userID, &actorID, &action, &details); err != nil {
			t.Fatalf("Failed to scan activity log: %v", err)
		}
		if userID != testTokenUserID || actorID != testTokenUserID {
			t.Errorf("entry for user %q by %q, want %q", userID, actorID, testTokenUserID)
		}
		if !strings.Contains(details, token.ID) {
			t.Errorf("details %q miss the token id", details)
		}
		if strings.Contains(details, "nimbus_") {
			t.Errorf("details %q leak the token", details)
		}
		actions = append(actions, action)
	}
	if got := strings.Join(actions, ","); got != "api_token_created,api_token_revoked" {
		t.Errorf("actions = %q", got)
	}
}
