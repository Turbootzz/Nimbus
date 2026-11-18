package utils

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"syscall"
	"time"
)

// safeClientMaxRedirects caps redirect chains for server-side fetches.
const safeClientMaxRedirects = 5

// blockedHosts are hosts we refuse to fetch from even though Nimbus is
// otherwise permissive about private addresses (homelab services often live
// on RFC1918). These hosts expose credentials on cloud VMs, so an authenticated
// user must not be able to coax the server into hitting them — directly or via
// a redirect from an attacker-controlled public URL.
var blockedHosts = map[string]struct{}{
	"169.254.169.254":          {}, // AWS / Azure / GCP / DO IMDS
	"fd00:ec2::254":            {}, // AWS IMDS over IPv6
	"169.254.170.2":            {}, // AWS ECS task metadata and credentials
	"100.100.100.200":          {}, // Alibaba Cloud metadata
	"168.63.129.16":            {}, // Azure WireServer
	"metadata.google.internal": {}, // GCP DNS alias for 169.254.169.254
}

// IsBlockedHost reports whether a hostname is on the cloud metadata deny list.
// This is the single deny list for server-side fetches (see url_validator.go).
func IsBlockedHost(host string) bool {
	_, blocked := blockedHosts[strings.ToLower(host)]
	return blocked
}

// blockedIPs are the actual resolved IPs we refuse to dial. The hostname
// check above is a first-pass filter; this one defeats DNS rebinding, where an
// attacker-controlled domain resolves to one of these addresses despite the
// hostname not matching the deny list.
//
// IPs are stored as parsed net.IP so the comparison covers IPv4-mapped IPv6
// representations (e.g. ::ffff:169.254.169.254) — a plain string-compare would
// miss those and a clever attacker could resolve to that form to bypass.
var blockedIPs = []net.IP{
	net.ParseIP("169.254.169.254"), // AWS / Azure / GCP / DO IMDS
	net.ParseIP("fd00:ec2::254"),   // AWS IMDS over IPv6
	net.ParseIP("169.254.170.2"),   // AWS ECS task metadata and credentials
	net.ParseIP("100.100.100.200"), // Alibaba Cloud metadata
	net.ParseIP("168.63.129.16"),   // Azure WireServer
}

// IsBlockedIP reports whether an IP literal is on the cloud metadata deny list.
func IsBlockedIP(addr string) bool {
	ip := net.ParseIP(addr)
	if ip == nil {
		return false
	}
	// Normalize IPv4-mapped IPv6 down to plain IPv4 so the .Equal compare
	// matches the IPv4 literals above. .To4() returns the IPv4 form or nil.
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	for _, blocked := range blockedIPs {
		if blocked == nil {
			continue
		}
		b := blocked
		if v4 := b.To4(); v4 != nil {
			b = v4
		}
		if b.Equal(ip) {
			return true
		}
	}
	return false
}

// ErrBlockedHost is returned when a request or redirect targets a blocked
// host. net/http wraps it in url.Error, so callers should use errors.Is.
var ErrBlockedHost = errors.New("blocked host")

// redirectSafeHeaders may follow a redirect to another host. Credentials
// travel in headers of any name (X-API-Key, X-Api-Key, ...), so everything
// else is dropped.
var redirectSafeHeaders = map[string]bool{
	"Accept":          true,
	"Accept-Language": true,
	"User-Agent":      true,
}

// dropCredentialsOnRedirect keeps a redirect from carrying the first
// request's credentials to another host, or from https to plain http.
// Go only strips Authorization and Cookie itself, and re-adds the first
// request's headers on every hop, hence the compare with the first request.
// Another port on the same host keeps them, so http://app:80 can still
// upgrade to https://app:443.
func dropCredentialsOnRedirect(req, first *http.Request) error {
	sameHost := strings.EqualFold(req.URL.Hostname(), first.URL.Hostname())
	downgrade := first.URL.Scheme == "https" && req.URL.Scheme != "https"
	if sameHost && !downgrade {
		return nil
	}
	// A 307 or 308 would send the body (a login form) along
	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		return errors.New("refused a redirect that sends the request body to another host")
	}
	for name := range req.Header {
		if !redirectSafeHeaders[name] {
			req.Header.Del(name)
		}
	}
	return nil
}

// NewSafeClient returns an HTTP client for fetching user-supplied URLs.
// Defense in depth: CheckRedirect blocks by hostname, the dialer's Control
// blocks by resolved IP (so DNS rebinding can't sneak through). A redirect
// to another host never gets the request's credentials.
// verifyTLS=false skips certificate checks for this client only.
func NewSafeClient(verifyTLS bool, timeout time.Duration) *http.Client {
	dialer := &net.Dialer{
		Timeout: timeout,
		Control: func(_, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				host = address
			}
			if IsBlockedIP(host) {
				return ErrBlockedHost
			}
			return nil
		},
	}

	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		TLSHandshakeTimeout:   timeout,
		ResponseHeaderTimeout: timeout,
	}
	if !verifyTLS {
		// Opt-in per integration for self-signed homelab certs.
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	}

	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= safeClientMaxRedirects {
				return fmt.Errorf("stopped after %d redirects", safeClientMaxRedirects)
			}
			if IsBlockedHost(req.URL.Hostname()) {
				return ErrBlockedHost
			}
			return dropCredentialsOnRedirect(req, via[0])
		},
		Transport: transport,
	}
}
