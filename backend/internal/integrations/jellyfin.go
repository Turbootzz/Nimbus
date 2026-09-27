package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nimbus/backend/internal/models"
)

type jellyfin struct{}

func init() { Register(jellyfin{}) }

func (jellyfin) Kind() string { return "jellyfin" }

func (jellyfin) Meta() Meta {
	return Meta{
		Name:        "Jellyfin",
		Icon:        "jellyfin",
		DefaultPort: 8096,
		AuthTypes:   []string{models.IntegrationAuthAPIKey},
		KPIs: []KPI{
			{Key: "movies", Label: "Movies"},
			{Key: "series", Label: "Series"},
			{Key: "episodes", Label: "Episodes"},
			{Key: "streams", Label: "Streams"},
		},
	}
}

func jellyfinAuth(conn *Conn) authFunc {
	return func(r *http.Request) {
		// Newer versions prefer the Authorization header, older ones X-Emby-Token
		r.Header.Set("Authorization", fmt.Sprintf("MediaBrowser Token=%q", conn.Creds.APIKey))
		r.Header.Set("X-Emby-Token", conn.Creds.APIKey)
	}
}

func (jellyfin) Test(ctx context.Context, conn *Conn) error {
	var info map[string]any
	return getJSON(ctx, conn, "/System/Info", jellyfinAuth(conn), &info)
}

func (jellyfin) Fetch(ctx context.Context, conn *Conn) (*Payload, error) {
	var counts struct {
		Movies   int64 `json:"MovieCount"`
		Series   int64 `json:"SeriesCount"`
		Episodes int64 `json:"EpisodeCount"`
	}
	if err := getJSON(ctx, conn, "/Items/Counts", jellyfinAuth(conn), &counts); err != nil {
		return nil, err
	}
	streams := 0
	err := eachJSON(ctx, conn, "/Sessions", jellyfinAuth(conn), func(s struct {
		NowPlayingItem json.RawMessage `json:"NowPlayingItem"`
	}) {
		if len(s.NowPlayingItem) > 0 && string(s.NowPlayingItem) != "null" {
			streams++
		}
	})
	if err != nil {
		return nil, err
	}
	return &Payload{KPIs: map[string]any{
		"movies":   counts.Movies,
		"series":   counts.Series,
		"episodes": counts.Episodes,
		"streams":  streams,
	}}, nil
}
