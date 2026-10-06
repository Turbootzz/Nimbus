package integrations

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/nimbus/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJellyfin(t *testing.T) {
	impl := registered(t, "jellyfin")
	auth := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Emby-Token") != "jf-key" || !strings.Contains(r.Header.Get("Authorization"), `Token="jf-key"`) {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			next(w, r)
		}
	}
	server := fakeApp(t, map[string]http.HandlerFunc{
		"GET /System/Info": auth(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `{"Version":"10.9"}`) }),
		"GET /Items/Counts": auth(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"MovieCount":310,"SeriesCount":42,"EpisodeCount":1800,"SongCount":0}`)
		}),
		"GET /Sessions": auth(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `[{"NowPlayingItem":{"Name":"A"}},{"UserName":"idle"},{"NowPlayingItem":null},{"NowPlayingItem":{"Name":"B"}}]`)
		}),
	})
	ctx := context.Background()
	conn := newTestConn(server.URL, models.IntegrationCredentials{APIKey: "jf-key"})

	require.NoError(t, impl.Test(ctx, conn))
	payload, err := impl.Fetch(ctx, conn)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"movies": int64(310), "series": int64(42), "episodes": int64(1800), "streams": 2}, payload.KPIs)
	assert.ErrorIs(t, impl.Test(ctx, newTestConn(server.URL, models.IntegrationCredentials{APIKey: "x"})), errRejected)
}
