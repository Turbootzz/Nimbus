package integrations

import (
	"context"
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

	assert.ErrorIs(t, impl.Test(ctx, newTestConn(server.URL, models.IntegrationCredentials{APIKey: "x"})), errRejected)
}
