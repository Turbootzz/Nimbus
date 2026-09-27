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

func TestAdguard(t *testing.T) {
	impl := registered(t, "adguard")
	auth := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if user, pass, ok := r.BasicAuth(); !ok || user != "admin" || pass != "pw" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			next(w, r)
		}
	}
	server := fakeApp(t, map[string]http.HandlerFunc{
		"GET /control/status": auth(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `{"running":true}`) }),
		"GET /control/stats": auth(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"num_dns_queries":2000,"num_blocked_filtering":500,"avg_processing_time":0.01234}`)
		}),
	})
	ctx := context.Background()
	conn := newTestConn(server.URL, models.IntegrationCredentials{Username: "admin", Password: "pw"})

	require.NoError(t, impl.Test(ctx, conn))
	payload, err := impl.Fetch(ctx, conn)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"queries": int64(2000), "blocked": int64(500), "blocked_percent": 25, "latency": 12.3}, payload.KPIs)

	assert.ErrorIs(t, impl.Test(ctx, newTestConn(server.URL, models.IntegrationCredentials{Username: "admin", Password: "x"})), errRejected)
}
