package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/nimbus/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const uptimeKumaMetrics = `# HELP monitor_status Monitor Status (1 = UP, 0= DOWN, 2= PENDING, 3= MAINTENANCE)
# TYPE monitor_status gauge
monitor_status{monitor_name="Plex",monitor_type="http",monitor_url="https://plex.lan",monitor_hostname="null",monitor_port="null"} 1
monitor_status{monitor_name="NAS",monitor_type="ping",monitor_url="https://",monitor_hostname="nas.lan",monitor_port="null"} 1
monitor_status{monitor_name="Printer 1 0",monitor_type="ping",monitor_url="https://",monitor_hostname="printer",monitor_port="null"} 0
monitor_status{monitor_name="Backup",monitor_type="push",monitor_url="https://",monitor_hostname="null",monitor_port="null"} 2
monitor_status{monitor_name="Old",monitor_type="http",monitor_url="https://old.lan",monitor_hostname="null",monitor_port="null"} 3
# HELP monitor_response_time Monitor Response Time (ms)
monitor_response_time{monitor_name="Plex",monitor_type="http",monitor_url="https://plex.lan",monitor_hostname="null",monitor_port="null"} 1
`

func TestUptimeKuma(t *testing.T) {
	impl := registered(t, "uptime_kuma")
	server := fakeApp(t, map[string]http.HandlerFunc{
		"GET /metrics": func(w http.ResponseWriter, r *http.Request) {
			if user, pass, ok := r.BasicAuth(); !ok || user != "" || pass != "uk-key" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			fmt.Fprint(w, uptimeKumaMetrics)
		},
	})
	ctx := context.Background()
	conn := newTestConn(server.URL, models.IntegrationCredentials{APIKey: "uk-key"})

	require.NoError(t, impl.Test(ctx, conn))
	payload, err := impl.Fetch(ctx, conn)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"up": 2, "down": 1, "up_percent": 67}, payload.KPIs)

	assert.Nil(t, payload.KPIDefs, "the kind's own KPIs apply")

	assert.ErrorIs(t, impl.Test(ctx, newTestConn(server.URL, models.IntegrationCredentials{APIKey: "x"})), errRejected)
	assertUnreachable(t, impl, "/metrics")
}

func TestUptimeKumaStatusPage(t *testing.T) {
	impl := registered(t, "uptime_kuma")
	public := func(body string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			assert.Empty(t, r.Header.Get("Authorization"), "the API key is not for public endpoints")
			fmt.Fprint(w, body)
		}
	}
	// Shapes of Uptime Kuma 1.23 and 2.x: heartbeats oldest first, uptime as a ratio
	server := fakeApp(t, map[string]http.HandlerFunc{
		"GET /api/status-page/homelab": public(`{"config":{"slug":"homelab","title":"Homelab"},"incident":null,
			"publicGroupList":[{"id":1,"name":"Services","monitorList":[{"id":1,"name":"Plex"},{"id":2,"name":"NAS"}]}],
			"maintenanceList":[]}`),
		"GET /api/status-page/heartbeat/homelab": public(`{
			"heartbeatList":{
				"1":[{"status":0,"time":"2026-11-18 10:00:00","msg":"","ping":null},{"status":1,"time":"2026-11-18 10:01:00","msg":"","ping":12}],
				"2":[{"status":1,"time":"2026-11-18 10:00:00","msg":"","ping":3},{"status":0,"time":"2026-11-18 10:01:00","msg":"","ping":null}],
				"3":[{"status":1,"time":"2026-11-18 10:01:00","msg":"","ping":5}],
				"4":[{"status":3,"time":"2026-11-18 10:01:00","msg":"","ping":null}],
				"5":[]
			},
			"uptimeList":{"1_24":0.9987,"2_24":0.5,"3_24":1,"4_24":1,"5_24":0}
		}`),
		"GET /api/status-page/gone": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"status":"fail","msg":"Status Page Not Found"}`)
		},
		"GET /api/status-page/other-app": public(`{"ok":true}`),
		// What the server sees when the slug is escaped: one odd segment
		"GET /api/status-page/../../metrics?x": func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/status-page/..%2F..%2Fmetrics%3Fx", r.RequestURI)
			w.WriteHeader(http.StatusNotFound)
		},
	})
	ctx := context.Background()
	conn := newTestConn(server.URL, models.IntegrationCredentials{APIKey: "unused"})
	conn.Options = json.RawMessage(`{"status_page_slug":"Homelab"}`)

	require.NoError(t, impl.Test(ctx, conn))
	payload, err := impl.Fetch(ctx, conn)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"up": 2, "down": 1, "up_percent": 67, "uptime": 69.97}, payload.KPIs)
	keys := []string{}
	for _, kpi := range payload.KPIDefs {
		keys = append(keys, kpi.Key)
	}
	assert.Equal(t, []string{"up", "down", "uptime"}, keys, "the tile shows real uptime, not the share")

	conn.Options = json.RawMessage(`{"status_page_slug":"gone"}`)
	assert.EqualError(t, impl.Test(ctx, conn), "/api/status-page/gone returned status 404")
	conn.Options = json.RawMessage(`{"status_page_slug":"other-app"}`)
	assert.EqualError(t, impl.Test(ctx, conn), "/api/status-page/other-app returned an unexpected response")

	// The slug can't leave its path segment
	conn.Options = json.RawMessage(`{"status_page_slug":"../../metrics?x"}`)
	assert.Error(t, impl.Test(ctx, conn))

	closed := newTestConn("http://127.0.0.1:1", models.IntegrationCredentials{})
	closed.Options = json.RawMessage(`{"status_page_slug":"homelab"}`)
	err = impl.Test(ctx, closed)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "/api/status-page/homelab could not be reached: ")
}

func TestUptimeKumaNeedsItsMetrics(t *testing.T) {
	impl := registered(t, "uptime_kuma")
	body := ""
	server := fakeApp(t, map[string]http.HandlerFunc{
		"GET /metrics": func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) },
	})
	ctx := context.Background()
	conn := newTestConn(server.URL, models.IntegrationCredentials{})

	// A login page or another app answering 200 is not Uptime Kuma
	for _, wrong := range []string{"", "<html>login</html>", "process_cpu_seconds_total 1\n"} {
		body = wrong
		assert.EqualError(t, impl.Test(ctx, conn), "/metrics returned an unexpected response", wrong)
	}

	// No monitors yet is fine
	body = "# HELP monitor_status Monitor Status\n# TYPE monitor_status gauge\n"
	payload, err := impl.Fetch(ctx, conn)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"up": 0, "down": 0, "up_percent": 0}, payload.KPIs)
}
