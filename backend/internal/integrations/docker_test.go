package integrations

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const dockerContainersFixture = `[
	{"Id":"1","Names":["/Web"],"Image":"nginx:1.27","State":"running","Status":"Up 3 days"},
	{"Id":"2","Names":["/backup"],"Image":"restic","State":"exited","Status":"Exited (0) 2 hours ago"},
	{"Id":"3","Names":["/db"],"Image":"postgres:18","State":"running","Status":"Up 3 days (healthy)"},
	{"Id":"4","Names":[],"Image":"busybox","State":"paused","Status":"Up 1 hour (Paused)"}
]`

func dockerRoutes(t *testing.T) http.Handler {
	t.Helper()
	routes := map[string]http.HandlerFunc{
		"GET /version": func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"Version":"27.3.1","ApiVersion":"1.47"}`)
		},
		"GET /containers/json": func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "1", r.URL.Query().Get("all"))
			fmt.Fprint(w, dockerContainersFixture)
		},
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler, ok := routes[r.Method+" "+r.URL.Path]
		if !ok {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		handler(w, r)
	})
}

func TestDockerOverHTTP(t *testing.T) {
	impl := registered(t, "docker")
	server := httptest.NewServer(dockerRoutes(t))
	t.Cleanup(server.Close)
	ctx := context.Background()
	conn := newTestConn(server.URL, models.IntegrationCredentials{})

	require.NoError(t, impl.Test(ctx, conn))
	payload, err := impl.Fetch(ctx, conn)
	require.NoError(t, err)
	// Paused is neither running nor stopped
	assert.Equal(t, map[string]any{"running": 2, "stopped": 1, "total": 4}, payload.KPIs)

	containers, err := DockerContainers(ctx, conn)
	require.NoError(t, err)
	assert.Equal(t, []DockerContainer{
		{ID: "3", Name: "db", Image: "postgres:18", State: "running", Status: "Up 3 days (healthy)"},
		{ID: "1", Name: "Web", Image: "nginx:1.27", State: "running", Status: "Up 3 days"},
		{ID: "4", Name: "", Image: "busybox", State: "paused", Status: "Up 1 hour (Paused)"},
		{ID: "2", Name: "backup", Image: "restic", State: "exited", Status: "Exited (0) 2 hours ago"},
	}, containers, "running first, then by name, ignoring case")
}

func TestDockerOverUnixSocket(t *testing.T) {
	impl := registered(t, "docker")
	dir, err := os.MkdirTemp("", "nimbus")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	path := filepath.Join(dir, "docker.sock")
	listener, err := net.Listen("unix", path)
	require.NoError(t, err)
	server := &http.Server{Handler: dockerRoutes(t)}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })

	conn := &Conn{BaseURL: "http://docker", Client: utils.NewUnixSocketClient(path, 2*time.Second), State: &State{}}
	defer conn.Client.CloseIdleConnections()
	require.NoError(t, impl.Test(context.Background(), conn))
	payload, err := impl.Fetch(context.Background(), conn)
	require.NoError(t, err)
	assert.Equal(t, 4, payload.KPIs["total"])
}

func TestDockerErrors(t *testing.T) {
	impl := registered(t, "docker")
	server := fakeApp(t, map[string]http.HandlerFunc{
		"GET /version": func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `{"status":"ok"}`) },
	})
	ctx := context.Background()
	assert.EqualError(t, impl.Test(ctx, newTestConn(server.URL, models.IntegrationCredentials{})), "/version did not answer like the Docker API")

	// A socket proxy that forbids the endpoint answers 403
	denied := fakeApp(t, map[string]http.HandlerFunc{
		"GET /containers/json": func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusForbidden) },
	})
	_, err := impl.Fetch(ctx, newTestConn(denied.URL, models.IntegrationCredentials{}))
	assert.ErrorIs(t, err, errRejected)

	_, err = impl.Fetch(ctx, newTestConn("http://127.0.0.1:1", models.IntegrationCredentials{}))
	assert.ErrorContains(t, err, "could not be reached")
}
