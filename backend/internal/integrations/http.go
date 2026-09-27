package integrations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Big lists (Home Assistant states, torrents) can be a few MB
const maxBodyBytes = 64 << 20

// errRejected means the app refused the credentials or session (401, 403)
var errRejected = errors.New("the credentials were rejected")

// authFunc adds a kind's credentials to a request
type authFunc func(*http.Request)

// doRequest sends a request to BaseURL+path and hands a 200 body to read.
// 401 and 403 become errRejected; other statuses an error with the path.
func doRequest(ctx context.Context, conn *Conn, method, path string, body io.Reader, auth authFunc, read func(*http.Response) error) error {
	req, err := http.NewRequestWithContext(ctx, method, conn.BaseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if auth != nil {
		auth(req)
	}

	resp, err := conn.Client.Do(req)
	if err != nil {
		return transportError(ctx, path, err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return errRejected
	case resp.StatusCode < 200 || resp.StatusCode > 299:
		return fmt.Errorf("%s returned status %d", pathOnly(path), resp.StatusCode)
	}
	if read == nil {
		return nil
	}
	resp.Body = io.NopCloser(io.LimitReader(resp.Body, maxBodyBytes))
	if err := read(resp); err != nil {
		if isTimeout(ctx, err) {
			return fmt.Errorf("%s took too long to answer", pathOnly(path))
		}
		return fmt.Errorf("%s returned an unexpected response", pathOnly(path))
	}
	return nil
}

// transportError describes a failed request without the URL Go puts in its
// errors, but keeps the cause: "connection refused" or a certificate error
// tells the user what to fix
func transportError(ctx context.Context, path string, err error) error {
	if isTimeout(ctx, err) {
		return fmt.Errorf("%s took too long to answer", pathOnly(path))
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		err = urlErr.Err
	}
	return fmt.Errorf("%s could not be reached: %w", pathOnly(path), err)
}

func isTimeout(ctx context.Context, err error) bool {
	var netErr net.Error
	return ctx.Err() != nil || (errors.As(err, &netErr) && netErr.Timeout())
}

// getJSON GETs path and decodes the JSON body into v
func getJSON(ctx context.Context, conn *Conn, path string, auth authFunc, v any) error {
	return doRequest(ctx, conn, http.MethodGet, path, nil, auth, func(resp *http.Response) error {
		return json.NewDecoder(resp.Body).Decode(v)
	})
}

// eachJSON GETs a JSON array and decodes it item by item into a new T, so
// a big list is never held in memory
func eachJSON[T any](ctx context.Context, conn *Conn, path string, auth authFunc, fn func(T)) error {
	return doRequest(ctx, conn, http.MethodGet, path, nil, auth, func(resp *http.Response) error {
		dec := json.NewDecoder(resp.Body)
		if tok, err := dec.Token(); err != nil || tok != json.Delim('[') {
			return errors.New("expected a JSON array")
		}
		for dec.More() {
			var item T
			if err := dec.Decode(&item); err != nil {
				return err
			}
			fn(item)
		}
		return nil
	})
}

// withSession runs call with the session stored under key, logging in when
// there is none. If a stored session was rejected (expired), it logs in
// once more and retries. Apps with few session seats (Pi-hole) need the
// reuse.
func withSession(conn *Conn, key string, login func() (string, error), call func(session string) error) error {
	session, stored := conn.State.Get(key)
	if !stored {
		fresh, err := login()
		if err != nil {
			return err
		}
		conn.State.Set(key, fresh)
		session = fresh
	}
	err := call(session)
	if !stored || !errors.Is(err, errRejected) {
		return err
	}
	conn.State.Delete(key)
	fresh, err := login()
	if err != nil {
		return err
	}
	conn.State.Set(key, fresh)
	return call(fresh)
}

// pathOnly drops the query, which may hold parameters an app needs
func pathOnly(path string) string {
	p, _, _ := strings.Cut(path, "?")
	return p
}

// percent returns part/total as a whole percentage, 0 when total is 0
func percent(part, total float64) int {
	if total <= 0 {
		return 0
	}
	return int(part/total*100 + 0.5)
}

// formatRate turns bytes per second into a short text like "1.5 MB/s".
// The unit is picked after rounding, so 999,950 B/s is "1.0 MB/s".
func formatRate(bytesPerSecond int64) string {
	units := []string{"B/s", "KB/s", "MB/s", "GB/s"}
	value := float64(bytesPerSecond)
	for unit := 0; ; unit++ {
		decimals := 1
		if unit == 0 || value >= 100 {
			decimals = 0
		}
		scale := math.Pow10(decimals)
		rounded := math.Round(value*scale) / scale
		if rounded < 1000 || unit == len(units)-1 {
			return strconv.FormatFloat(rounded, 'f', decimals, 64) + " " + units[unit]
		}
		value /= 1000
	}
}
