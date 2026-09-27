package integrations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/nimbus/backend/internal/models"
)

// Library lists (all series or movies) can be a few MB on big setups
const maxArrBodyBytes = 64 << 20

// arrKind is the shared client for the *arr apps on API v3 (Sonarr, Radarr).
// They differ only in name, port and which library they count.
type arrKind struct {
	kind        string
	name        string
	port        int
	libraryPath string // e.g. /api/v3/series
	library     KPI
}

func (a arrKind) Kind() string { return a.kind }

func (a arrKind) Meta() Meta {
	return Meta{
		Name:        a.name,
		Icon:        a.kind,
		DefaultPort: a.port,
		AuthTypes:   []string{models.IntegrationAuthAPIKey},
		KPIs: []KPI{
			{Key: "wanted", Label: "Wanted"},
			{Key: "queued", Label: "Queued"},
			a.library,
		},
	}
}

func (a arrKind) Test(ctx context.Context, conn *Conn) error {
	var status struct {
		Version string `json:"version"`
	}
	return arrGet(ctx, conn, "/api/v3/system/status", func(body io.Reader) error {
		return json.NewDecoder(body).Decode(&status)
	})
}

func (a arrKind) Fetch(ctx context.Context, conn *Conn) (*Payload, error) {
	wanted, err := arrTotal(ctx, conn, "/api/v3/wanted/missing?pageSize=1")
	if err != nil {
		return nil, err
	}
	queued, err := arrTotal(ctx, conn, "/api/v3/queue?pageSize=1")
	if err != nil {
		return nil, err
	}
	var library int
	err = arrGet(ctx, conn, a.libraryPath, func(body io.Reader) error {
		library, err = countJSONArray(body)
		return err
	})
	if err != nil {
		return nil, err
	}
	return &Payload{KPIs: map[string]any{"wanted": wanted, "queued": queued, a.library.Key: library}}, nil
}

// arrTotal reads totalRecords from a paged endpoint
func arrTotal(ctx context.Context, conn *Conn, path string) (int, error) {
	var page struct {
		TotalRecords int `json:"totalRecords"`
	}
	err := arrGet(ctx, conn, path, func(body io.Reader) error {
		return json.NewDecoder(body).Decode(&page)
	})
	return page.TotalRecords, err
}

// arrGet calls an API path with the key in the X-Api-Key header (never in
// the URL, so it can't end up in errors) and hands the body to read
func arrGet(ctx context.Context, conn *Conn, path string, read func(io.Reader) error) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, conn.BaseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Api-Key", conn.Creds.APIKey)
	req.Header.Set("Accept", "application/json")

	resp, err := conn.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		return errors.New("the API key was rejected")
	case resp.StatusCode != http.StatusOK:
		return fmt.Errorf("%s returned status %d", path, resp.StatusCode)
	}
	if err := read(io.LimitReader(resp.Body, maxArrBodyBytes)); err != nil {
		return fmt.Errorf("%s returned an unexpected response", path)
	}
	return nil
}

// countJSONArray counts the elements of a JSON array without keeping them
func countJSONArray(r io.Reader) (int, error) {
	dec := json.NewDecoder(r)
	if tok, err := dec.Token(); err != nil || tok != json.Delim('[') {
		return 0, errors.New("expected a JSON array")
	}
	n := 0
	for dec.More() {
		var skip json.RawMessage
		if err := dec.Decode(&skip); err != nil {
			return 0, err
		}
		n++
	}
	return n, nil
}
