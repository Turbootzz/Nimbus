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

func TestPihole(t *testing.T) {
	impl := registered(t, "pihole")
	logins, logouts := 0, 0
	validSID := "sid-1"
	server := fakeApp(t, map[string]http.HandlerFunc{
		"POST /api/auth": func(w http.ResponseWriter, r *http.Request) {
			var body struct{ Password string }
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.Password != "app-pass" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			logins++
			fmt.Fprintf(w, `{"session":{"valid":true,"sid":%q,"validity":1800}}`, validSID)
		},
		"DELETE /api/auth": func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-FTL-SID") != validSID {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			logouts++
			w.WriteHeader(http.StatusNoContent)
		},
		"GET /api/info/version": func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-FTL-SID") != validSID {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			fmt.Fprint(w, `{"version":{"core":{"local":{"version":"v6.0"}}}}`)
		},
		"GET /api/stats/summary": func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-FTL-SID") != validSID {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			fmt.Fprint(w, `{"queries":{"total":1000,"blocked":123,"percent_blocked":12.3456},"gravity":{"domains_being_blocked":150000}}`)
		},
	})
	ctx := context.Background()
	conn := newTestConn(server.URL, models.IntegrationCredentials{Token: "app-pass"})

	require.NoError(t, impl.Test(ctx, conn))
	assert.Equal(t, 1, logouts, "Test gives its session seat back")

	payload, err := impl.Fetch(ctx, conn)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"queries": int64(1000), "blocked": int64(123), "blocked_percent": 12.3, "gravity": int64(150000)}, payload.KPIs)
	_, err = impl.Fetch(ctx, conn)
	require.NoError(t, err)
	assert.Equal(t, 2, logins, "one login for Test, one reused for both fetches")

	validSID = "sid-2" // the stored session expired
	_, err = impl.Fetch(ctx, conn)
	require.NoError(t, err)
	assert.Equal(t, 3, logins)

	wrong := newTestConn(server.URL, models.IntegrationCredentials{Token: "nope"})
	assert.ErrorIs(t, impl.Test(ctx, wrong), errRejected)

	// Deleting the integration logs the kept session out, once
	closer := impl.(Closer)
	require.NoError(t, closer.Close(ctx, conn))
	assert.Equal(t, 2, logouts)
	require.NoError(t, closer.Close(ctx, conn))
	assert.Equal(t, 2, logouts, "no session left to close")

	// A session that had expired already is fine
	_, err = impl.Fetch(ctx, conn)
	require.NoError(t, err)
	validSID = "sid-3"
	assert.NoError(t, closer.Close(ctx, conn))
}

func TestPiholeWithoutPassword(t *testing.T) {
	impl := registered(t, "pihole")
	server := fakeApp(t, map[string]http.HandlerFunc{
		"GET /api/stats/summary": func(w http.ResponseWriter, r *http.Request) {
			assert.Empty(t, r.Header.Get("X-FTL-SID"))
			fmt.Fprint(w, `{"queries":{"total":5}}`)
		},
	})
	payload, err := impl.Fetch(context.Background(), newTestConn(server.URL, models.IntegrationCredentials{}))
	require.NoError(t, err)
	assert.Equal(t, int64(5), payload.KPIs["queries"])
}
