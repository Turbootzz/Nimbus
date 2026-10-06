package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/nimbus/backend/internal/integrations"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/widgets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// liveCounter polls nothing and returns a fixed payload
type liveCounter struct{}

func (liveCounter) Type() string { return "svc-counter" }
func (liveCounter) Meta() widgets.Meta {
	return widgets.Meta{Name: "Counter", Category: widgets.CategoryInfo, DefaultSize: "2x1", AllowedSizes: []string{"2x1"}}
}
func (liveCounter) Validate(c json.RawMessage) (json.RawMessage, error) { return c, nil }
func (liveCounter) Fetch(_ context.Context, req *widgets.FetchRequest) (any, error) {
	if req.Client == nil {
		return nil, errors.New("no client")
	}
	return map[string]any{"count": 42, "config": req.Config}, nil
}

// liveQueue needs an svc-fake integration and calls BaseURL/queue with the
// API key in the query, so redaction of net/http errors is exercised
type liveQueue struct{}

func (liveQueue) Type() string { return "svc-queue" }
func (liveQueue) Meta() widgets.Meta {
	return widgets.Meta{Name: "Queue", Category: widgets.CategoryInfo, DefaultSize: "2x1", AllowedSizes: []string{"2x1"}, IntegrationKinds: []string{"svc-fake"}}
}
func (liveQueue) Validate(c json.RawMessage) (json.RawMessage, error) { return c, nil }
func (liveQueue) Fetch(ctx context.Context, req *widgets.FetchRequest) (any, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, req.Integration.BaseURL+"/queue?apikey="+req.Integration.Creds.APIKey, nil)
	if err != nil {
		return nil, err
	}
	resp, err := req.Integration.Client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: status %d", httpReq.URL, resp.StatusCode)
	}
	return map[string]int{"queued": 3}, nil
}

// statefulKind counts its fetches in the integration's session state
type statefulKind struct{ fakeKind }

func (statefulKind) Kind() string { return "svc-stateful" }
func (statefulKind) Fetch(_ context.Context, conn *integrations.Conn) (*integrations.Payload, error) {
	n := 0
	if v, ok := conn.State.Get("fetches"); ok {
		n, _ = strconv.Atoi(v)
	}
	n++
	conn.State.Set("fetches", strconv.Itoa(n))
	return &integrations.Payload{KPIs: map[string]any{"fetches": n}}, nil
}

func init() {
	widgets.Register(liveCounter{})
	widgets.Register(liveQueue{})
	integrations.Register(statefulKind{})
}

// fakeWidgetRepo only serves the poller's list; widget CRUD is tested elsewhere
type fakeWidgetRepo struct {
	widgetRepoStub
	enabled []models.Widget
}

func (r *fakeWidgetRepo) ListEnabled(context.Context) ([]models.Widget, error) {
	return r.enabled, nil
}

// widgetRepoStub satisfies the rest of WidgetRepositoryInterface
type widgetRepoStub struct{}

func (widgetRepoStub) Create(context.Context, *models.Widget, int) error { return nil }
func (widgetRepoStub) GetByID(context.Context, string, string) (*models.Widget, error) {
	return nil, errors.New("not implemented")
}
func (widgetRepoStub) ListByUserID(context.Context, string) ([]models.Widget, error) {
	return nil, nil
}
func (widgetRepoStub) Update(context.Context, *models.Widget) error { return nil }
func (widgetRepoStub) Delete(context.Context, string, string) error { return nil }
func (widgetRepoStub) ReorderTiles(context.Context, string, []models.TilePosition) error {
	return nil
}

type fakeSnapshotRepo struct {
	saved map[string]models.Snapshot
	fail  bool
}

func (r *fakeSnapshotRepo) Upsert(_ context.Context, snap *models.Snapshot) error {
	if r.fail {
		return errors.New("db down")
	}
	r.saved[snap.Key()] = *snap
	return nil
}
func (r *fakeSnapshotRepo) ListAll(context.Context) ([]models.Snapshot, error) {
	list := []models.Snapshot{}
	for _, s := range r.saved {
		list = append(list, s)
	}
	return list, nil
}
func (r *fakeSnapshotRepo) PruneOrphans(context.Context) (int64, error) { return 0, nil }

type liveFixture struct {
	live         *LiveDataService
	widgets      *fakeWidgetRepo
	integrations *fakeIntegrationRepo
	snapshots    *fakeSnapshotRepo
	hub          *SSEHub
	service      *IntegrationService
}

func newLiveFixture(t *testing.T) *liveFixture {
	t.Helper()
	service, integrationRepo := newTestIntegrationService(t)
	f := &liveFixture{
		widgets:      &fakeWidgetRepo{},
		integrations: integrationRepo,
		snapshots:    &fakeSnapshotRepo{saved: map[string]models.Snapshot{}},
		hub:          NewSSEHub(),
		service:      service,
	}
	f.live = NewLiveDataService(f.widgets, integrationRepo, service, f.snapshots, f.hub)
	f.live.now = func() time.Time { return time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC) }
	return f
}

func sourceByKey(t *testing.T, sources []LiveSource, key string) LiveSource {
	t.Helper()
	for _, src := range sources {
		if src.Key() == key {
			return src
		}
	}
	t.Fatalf("no source %s", key)
	return LiveSource{}
}

func TestLiveData_SourcesSkipStaticAndUnknownTypes(t *testing.T) {
	f := newLiveFixture(t)
	f.widgets.enabled = []models.Widget{
		{ID: "w-clock", UserID: "u1", Type: "clock", RefreshSeconds: 300},
		{ID: "w-gone", UserID: "u1", Type: "removed-type", RefreshSeconds: 300},
		{ID: "w-count", UserID: "u1", Type: "svc-counter", Config: json.RawMessage(`{"a":1}`), RefreshSeconds: 30},
	}

	sources, err := f.live.Sources(context.Background())
	require.NoError(t, err)
	require.Len(t, sources, 1)
	src := sources[0]
	assert.Equal(t, "widget:w-count", src.Key())
	assert.Equal(t, "u1", src.UserID)
	assert.Equal(t, 30*time.Second, src.Interval)

	payload, err := src.Fetch(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 42, payload.(map[string]any)["count"])
}

func TestLiveData_IntegrationWidgetFetchRedactsSecrets(t *testing.T) {
	f := newLiveFixture(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(server.Close)

	integration, err := f.service.Create(context.Background(), "u1", false, apiKeyRequest(server.URL))
	require.NoError(t, err)
	f.widgets.enabled = []models.Widget{
		{ID: "w-queue", UserID: "u1", Type: "svc-queue", IntegrationID: &integration.ID, RefreshSeconds: 60},
		{ID: "w-orphan", UserID: "u1", Type: "svc-queue", RefreshSeconds: 60},
	}

	sources, err := f.live.Sources(context.Background())
	require.NoError(t, err)
	assert.Len(t, sources, 3, "the integration itself is polled too")
	sourceByKey(t, sources, "integration:"+integration.ID)

	_, err = sourceByKey(t, sources, "widget:w-queue").Fetch(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 401")
	assert.NotContains(t, err.Error(), testSecret)

	_, err = sourceByKey(t, sources, "widget:w-orphan").Fetch(context.Background())
	assert.EqualError(t, err, "this widget has no integration")
}

func TestLiveData_RecordKeepsLastGoodPayload(t *testing.T) {
	f := newLiveFixture(t)
	sub, err := f.hub.Subscribe("u1")
	require.NoError(t, err)
	src := LiveSource{Kind: models.SnapshotSourceWidget, ID: "w1", UserID: "u1"}

	f.live.Record(src, map[string]int{"n": 1}, nil)
	good := f.live.Snapshots("u1")
	require.Len(t, good, 1)
	assert.JSONEq(t, `{"n":1}`, string(good[0].Payload))
	assert.False(t, good[0].Stale)
	assert.Equal(t, EventSnapshot, (<-sub.Events()).Name)

	later := f.live.now().Add(time.Minute)
	f.live.now = func() time.Time { return later }
	f.live.Record(src, nil, errors.New("connection refused"))

	snap := f.live.Snapshots("u1")[0]
	assert.JSONEq(t, `{"n":1}`, string(snap.Payload), "last good payload is kept")
	assert.Equal(t, good[0].FetchedAt, snap.FetchedAt, "and so is its time")
	assert.Equal(t, "connection refused", snap.Error)
	assert.True(t, snap.Stale)
	assert.Equal(t, snap, f.snapshots.saved["widget:w1"], "the database gets the same snapshot")

	event := <-sub.Events()
	assert.Equal(t, snap, event.Data)
	assert.Empty(t, f.live.Snapshots("u2"), "other users see nothing")
}

func TestLiveData_RecordWithoutPreviousPayload(t *testing.T) {
	f := newLiveFixture(t)
	f.snapshots.fail = true // a database error doesn't stop the push
	f.live.Record(LiveSource{Kind: models.SnapshotSourceIntegration, ID: "i1", UserID: "u1"}, nil, errors.New("timeout"))

	snap := f.live.Snapshots("u1")[0]
	assert.Nil(t, snap.Payload)
	assert.Equal(t, "timeout", snap.Error)
	assert.Equal(t, f.live.now(), snap.FetchedAt)
}

func TestLiveData_WarmAndRetain(t *testing.T) {
	f := newLiveFixture(t)
	f.snapshots.saved["widget:w1"] = models.Snapshot{SourceKind: "widget", SourceID: "w1", UserID: "u1", Payload: json.RawMessage(`{}`)}
	f.snapshots.saved["widget:w2"] = models.Snapshot{SourceKind: "widget", SourceID: "w2", UserID: "u1", Payload: json.RawMessage(`{}`)}

	require.NoError(t, f.live.Warm(context.Background()))
	snaps := f.live.Snapshots("u1")
	require.Len(t, snaps, 2)
	assert.True(t, snaps[0].Stale, "warmed snapshots are stale until fetched again")

	f.live.Retain(map[string]bool{"widget:w2": true})
	snaps = f.live.Snapshots("u1")
	require.Len(t, snaps, 1)
	assert.Equal(t, "w2", snaps[0].SourceID)
}

func TestLiveData_PublishServiceStatus(t *testing.T) {
	f := newLiveFixture(t)
	sub, err := f.hub.Subscribe("u1")
	require.NoError(t, err)
	ms := 12

	f.live.PublishServiceStatus("u1", "s1", models.StatusOnline, &ms)
	event := <-sub.Events()
	assert.Equal(t, EventServiceStatus, event.Name)
	assert.Equal(t, ServiceStatusEvent{ID: "s1", Status: "online", ResponseTime: &ms}, event.Data)
}

func TestIntegrationService_FetchKeepsSessionStateUntilReconnect(t *testing.T) {
	service, _ := newTestIntegrationService(t)
	ctx := context.Background()
	integration, err := service.Create(ctx, "u1", false, &models.IntegrationRequest{Kind: "svc-stateful", Name: "S", BaseURL: "http://app.lan", AuthType: "none"})
	require.NoError(t, err)

	fetchCount := func() any {
		payload, err := service.Fetch(ctx, integration)
		require.NoError(t, err)
		return payload.KPIs["fetches"]
	}
	assert.Equal(t, 1, fetchCount())
	assert.Equal(t, 2, fetchCount(), "state survives between fetches")

	newURL := "http://other.lan"
	integration, err = service.Update(ctx, integration.ID, "u1", false, &models.IntegrationRequest{BaseURL: newURL})
	require.NoError(t, err)
	assert.Equal(t, 1, fetchCount(), "a new connection starts a new session")

	_, err = service.Fetch(ctx, &models.Integration{ID: "x", UserID: "u1", Kind: "gone"})
	assert.EqualError(t, err, `integration kind "gone" is not available`)
}

func TestIntegrationService_FetchRecoversFromPanics(t *testing.T) {
	service, _ := newTestIntegrationService(t)
	integration, err := service.Create(context.Background(), "u1", false, &models.IntegrationRequest{Kind: "svc-panic", Name: "P", BaseURL: "http://app.lan", AuthType: "none"})
	require.NoError(t, err)

	_, err = service.Fetch(context.Background(), integration)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "crashed")
}

func TestLiveData_VersionIgnoresLayoutChanges(t *testing.T) {
	f := newLiveFixture(t)
	widget := models.Widget{ID: "w", UserID: "u1", Type: "svc-counter", Config: json.RawMessage(`{"a":1}`), RefreshSeconds: 60}
	versionOf := func(w models.Widget) string {
		f.widgets.enabled = []models.Widget{w}
		sources, err := f.live.Sources(context.Background())
		require.NoError(t, err)
		return sources[0].Version
	}
	base := versionOf(widget)

	moved := widget
	moved.Position, moved.Title, moved.CardSize = 9, "Renamed", "2x2"
	moved.UpdatedAt = time.Now()
	assert.Equal(t, base, versionOf(moved), "reorder, rename and resize don't refetch")

	edited := widget
	edited.Config = json.RawMessage(`{"a":2}`)
	assert.NotEqual(t, base, versionOf(edited))
	slower := widget
	slower.RefreshSeconds = 120
	assert.NotEqual(t, base, versionOf(slower))
}

// multiWidget lists its integrations in the config, like the calendar
type multiWidget struct{}

func (multiWidget) Type() string { return "svc-multi" }
func (multiWidget) Meta() widgets.Meta {
	return widgets.Meta{Name: "Multi", Category: widgets.CategoryInfo, DefaultSize: "2x2", AllowedSizes: []string{"2x2"}, ConfigIntegrationKinds: []string{"svc-fake"}}
}
func (multiWidget) Validate(c json.RawMessage) (json.RawMessage, error) { return c, nil }
func (multiWidget) IntegrationIDs(c json.RawMessage) []string {
	var cfg struct{ IDs []string }
	_ = json.Unmarshal(c, &cfg)
	return cfg.IDs
}
func (multiWidget) SetIntegrationIDs(c json.RawMessage, ids []string) (json.RawMessage, error) {
	return json.Marshal(map[string]any{"IDs": ids})
}
func (multiWidget) Fetch(_ context.Context, req *widgets.FetchRequest) (any, error) {
	names := []string{}
	for _, l := range req.Linked {
		names = append(names, l.Name+"@"+l.Conn.BaseURL)
	}
	// A careless widget quoting a key in a message
	leak := ""
	if len(req.Linked) > 0 {
		leak = "failed with key " + req.Linked[0].Conn.Creds.APIKey
	}
	return map[string]any{"linked": names, "unlinked": req.Unlinked, "leak": leak}, nil
}

func init() { widgets.Register(multiWidget{}) }

func TestLiveData_ConfigIntegrationsAreConnected(t *testing.T) {
	f := newLiveFixture(t)
	ctx := context.Background()
	mine, err := f.service.Create(ctx, "u1", false, apiKeyRequest("http://nas.lan"))
	require.NoError(t, err)
	theirs, err := f.service.Create(ctx, "u2", false, apiKeyRequest("http://other.lan"))
	require.NoError(t, err)
	f.integrations.unused = map[string]bool{mine.ID: true, theirs.ID: true}
	f.widgets.enabled = []models.Widget{{ID: "w-multi", UserID: "u1", Type: "svc-multi",
		Config: json.RawMessage(`{"IDs":["` + mine.ID + `","` + theirs.ID + `"]}`), RefreshSeconds: 300}}

	sources, err := f.live.Sources(ctx)
	require.NoError(t, err)
	payload, err := sourceByKey(t, sources, "widget:w-multi").Fetch(ctx)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"linked":   []any{mine.Name + "@http://nas.lan"},
		"unlinked": []any{"an integration that was removed"},
		"leak":     "failed with key [redacted]",
	}, payload, "only the owner's own integration is connected, and keys are redacted")

	// Editing a listed integration changes the widget's version
	before := sourceByKey(t, sources, "widget:w-multi").Version
	edited := f.integrations.items[mine.ID]
	edited.UpdatedAt = edited.UpdatedAt.Add(time.Minute) // as the repository does on update
	f.integrations.items[mine.ID] = edited
	sources, err = f.live.Sources(ctx)
	require.NoError(t, err)
	assert.NotEqual(t, before, sourceByKey(t, sources, "widget:w-multi").Version)
}
