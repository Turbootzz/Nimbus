package widgets

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nimbus/backend/internal/integrations"
	"github.com/nimbus/backend/internal/utils"
)

const (
	maxCalendarIntegrations   = 4
	maxCalendarFeeds          = 5
	defaultCalendarDays       = 14
	maxCalendarDays           = 60
	calendarMinRefreshSeconds = 300
	// Calendars with years of history are bigger than other feeds
	maxICalBytes = 8 << 20
)

type calendarConfig struct {
	Integrations []string `json:"integrations"` // Sonarr and Radarr ids
	ICalURLs     []string `json:"ical_urls"`
	Days         int      `json:"days"` // how far ahead the agenda goes
	tlsOption
}

type calendarSource struct {
	Name string `json:"name"`
	Kind string `json:"kind"` // sonarr, radarr or ical
}

type calendarEvent struct {
	Title  string `json:"title"`
	Start  string `json:"start"` // RFC 3339, or YYYY-MM-DD when all day
	AllDay bool   `json:"all_day"`
	Source int    `json:"source"` // index in sources
	at     time.Time
}

type calendarPayload struct {
	// From is the first day covered: the 1st of this month, for the grid
	From    string           `json:"from"`
	Sources []calendarSource `json:"sources"`
	Events  []calendarEvent  `json:"events"`
	// Failed lists sources that could not be loaded while others could
	Failed []string `json:"failed,omitempty"`
}

// calendar shows what Sonarr and Radarr bring next, and iCal feeds
type calendar struct{}

func init() { Register(calendar{}) }

func (calendar) Type() string { return "calendar" }

func (calendar) Meta() Meta {
	return Meta{
		Name:                   "Calendar",
		Category:               CategoryInfo,
		DefaultSize:            "2x2",
		AllowedSizes:           []string{"2x1", "1x2", "2x2"},
		ConfigIntegrationKinds: []string{"sonarr", "radarr"},
		MinRefreshSeconds:      calendarMinRefreshSeconds,
	}
}

func (calendar) Validate(config json.RawMessage) (json.RawMessage, error) {
	cfg, err := decodeConfig[calendarConfig](config)
	if err != nil {
		return nil, err
	}

	ids := []string{}
	for _, id := range cfg.Integrations {
		if id = strings.TrimSpace(id); id != "" && !slices.Contains(ids, id) {
			if _, err := uuid.Parse(id); err != nil {
				return nil, errors.New("integration not found")
			}
			ids = append(ids, id)
		}
	}
	if len(ids) > maxCalendarIntegrations {
		return nil, fmt.Errorf("at most %d integrations are allowed", maxCalendarIntegrations)
	}

	// Count first: checking a URL looks up its host
	urls := []string{}
	for _, u := range cfg.ICalURLs {
		if u = strings.TrimSpace(u); u != "" {
			urls = append(urls, u)
		}
	}
	if len(urls) > maxCalendarFeeds {
		return nil, fmt.Errorf("at most %d calendar feeds are allowed", maxCalendarFeeds)
	}
	for i, u := range urls {
		if strings.HasSuffix(u, "/"+redactedHeaderValue) {
			return nil, fmt.Errorf("enter calendar URL %d again", i+1)
		}
		if err := utils.ValidateWebhookURL(u); err != nil {
			return nil, fmt.Errorf("invalid calendar URL %q: %s", u, err.Error())
		}
	}
	if len(ids) == 0 && len(urls) == 0 {
		return nil, errors.New("pick an integration or add a calendar URL")
	}

	if cfg.Days == 0 {
		cfg.Days = defaultCalendarDays
	}
	if cfg.Days < 1 || cfg.Days > maxCalendarDays {
		return nil, fmt.Errorf("days must be between 1 and %d", maxCalendarDays)
	}
	cfg.Integrations, cfg.ICalURLs = ids, urls
	cfg.tlsOption.normalise()
	return encodeConfig(cfg)
}

func (calendar) IntegrationIDs(config json.RawMessage) []string {
	cfg, err := decodeConfig[calendarConfig](config)
	if err != nil {
		return nil
	}
	return cfg.Integrations
}

func (calendar) SetIntegrationIDs(config json.RawMessage, ids []string) (json.RawMessage, error) {
	cfg, err := decodeConfig[calendarConfig](config)
	if err != nil {
		return nil, err
	}
	cfg.Integrations = ids
	return encodeConfig(cfg)
}

// maskFeedURL keeps the host of a feed URL and hides the rest: secret
// calendar addresses carry their key in the path or query
func maskFeedURL(feedURL string) string {
	u, err := url.Parse(feedURL)
	if err != nil || u.Host == "" {
		return redactedHeaderValue
	}
	return u.Scheme + "://" + u.Host + "/" + redactedHeaderValue
}

// Redact masks the feed URLs, which are often secret addresses
func (calendar) Redact(config json.RawMessage) json.RawMessage {
	cfg, err := decodeConfig[calendarConfig](config)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	for i, feedURL := range cfg.ICalURLs {
		cfg.ICalURLs[i] = maskFeedURL(feedURL)
	}
	out, err := encodeConfig(cfg)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return out
}

// Unredact puts back each masked URL that was sent unchanged: the first
// stored URL with the same mask that isn't taken yet
func (calendar) Unredact(config, stored json.RawMessage) json.RawMessage {
	cfg, err := decodeConfig[calendarConfig](config)
	if err != nil {
		return config
	}
	old, err := decodeConfig[calendarConfig](stored)
	if err != nil {
		return config
	}
	taken := make([]bool, len(old.ICalURLs))
	for i, feedURL := range cfg.ICalURLs {
		if !strings.HasSuffix(feedURL, redactedHeaderValue) {
			continue
		}
		for j, oldURL := range old.ICalURLs {
			if !taken[j] && maskFeedURL(oldURL) == strings.TrimSpace(feedURL) {
				cfg.ICalURLs[i], taken[j] = oldURL, true
				break
			}
		}
	}
	out, err := encodeConfig(cfg)
	if err != nil {
		return config
	}
	return out
}

// calendarNow is a variable so tests can fix the date
var calendarNow = time.Now

func (calendar) Fetch(ctx context.Context, req *FetchRequest) (any, error) {
	cfg, err := decodeConfig[calendarConfig](req.Config)
	if err != nil {
		return nil, errors.New("widget has no valid config")
	}

	// From the 1st of this month (the grid) to the end of it, or further
	// when the agenda reaches beyond. In UTC dates, like all-day events, so
	// those on the 1st stay in whatever zone the server is in. Events start
	// two days earlier: a browser up to 26 hours behind the server can still
	// be in the month before, and its agenda starts today.
	now := calendarNow()
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	eventsFrom := from.AddDate(0, 0, -2)
	to := from.AddDate(0, 1, 0)
	if agendaEnd := now.AddDate(0, 0, cfg.Days+1); agendaEnd.After(to) {
		to = agendaEnd
	}

	type result struct {
		source calendarSource
		events []integrations.CalendarEvent
		err    error
	}
	results := make([]result, len(req.Linked)+len(cfg.ICalURLs))
	var wg sync.WaitGroup
	for i, linked := range req.Linked {
		results[i].source = calendarSource{Name: linked.Name, Kind: linked.Kind}
		wg.Go(func() {
			defer recoverInto(&results[i].err)
			results[i].events, results[i].err = integrations.ArrCalendar(ctx, linked.Conn, linked.Kind, eventsFrom, to)
		})
	}
	for j, feedURL := range cfg.ICalURLs {
		i := len(req.Linked) + j
		results[i].source = calendarSource{Name: hostOf(feedURL), Kind: "ical"}
		wg.Go(func() {
			defer recoverInto(&results[i].err)
			results[i].events, results[i].err = fetchICal(ctx, req.Client, feedURL)
		})
	}
	wg.Wait()

	payload := calendarPayload{From: from.Format(time.DateOnly), Sources: []calendarSource{}, Events: []calendarEvent{}, Failed: req.Unlinked}
	for _, r := range results {
		if r.err != nil {
			payload.Failed = append(payload.Failed, fmt.Sprintf("%s: %s", r.source.Name, r.err))
			continue
		}
		payload.Sources = append(payload.Sources, r.source)
		for _, e := range r.events {
			if e.Start.Before(eventsFrom) || !e.Start.Before(to) {
				continue
			}
			event := calendarEvent{Title: shortText(e.Title), AllDay: e.AllDay, Source: len(payload.Sources) - 1, at: e.Start}
			if e.AllDay {
				event.Start = e.Start.Format(time.DateOnly)
			} else {
				event.Start = e.Start.UTC().Format(time.RFC3339)
			}
			payload.Events = append(payload.Events, event)
		}
	}
	if len(payload.Sources) == 0 {
		return nil, errors.New(strings.Join(payload.Failed, "; "))
	}
	slices.SortStableFunc(payload.Events, func(a, b calendarEvent) int { return cmp.Compare(a.at.Unix(), b.at.Unix()) })
	return payload, nil
}

// fetchICal loads one iCalendar feed
func fetchICal(ctx context.Context, client *http.Client, feedURL string) ([]integrations.CalendarEvent, error) {
	body, err := getBodyLimit(ctx, client, feedURL, http.Header{"Accept": {"text/calendar, */*;q=0.8"}}, maxICalBytes)
	if err != nil {
		return nil, err
	}
	parsed, err := parseICal(body)
	if err != nil {
		return nil, err
	}
	events := make([]integrations.CalendarEvent, len(parsed))
	for i, e := range parsed {
		events[i] = integrations.CalendarEvent{Title: e.Summary, Start: e.Start, AllDay: e.AllDay}
	}
	return events, nil
}
