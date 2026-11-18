package utils

import (
	"errors"
	"net/http"
	"net/http/httptest"
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

func TestNewSafeClient_RedirectsKeepCredentialsOnTheSameHostOnly(t *testing.T) {
	type seen struct{ apiKey, auth, accept string }
	got := make(chan seen, 1)
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got <- seen{r.Header.Get("X-API-Key"), r.Header.Get("Authorization"), r.Header.Get("Accept")}
	}))
	defer target.Close()
	// httptest servers all listen on 127.0.0.1; "localhost" makes the
	// target another host
	otherHost := strings.Replace(target.URL, "127.0.0.1", "localhost", 1)

	location := ""
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, location, http.StatusTemporaryRedirect)
	}))
	defer origin.Close()

	do := func(method string) error {
		req, err := http.NewRequest(method, origin.URL, strings.NewReader("password=hunter2"))
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

	// Another port on the same host is still the same app
	location = target.URL + "/same-host"
	require.NoError(t, do(http.MethodGet))
	assert.Equal(t, seen{"secret", "PVEAPIToken=secret", "application/json"}, <-got)

	location = otherHost + "/other-host"
	require.NoError(t, do(http.MethodGet))
	assert.Equal(t, seen{"", "", "application/json"}, <-got, "credentials stay behind, harmless headers go along")

	// A 307 would resend the login form
	err := do(http.MethodPost)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "refused a redirect")
	assert.Empty(t, got, "the other host was never called")
}

func TestDropCredentialsOnRedirect_HTTPSDowngrade(t *testing.T) {
	first := httptest.NewRequest(http.MethodGet, "https://app.lan/api", nil)
	upgrade := httptest.NewRequest(http.MethodGet, "http://app.lan/api", nil)
	upgrade.Header.Set("X-API-Key", "secret")

	// http to https on the same host keeps the key
	next := httptest.NewRequest(http.MethodGet, "https://APP.lan:8443/api", nil)
	next.Header.Set("X-API-Key", "secret")
	require.NoError(t, dropCredentialsOnRedirect(next, upgrade))
	assert.Equal(t, "secret", next.Header.Get("X-API-Key"))

	// https to http never does
	require.NoError(t, dropCredentialsOnRedirect(upgrade, first))
	assert.Empty(t, upgrade.Header.Get("X-API-Key"))
}
