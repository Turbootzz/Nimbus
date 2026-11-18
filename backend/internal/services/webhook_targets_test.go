package services

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/utils"
)

const (
	testTelegramToken = "123456789:AAbbCCddEEffGGhhIIjjKKllMMnnOOppQQr"
	testPushoverToken = "azGDORePK8gMaC0QOYAMyEEuzJnyUi"
	testPushoverUser  = "uQiRzpo4DXghDmr9QzzfQu27cmVRsG"
)

func targetEvent(status string) NotificationEvent {
	event := NotificationEvent{
		ServiceID: "svc-1", ServiceName: "Jellyfin", ServiceURL: "http://jellyfin.lan:8096",
		OldStatus: models.StatusOnline, NewStatus: status,
		UserID: "user-1", Timestamp: time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC),
	}
	if status == models.StatusOffline {
		errMsg := "connection refused"
		event.ErrorMsg = &errMsg
	} else {
		event.OldStatus = models.StatusOffline
	}
	return event
}

func TestBuildNtfyPayload(t *testing.T) {
	offline := buildNtfyPayload("alerts", 5, targetEvent(models.StatusOffline))
	want := map[string]interface{}{
		"topic":    "alerts",
		"title":    "Jellyfin is offline",
		"message":  "connection refused",
		"priority": 5,
		"tags":     []string{"rotating_light"},
		"click":    "http://jellyfin.lan:8096",
	}
	if !reflect.DeepEqual(offline, want) {
		t.Errorf("offline payload = %v, want %v", offline, want)
	}

	event := targetEvent(models.StatusOnline)
	event.ServiceURL = ""
	online := buildNtfyPayload("alerts", 0, event)
	if online["title"] != "Jellyfin is back online" {
		t.Errorf("online title = %v", online["title"])
	}
	if online["priority"] != ntfyDefaultPriority {
		t.Errorf("unset priority = %v, want the default %d", online["priority"], ntfyDefaultPriority)
	}
	if !reflect.DeepEqual(online["tags"], []string{"heavy_check_mark"}) {
		t.Errorf("online tags = %v", online["tags"])
	}
	if _, ok := online["click"]; ok {
		t.Error("click must be left out without a service URL")
	}
}

func TestBuildGotifyPayload(t *testing.T) {
	offline := buildGotifyPayload(targetEvent(models.StatusOffline))
	want := map[string]interface{}{
		"title":    "Jellyfin is offline",
		"message":  "connection refused",
		"priority": gotifyPriorityOffline,
		"extras": map[string]interface{}{
			"client::notification": map[string]interface{}{
				"click": map[string]interface{}{"url": "http://jellyfin.lan:8096"},
			},
		},
	}
	if !reflect.DeepEqual(offline, want) {
		t.Errorf("offline payload = %v, want %v", offline, want)
	}

	online := buildGotifyPayload(targetEvent(models.StatusOnline))
	if online["priority"] != gotifyPriorityOnline {
		t.Errorf("online priority = %v, want %d", online["priority"], gotifyPriorityOnline)
	}
	if online["message"] != "The service is back up and responding normally." {
		t.Errorf("online message = %v", online["message"])
	}
}

func TestBuildTelegramPayload(t *testing.T) {
	offline := buildTelegramPayload("-1001234", targetEvent(models.StatusOffline))
	want := map[string]interface{}{
		"chat_id": "-1001234",
		"text":    "🔴 Jellyfin is offline\nconnection refused\nhttp://jellyfin.lan:8096",
	}
	if !reflect.DeepEqual(offline, want) {
		t.Errorf("offline payload = %v, want %v", offline, want)
	}

	// No parse_mode: markup characters in names must go out untouched
	event := targetEvent(models.StatusOnline)
	event.ServiceName = "my_app <v2> & *co*"
	online := buildTelegramPayload("@homelab", event)
	if !strings.HasPrefix(online["text"].(string), "🟢 my_app <v2> & *co* is back online\n") {
		t.Errorf("online text = %q", online["text"])
	}
	if _, ok := online["parse_mode"]; ok {
		t.Error("parse_mode must not be set")
	}

	long := strings.Repeat("é", 5000)
	event.ErrorMsg = &long
	event.NewStatus = models.StatusOffline
	if got := len([]rune(buildTelegramPayload("1", event)["text"].(string))); got != telegramMaxText {
		t.Errorf("long text has %d characters, want %d", got, telegramMaxText)
	}
}

func TestBuildPushoverPayload(t *testing.T) {
	creds := models.WebhookCredentials{Token: testPushoverToken, UserKey: testPushoverUser}
	offline := buildPushoverPayload(creds, targetEvent(models.StatusOffline))
	want := map[string]interface{}{
		"token":     testPushoverToken,
		"user":      testPushoverUser,
		"title":     "Jellyfin is offline",
		"message":   "connection refused",
		"url":       "http://jellyfin.lan:8096",
		"url_title": "Open Jellyfin",
	}
	if !reflect.DeepEqual(offline, want) {
		t.Errorf("offline payload = %v, want %v", offline, want)
	}

	// Pushover rejects oversized fields, so they are cut or left out
	event := targetEvent(models.StatusOffline)
	long := strings.Repeat("x", 2000)
	event.ErrorMsg = &long
	event.ServiceURL = "https://example.com/" + strings.Repeat("a", 600)
	payload := buildPushoverPayload(creds, event)
	if got := len([]rune(payload["message"].(string))); got != pushoverMaxMessage {
		t.Errorf("message has %d characters, want %d", got, pushoverMaxMessage)
	}
	if _, ok := payload["url"]; ok {
		t.Error("a URL over the limit must be left out")
	}
}

func TestSplitNtfyTopicURL(t *testing.T) {
	tests := []struct {
		url, server, topic string
		ok                 bool
	}{
		{"https://ntfy.sh/alerts", "https://ntfy.sh/", "alerts", true},
		{"https://ntfy.sh/alerts/", "https://ntfy.sh/", "alerts", true},
		{"http://192.168.1.5:8080/ntfy/home_lab-1?x=1#y", "http://192.168.1.5:8080/ntfy/", "home_lab-1", true},
		{"https://ntfy.sh", "", "", false},
		{"https://ntfy.sh/", "", "", false},
		{"https://ntfy.sh/bad topic", "", "", false},
		{"ntfy.sh/alerts", "", "", false},
		{"", "", "", false},
	}
	for _, tt := range tests {
		server, topic, ok := splitNtfyTopicURL(tt.url)
		if server != tt.server || topic != tt.topic || ok != tt.ok {
			t.Errorf("splitNtfyTopicURL(%q) = %q, %q, %v; want %q, %q, %v",
				tt.url, server, topic, ok, tt.server, tt.topic, tt.ok)
		}
	}
}

func TestPrepareWebhookTarget(t *testing.T) {
	tests := []struct {
		name       string
		webhook    models.Webhook
		creds      *models.WebhookCredentials
		wantFields []string
	}{
		{
			name:    "ntfy without token",
			webhook: models.Webhook{Format: models.WebhookFormatNtfy, URL: "https://ntfy.sh/alerts"},
			creds:   &models.WebhookCredentials{},
		},
		{
			name: "ntfy bad topic, priority and token",
			webhook: models.Webhook{Format: models.WebhookFormatNtfy, URL: "https://ntfy.sh/",
				Options: models.WebhookOptions{Priority: 9}},
			creds:      &models.WebhookCredentials{Token: "has space"},
			wantFields: []string{"priority", "token", "url"},
		},
		{
			name:       "gotify needs a token",
			webhook:    models.Webhook{Format: models.WebhookFormatGotify, URL: "http://gotify.lan"},
			creds:      &models.WebhookCredentials{},
			wantFields: []string{"token"},
		},
		{
			name:    "gotify valid",
			webhook: models.Webhook{Format: models.WebhookFormatGotify, URL: "http://gotify.lan"},
			creds:   &models.WebhookCredentials{Token: "AbC.dEf-123"},
		},
		{
			name:       "telegram needs token and chat",
			webhook:    models.Webhook{Format: models.WebhookFormatTelegram},
			creds:      &models.WebhookCredentials{},
			wantFields: []string{"chat_id", "token"},
		},
		{
			name: "telegram token that would change the URL path",
			webhook: models.Webhook{Format: models.WebhookFormatTelegram,
				Options: models.WebhookOptions{ChatID: "12345"}},
			creds:      &models.WebhookCredentials{Token: "123:abc/../getMe?x="},
			wantFields: []string{"token"},
		},
		{
			name: "telegram bad chat id",
			webhook: models.Webhook{Format: models.WebhookFormatTelegram,
				Options: models.WebhookOptions{ChatID: "my chat"}},
			creds:      &models.WebhookCredentials{Token: testTelegramToken},
			wantFields: []string{"chat_id"},
		},
		{
			name: "telegram valid",
			webhook: models.Webhook{Format: models.WebhookFormatTelegram,
				Options: models.WebhookOptions{ChatID: " -1001234 "}},
			creds: &models.WebhookCredentials{Token: testTelegramToken},
		},
		{
			name:       "pushover needs both keys",
			webhook:    models.Webhook{Format: models.WebhookFormatPushover},
			creds:      &models.WebhookCredentials{Token: "short"},
			wantFields: []string{"token", "user_key"},
		},
		{
			name:    "pushover valid",
			webhook: models.Webhook{Format: models.WebhookFormatPushover},
			creds:   &models.WebhookCredentials{Token: testPushoverToken, UserKey: testPushoverUser},
		},
		{
			name: "kept credentials are not checked",
			webhook: models.Webhook{Format: models.WebhookFormatTelegram,
				Options: models.WebhookOptions{ChatID: "12345"}},
			creds: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := PrepareWebhookTarget(&tt.webhook, tt.creds)
			if len(fields) != len(tt.wantFields) {
				t.Fatalf("fields = %v, want errors for %v", fields, tt.wantFields)
			}
			for _, name := range tt.wantFields {
				if fields[name] == "" {
					t.Errorf("missing error for %q in %v", name, fields)
				}
			}
		})
	}
}

func TestPrepareWebhookTarget_Normalises(t *testing.T) {
	telegram := models.Webhook{Format: models.WebhookFormatTelegram, URL: "https://evil.example",
		Options: models.WebhookOptions{ChatID: " 42 ", Priority: 4}}
	creds := models.WebhookCredentials{Token: testTelegramToken, UserKey: "unused"}
	PrepareWebhookTarget(&telegram, &creds)
	if telegram.URL != models.TelegramAPIURL {
		t.Errorf("Telegram URL = %q, want the fixed API URL", telegram.URL)
	}
	if telegram.Options != (models.WebhookOptions{ChatID: "42"}) {
		t.Errorf("Telegram options = %+v", telegram.Options)
	}
	if creds.UserKey != "" {
		t.Error("Telegram must not keep a user key")
	}

	pushover := models.Webhook{Format: models.WebhookFormatPushover}
	PrepareWebhookTarget(&pushover, nil)
	if pushover.URL != models.PushoverAPIURL {
		t.Errorf("Pushover URL = %q, want the fixed API URL", pushover.URL)
	}

	ntfy := models.Webhook{Format: models.WebhookFormatNtfy, URL: "https://ntfy.sh/alerts"}
	PrepareWebhookTarget(&ntfy, nil)
	if ntfy.Options.Priority != ntfyDefaultPriority {
		t.Errorf("ntfy priority = %d, want the default", ntfy.Options.Priority)
	}

	// Plain webhooks store no options or credentials
	generic := models.Webhook{Format: models.WebhookFormatGeneric, URL: "https://example.com/hook",
		Options: models.WebhookOptions{ChatID: "1", Priority: 2}}
	creds = models.WebhookCredentials{Token: "abc", UserKey: "def"}
	if fields := PrepareWebhookTarget(&generic, &creds); len(fields) != 0 {
		t.Errorf("generic fields = %v", fields)
	}
	if generic.Options != (models.WebhookOptions{}) || creds != (models.WebhookCredentials{}) {
		t.Errorf("generic kept options %+v or credentials", generic.Options)
	}
	if generic.URL != "https://example.com/hook" {
		t.Errorf("generic URL changed to %q", generic.URL)
	}
}

// capturedRequest is what a fake notification service received
type capturedRequest struct {
	Method string
	Path   string
	Header http.Header
	Body   map[string]interface{}
}

// targetServer records every request and answers with the given status and body
func targetServer(t *testing.T, status int, body string) (*httptest.Server, func() []capturedRequest) {
	t.Helper()
	var mu sync.Mutex
	var requests []capturedRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured := capturedRequest{Method: r.Method, Path: r.URL.Path, Header: r.Header.Clone()}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &captured.Body)
		mu.Lock()
		requests = append(requests, captured)
		mu.Unlock()
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(server.Close)
	return server, func() []capturedRequest {
		mu.Lock()
		defer mu.Unlock()
		return append([]capturedRequest(nil), requests...)
	}
}

// newTargetNotifier builds a notifier with a cipher on an in-memory database
func newTargetNotifier(t *testing.T, serviceRepo repository.ServiceRepositoryInterface) *WebhookNotifier {
	t.Helper()
	cipher, err := utils.NewCipher(bytes.Repeat([]byte{9}, 32))
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}
	notifier := NewWebhookNotifier(setupWebhookNotifierTestDB(t), serviceRepo)
	notifier.cipher = cipher
	return notifier
}

// saveTarget stores a webhook with encrypted credentials, like the handler does
func saveTarget(t *testing.T, notifier *WebhookNotifier, webhook *models.Webhook, creds models.WebhookCredentials) *models.Webhook {
	t.Helper()
	webhook.ID = "wh-" + webhook.Format
	webhook.UserID = "user-1"
	webhook.Name = webhook.Format + " target"
	webhook.Enabled = true
	webhook.Triggers = models.WebhookTriggers{OnOffline: true, OnOnline: true}
	blob, err := notifier.encryptCredentials(creds, webhook.ID)
	if err != nil {
		t.Fatalf("encryptCredentials: %v", err)
	}
	webhook.CredentialsEnc = blob
	if err := notifier.webhookRepo.Create(context.Background(), webhook); err != nil {
		t.Fatalf("Create: %v", err)
	}
	return webhook
}

func assertJSONRequest(t *testing.T, requests []capturedRequest, path string) capturedRequest {
	t.Helper()
	if len(requests) != 1 {
		t.Fatalf("got %d requests, want 1", len(requests))
	}
	req := requests[0]
	if req.Method != http.MethodPost {
		t.Errorf("method = %s, want POST", req.Method)
	}
	if req.Path != path {
		t.Errorf("path = %q, want %q", req.Path, path)
	}
	if got := req.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q", got)
	}
	return req
}

func TestDeliver_Ntfy(t *testing.T) {
	server, requests := targetServer(t, http.StatusOK, `{"id":"abc"}`)
	notifier := newTargetNotifier(t, nil)
	webhook := saveTarget(t, notifier, &models.Webhook{
		Format: models.WebhookFormatNtfy, URL: server.URL + "/alerts",
		Options: models.WebhookOptions{Priority: 4},
	}, models.WebhookCredentials{Token: "tk_secret123"})

	status, errMsg := notifier.deliver(context.Background(), webhook, targetEvent(models.StatusOffline))
	if status != http.StatusOK || errMsg != "" {
		t.Fatalf("deliver = %d, %q", status, errMsg)
	}

	// JSON publishing posts to the server root, with the topic in the body
	req := assertJSONRequest(t, requests(), "/")
	if got := req.Header.Get("Authorization"); got != "Bearer tk_secret123" {
		t.Errorf("Authorization = %q", got)
	}
	want := map[string]interface{}{
		"topic": "alerts", "title": "Jellyfin is offline", "message": "connection refused",
		"priority": float64(4), "tags": []interface{}{"rotating_light"}, "click": "http://jellyfin.lan:8096",
	}
	if !reflect.DeepEqual(req.Body, want) {
		t.Errorf("body = %v, want %v", req.Body, want)
	}
}

func TestDeliver_NtfyWithoutToken(t *testing.T) {
	server, requests := targetServer(t, http.StatusOK, `{}`)
	notifier := newTargetNotifier(t, nil)
	webhook := saveTarget(t, notifier, &models.Webhook{
		Format: models.WebhookFormatNtfy, URL: server.URL + "/alerts",
	}, models.WebhookCredentials{})

	if _, errMsg := notifier.deliver(context.Background(), webhook, targetEvent(models.StatusOnline)); errMsg != "" {
		t.Fatalf("deliver failed: %s", errMsg)
	}
	req := assertJSONRequest(t, requests(), "/")
	if _, ok := req.Header["Authorization"]; ok {
		t.Error("a public topic must be sent without an Authorization header")
	}
}

func TestDeliver_Gotify(t *testing.T) {
	server, requests := targetServer(t, http.StatusOK, `{"id":1}`)
	notifier := newTargetNotifier(t, nil)
	// Gotify behind a sub path: the message endpoint is appended to it
	webhook := saveTarget(t, notifier, &models.Webhook{
		Format: models.WebhookFormatGotify, URL: server.URL + "/gotify/",
	}, models.WebhookCredentials{Token: "AppToken.123"})

	status, errMsg := notifier.deliver(context.Background(), webhook, targetEvent(models.StatusOffline))
	if status != http.StatusOK || errMsg != "" {
		t.Fatalf("deliver = %d, %q", status, errMsg)
	}

	req := assertJSONRequest(t, requests(), "/gotify/message")
	if got := req.Header.Get("X-Gotify-Key"); got != "AppToken.123" {
		t.Errorf("X-Gotify-Key = %q", got)
	}
	if req.Body["title"] != "Jellyfin is offline" || req.Body["message"] != "connection refused" ||
		req.Body["priority"] != float64(gotifyPriorityOffline) {
		t.Errorf("body = %v", req.Body)
	}
}

func TestDeliver_Telegram(t *testing.T) {
	server, requests := targetServer(t, http.StatusOK, `{"ok":true}`)
	notifier := newTargetNotifier(t, nil)
	webhook := saveTarget(t, notifier, &models.Webhook{
		Format: models.WebhookFormatTelegram, URL: server.URL,
		Options: models.WebhookOptions{ChatID: "-1001234"},
	}, models.WebhookCredentials{Token: testTelegramToken})

	status, errMsg := notifier.deliver(context.Background(), webhook, targetEvent(models.StatusOffline))
	if status != http.StatusOK || errMsg != "" {
		t.Fatalf("deliver = %d, %q", status, errMsg)
	}

	req := assertJSONRequest(t, requests(), "/bot"+testTelegramToken+"/sendMessage")
	want := map[string]interface{}{
		"chat_id": "-1001234",
		"text":    "🔴 Jellyfin is offline\nconnection refused\nhttp://jellyfin.lan:8096",
	}
	if !reflect.DeepEqual(req.Body, want) {
		t.Errorf("body = %v, want %v", req.Body, want)
	}
}

func TestDeliver_Pushover(t *testing.T) {
	server, requests := targetServer(t, http.StatusOK, `{"status":1,"request":"r1"}`)
	notifier := newTargetNotifier(t, nil)
	webhook := saveTarget(t, notifier, &models.Webhook{
		Format: models.WebhookFormatPushover, URL: server.URL + "/1/messages.json",
	}, models.WebhookCredentials{Token: testPushoverToken, UserKey: testPushoverUser})

	status, errMsg := notifier.deliver(context.Background(), webhook, targetEvent(models.StatusOnline))
	if status != http.StatusOK || errMsg != "" {
		t.Fatalf("deliver = %d, %q", status, errMsg)
	}

	req := assertJSONRequest(t, requests(), "/1/messages.json")
	want := map[string]interface{}{
		"token": testPushoverToken, "user": testPushoverUser,
		"title":   "Jellyfin is back online",
		"message": "The service is back up and responding normally.",
		"url":     "http://jellyfin.lan:8096", "url_title": "Open Jellyfin",
	}
	if !reflect.DeepEqual(req.Body, want) {
		t.Errorf("body = %v, want %v", req.Body, want)
	}
}

// A rejected notification is recorded as a failure with the reason the
// service gave, for every target's error format.
func TestSendWebhook_TargetFailureIsRecorded(t *testing.T) {
	tests := []struct {
		format   string
		status   int
		response string
		wantErr  string
	}{
		{models.WebhookFormatTelegram, 400, `{"ok":false,"error_code":400,"description":"Bad Request: chat not found"}`, "HTTP 400: Bad Request: chat not found"},
		{models.WebhookFormatPushover, 400, `{"status":0,"errors":["user identifier is invalid"]}`, "HTTP 400: user identifier is invalid"},
		{models.WebhookFormatGotify, 401, `{"error":"Unauthorized","errorCode":401,"errorDescription":"you need to provide a valid access token"}`, "HTTP 401: you need to provide a valid access token"},
		{models.WebhookFormatNtfy, 403, `{"code":40301,"http":403,"error":"forbidden"}`, "HTTP 403: forbidden"},
		{models.WebhookFormatNtfy, 502, `<html>Bad Gateway</html>`, "HTTP 502"},
	}

	for _, tt := range tests {
		t.Run(tt.format+" "+tt.wantErr, func(t *testing.T) {
			server, requests := targetServer(t, tt.status, tt.response)
			notifier := newTargetNotifier(t, nil)
			webhook := &models.Webhook{Format: tt.format, URL: server.URL,
				Options: models.WebhookOptions{ChatID: "1"}}
			if tt.format == models.WebhookFormatNtfy {
				webhook.URL += "/alerts"
			}
			saveTarget(t, notifier, webhook, models.WebhookCredentials{Token: testTelegramToken, UserKey: testPushoverUser})

			notifier.sendWebhook(context.Background(), webhook, targetEvent(models.StatusOffline))

			if len(requests()) != 1 {
				t.Fatalf("got %d requests, want exactly 1 (a failed delivery is not resent)", len(requests()))
			}
			ctx := context.Background()
			logs, err := notifier.webhookRepo.GetLogsByWebhookID(ctx, webhook.ID, 10)
			if err != nil || len(logs) != 1 {
				t.Fatalf("logs = %v, err = %v", logs, err)
			}
			entry := logs[0]
			if entry.Success || entry.StatusCode == nil || *entry.StatusCode != tt.status {
				t.Errorf("log success = %v, status = %v", entry.Success, entry.StatusCode)
			}
			if entry.ErrorMessage == nil || *entry.ErrorMessage != tt.wantErr {
				t.Errorf("log error = %v, want %q", entry.ErrorMessage, tt.wantErr)
			}
			stored, err := notifier.webhookRepo.GetByID(ctx, webhook.ID, "user-1")
			if err != nil {
				t.Fatalf("GetByID: %v", err)
			}
			if stored.TotalFailed != 1 || stored.ConsecutiveFailures != 1 {
				t.Errorf("failed = %d, consecutive = %d, want 1 and 1", stored.TotalFailed, stored.ConsecutiveFailures)
			}
		})
	}
}

// Telegram's bot token is part of the request URL. A transport error must
// not carry that URL into the server log, the delivery log or the test result.
func TestTelegramTokenNeverLeaks(t *testing.T) {
	var logged bytes.Buffer
	log.SetOutput(&logged)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	dead := httptest.NewServer(http.NotFoundHandler())
	dead.Close()

	notifier := newTargetNotifier(t, nil)
	webhook := saveTarget(t, notifier, &models.Webhook{
		Format: models.WebhookFormatTelegram, URL: dead.URL,
		Options: models.WebhookOptions{ChatID: "1"},
	}, models.WebhookCredentials{Token: testTelegramToken})
	ctx := context.Background()

	notifier.sendWebhook(ctx, webhook, targetEvent(models.StatusOffline))
	result, err := notifier.TestWebhook(ctx, webhook.ID, "user-1")
	if err != nil {
		t.Fatalf("TestWebhook: %v", err)
	}
	if result.Success || result.Error == "" {
		t.Fatalf("expected a failed test with an error, got %+v", result)
	}

	// Stop logging into the buffer before reading it: goroutines of earlier
	// tests may still log
	log.SetOutput(os.Stderr)

	logs, err := notifier.webhookRepo.GetLogsByWebhookID(ctx, webhook.ID, 10)
	if err != nil || len(logs) != 1 || logs[0].ErrorMessage == nil {
		t.Fatalf("logs = %v, err = %v", logs, err)
	}
	for name, text := range map[string]string{
		"server log":   logged.String(),
		"delivery log": *logs[0].ErrorMessage,
		"test result":  result.Error,
	} {
		if strings.Contains(text, testTelegramToken) || strings.Contains(text, "/bot") {
			t.Errorf("%s leaks the bot token URL: %s", name, text)
		}
	}
	if !strings.Contains(result.Error, "connect") {
		t.Errorf("error should still say what went wrong, got %q", result.Error)
	}
}

func TestResponseError_ScrubsEchoedSecrets(t *testing.T) {
	server, _ := targetServer(t, http.StatusBadRequest, `{"errors":["token `+testPushoverToken+` is invalid"]}`)
	notifier := newTargetNotifier(t, nil)
	webhook := saveTarget(t, notifier, &models.Webhook{
		Format: models.WebhookFormatPushover, URL: server.URL,
	}, models.WebhookCredentials{Token: testPushoverToken, UserKey: testPushoverUser})

	_, errMsg := notifier.deliver(context.Background(), webhook, targetEvent(models.StatusOffline))
	if errMsg != "HTTP 400: token [redacted] is invalid" {
		t.Errorf("error = %q", errMsg)
	}
}

func TestDeliver_UnreadableCredentials(t *testing.T) {
	server, requests := targetServer(t, http.StatusOK, `{}`)
	notifier := newTargetNotifier(t, nil)
	webhook := saveTarget(t, notifier, &models.Webhook{
		Format: models.WebhookFormatGotify, URL: server.URL,
	}, models.WebhookCredentials{Token: "AppToken.123"})

	// A blob moved to another webhook is bound to the wrong ID
	moved := *webhook
	moved.ID = "wh-other"
	if _, errMsg := notifier.deliver(context.Background(), &moved, targetEvent(models.StatusOffline)); errMsg == "" {
		t.Error("expected an error for a blob bound to another webhook")
	}

	// Without ENCRYPTION_KEY the secrets can't be read either
	notifier.cipher = nil
	if _, errMsg := notifier.deliver(context.Background(), webhook, targetEvent(models.StatusOffline)); errMsg == "" {
		t.Error("expected an error without a cipher")
	}

	if len(requests()) != 0 {
		t.Errorf("got %d requests, want none without credentials", len(requests()))
	}
}

func TestEncryptCredentials(t *testing.T) {
	notifier := newTargetNotifier(t, nil)
	creds := models.WebhookCredentials{Token: testPushoverToken, UserKey: testPushoverUser}

	blob, err := notifier.encryptCredentials(creds, "wh-1")
	if err != nil {
		t.Fatalf("encryptCredentials: %v", err)
	}
	if bytes.Contains(blob, []byte(testPushoverToken)) || bytes.Contains(blob, []byte(testPushoverUser)) {
		t.Error("the stored blob contains a secret in plaintext")
	}
	got, err := notifier.decryptCredentials(&models.Webhook{ID: "wh-1", CredentialsEnc: blob})
	if err != nil || got != creds {
		t.Errorf("round trip = %+v, %v", got, err)
	}

	if blob, err := notifier.encryptCredentials(models.WebhookCredentials{}, "wh-1"); blob != nil || err != nil {
		t.Errorf("empty credentials = %v, %v; want nothing stored", blob, err)
	}

	notifier.cipher = nil
	if _, err := notifier.encryptCredentials(creds, "wh-1"); err == nil {
		t.Error("expected an error without a cipher")
	}
}

// The offline re-check ("notification retries") is shared with plain
// webhooks: a target is only notified once the service is confirmed offline.
func TestNotifyWithRetry_Target(t *testing.T) {
	tests := []struct {
		name         string
		statusOnPoll string
		wantRequests int
	}{
		{"still offline sends once", models.StatusOffline, 1},
		{"recovered sends nothing", models.StatusOnline, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearRetryPending()
			rateLimitMu.Lock()
			rateLimitCache = make(map[rateLimitKey]time.Time)
			rateLimitMu.Unlock()

			server, requests := targetServer(t, http.StatusOK, `{}`)
			notifier := newTargetNotifier(t, &retryTestServiceRepo{status: tt.statusOnPoll})
			webhook := saveTarget(t, notifier, &models.Webhook{
				Format: models.WebhookFormatNtfy, URL: server.URL + "/alerts",
				RetryCount: 2, RetryDelaySeconds: 0,
			}, models.WebhookCredentials{Token: "tk_secret123"})

			notifier.notifyWithRetry(context.Background(), webhook, targetEvent(models.StatusOffline))

			got := requests()
			if len(got) != tt.wantRequests {
				t.Fatalf("got %d requests, want %d", len(got), tt.wantRequests)
			}
			if tt.wantRequests == 1 && got[0].Body["topic"] != "alerts" {
				t.Errorf("body = %v", got[0].Body)
			}
		})
	}
}
