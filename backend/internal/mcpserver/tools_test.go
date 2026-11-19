package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/widgets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	alice = "user-alice"
	bob   = "user-bob"
)

// fakeData stands in for every repository and service the tools read
type fakeData struct {
	services     map[string][]*models.Service
	groups       map[string][]*models.Group
	logs         map[string][]*models.StatusLog
	stats        map[string]map[string]interface{}
	widgets      map[string][]models.Widget
	snapshots    map[string][]models.Snapshot
	integrations map[string][]models.Integration

	statsFrom, statsTo time.Time
	checked            []string
	failServices       bool
}

func (f *fakeData) deps() Deps {
	return Deps{
		Services:     fakeServices{f},
		Groups:       fakeGroups{f},
		StatusLogs:   f,
		Widgets:      fakeWidgets{f},
		Snapshots:    f,
		Integrations: fakeIntegrations{f},
		Health:       f,
	}
}

type fakeServices struct{ *fakeData }

func (f fakeServices) GetAllByUserID(_ context.Context, userID string) ([]*models.Service, error) {
	if f.failServices {
		return nil, errors.New("pq: connection refused to db.internal:5432")
	}
	// A copy, like a repository returning fresh rows
	return append([]*models.Service(nil), f.services[userID]...), nil
}

type fakeGroups struct{ *fakeData }

func (f fakeGroups) GetAllByUserID(_ context.Context, userID string) ([]*models.Group, error) {
	return f.groups[userID], nil
}

type fakeWidgets struct{ *fakeData }

func (f fakeWidgets) List(_ context.Context, userID string) ([]models.Widget, error) {
	return f.widgets[userID], nil
}

type fakeIntegrations struct{ *fakeData }

func (f fakeIntegrations) List(_ context.Context, userID string) ([]models.Integration, error) {
	return f.integrations[userID], nil
}

func (f *fakeData) GetUptimeStats(_ context.Context, serviceID string, start, end time.Time) (map[string]interface{}, error) {
	f.statsFrom, f.statsTo = start, end
	if stats, ok := f.stats[serviceID]; ok {
		return stats, nil
	}
	return map[string]interface{}{
		"total_checks": 0, "online_count": 0, "offline_count": 0, "uptime_percentage": 0.0,
		"avg_response_time": 0.0, "min_response_time": 0.0, "max_response_time": 0.0,
	}, nil
}

func (f *fakeData) GetLatestByServiceID(_ context.Context, serviceID string, limit int) ([]*models.StatusLog, error) {
	logs := f.logs[serviceID]
	if len(logs) > limit {
		logs = logs[:limit]
	}
	return logs, nil
}

func (f *fakeData) Snapshots(userID string) []models.Snapshot {
	return f.snapshots[userID]
}

// CheckService records the check and brings the service online
func (f *fakeData) CheckService(_ context.Context, service *models.Service) error {
	f.checked = append(f.checked, service.ID)
	service.Status = models.StatusOnline
	return nil
}

func ptr[T any](v T) *T { return &v }

func newFakeData() *fakeData {
	now := time.Now()
	return &fakeData{
		services: map[string][]*models.Service{
			alice: {
				{ID: "svc-jellyfin", UserID: alice, Name: "Jellyfin", URL: "http://jellyfin.lan:8096", Description: "Movies and shows", Status: models.StatusOnline, ResponseTime: ptr(42), GroupID: ptr("grp-media"), MonitoringEnabled: true},
				{ID: "svc-sonarr", UserID: alice, Name: "Sonarr", URL: "http://sonarr.lan", Status: models.StatusOffline, GroupID: ptr("grp-media"), MonitoringEnabled: true},
				// Monitored itself, but its group is not: the status is stale
				{ID: "svc-router", UserID: alice, Name: "Router", URL: "http://192.168.1.1", Status: models.StatusOnline, GroupID: ptr("grp-infra"), MonitoringEnabled: true},
				{ID: "svc-dup-1", UserID: alice, Name: "Pi-hole", URL: "http://pihole-1.lan", Status: models.StatusOnline, MonitoringEnabled: true},
				{ID: "svc-dup-2", UserID: alice, Name: "pi-hole", URL: "http://pihole-2.lan", Status: models.StatusOnline, MonitoringEnabled: true},
				{ID: "svc-printer", UserID: alice, Name: "Printer", URL: "http://printer.lan", Status: models.StatusOffline, MonitoringEnabled: false},
			},
			bob: {
				{ID: "svc-bob", UserID: bob, Name: "Vaultwarden", URL: "http://vault.bob.lan", Status: models.StatusOffline, MonitoringEnabled: true},
			},
		},
		groups: map[string][]*models.Group{
			alice: {
				{ID: "grp-media", Name: "Media", MonitoringEnabled: true},
				{ID: "grp-infra", Name: "Infrastructure", MonitoringEnabled: false},
				{ID: "grp-home", Name: "Home", MonitoringEnabled: true, IsDefault: true},
			},
		},
		logs: map[string][]*models.StatusLog{
			"svc-sonarr": {
				{Status: models.StatusOffline, ErrorMessage: ptr("connection refused"), CheckedAt: now},
				{Status: models.StatusOnline, ResponseTime: ptr(30), CheckedAt: now.Add(-time.Minute)},
			},
		},
		stats: map[string]map[string]interface{}{
			"svc-jellyfin": {
				"total_checks": 200, "online_count": 199, "offline_count": 1, "uptime_percentage": 99.5,
				"avg_response_time": 40.5, "min_response_time": 12.0, "max_response_time": 300.0,
			},
		},
		widgets: map[string][]models.Widget{
			alice: {
				{ID: "wid-weather", UserID: alice, Type: "weather", Title: "Home", Enabled: true, Config: json.RawMessage(`{"city":"Utrecht"}`)},
				{ID: "wid-clock", UserID: alice, Type: "clock", Enabled: true},
			},
			bob: {{ID: "wid-bob", UserID: bob, Type: "note", Title: "Private", Enabled: true, Config: json.RawMessage(`{"text":"bob's note"}`)}},
		},
		snapshots: map[string][]models.Snapshot{
			alice: {
				{SourceKind: models.SnapshotSourceWidget, SourceID: "wid-weather", UserID: alice, Payload: json.RawMessage(`{"temperature":11}`), FetchedAt: now, Stale: true},
				{SourceKind: models.SnapshotSourceIntegration, SourceID: "wid-clock", UserID: alice, Payload: json.RawMessage(`{"wrong":"kind"}`), FetchedAt: now},
			},
		},
		integrations: map[string][]models.Integration{
			alice: {
				{ID: "int-sonarr", UserID: alice, Kind: "sonarr", Name: "Sonarr", BaseURL: "http://sonarr.internal:8989", AuthType: models.IntegrationAuthAPIKey, HasCredentials: true, Options: json.RawMessage(`{"secret_option":"hunter2"}`), LastTestOK: ptr(true), LastTestAt: &now},
				{ID: "int-pve", UserID: alice, Kind: "proxmox", Name: "Proxmox", BaseURL: "https://pve.internal:8006", LastTestOK: ptr(false), LastError: ptr("dial tcp 10.0.0.9:8006: refused")},
				{ID: "int-new", UserID: alice, Kind: "pihole", Name: "Pi-hole", BaseURL: "http://pihole.internal"},
			},
		},
	}
}

// requestAs builds a tool request as the transport does after verifying a token
func requestAs(userID string, scopes ...string) *mcp.CallToolRequest {
	return &mcp.CallToolRequest{Extra: &mcp.RequestExtra{TokenInfo: &auth.TokenInfo{UserID: userID, Scopes: scopes}}}
}

func names(list ServiceList) []string {
	out := make([]string, len(list.Services))
	for i, service := range list.Services {
		out[i] = service.Name
	}
	return out
}

func TestListServices(t *testing.T) {
	tools := &tools{newFakeData().deps()}
	ctx := context.Background()

	_, list, err := tools.listServices(ctx, requestAs(alice, models.ScopeMCP), listServicesInput{})
	require.NoError(t, err)
	assert.Equal(t, []string{"Jellyfin", "Sonarr", "Router", "Pi-hole", "pi-hole", "Printer"}, names(list))
	// Counted as the dashboard does: unmonitored services are neither online nor offline
	assert.Equal(t, ServiceList{Services: list.Services, Total: 6, Online: 3, Offline: 1, NotMonitored: 2}, list)
	assert.Equal(t, ServiceInfo{
		ID: "svc-jellyfin", Name: "Jellyfin", URL: "http://jellyfin.lan:8096", Description: "Movies and shows",
		Group: "Media", Status: "online", ResponseTimeMs: ptr(42), MonitoringEnabled: true,
	}, list.Services[0])
	// Not monitored by its group's setting or its own; ungrouped is in the default group
	assert.Equal(t, ServiceInfo{
		ID: "svc-router", Name: "Router", URL: "http://192.168.1.1", Group: "Infrastructure", Status: "not_monitored",
	}, list.Services[2])
	assert.Equal(t, ServiceInfo{
		ID: "svc-printer", Name: "Printer", URL: "http://printer.lan", Group: "Home", Status: "not_monitored",
	}, list.Services[5])

	// Another user only ever sees their own services
	_, list, err = tools.listServices(ctx, requestAs(bob, models.ScopeMCP), listServicesInput{})
	require.NoError(t, err)
	assert.Equal(t, []string{"Vaultwarden"}, names(list))

	// A user without services gets an empty list, not null
	_, list, err = tools.listServices(ctx, requestAs("nobody", models.ScopeMCP), listServicesInput{})
	require.NoError(t, err)
	assert.NotNil(t, list.Services)
	assert.Empty(t, list.Services)
}

func TestListServices_Group(t *testing.T) {
	tools := &tools{newFakeData().deps()}
	ctx := context.Background()

	for _, group := range []string{"Media", "media", "grp-media", " Media "} {
		_, list, err := tools.listServices(ctx, requestAs(alice), listServicesInput{Group: group})
		require.NoError(t, err, group)
		assert.Equal(t, []string{"Jellyfin", "Sonarr"}, names(list), group)
		assert.Equal(t, 2, list.Total)
	}

	// The default group holds the ungrouped services, as on the dashboard
	_, list, err := tools.listServices(ctx, requestAs(alice), listServicesInput{Group: "Home"})
	require.NoError(t, err)
	assert.Equal(t, []string{"Pi-hole", "pi-hole", "Printer"}, names(list))

	_, _, err = tools.listServices(ctx, requestAs(alice), listServicesInput{Group: "Games"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Home, Infrastructure, Media", "the error names the groups that exist")
}

func TestSearchServices(t *testing.T) {
	tools := &tools{newFakeData().deps()}
	ctx := context.Background()

	tests := []struct {
		q    string
		want []string
	}{
		{"jelly", []string{"Jellyfin"}},
		{"SHOWS", []string{"Jellyfin"}},             // description
		{"192.168", []string{"Router"}},             // URL
		{"media", []string{"Jellyfin", "Sonarr"}},   // group
		{"pi-hole", []string{"Pi-hole", "pi-hole"}}, // several
		{"vaultwarden", []string{}},                 // another user's service
		{"  sonarr  ", []string{"Sonarr"}},          // trimmed
		{"nothing-matches-this", []string{}},
	}
	for _, tt := range tests {
		_, list, err := tools.searchServices(ctx, requestAs(alice), searchInput{Q: tt.q})
		require.NoError(t, err, tt.q)
		assert.Equal(t, tt.want, names(list), tt.q)
	}

	_, _, err := tools.searchServices(ctx, requestAs(alice), searchInput{Q: "  "})
	assert.Error(t, err)
}

func TestGetServiceStatus(t *testing.T) {
	tools := &tools{newFakeData().deps()}
	ctx := context.Background()

	for _, ref := range []string{"svc-sonarr", "Sonarr", "sonarr", " SONARR "} {
		_, status, err := tools.getServiceStatus(ctx, requestAs(alice), serviceInput{Service: ref})
		require.NoError(t, err, ref)
		assert.Equal(t, "svc-sonarr", status.Service.ID, ref)
		assert.Equal(t, "offline", status.Service.Status)
		require.Len(t, status.RecentChecks, 2)
		assert.Equal(t, "connection refused", status.RecentChecks[0].Error)
		assert.Equal(t, ptr(30), status.RecentChecks[1].ResponseTimeMs)
	}

	// No history yet: an empty list, not null
	_, status, err := tools.getServiceStatus(ctx, requestAs(alice), serviceInput{Service: "Jellyfin"})
	require.NoError(t, err)
	assert.NotNil(t, status.RecentChecks)
	assert.Empty(t, status.RecentChecks)

	// A service in a group with monitoring off has no current status
	_, status, err = tools.getServiceStatus(ctx, requestAs(alice), serviceInput{Service: "Router"})
	require.NoError(t, err)
	assert.Equal(t, "not_monitored", status.Service.Status)
	assert.False(t, status.Service.MonitoringEnabled)
	assert.Nil(t, status.Service.ResponseTimeMs)

	errorCases := map[string]string{
		"":            "required",
		"Plex":        "search_services",
		"pi-hole":     "svc-dup-1, svc-dup-2", // ambiguous name lists the IDs
		"svc-bob":     "no service",           // another user's ID
		"Vaultwarden": "no service",           // another user's name
	}
	for ref, want := range errorCases {
		_, _, err := tools.getServiceStatus(ctx, requestAs(alice), serviceInput{Service: ref})
		require.Error(t, err, ref)
		assert.Contains(t, err.Error(), want, ref)
	}
}

func TestGetUptime(t *testing.T) {
	data := newFakeData()
	tools := &tools{data.deps()}
	ctx := context.Background()

	_, uptime, err := tools.getUptime(ctx, requestAs(alice), uptimeInput{Service: "jellyfin", Range: "7d"})
	require.NoError(t, err)
	assert.Equal(t, Uptime{
		Service: uptime.Service, Range: "7d", UptimePercentage: 99.5, TotalChecks: 200, OnlineChecks: 199, OfflineChecks: 1,
		AvgResponseTimeMs: 40.5, MinResponseTimeMs: 12, MaxResponseTimeMs: 300,
	}, uptime)
	assert.Equal(t, "svc-jellyfin", uptime.Service.ID)
	assert.Equal(t, 7*24*time.Hour, data.statsTo.Sub(data.statsFrom))
	assert.WithinDuration(t, time.Now(), data.statsTo, time.Minute)

	// Every range, and 24h when none is given
	for input, span := range map[string]time.Duration{"1h": time.Hour, "24h": 24 * time.Hour, "30d": 30 * 24 * time.Hour, "": 24 * time.Hour} {
		_, uptime, err := tools.getUptime(ctx, requestAs(alice), uptimeInput{Service: "svc-jellyfin", Range: input})
		require.NoError(t, err, input)
		assert.Equal(t, span, data.statsTo.Sub(data.statsFrom), input)
		assert.Equal(t, span, uptimeRanges[uptime.Range], input)
	}

	// No data is reported as zero checks
	_, uptime, err = tools.getUptime(ctx, requestAs(alice), uptimeInput{Service: "Sonarr"})
	require.NoError(t, err)
	assert.Zero(t, uptime.TotalChecks)

	_, _, err = tools.getUptime(ctx, requestAs(alice), uptimeInput{Service: "Jellyfin", Range: "1y"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "1h, 24h, 7d or 30d")

	_, _, err = tools.getUptime(ctx, requestAs(alice), uptimeInput{Service: "svc-bob"})
	assert.Error(t, err, "another user's service")
}

func TestGetWidgetData(t *testing.T) {
	tools := &tools{newFakeData().deps()}
	ctx := context.Background()

	_, widget, err := tools.getWidgetData(ctx, requestAs(alice), idInput{ID: "wid-weather"})
	require.NoError(t, err)
	assert.Equal(t, "weather", widget.Type)
	assert.Equal(t, "Home", widget.Title)
	assert.Equal(t, map[string]any{"city": "Utrecht"}, widget.Config)
	assert.Equal(t, map[string]any{"temperature": float64(11)}, widget.Data)
	assert.True(t, widget.Stale)
	assert.NotNil(t, widget.FetchedAt)

	// A widget without data of its own; an integration snapshot with the same ID is not its data
	_, widget, err = tools.getWidgetData(ctx, requestAs(alice), idInput{ID: "wid-clock"})
	require.NoError(t, err)
	assert.Nil(t, widget.Data)
	assert.Nil(t, widget.Config)
	assert.Nil(t, widget.FetchedAt)

	// Unknown and foreign IDs list the caller's own widgets, nothing else
	for _, id := range []string{"", "nope", "wid-bob"} {
		_, _, err = tools.getWidgetData(ctx, requestAs(alice), idInput{ID: id})
		require.Error(t, err, id)
		assert.Contains(t, err.Error(), "wid-weather (weather: Home); wid-clock (clock)", id)
		assert.NotContains(t, err.Error(), "Private", id)
	}

	_, _, err = tools.getWidgetData(ctx, requestAs("nobody"), idInput{ID: "wid-weather"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no widgets")
}

// The config goes through the same masking as the REST API (the real Redact
// of each type) and then loses what URLs carry in their query or user info
func TestGetWidgetData_NoSecrets(t *testing.T) {
	configs := map[string]string{
		"custom_api": `{"url":"https://api.example.com/v1/stats?apikey=QUERYSECRET","headers":[{"name":"X-Api-Key","value":"HEADERSECRET"}],"fields":[{"label":"Users","path":"users"}]}`,
		"calendar":   `{"ical_urls":["https://calendar.example.com/private-ICALSECRET/basic.ics"]}`,
		"rss":        `{"feeds":["https://USERSECRET:PASSSECRET@feeds.example.com/private.xml?token=FEEDSECRET"]}`,
		"iframe":     `{"url":"https://grafana.example.com/d/abc?auth_token=IFRAMESECRET"}`,
	}
	data := newFakeData()
	for widgetType, config := range configs {
		raw := json.RawMessage(config)
		// What WidgetService.List does before a widget leaves the server
		if registered, ok := widgets.Get(widgetType); ok {
			if secret, ok := registered.(widgets.SecretConfig); ok {
				raw = secret.Redact(raw)
			}
		}
		data.widgets[alice] = append(data.widgets[alice], models.Widget{ID: "wid-" + widgetType, UserID: alice, Type: widgetType, Config: raw})
	}
	tools := &tools{data.deps()}

	for widgetType := range configs {
		_, widget, err := tools.getWidgetData(context.Background(), requestAs(alice), idInput{ID: "wid-" + widgetType})
		require.NoError(t, err)
		out, err := json.Marshal(widget)
		require.NoError(t, err)
		assert.NotContains(t, string(out), "SECRET", widgetType)
		assert.Contains(t, string(out), "example.com", "%s keeps the host", widgetType)
	}
}

func TestListIntegrations_NoSecretsOrURLs(t *testing.T) {
	tools := &tools{newFakeData().deps()}

	_, list, err := tools.listIntegrations(context.Background(), requestAs(alice), noInput{})
	require.NoError(t, err)
	require.Len(t, list.Integrations, 3)
	assert.Equal(t, "ok", list.Integrations[0].Status)
	assert.Equal(t, "failing", list.Integrations[1].Status)
	assert.Equal(t, "untested", list.Integrations[2].Status)
	assert.NotNil(t, list.Integrations[0].LastTestAt)

	raw, err := json.Marshal(list)
	require.NoError(t, err)
	for _, leak := range []string{"internal", "http", "hunter2", "secret_option", "10.0.0.9", "credentials", "api_key"} {
		assert.NotContains(t, string(raw), leak)
	}

	_, list, err = tools.listIntegrations(context.Background(), requestAs(bob), noInput{})
	require.NoError(t, err)
	assert.NotNil(t, list.Integrations)
	assert.Empty(t, list.Integrations)
}

func TestRecheckService(t *testing.T) {
	data := newFakeData()
	tools := &tools{data.deps()}
	ctx := context.Background()

	// The mcp scope alone is read-only
	_, _, err := tools.recheckService(ctx, requestAs(alice, models.ScopeMCP, models.ScopeReadServices), recheckInput{ID: "svc-sonarr"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write:services")
	assert.Empty(t, data.checked)

	writer := requestAs(alice, models.ScopeMCP, models.ScopeWriteServices)

	_, status, err := tools.recheckService(ctx, writer, recheckInput{ID: "svc-sonarr"})
	require.NoError(t, err)
	assert.Equal(t, []string{"svc-sonarr"}, data.checked)
	assert.Equal(t, "online", status.Service.Status, "returns the status after the check")

	// Not for services Nimbus does not monitor (by their group's setting or
	// their own), and never for another user's
	for _, unmonitored := range []string{"Router", "Printer"} {
		_, _, err = tools.recheckService(ctx, writer, recheckInput{ID: unmonitored})
		require.Error(t, err, unmonitored)
		assert.Contains(t, err.Error(), "monitoring is disabled", unmonitored)
	}
	_, _, err = tools.recheckService(ctx, writer, recheckInput{ID: "svc-bob"})
	require.Error(t, err)
	assert.Equal(t, []string{"svc-sonarr"}, data.checked)
}

func TestTools_RequireVerifiedToken(t *testing.T) {
	tools := &tools{newFakeData().deps()}
	ctx := context.Background()

	for name, req := range map[string]*mcp.CallToolRequest{
		"no extra":      {},
		"no token info": {Extra: &mcp.RequestExtra{}},
		"no user":       {Extra: &mcp.RequestExtra{TokenInfo: &auth.TokenInfo{Scopes: []string{models.ScopeMCP}}}},
	} {
		_, _, err := tools.listServices(ctx, req, listServicesInput{})
		assert.Error(t, err, name)
		_, _, err = tools.listIntegrations(ctx, req, noInput{})
		assert.Error(t, err, name)
		_, _, err = tools.recheckService(ctx, req, recheckInput{ID: "svc-sonarr"})
		assert.Error(t, err, name)
	}
	_, err := tools.readDashboard(ctx, &mcp.ReadResourceRequest{})
	assert.Error(t, err)
}

func TestTools_HideInternalErrors(t *testing.T) {
	data := newFakeData()
	data.failServices = true
	tools := &tools{data.deps()}

	_, _, err := tools.listServices(context.Background(), requestAs(alice), listServicesInput{})
	require.Error(t, err)
	assert.Equal(t, "failed to load services", err.Error())
}

func TestReadDashboard(t *testing.T) {
	tools := &tools{newFakeData().deps()}

	result, err := tools.readDashboard(context.Background(), &mcp.ReadResourceRequest{
		Extra: &mcp.RequestExtra{TokenInfo: &auth.TokenInfo{UserID: alice}},
	})
	require.NoError(t, err)
	require.Len(t, result.Contents, 1)
	assert.Equal(t, DashboardURI, result.Contents[0].URI)

	text := result.Contents[0].Text
	for _, want := range []string{
		"6 services: 3 online, 1 offline, 0 unknown, 2 not monitored.",
		"## Offline\n- Sonarr (http://sonarr.lan)\n\n",
		"- Media: 2 services",
		"- Home: 3 services",
		"- wid-weather (weather: Home)",
		"- Proxmox (proxmox): failing",
	} {
		assert.Contains(t, text, want)
	}
	for _, leak := range []string{"Vaultwarden", "internal", "hunter2", "wid-bob"} {
		assert.NotContains(t, text, leak)
	}
	assert.False(t, strings.Contains(text, "## Offline") && strings.Contains(text, "Jellyfin ("), "online services are not listed as offline")
}
