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
			if libraryPath == "/api/v3/movie" {
				w.WriteHeader(http.StatusNotFound) // like Radarr 3 and 4
				return
			}
			assert.Equal(t, "1", r.URL.Query().Get("pageSize"))
			fmt.Fprint(w, `{"page":1,"pageSize":1,"totalRecords":12,"records":[{}]}`)
		case "/sub/api/v3/queue":
			fmt.Fprint(w, `{"totalRecords":2,"records":[{}]}`)
		case "/sub" + libraryPath:
			// One wanted item: monitored, no file, released
			fmt.Fprint(w, `[
				{"id":1,"monitored":true,"hasFile":false,"isAvailable":true},
				{"id":2,"monitored":true,"hasFile":true,"isAvailable":true,"seasons":[{"x":[1,2]}]},
				{"id":3,"monitored":false,"hasFile":false,"isAvailable":true},
				{"id":4,"monitored":true,"hasFile":false,"isAvailable":false}
			]`)
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
		port, wanted                  int
	}{
		{"sonarr", "/api/v3/series", "series", 8989, 12}, // from /wanted/missing
		{"radarr", "/api/v3/movie", "movies", 7878, 1},   // counted in the movie list
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
			assert.Equal(t, map[string]any{"wanted": tc.wanted, "queued": 2, tc.libraryKey: 4}, payload.KPIs)

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

func TestCountLibrary(t *testing.T) {
	total, wanted, err := countLibrary(strings.NewReader(`[]`))
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Equal(t, 0, wanted)

	total, wanted, err = countLibrary(strings.NewReader(` [{"monitored":true,"isAvailable":true,"extra":[1,{"a":2}]}, {}, {"hasFile":true}]`))
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Equal(t, 1, wanted)

	for _, bad := range []string{`{}`, `"x"`, ``, `[{},`, `[1]`} {
		_, _, err := countLibrary(strings.NewReader(bad))
		assert.Error(t, err, bad)
	}
}
