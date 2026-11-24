package utils

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsBlockedHost(t *testing.T) {
	cases := map[string]bool{
		"169.254.169.254":          true,
		"METADATA.GOOGLE.INTERNAL": true, // lookup is case-insensitive
		"100.100.100.200":          true,
		"github.com":               false,
		"192.168.1.5":              false, // private but not metadata - homelab use case
		"127.0.0.1":                false,
		"":                         false,
	}
	for host, want := range cases {
		assert.Equal(t, want, IsBlockedHost(host), "host=%q", host)
	}
}

func TestIsBlockedIP(t *testing.T) {
	cases := map[string]bool{
		// Direct hits.
		"169.254.169.254": true,
		"100.100.100.200": true,
		"168.63.129.16":   true,
		"169.254.170.2":   true, // ECS task credentials
		// IPv4-mapped IPv6: an attacker resolving to ::ffff:169.254.169.254
		// must NOT bypass the deny list.
		"::ffff:169.254.169.254": true,
		"::ffff:100.100.100.200": true,
		// Same metadata octets in different IPv6 forms.
		"::ffff:a9fe:a9fe": true, // hex form of 169.254.169.254
		// Pass-throughs.
		"127.0.0.1":   false,
		"192.168.1.5": false, // private but allowed for homelab use
		"::1":         false,
		"github.com":  false, // unparseable as IP → not blocked
		"":            false,
	}
	for addr, want := range cases {
		assert.Equal(t, want, IsBlockedIP(addr), "addr=%q", addr)
	}
}

func TestNewSafeClient_BlocksMetadataIPAtDial(t *testing.T) {
	client := NewSafeClient(true, 2*time.Second)
	_, err := client.Get("http://169.254.169.254/latest/meta-data/")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBlockedHost), "expected ErrBlockedHost, got %v", err)
}

func TestNewSafeClient_BlocksRedirectToMetadataHost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://metadata.google.internal/computeMetadata/v1/", http.StatusFound)
	}))
	defer server.Close()

	_, err := NewSafeClient(true, 2*time.Second).Get(server.URL)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBlockedHost), "expected ErrBlockedHost, got %v", err)
}

func TestNewSafeClient_StopsRedirectLoops(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/again", http.StatusFound)
	}))
	defer server.Close()

	_, err := NewSafeClient(true, 2*time.Second).Get(server.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stopped after 5 redirects")
}

func TestNewSafeClient_VerifyTLS(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Self-signed cert: rejected when verifying, accepted when opted out.
	_, err := NewSafeClient(true, 2*time.Second).Get(server.URL)
	assert.Error(t, err, "expected certificate error with verifyTLS=true")

	resp, err := NewSafeClient(false, 2*time.Second).Get(server.URL)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestNewSafeClient_RedirectsKeepCredentialsOnTheSameOriginOnly(t *testing.T) {
	type seen struct{ apiKey, auth, accept, body string }
	got := make(chan seen, 1)
	record := func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got <- seen{r.Header.Get("X-API-Key"), r.Header.Get("Authorization"), r.Header.Get("Accept"), string(body)}
	}
	target := httptest.NewServer(http.HandlerFunc(record))
	defer target.Close()

	location := ""
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/moved" {
			record(w, r)
			return
		}
		http.Redirect(w, r, location, http.StatusTemporaryRedirect)
	}))
	defer origin.Close()

	const form = "password=hunter2"
	do := func(method, body string) error {
		req, err := http.NewRequest(method, origin.URL, strings.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("X-API-Key", "secret")
		req.Header.Set("Authorization", "PVEAPIToken=secret")
		req.Header.Set("Accept", "application/json")
		resp, err := NewSafeClient(true, 2*time.Second).Do(req)
		if err == nil {
			resp.Body.Close()
		}
		return err
	}
	all := seen{"secret", "PVEAPIToken=secret", "application/json", form}
	harmless := seen{"", "", "application/json", ""}

	// The app moving its own path keeps everything, body included
	location = origin.URL + "/moved"
	require.NoError(t, do(http.MethodGet, form))
	assert.Equal(t, all, <-got)
	require.NoError(t, do(http.MethodPost, form))
	assert.Equal(t, all, <-got)

	// Another port on the same host can be another app; so can another
	// name (httptest servers all listen on 127.0.0.1)
	for _, other := range []string{target.URL, strings.Replace(target.URL, "127.0.0.1", "localhost", 1)} {
		location = other + "/elsewhere"
		require.NoError(t, do(http.MethodGet, ""))
		assert.Equal(t, harmless, <-got, "credentials stay behind, harmless headers go along: %s", other)

		// A 307 would resend the body, whatever the method
		for _, method := range []string{http.MethodPost, http.MethodGet} {
			err := do(method, form)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "refused a redirect")
			assert.Empty(t, got, "the other app was never called")
		}
	}
}

func TestSameService(t *testing.T) {
	for _, c := range []struct {
		first, next string
		want        bool
	}{
		{"https://app.lan/api", "https://APP.lan:443/v2", true},
		{"http://app.lan/api", "http://app.lan:80/v2", true},
		{"http://app.lan/api", "https://app.lan/api", true}, // standard upgrade
		{"http://app.lan:80/api", "https://app.lan:443/api", true},
		{"http://app.lan:5000/api", "https://app.lan:5001/api", false}, // another port can be another app
		{"http://app.lan/api", "https://app.lan:8443/api", false},
		{"http://app.lan:8080/api", "https://app.lan/api", false},
		{"https://app.lan/api", "http://app.lan/api", false}, // downgrade
		{"https://app.lan/api", "https://app.lan:8443/api", false},
		{"http://app.lan:8080/api", "http://app.lan:9000/api", false},
		{"https://app.lan/api", "https://sub.app.lan/api", false},
		{"http://app.lan/api", "https://other.lan/api", false},
		{"http://[::1]:80/api", "http://[::1]/api", true},
	} {
		first, err := url.Parse(c.first)
		require.NoError(t, err)
		next, err := url.Parse(c.next)
		require.NoError(t, err)
		assert.Equal(t, c.want, sameService(first, next), "%s to %s", c.first, c.next)
	}
}

func TestDropCredentialsOnRedirect_StripsAnyCasing(t *testing.T) {
	first := httptest.NewRequest(http.MethodGet, "https://app.lan/api", nil)
	next := httptest.NewRequest(http.MethodGet, "http://app.lan/api", nil)
	next.Header.Set("X-API-Key", "secret")
	next.Header["x-odd-key"] = []string{"secret"} // set past canonical casing
	next.Header["accept"] = []string{"text/plain"}

	require.NoError(t, dropCredentialsOnRedirect(next, first))
	assert.Equal(t, http.Header{"accept": {"text/plain"}}, next.Header)
}
