package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// CalendarEvent is one dated item for the calendar widget
type CalendarEvent struct {
	Title  string
	Start  time.Time
	AllDay bool // Start is a date; the time means nothing
}

// ArrCalendar lists what Sonarr airs or Radarr releases between from and to.
// Episodes have an air time; movie releases are dates.
func ArrCalendar(ctx context.Context, conn *Conn, kind string, from, to time.Time) ([]CalendarEvent, error) {
	path := fmt.Sprintf("/api/v3/calendar?start=%s&end=%s", from.Format(time.DateOnly), to.Format(time.DateOnly))
	switch kind {
	case "sonarr":
		var episodes []struct {
			SeasonNumber  int       `json:"seasonNumber"`
			EpisodeNumber int       `json:"episodeNumber"`
			AirDateUTC    time.Time `json:"airDateUtc"`
			Series        struct {
				Title string `json:"title"`
			} `json:"series"`
		}
		if err := arrGet(ctx, conn, path+"&includeSeries=true", decodeInto(&episodes)); err != nil {
			return nil, err
		}
		events := make([]CalendarEvent, 0, len(episodes))
		for _, e := range episodes {
			if e.AirDateUTC.IsZero() {
				continue
			}
			title := fmt.Sprintf("%s S%02dE%02d", e.Series.Title, e.SeasonNumber, e.EpisodeNumber)
			events = append(events, CalendarEvent{Title: title, Start: e.AirDateUTC})
		}
		return events, nil

	case "radarr":
		var movies []struct {
			Title           string     `json:"title"`
			InCinemas       *time.Time `json:"inCinemas"`
			DigitalRelease  *time.Time `json:"digitalRelease"`
			PhysicalRelease *time.Time `json:"physicalRelease"`
		}
		if err := arrGet(ctx, conn, path, decodeInto(&movies)); err != nil {
			return nil, err
		}
		var events []CalendarEvent
		for _, m := range movies {
			for _, release := range []struct {
				date *time.Time
				what string
			}{{m.InCinemas, "in cinemas"}, {m.DigitalRelease, "digital"}, {m.PhysicalRelease, "physical"}} {
				// The calendar returns a movie when any of its dates is in range
				if release.date == nil || release.date.Before(from) || !release.date.Before(to) {
					continue
				}
				events = append(events, CalendarEvent{Title: m.Title + " (" + release.what + ")", Start: *release.date, AllDay: true})
			}
		}
		return events, nil
	}
	return nil, fmt.Errorf("%s has no calendar", kind)
}

func decodeInto(v any) func(io.Reader) error {
	return func(body io.Reader) error { return json.NewDecoder(body).Decode(v) }
}
