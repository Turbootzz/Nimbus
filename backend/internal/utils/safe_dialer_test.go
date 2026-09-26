package utils

import (
	"errors"
	"net/http"
	"net/http/httptest"
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
