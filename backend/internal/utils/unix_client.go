package utils

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// NewUnixSocketClient returns a client that sends every request to the unix
// socket at path, whatever host the URL names. It only allows GET and HEAD:
// mounting a socket read-only does not make the API behind it read-only.
func NewUnixSocketClient(path string, timeout time.Duration) *http.Client {
	dialer := &net.Dialer{Timeout: timeout}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return dialer.DialContext(ctx, "unix", path)
		},
		ResponseHeaderTimeout: timeout,
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: readOnlyTransport{transport},
		// A redirect from a local socket is never expected
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

type readOnlyTransport struct {
	next *http.Transport
}

func (t readOnlyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		return nil, fmt.Errorf("%s requests are not allowed on the socket", req.Method)
	}
	return t.next.RoundTrip(req)
}

// CloseIdleConnections lets http.Client.CloseIdleConnections reach the transport
func (t readOnlyTransport) CloseIdleConnections() {
	t.next.CloseIdleConnections()
}
