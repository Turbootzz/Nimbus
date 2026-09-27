package integrations

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/nimbus/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQbittorrent(t *testing.T) {
	impl := registered(t, "qbittorrent")
	logins := 0
	validSID := "one"
	authed := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("QBT_SID_8080")
			if err != nil || cookie.Value != validSID {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			next(w, r)
		}
	}
	server := fakeApp(t, map[string]http.HandlerFunc{
		"POST /api/v2/auth/login": func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			form, _ := url.ParseQuery(string(body))
			if form.Get("username") != "admin" || form.Get("password") != "pw" {
				fmt.Fprint(w, "Fails.") // qBittorrent answers 200 here
				return
			}
			logins++
			http.SetCookie(w, &http.Cookie{Name: "QBT_SID_8080", Value: validSID})
			fmt.Fprint(w, "Ok.")
		},
		"POST /api/v2/auth/logout": authed(func(w http.ResponseWriter, r *http.Request) {}),
		"GET /api/v2/app/version":  authed(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "v5.0.0") }),
		"GET /api/v2/transfer/info": authed(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"dl_info_speed":2500000,"up_info_speed":800}`)
		}),
		"GET /api/v2/torrents/info": authed(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `[{"progress":0.5},{"progress":1},{"progress":1},{"progress":0}]`)
		}),
	})
	ctx := context.Background()
	conn := newTestConn(server.URL, models.IntegrationCredentials{Username: "admin", Password: "pw"})

	require.NoError(t, impl.Test(ctx, conn))
	payload, err := impl.Fetch(ctx, conn)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"leech": 2, "download": "2.5 MB/s", "seed": 2, "upload": "800 B/s"}, payload.KPIs)

	validSID = "two" // session expired: 403, then one new login
	_, err = impl.Fetch(ctx, conn)
	require.NoError(t, err)
	assert.Equal(t, 3, logins)

	wrong := newTestConn(server.URL, models.IntegrationCredentials{Username: "admin", Password: "x"})
	assert.ErrorIs(t, impl.Test(ctx, wrong), errRejected)
}
