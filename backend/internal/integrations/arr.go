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
	// wantedFromLibrary counts wanted items in the library list instead of
	// calling /wanted/missing, which not every Radarr version has
	wantedFromLibrary bool
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
	queued, err := arrTotal(ctx, conn, "/api/v3/queue?pageSize=1")
	if err != nil {
		return nil, err
	}
	var library, wanted int
	err = arrGet(ctx, conn, a.libraryPath, func(body io.Reader) error {
		library, wanted, err = countLibrary(body)
		return err
	})
	if err != nil {
		return nil, err
	}
	if !a.wantedFromLibrary {
		if wanted, err = arrTotal(ctx, conn, "/api/v3/wanted/missing?pageSize=1"); err != nil {
			return nil, err
		}
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

// countLibrary counts the items of a library list, and those that are
// wanted: monitored, not downloaded and released. It reads the list item by
// item, so a big library is never held in memory.
func countLibrary(r io.Reader) (total, wanted int, err error) {
	dec := json.NewDecoder(r)
	if tok, err := dec.Token(); err != nil || tok != json.Delim('[') {
		return 0, 0, errors.New("expected a JSON array")
	}
	for dec.More() {
		var item struct {
			Monitored   bool `json:"monitored"`
			HasFile     bool `json:"hasFile"`
			IsAvailable bool `json:"isAvailable"`
		}
		if err := dec.Decode(&item); err != nil {
			return 0, 0, err
		}
		total++
		if item.Monitored && !item.HasFile && item.IsAvailable {
			wanted++
		}
	}
	return total, wanted, nil
}
