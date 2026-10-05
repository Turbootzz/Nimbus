package widgets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
)

// Widgets fetch small documents; anything bigger is a mistake
const maxFetchBodyBytes = 1 << 20

// getBody GETs rawURL with the given headers and returns a body of at most
// maxFetchBodyBytes
func getBody(ctx context.Context, client *http.Client, rawURL string, header http.Header) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	maps.Copy(req.Header, header)
	resp, err := client.Do(req)
	if err != nil {
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
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxFetchBodyBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxFetchBodyBytes {
		return nil, errors.New("response is larger than 1 MB")
	}
	return body, nil
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
