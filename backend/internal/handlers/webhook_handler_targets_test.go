package handlers

import (
	"bytes"
	"context"
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
	"github.com/nimbus/backend/internal/utils"
)

const (
	targetTestUser      = "cccccccc-cccc-cccc-cccc-ccccccccccc1"
	targetTelegramToken = "123456789:AAbbCCddEEffGGhhIIjjKKllMMnnOOppQQr"
	targetPushoverToken = "azGDORePK8gMaC0QOYAMyEEuzJnyUi"
	targetPushoverUser  = "uQiRzpo4DXghDmr9QzzfQu27cmVRsG"
)

// targetTestApp wires the webhook routes to a handler that can encrypt credentials
func targetTestApp(t *testing.T) (*fiber.App, *repository.WebhookRepository) {
	t.Helper()
	db := setupWebhookTestDB(t)
	t.Cleanup(func() { db.Close() })

	cipher, err := utils.NewCipher(bytes.Repeat([]byte{9}, 32))
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}
	webhookRepo := repository.NewWebhookRepository(db)
	notificationService := services.NewNotificationService(webhookRepo, nil)
	notificationService.SetCipher(cipher)
	handler := NewWebhookHandler(webhookRepo, notificationService)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", targetTestUser)
		return c.Next()
	})
	app.Post("/webhooks", handler.CreateWebhook)
	app.Get("/webhooks", handler.GetWebhooks)
	app.Put("/webhooks/:id", handler.UpdateWebhook)
	app.Post("/webhooks/:id/test", handler.TestWebhook)
	return app, webhookRepo
}

// targetRequest sends a JSON request and returns the status and raw body
func targetRequest(t *testing.T, app *fiber.App, method, path string, body any) (int, string) {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	return resp.StatusCode, string(raw)
}

func decodeWebhook(t *testing.T, raw string) models.WebhookResponse {
	t.Helper()
	var webhook models.WebhookResponse
	if err := json.Unmarshal([]byte(raw), &webhook); err != nil {
		t.Fatalf("Unmarshal %s: %v", raw, err)
	}
	return webhook
}

func decodeFields(t *testing.T, raw string) map[string]string {
	t.Helper()
	var body struct {
		Error  string            `json:"error"`
		Fields map[string]string `json:"fields"`
	}
	if err := json.Unmarshal([]byte(raw), &body); err != nil {
		t.Fatalf("Unmarshal %s: %v", raw, err)
	}
	if body.Error == "" {
		t.Errorf("expected an error summary in %s", raw)
	}
	return body.Fields
}

func TestWebhookHandler_CreateTargets(t *testing.T) {
	tests := []struct {
		name            string
		body            map[string]any
		secrets         []string
		wantURL         string
		wantOptions     models.WebhookOptions
		wantCredentials bool
	}{
		{
			name: "ntfy with token",
			body: map[string]any{"name": "ntfy", "format": "ntfy", "url": "https://ntfy.sh/alerts",
				"options": map[string]any{"priority": 5}, "credentials": map[string]any{"token": "tk_secret123"}},
			secrets: []string{"tk_secret123"}, wantURL: "https://ntfy.sh/alerts",
			wantOptions: models.WebhookOptions{Priority: 5}, wantCredentials: true,
		},
		{
			name:    "ntfy public topic",
			body:    map[string]any{"name": "ntfy", "format": "ntfy", "url": "http://192.168.1.5/alerts"},
			wantURL: "http://192.168.1.5/alerts", wantOptions: models.WebhookOptions{Priority: 3},
		},
		{
			name: "gotify",
			body: map[string]any{"name": "gotify", "format": "gotify", "url": "http://192.168.1.6:8080",
				"credentials": map[string]any{"token": "AppToken.123"}},
			secrets: []string{"AppToken.123"}, wantURL: "http://192.168.1.6:8080", wantCredentials: true,
		},
		{
			name: "telegram ignores a client URL",
			body: map[string]any{"name": "telegram", "format": "telegram", "url": "https://evil.example/steal",
				"options": map[string]any{"chat_id": "-1001234"}, "credentials": map[string]any{"token": targetTelegramToken}},
			secrets: []string{targetTelegramToken}, wantURL: models.TelegramAPIURL,
			wantOptions: models.WebhookOptions{ChatID: "-1001234"}, wantCredentials: true,
		},
		{
			name: "pushover without URL",
			body: map[string]any{"name": "pushover", "format": "pushover",
				"credentials": map[string]any{"token": targetPushoverToken, "user_key": targetPushoverUser}},
			secrets: []string{targetPushoverToken, targetPushoverUser}, wantURL: models.PushoverAPIURL,
			wantCredentials: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, repo := targetTestApp(t)

			status, raw := targetRequest(t, app, http.MethodPost, "/webhooks", tt.body)
			if status != http.StatusCreated {
				t.Fatalf("status = %d, body = %s", status, raw)
			}
			created := decodeWebhook(t, raw)
			if created.URL != tt.wantURL || created.Options != tt.wantOptions || created.HasCredentials != tt.wantCredentials {
				t.Errorf("created = %+v", created)
			}

			// Secrets are neither echoed back nor stored in plaintext
			_, list := targetRequest(t, app, http.MethodGet, "/webhooks", nil)
			stored, err := repo.GetByID(context.Background(), created.ID, targetTestUser)
			if err != nil {
				t.Fatalf("GetByID: %v", err)
			}
			for _, secret := range tt.secrets {
				if strings.Contains(raw, secret) || strings.Contains(list, secret) {
					t.Errorf("a response contains the secret %q", secret)
				}
				if bytes.Contains(stored.CredentialsEnc, []byte(secret)) || strings.Contains(stored.URL, secret) {
					t.Errorf("the secret %q is stored in plaintext", secret)
				}
			}
			if (stored.CredentialsEnc != nil) != tt.wantCredentials {
				t.Errorf("stored credentials = %v, want present = %v", stored.CredentialsEnc != nil, tt.wantCredentials)
			}
		})
	}
}

func TestWebhookHandler_CreateTargets_FieldErrors(t *testing.T) {
	tests := []struct {
		name       string
		body       map[string]any
		wantFields []string
	}{
		{"ntfy without URL", map[string]any{"name": "n", "format": "ntfy"}, []string{"url"}},
		{"ntfy without topic", map[string]any{"name": "n", "format": "ntfy", "url": "https://ntfy.sh"}, []string{"url"}},
		{"ntfy metadata URL", map[string]any{"name": "n", "format": "ntfy", "url": "http://169.254.169.254/alerts"}, []string{"url"}},
		{"ntfy bad priority", map[string]any{"name": "n", "format": "ntfy", "url": "https://ntfy.sh/alerts",
			"options": map[string]any{"priority": 6}}, []string{"priority"}},
		{"gotify without token", map[string]any{"name": "g", "format": "gotify", "url": "http://gotify.lan"}, []string{"token"}},
		{"telegram without anything", map[string]any{"name": "t", "format": "telegram"}, []string{"chat_id", "token"}},
		{"telegram malformed token", map[string]any{"name": "t", "format": "telegram",
			"options": map[string]any{"chat_id": "1"}, "credentials": map[string]any{"token": "123:abc/../getMe"}}, []string{"token"}},
		{"pushover without user key", map[string]any{"name": "p", "format": "pushover",
			"credentials": map[string]any{"token": targetPushoverToken}}, []string{"user_key"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := targetTestApp(t)

			status, raw := targetRequest(t, app, http.MethodPost, "/webhooks", tt.body)
			if status != http.StatusBadRequest {
				t.Fatalf("status = %d, body = %s", status, raw)
			}
			fields := decodeFields(t, raw)
			if len(fields) != len(tt.wantFields) {
				t.Fatalf("fields = %v, want errors for %v", fields, tt.wantFields)
			}
			for _, name := range tt.wantFields {
				if fields[name] == "" {
					t.Errorf("missing error for %q in %v", name, fields)
				}
			}
			if strings.Contains(raw, "abc/../getMe") {
				t.Errorf("the rejected token is echoed back: %s", raw)
			}
		})
	}
}

// Without ENCRYPTION_KEY wiring a secret must be refused, not stored in plaintext
func TestWebhookHandler_CreateTarget_NoCipher(t *testing.T) {
	db := setupWebhookTestDB(t)
	defer db.Close()
	handler := NewWebhookHandler(repository.NewWebhookRepository(db), nil)

	app := fiber.New()
	app.Post("/webhooks", func(c *fiber.Ctx) error {
		c.Locals("user_id", targetTestUser)
		return handler.CreateWebhook(c)
	})

	status, raw := targetRequest(t, app, http.MethodPost, "/webhooks", map[string]any{
		"name": "gotify", "format": "gotify", "url": "http://gotify.lan",
		"credentials": map[string]any{"token": "AppToken.123"},
	})
	if status != http.StatusInternalServerError {
		t.Fatalf("status = %d, body = %s", status, raw)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM webhooks`).Scan(&count); err != nil || count != 0 {
		t.Errorf("stored %d webhooks (err %v), want none", count, err)
	}
}

func TestWebhookHandler_UpdateTargets(t *testing.T) {
	app, repo := targetTestApp(t)
	ctx := context.Background()

	_, raw := targetRequest(t, app, http.MethodPost, "/webhooks", map[string]any{
		"name": "pushover", "format": "pushover",
		"credentials": map[string]any{"token": targetPushoverToken, "user_key": targetPushoverUser},
	})
	id := decodeWebhook(t, raw).ID
	original, err := repo.GetByID(ctx, id, targetTestUser)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	path := "/webhooks/" + id

	// The enable toggle sends only {enabled}: the stored secrets must survive
	status, raw := targetRequest(t, app, http.MethodPut, path, map[string]any{"enabled": false})
	if status != http.StatusOK || !decodeWebhook(t, raw).HasCredentials {
		t.Fatalf("toggle: status = %d, body = %s", status, raw)
	}
	stored, _ := repo.GetByID(ctx, id, targetTestUser)
	if !bytes.Equal(stored.CredentialsEnc, original.CredentialsEnc) {
		t.Error("an update without credentials changed the stored blob")
	}

	// Sent credentials replace the stored ones and are validated
	status, raw = targetRequest(t, app, http.MethodPut, path, map[string]any{
		"credentials": map[string]any{"token": targetPushoverToken},
	})
	if status != http.StatusBadRequest || decodeFields(t, raw)["user_key"] == "" {
		t.Errorf("partial credentials: status = %d, body = %s", status, raw)
	}
	status, raw = targetRequest(t, app, http.MethodPut, path, map[string]any{
		"credentials": map[string]any{"token": targetPushoverUser, "user_key": targetPushoverToken},
	})
	if status != http.StatusOK {
		t.Fatalf("replace credentials: status = %d, body = %s", status, raw)
	}
	stored, _ = repo.GetByID(ctx, id, targetTestUser)
	if bytes.Equal(stored.CredentialsEnc, original.CredentialsEnc) {
		t.Error("new credentials were not stored")
	}

	// Another format needs its own settings: the Pushover keys don't carry over
	status, raw = targetRequest(t, app, http.MethodPut, path, map[string]any{"format": "telegram"})
	if status != http.StatusBadRequest {
		t.Fatalf("format change without settings: status = %d, body = %s", status, raw)
	}
	if fields := decodeFields(t, raw); fields["token"] == "" || fields["chat_id"] == "" {
		t.Errorf("fields = %v, want token and chat_id", fields)
	}

	// Leaving a fixed-endpoint format needs a URL from the user
	status, raw = targetRequest(t, app, http.MethodPut, path, map[string]any{"format": "generic"})
	if status != http.StatusBadRequest || decodeFields(t, raw)["url"] == "" {
		t.Errorf("format change without URL: status = %d, body = %s", status, raw)
	}
	status, raw = targetRequest(t, app, http.MethodPut, path, map[string]any{
		"format": "generic", "url": "https://example.com/hook",
	})
	if status != http.StatusOK {
		t.Fatalf("format change: status = %d, body = %s", status, raw)
	}
	updated := decodeWebhook(t, raw)
	if updated.URL != "https://example.com/hook" || updated.HasCredentials {
		t.Errorf("updated = %+v, want the new URL and no credentials", updated)
	}
}

// An ntfy token is optional, so an empty credentials object removes it
func TestWebhookHandler_UpdateTarget_ClearNtfyToken(t *testing.T) {
	app, _ := targetTestApp(t)

	_, raw := targetRequest(t, app, http.MethodPost, "/webhooks", map[string]any{
		"name": "ntfy", "format": "ntfy", "url": "https://ntfy.sh/alerts",
		"credentials": map[string]any{"token": "tk_secret123"},
	})
	created := decodeWebhook(t, raw)
	if !created.HasCredentials {
		t.Fatalf("created = %+v, want credentials", created)
	}

	status, raw := targetRequest(t, app, http.MethodPut, "/webhooks/"+created.ID, map[string]any{
		"credentials": map[string]any{},
	})
	if status != http.StatusOK || decodeWebhook(t, raw).HasCredentials {
		t.Errorf("clear token: status = %d, body = %s", status, raw)
	}
}

// The test button works for a target: the stored token is decrypted and sent
func TestWebhookHandler_TestTarget(t *testing.T) {
	var gotKey, gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey, gotPath = r.Header.Get("X-Gotify-Key"), r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	app, _ := targetTestApp(t)
	_, raw := targetRequest(t, app, http.MethodPost, "/webhooks", map[string]any{
		"name": "gotify", "format": "gotify", "url": server.URL,
		"credentials": map[string]any{"token": "AppToken.123"},
	})
	id := decodeWebhook(t, raw).ID

	status, raw := targetRequest(t, app, http.MethodPost, "/webhooks/"+id+"/test", nil)
	if status != http.StatusOK || !strings.Contains(raw, `"success":true`) {
		t.Fatalf("status = %d, body = %s", status, raw)
	}
	if gotKey != "AppToken.123" || gotPath != "/message" {
		t.Errorf("Gotify got key %q on %q", gotKey, gotPath)
	}

	// A failing target reports the reason and never the token
	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"error":"Unauthorized","errorDescription":"invalid token AppToken.123"}`)
	})
	status, raw = targetRequest(t, app, http.MethodPost, "/webhooks/"+id+"/test", nil)
	if status != http.StatusBadGateway || !strings.Contains(raw, "HTTP 401: invalid token [redacted]") {
		t.Errorf("status = %d, body = %s", status, raw)
	}
}

// verify_tls defaults to on, can be turned off for a self-hosted target and
// survives an update that doesn't mention it; fixed public APIs always verify
func TestWebhookHandler_VerifyTLS(t *testing.T) {
	app, _ := targetTestApp(t)

	_, raw := targetRequest(t, app, http.MethodPost, "/webhooks", map[string]any{
		"name": "gotify", "format": "gotify", "url": "https://gotify.lan",
		"credentials": map[string]any{"token": "AbC123"},
	})
	created := decodeWebhook(t, raw)
	if !created.VerifyTLS {
		t.Fatalf("created = %+v, want verify_tls on by default", created)
	}

	path := "/webhooks/" + created.ID
	_, raw = targetRequest(t, app, http.MethodPut, path, map[string]any{"verify_tls": false})
	if decodeWebhook(t, raw).VerifyTLS {
		t.Errorf("verify_tls was not turned off: %s", raw)
	}
	_, raw = targetRequest(t, app, http.MethodPut, path, map[string]any{"enabled": false})
	if decodeWebhook(t, raw).VerifyTLS {
		t.Errorf("a toggle reset verify_tls: %s", raw)
	}

	_, raw = targetRequest(t, app, http.MethodPost, "/webhooks", map[string]any{
		"name": "telegram", "format": "telegram", "verify_tls": false,
		"options":     map[string]any{"chat_id": "42"},
		"credentials": map[string]any{"token": "123456789:AAbbCCddEEffGGhhIIjjKKllMMnnOOppQQr"},
	})
	if !decodeWebhook(t, raw).VerifyTLS {
		t.Errorf("Telegram must always verify TLS: %s", raw)
	}
}
