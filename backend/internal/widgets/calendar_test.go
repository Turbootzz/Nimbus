package widgets

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nimbus/backend/internal/integrations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sonarrID = "11111111-1111-1111-1111-111111111111"

func TestCalendarValidate(t *testing.T) {
	runConfigCases(t, "calendar", []configCase{
		{"integration only", `{"integrations":[" ` + sonarrID + ` ","` + sonarrID + `"]}`,
			`{"integrations":["` + sonarrID + `"],"ical_urls":[],"days":14,"verify_tls":true}`, ""},
		{"feed only", `{"ical_urls":[" http://10.0.0.2/cal.ics ",""],"days":30}`,
			`{"integrations":[],"ical_urls":["http://10.0.0.2/cal.ics"],"days":30,"verify_tls":true}`, ""},
		{"nothing", `{"ical_urls":[" "]}`, "", "pick an integration or add a calendar URL"},
		{"bad id", `{"integrations":["sonarr"]}`, "", "integration not found"},
		{"not http", `{"ical_urls":["webcal://a.test/x.ics"]}`, "", "invalid calendar URL"},
		{"too many feeds", `{"ical_urls":["http://a.test/1","http://a.test/2","http://a.test/3","http://a.test/4","http://a.test/5","http://a.test/6"]}`, "", "at most 5"},
		{"days", `{"ical_urls":["http://10.0.0.2/c"],"days":61}`, "", "between 1 and 60"},
	})
}

func TestCalendarFetch(t *testing.T) {
	old := calendarNow
	calendarNow = func() time.Time { return time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { calendarNow = old })

	sonarr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "start=2026-10-01&end=2026-11-01&includeSeries=true", r.URL.RawQuery)
		fmt.Fprint(w, `[{"seasonNumber":1,"episodeNumber":3,"airDateUtc":"2026-10-07T01:00:00Z","series":{"title":"Andor"}}]`)
	}))
	t.Cleanup(sonarr.Close)
	feeds := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cal.ics" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		fmt.Fprint(w, icalFixture)
	}))
	t.Cleanup(feeds.Close)

	w, _ := Get("calendar")
	config := `{"integrations":["` + sonarrID + `"],"ical_urls":["` + feeds.URL + `/cal.ics","` + feeds.URL + `/gone.ics"],"days":14}`
	got, err := w.(Fetcher).Fetch(context.Background(), &FetchRequest{
		Config: json.RawMessage(config),
		Client: http.DefaultClient,
		Linked: []LinkedIntegration{{ID: sonarrID, Kind: "sonarr", Name: "Sonarr",
			Conn: &integrations.Conn{BaseURL: sonarr.URL, Client: http.DefaultClient, State: &integrations.State{}}}},
		Unlinked: []string{"an integration that was removed"},
	})
	require.NoError(t, err)
	payload := got.(calendarPayload)

	assert.Equal(t, "2026-10-01", payload.From)
	assert.Equal(t, []calendarSource{{Name: "Sonarr", Kind: "sonarr"}, {Name: hostOf(feeds.URL), Kind: "ical"}}, payload.Sources)
	assert.Len(t, payload.Failed, 2, "the removed integration and the missing feed")

	var titles []string
	for _, e := range payload.Events {
		titles = append(titles, e.Title)
	}
	assert.Equal(t, []string{"Andor S01E03", "Dentist appointment with a long name that is folded over two lines", "Standup", "Bin day, green"}, titles,
		"sorted by start")
	assert.Equal(t, calendarEvent{Title: "Bin day, green", Start: "2026-10-09", AllDay: true, Source: 1}, calendarEvent{
		Title: payload.Events[3].Title, Start: payload.Events[3].Start, AllDay: payload.Events[3].AllDay, Source: payload.Events[3].Source,
	})
	assert.Equal(t, "2026-10-07T01:00:00Z", payload.Events[0].Start)
}

func TestCalendarFetchAllFailed(t *testing.T) {
	w, _ := Get("calendar")
	_, err := w.(Fetcher).Fetch(context.Background(), &FetchRequest{
		Config:   json.RawMessage(`{"integrations":["` + sonarrID + `"],"ical_urls":[]}`),
		Client:   http.DefaultClient,
		Unlinked: []string{"an integration that was removed"},
	})
	assert.EqualError(t, err, "an integration that was removed")
}
