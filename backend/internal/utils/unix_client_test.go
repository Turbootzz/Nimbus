package utils

import (
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnixSocketClient(t *testing.T) {
	// Unix socket paths are limited to about 100 bytes, so no t.TempDir()
	dir, err := os.MkdirTemp("", "nimbus")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	path := filepath.Join(dir, "api.sock")

	listener, err := net.Listen("unix", path)
	require.NoError(t, err)
	var methods []string
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		_, _ = io.WriteString(w, "ok "+r.URL.Path)
	})}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })

	client := NewUnixSocketClient(path, 2*time.Second)
	defer client.CloseIdleConnections()

	resp, err := client.Get("http://docker/version")
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	assert.Equal(t, "ok /version", string(body))

	_, err = client.Post("http://docker/containers/x/stop", "text/plain", strings.NewReader(""))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "POST requests are not allowed")
	assert.Equal(t, []string{http.MethodGet}, methods, "the POST never reached the socket")
}
