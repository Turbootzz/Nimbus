package integrations

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nimbus/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestConn(baseURL string, creds models.IntegrationCredentials) *Conn {
	return &Conn{BaseURL: baseURL, Creds: creds, Client: http.DefaultClient, State: &State{}}
}

// fakeApp serves one handler per path and fails the test on anything else
func fakeApp(t *testing.T, routes map[string]http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler, ok := routes[r.Method+" "+r.URL.Path]
		if !ok {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		handler(w, r)
	}))
	t.Cleanup(server.Close)
	return server
}

// registered returns a kind and checks its metadata is usable
func registered(t *testing.T, kind string) Integration {
	t.Helper()
	impl, ok := Get(kind)
	require.True(t, ok, kind)
	meta := impl.Meta()
	assert.NotEmpty(t, meta.Name)
	assert.NotEmpty(t, meta.KPIs)
	assert.NotZero(t, meta.DefaultPort)
	return impl
}

func TestDoRequestStatuses(t *testing.T) {
	server := fakeApp(t, map[string]http.HandlerFunc{
		"GET /unauthorized": func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusUnauthorized) },
		"GET /forbidden":    func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusForbidden) },
		"GET /broken":       func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusBadGateway) },
		"GET /html":         func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("<html>")) },
	})
	conn := newTestConn(server.URL, models.IntegrationCredentials{})
	ctx := context.Background()
	var v map[string]any

	assert.ErrorIs(t, getJSON(ctx, conn, "/unauthorized", nil, &v), errRejected)
	assert.ErrorIs(t, getJSON(ctx, conn, "/forbidden", nil, &v), errRejected)
	assert.EqualError(t, getJSON(ctx, conn, "/broken?token=x", nil, &v), "/broken returned status 502", "no query in errors")
	assert.EqualError(t, getJSON(ctx, conn, "/html", nil, &v), "/html returned an unexpected response")
}

func TestWithSessionRelogsOnceWhenRejected(t *testing.T) {
	conn := newTestConn("", models.IntegrationCredentials{})
	logins := 0
	login := func() (string, error) {
		logins++
		return "s" + string(rune('0'+logins)), nil
	}

	var seen []string
	call := func(session string) error {
		seen = append(seen, session)
		if session == "s1" && len(seen) > 1 {
			return errRejected // the stored session expired
		}
		return nil
	}

	require.NoError(t, withSession(conn, "sid", login, call))
	require.NoError(t, withSession(conn, "sid", login, call))
	assert.Equal(t, []string{"s1", "s1", "s2"}, seen, "reused, then renewed once")
	assert.Equal(t, 2, logins)

	// A fresh login that is rejected is not retried
	conn.State.Delete("sid")
	err := withSession(conn, "sid", login, func(string) error { return errRejected })
	assert.ErrorIs(t, err, errRejected)
	assert.Equal(t, 3, logins)

	failing := errors.New("down")
	conn.State.Delete("sid")
	assert.ErrorIs(t, withSession(conn, "sid", func() (string, error) { return "", failing }, call), failing)
}

func TestFormatRateAndPercent(t *testing.T) {
	assert.Equal(t, "0 B/s", formatRate(0))
	assert.Equal(t, "999 B/s", formatRate(999))
	assert.Equal(t, "1.5 KB/s", formatRate(1500))
	assert.Equal(t, "12.3 MB/s", formatRate(12_345_678))
	assert.Equal(t, "250 MB/s", formatRate(250_000_000))
	assert.Equal(t, "3.0 GB/s", formatRate(3_000_000_000))
	assert.Equal(t, 0, percent(5, 0))
	assert.Equal(t, 33, percent(1, 3))
}
