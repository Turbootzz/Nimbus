package widgets

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net"
	"net/http"
	"net/url"
)

// Widgets fetch small documents; anything bigger is a mistake
const maxFetchBodyBytes = 1 << 20

var utf8BOM = []byte("\xef\xbb\xbf")

// tlsOption is part of the config of types that fetch a URL the user gives,
// which may be a LAN host with a self-signed certificate
type tlsOption struct {
	VerifyTLS *bool `json:"verify_tls"`
}

// normalise turns certificate checks on unless they were turned off
func (o *tlsOption) normalise() {
	if o.VerifyTLS == nil {
		verify := true
		o.VerifyTLS = &verify
	}
}

// VerifiesTLS reports whether fetches for this config check certificates.
// Only an explicit "verify_tls": false turns the check off.
func VerifiesTLS(config json.RawMessage) bool {
	var o tlsOption
	return json.Unmarshal(config, &o) != nil || o.VerifyTLS == nil || *o.VerifyTLS
}

// getBody GETs rawURL with the given headers and returns a body of at most
// maxFetchBodyBytes, without a UTF-8 byte order mark
func getBody(ctx context.Context, client *http.Client, rawURL string, header http.Header) ([]byte, error) {
	return getBodyLimit(ctx, client, rawURL, header, maxFetchBodyBytes)
}

// getBodyLimit is getBody with another size limit, for documents that are
// big by nature (calendars with years of history)
func getBodyLimit(ctx context.Context, client *http.Client, rawURL string, header http.Header, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	maps.Copy(req.Header, header)
	resp, err := client.Do(req)
	if err != nil {
		var netErr net.Error
		if ctx.Err() != nil || (errors.As(err, &netErr) && netErr.Timeout()) {
			return nil, errors.New("took too long to answer")
		}
		// Go puts the full URL in the error, and a query may hold a key
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			return nil, urlErr.Err
		}
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("response is larger than %d MB", limit>>20)
	}
	return bytes.TrimPrefix(body, utf8BOM), nil
}

// getJSON GETs rawURL and decodes the JSON body into v
func getJSON(ctx context.Context, client *http.Client, rawURL string, v any) error {
	body, err := getBody(ctx, client, rawURL, http.Header{"Accept": {"application/json"}})
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, v); err != nil {
		return errors.New("response is not valid JSON")
	}
	return nil
}
