package integrations

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nimbus/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const arrTestKey = "arr-s3cret"

// fakeArr serves the v3 endpoints the way Sonarr and Radarr do, with 3
// items in the library list
func fakeArr(t *testing.T, libraryPath string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != arrTestKey {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		assert.NotContains(t, r.URL.RawQuery, arrTestKey, "the key must not be in the URL")
		switch r.URL.Path {
		case "/sub/api/v3/system/status":
			fmt.Fprint(w, `{"version":"4.0.0"}`)
		case "/sub/api/v3/wanted/missing":
			assert.Equal(t, "1", r.URL.Query().Get("pageSize"))
			fmt.Fprint(w, `{"page":1,"pageSize":1,"totalRecords":12,"records":[{}]}`)
		case "/sub/api/v3/queue":
			fmt.Fprint(w, `{"totalRecords":2,"records":[{}]}`)
		case "/sub" + libraryPath:
			fmt.Fprint(w, `[{"id":1,"title":"a"},{"id":2,"title":"b","seasons":[{"x":[1,2]}]},{"id":3}]`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func arrConn(baseURL, key string) *Conn {
	return &Conn{
		BaseURL: baseURL,
		Creds:   models.IntegrationCredentials{APIKey: key},
		Client:  http.DefaultClient,
		State:   &State{},
	}
}

func TestArrKinds(t *testing.T) {
	cases := []struct {
		kind, libraryPath, libraryKey string
		port                          int
	}{
		{"sonarr", "/api/v3/series", "series", 8989},
		{"radarr", "/api/v3/movie", "movies", 7878},
	}
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			impl, ok := Get(tc.kind)
			require.True(t, ok)
			meta := impl.Meta()
			assert.Equal(t, tc.port, meta.DefaultPort)
			assert.Equal(t, []string{models.IntegrationAuthAPIKey}, meta.AuthTypes)
			var keys []string
			for _, kpi := range meta.KPIs {
				keys = append(keys, kpi.Key)
			}
			assert.Equal(t, []string{"wanted", "queued", tc.libraryKey}, keys)

			server := fakeArr(t, tc.libraryPath)
			// A base URL with a path, like a reverse proxy sub path
			conn := arrConn(server.URL+"/sub", arrTestKey)
			ctx := context.Background()

			require.NoError(t, impl.Test(ctx, conn))
			payload, err := impl.Fetch(ctx, conn)
			require.NoError(t, err)
			assert.Equal(t, map[string]any{"wanted": 12, "queued": 2, tc.libraryKey: 3}, payload.KPIs)

			wrongKey := arrConn(server.URL+"/sub", "nope")
			assert.EqualError(t, impl.Test(ctx, wrongKey), "the API key was rejected")
			_, err = impl.Fetch(ctx, wrongKey)
			assert.EqualError(t, err, "the API key was rejected")
		})
	}
}

func TestArrErrors(t *testing.T) {
	sonarr, _ := Get("sonarr")
	ctx := context.Background()

	// Unreachable host
	server := httptest.NewServer(http.NotFoundHandler())
	url := server.URL
	server.Close()
	err := sonarr.Test(ctx, arrConn(url, arrTestKey))
	require.Error(t, err)
	assert.NotContains(t, err.Error(), arrTestKey)

	// Not an *arr app: wrong status, then a body that isn't JSON
	server = httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(server.Close)
	assert.EqualError(t, sonarr.Test(ctx, arrConn(server.URL, arrTestKey)), "/api/v3/system/status returned status 404")

	html := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<html>login</html>")
	}))
	t.Cleanup(html.Close)
	assert.EqualError(t, sonarr.Test(ctx, arrConn(html.URL, arrTestKey)), "/api/v3/system/status returned an unexpected response")
	_, err = sonarr.Fetch(ctx, arrConn(html.URL, arrTestKey))
	assert.Error(t, err)
}

func TestCountJSONArray(t *testing.T) {
	n, err := countJSONArray(strings.NewReader(`[]`))
	require.NoError(t, err)
	assert.Equal(t, 0, n)

	n, err = countJSONArray(strings.NewReader(` [1, {"a":[2,3]}, "x", null]`))
	require.NoError(t, err)
	assert.Equal(t, 4, n)

	for _, bad := range []string{`{}`, `"x"`, ``, `[1,`} {
		_, err := countJSONArray(strings.NewReader(bad))
		assert.Error(t, err, bad)
	}
}
