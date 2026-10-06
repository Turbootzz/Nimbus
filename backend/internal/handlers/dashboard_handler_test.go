package handlers

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/services"
	"github.com/nimbus/backend/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dashboardFixture struct {
	db   *sql.DB
	live *services.LiveDataService
	hub  *services.SSEHub
}

func newDashboardFixture(t *testing.T) *dashboardFixture {
	t.Helper()
	db := setupWidgetTestDB(t)
	_, err := db.Exec(`CREATE TABLE widget_snapshots (
		source_kind TEXT NOT NULL, source_id TEXT NOT NULL, user_id TEXT NOT NULL,
		payload TEXT, error TEXT, fetched_at TIMESTAMP NOT NULL,
		PRIMARY KEY (source_kind, source_id))`)
	require.NoError(t, err)

	cipher, err := utils.NewCipher(bytes.Repeat([]byte{5}, 32))
	require.NoError(t, err)
	integrationRepo := repository.NewIntegrationRepository(db)
	hub := services.NewSSEHub()
	t.Cleanup(hub.Close)
	live := services.NewLiveDataService(
		repository.NewWidgetRepository(db),
		integrationRepo,
		services.NewIntegrationService(integrationRepo, cipher, ""),
		repository.NewSnapshotRepository(db),
		hub,
	)
	return &dashboardFixture{db: db, live: live, hub: hub}
}

func (f *dashboardFixture) app(userID string, pingInterval time.Duration) *fiber.App {
	handler := NewDashboardHandler(f.live, f.hub)
	handler.pingInterval = pingInterval
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", userID)
		return c.Next()
	})
	app.Get("/dashboard/data", handler.Data)
	app.Get("/dashboard/stream", handler.Stream)
	return app
}

func widgetSource(id, userID string) services.LiveSource {
	return services.LiveSource{Kind: models.SnapshotSourceWidget, ID: id, UserID: userID}
}

func TestDashboardHandler_DataIsPerUser(t *testing.T) {
	f := newDashboardFixture(t)
	f.live.Record(widgetSource("w1", widgetOwnerID), map[string]float64{"temperature": 12.5}, nil)
	f.live.Record(widgetSource("w2", widgetOtherID), map[string]int{"x": 1}, nil)

	status, body := doWidgetRequest(t, f.app(widgetOwnerID, time.Minute), http.MethodGet, "/dashboard/data", "")
	require.Equal(t, fiber.StatusOK, status)
	var snaps []models.Snapshot
	require.NoError(t, json.Unmarshal([]byte(body), &snaps))
	require.Len(t, snaps, 1)
	assert.Equal(t, "w1", snaps[0].SourceID)
	assert.JSONEq(t, `{"temperature":12.5}`, string(snaps[0].Payload))
	assert.NotContains(t, body, widgetOwnerID, "user_id is not in the response")

	status, body = doWidgetRequest(t, f.app("dddddddd-dddd-dddd-dddd-ddddddddddd9", time.Minute), http.MethodGet, "/dashboard/data", "")
	require.Equal(t, fiber.StatusOK, status)
	assert.Equal(t, "[]", body)
}

// sseReader reads events from a live stream
type sseReader struct {
	t *testing.T
	r *bufio.Reader
}

// next returns the next event or comment block, e.g. "event: hello\ndata: {}"
func (s *sseReader) next() string {
	s.t.Helper()
	var lines []string
	for {
		line, err := s.r.ReadString('\n')
		require.NoError(s.t, err)
		line = strings.TrimRight(line, "\n")
		if line == "" {
			return strings.Join(lines, "\n")
		}
		lines = append(lines, line)
	}
}

func TestDashboardHandler_Stream(t *testing.T) {
	f := newDashboardFixture(t)
	app := f.app(widgetOwnerID, 100*time.Millisecond)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go func() { _ = app.Listener(ln) }()
	t.Cleanup(func() { _ = app.Shutdown() })

	resp, err := http.Get("http://" + ln.Addr().String() + "/dashboard/stream")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))
	assert.Equal(t, "no", resp.Header.Get("X-Accel-Buffering"))
	assert.Equal(t, "no-cache", resp.Header.Get("Cache-Control"))

	events := &sseReader{t: t, r: bufio.NewReader(resp.Body)}
	assert.Equal(t, "event: hello\ndata: {}", events.next())

	f.live.Record(widgetSource("w1", widgetOwnerID), map[string]int{"n": 1}, nil)
	f.live.Record(widgetSource("w2", widgetOtherID), map[string]int{"n": 2}, nil) // not ours
	snapshot := events.next()
	assert.True(t, strings.HasPrefix(snapshot, "event: snapshot\ndata: "), snapshot)
	assert.Contains(t, snapshot, `"source_id":"w1"`)
	assert.Contains(t, snapshot, `"payload":{"n":1}`)

	ms := 42
	f.live.PublishServiceStatus(widgetOwnerID, "s1", models.StatusOffline, &ms)
	assert.Equal(t, `event: service_status`+"\n"+`data: {"id":"s1","status":"offline","response_time":42}`, events.next())

	assert.Equal(t, ": ping", events.next(), "keeps the connection alive")

	// Closing the hub (server shutdown) ends the stream
	f.hub.Close()
	ended := make(chan struct{})
	go func() {
		_, _ = io.Copy(io.Discard, events.r)
		close(ended)
	}()
	select {
	case <-ended:
	case <-time.After(2 * time.Second):
		t.Fatal("stream did not end after the hub closed")
	}
}

func TestDashboardHandler_StreamLimitPerUser(t *testing.T) {
	f := newDashboardFixture(t)
	for range 20 {
		_, err := f.hub.Subscribe(widgetOwnerID)
		require.NoError(t, err)
	}

	req := httptest.NewRequest(http.MethodGet, "/dashboard/stream", nil)
	resp, err := f.app(widgetOwnerID, time.Minute).Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusTooManyRequests, resp.StatusCode)
}

// recordingPoller remembers what the widget service asked for and accepts
// one refresh per key
type recordingPoller struct {
	kicks     int
	refreshed []string
}

func (p *recordingPoller) Kick() { p.kicks++ }
func (p *recordingPoller) Refresh(key string) bool {
	for _, k := range p.refreshed {
		if k == key {
			return false
		}
	}
	p.refreshed = append(p.refreshed, key)
	return true
}

func TestWidgetHandler_RefreshAndKicks(t *testing.T) {
	db := setupWidgetTestDB(t)
	service := services.NewWidgetService(repository.NewWidgetRepository(db), repository.NewGroupRepository(db), repository.NewIntegrationRepository(db))
	poller := &recordingPoller{}
	service.SetPoller(poller)
	handler := NewWidgetHandler(service)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", c.Get("X-User", widgetOwnerID))
		return c.Next()
	})
	app.Post("/widgets", handler.CreateWidget)
	app.Put("/widgets/:id", handler.UpdateWidget)
	app.Delete("/widgets/:id", handler.DeleteWidget)
	app.Post("/widgets/:id/refresh", handler.RefreshWidget)

	weather := createWidget(t, app, `{"type":"weather","config":{"latitude":52.4,"longitude":4.9}}`)
	clock := createWidget(t, app, `{"type":"clock"}`)
	assert.Equal(t, 2, poller.kicks, "every create kicks the poller")
	assert.Equal(t, 600, weather.RefreshSeconds, "weather's minimum refresh")

	status, _ := doWidgetRequest(t, app, http.MethodPost, "/widgets/"+weather.ID+"/refresh", "")
	assert.Equal(t, fiber.StatusAccepted, status)
	assert.Equal(t, []string{"widget:" + weather.ID}, poller.refreshed)
	status, body := doWidgetRequest(t, app, http.MethodPost, "/widgets/"+weather.ID+"/refresh", "")
	assert.Equal(t, fiber.StatusTooManyRequests, status)
	assert.Contains(t, body, "moments ago")

	status, body = doWidgetRequest(t, app, http.MethodPost, "/widgets/"+clock.ID+"/refresh", "")
	assert.Equal(t, fiber.StatusBadRequest, status)
	assert.Contains(t, body, "no data to refresh")

	req := httptest.NewRequest(http.MethodPost, "/widgets/"+weather.ID+"/refresh", nil)
	req.Header.Set("X-User", widgetOtherID)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)

	status, _ = doWidgetRequest(t, app, http.MethodPut, "/widgets/"+weather.ID, `{"enabled":false}`)
	require.Equal(t, fiber.StatusOK, status)
	status, body = doWidgetRequest(t, app, http.MethodPost, "/widgets/"+weather.ID+"/refresh", "")
	assert.Equal(t, fiber.StatusBadRequest, status)
	assert.Contains(t, body, "disabled")

	status, _ = doWidgetRequest(t, app, http.MethodDelete, "/widgets/"+weather.ID, "")
	require.Equal(t, fiber.StatusNoContent, status)
	assert.Equal(t, 4, poller.kicks, "update and delete kick too")
	assert.Len(t, poller.refreshed, 1)
}
