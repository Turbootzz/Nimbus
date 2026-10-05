package integrations

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nimbus/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArrCalendar(t *testing.T) {
	var query string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, arrTestKey, r.Header.Get("X-Api-Key"))
		require.Equal(t, "/api/v3/calendar", r.URL.Path)
		query = r.URL.RawQuery
		if r.URL.Query().Get("includeSeries") == "true" {
			fmt.Fprint(w, `[
				{"seasonNumber":2,"episodeNumber":5,"airDateUtc":"2026-10-07T01:00:00Z","series":{"title":"Severance"}},
				{"seasonNumber":1,"episodeNumber":1,"series":{"title":"No date yet"}}
			]`)
			return
		}
		fmt.Fprint(w, `[{"title":"Dune","inCinemas":"2026-09-01T00:00:00Z","digitalRelease":"2026-10-12T00:00:00Z","physicalRelease":"2026-12-01T00:00:00Z"}]`)
	}))
	t.Cleanup(server.Close)
	conn := newTestConn(server.URL, models.IntegrationCredentials{APIKey: arrTestKey})
	from := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)

	episodes, err := ArrCalendar(context.Background(), conn, "sonarr", from, to)
	require.NoError(t, err)
	assert.Equal(t, "start=2026-10-01&end=2026-11-01&includeSeries=true", query)
	assert.Equal(t, []CalendarEvent{{Title: "Severance S02E05", Start: time.Date(2026, 10, 7, 1, 0, 0, 0, time.UTC)}}, episodes)

	movies, err := ArrCalendar(context.Background(), conn, "radarr", from, to)
	require.NoError(t, err)
	assert.Equal(t, []CalendarEvent{{Title: "Dune (digital)", Start: time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC), AllDay: true}}, movies,
		"only the release dates in range")

	_, err = ArrCalendar(context.Background(), conn, "pihole", from, to)
	assert.ErrorContains(t, err, "pihole has no calendar")
}
