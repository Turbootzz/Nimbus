package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
		CREATE TABLE IF NOT EXISTS api_tokens (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			name TEXT NOT NULL,
			token_hash TEXT NOT NULL UNIQUE,
			token_prefix TEXT NOT NULL,
			read_only INTEGER NOT NULL DEFAULT 1,
			last_used_at TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
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
