package widgets

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/nimbus/backend/internal/utils"
	"golang.org/x/net/html/charset"
)

const (
	maxRSSFeeds          = 3
	defaultRSSLimit      = 10
	maxRSSLimit          = 50
	rssMinRefreshSeconds = 300
	maxRSSTitleRunes     = 300
)

type rssConfig struct {
	Feeds []string `json:"feeds"`
	Limit int      `json:"limit"` // items shown, newest first
}

type rssItem struct {
	Title     string    `json:"title"`
	Link      string    `json:"link,omitempty"`
	Date      string    `json:"date,omitempty"` // RFC 3339; empty when the feed has none
	Source    string    `json:"source"`         // feed title
	published time.Time // for sorting
}

type rssPayload struct {
	Items []rssItem `json:"items"`
	// Failed lists the feeds that could not be loaded while others could
	Failed []string `json:"failed,omitempty"`
}

type rss struct{}

func init() { Register(rss{}) }

func (rss) Type() string { return "rss" }

func (rss) Meta() Meta {
	return Meta{
		Name:              "RSS",
		Category:          CategoryInfo,
		DefaultSize:       "2x2",
		AllowedSizes:      allSizes,
		MinRefreshSeconds: rssMinRefreshSeconds,
	}
}

func (rss) Validate(config json.RawMessage) (json.RawMessage, error) {
	cfg, err := decodeConfig[rssConfig](config)
	if err != nil {
		return nil, err
	}

	feeds := []string{}
	for _, feed := range cfg.Feeds {
		if feed = strings.TrimSpace(feed); feed == "" {
			continue
		}
		if err := utils.ValidateWebhookURL(feed); err != nil {
			return nil, fmt.Errorf("invalid feed URL %q: %s", feed, err.Error())
		}
		feeds = append(feeds, feed)
	}
	if len(feeds) == 0 || len(feeds) > maxRSSFeeds {
		return nil, fmt.Errorf("add 1 to %d feeds", maxRSSFeeds)
	}
	cfg.Feeds = feeds

	if cfg.Limit == 0 {
		cfg.Limit = defaultRSSLimit
	}
	if cfg.Limit < 1 || cfg.Limit > maxRSSLimit {
		return nil, fmt.Errorf("number of items must be between 1 and %d", maxRSSLimit)
	}
	return encodeConfig(cfg)
}

func (rss) Fetch(ctx context.Context, req *FetchRequest) (any, error) {
	cfg, err := decodeConfig[rssConfig](req.Config)
	if err != nil || len(cfg.Feeds) == 0 {
		return nil, errors.New("widget has no feeds")
	}

	items := make([][]rssItem, len(cfg.Feeds))
	errs := make([]error, len(cfg.Feeds))
	var wg sync.WaitGroup
	for i, feed := range cfg.Feeds {
		wg.Go(func() { items[i], errs[i] = fetchFeed(ctx, req.Client, feed) })
	}
	wg.Wait()

	payload := rssPayload{Items: []rssItem{}}
	for i, feed := range cfg.Feeds {
		if errs[i] != nil {
			payload.Failed = append(payload.Failed, fmt.Sprintf("%s: %s", hostOf(feed), errs[i]))
			continue
		}
		payload.Items = append(payload.Items, items[i]...)
	}
	if len(payload.Failed) == len(cfg.Feeds) {
		return nil, errors.New(payload.Failed[0])
	}

	// Newest first; items without a date go last in feed order
	slices.SortStableFunc(payload.Items, func(a, b rssItem) int { return b.published.Compare(a.published) })
	payload.Items = payload.Items[:min(len(payload.Items), cfg.Limit)]
	return payload, nil
}

// fetchFeed loads one RSS, Atom or JSON Feed
func fetchFeed(ctx context.Context, client *http.Client, feedURL string) ([]rssItem, error) {
	body, err := getBody(ctx, client, feedURL, http.Header{
		"Accept": {"application/rss+xml, application/atom+xml, application/feed+json, application/xml;q=0.9, */*;q=0.8"},
	})
	if err != nil {
		return nil, err
	}
	base, err := url.Parse(feedURL)
	if err != nil {
		return nil, err
	}
	if bytes.HasPrefix(bytes.TrimLeft(body, " \t\r\n\ufeff"), []byte("{")) {
		return parseJSONFeed(body, base)
	}
	return parseXMLFeed(body, base)
}

// xmlFeed covers RSS 2.0, RSS 1.0 (RDF) and Atom
type xmlFeed struct {
	XMLName xml.Name
	Channel struct {
		Title string    `xml:"title"`
		Items []xmlItem `xml:"item"`
	} `xml:"channel"`
	// RSS 1.0 keeps the items next to the channel
	Items []xmlItem `xml:"item"`
	// Atom
	Title   string      `xml:"title"`
	Entries []atomEntry `xml:"entry"`
}

type xmlItem struct {
	Title   string `xml:"title"`
	Link    string `xml:"link"`
	PubDate string `xml:"pubDate"`
	DCDate  string `xml:"http://purl.org/dc/elements/1.1/ date"`
}

type atomEntry struct {
	Title string `xml:"title"`
	Links []struct {
		Href string `xml:"href,attr"`
		Rel  string `xml:"rel,attr"`
	} `xml:"link"`
	Published string `xml:"published"`
	Updated   string `xml:"updated"`
}

func parseXMLFeed(body []byte, base *url.URL) ([]rssItem, error) {
	dec := xml.NewDecoder(bytes.NewReader(body))
	// Feeds in the wild use other charsets and HTML entities
	dec.CharsetReader = charset.NewReaderLabel
	dec.Strict = false
	dec.Entity = xml.HTMLEntity

	var feed xmlFeed
	if err := dec.Decode(&feed); err != nil {
		return nil, errors.New("not an RSS, Atom or JSON feed")
	}

	var items []rssItem
	switch feed.XMLName.Local {
	case "rss", "RDF":
		source := feedSource(feed.Channel.Title, base)
		for _, it := range append(feed.Channel.Items, feed.Items...) {
			items = append(items, newRSSItem(it.Title, resolveLink(base, it.Link), firstNonEmpty(it.PubDate, it.DCDate), source))
		}
	case "feed":
		source := feedSource(feed.Title, base)
		for _, entry := range feed.Entries {
			link := ""
			for _, l := range entry.Links {
				if l.Rel == "" || l.Rel == "alternate" {
					link = l.Href
					break
				}
			}
			items = append(items, newRSSItem(entry.Title, resolveLink(base, link), firstNonEmpty(entry.Published, entry.Updated), source))
		}
	default:
		return nil, errors.New("not an RSS, Atom or JSON feed")
	}
	return items, nil
}

type jsonFeed struct {
	Version string `json:"version"`
	Title   string `json:"title"`
	Items   []struct {
		Title         string `json:"title"`
		URL           string `json:"url"`
		ExternalURL   string `json:"external_url"`
		Summary       string `json:"summary"`
		ContentText   string `json:"content_text"`
		DatePublished string `json:"date_published"`
		DateModified  string `json:"date_modified"`
	} `json:"items"`
}

func parseJSONFeed(body []byte, base *url.URL) ([]rssItem, error) {
	var feed jsonFeed
	if err := json.Unmarshal(body, &feed); err != nil || !strings.Contains(feed.Version, "jsonfeed.org") {
		return nil, errors.New("not an RSS, Atom or JSON feed")
	}
	source := feedSource(feed.Title, base)
	items := make([]rssItem, 0, len(feed.Items))
	for _, it := range feed.Items {
		// Microblog items have no title, only text
		title := firstNonEmpty(it.Title, it.Summary, it.ContentText)
		link := resolveLink(base, firstNonEmpty(it.URL, it.ExternalURL))
		items = append(items, newRSSItem(title, link, firstNonEmpty(it.DatePublished, it.DateModified), source))
	}
	return items, nil
}

func newRSSItem(title, link, date, source string) rssItem {
	item := rssItem{Title: shortText(title), Link: link, Source: source}
	if t, ok := parseFeedDate(date); ok {
		item.published = t
		item.Date = t.UTC().Format(time.RFC3339)
	}
	return item
}

// feedDateLayouts are the date formats seen in feeds: RFC 822 variants for
// RSS and RFC 3339 for Atom and JSON Feed
var feedDateLayouts = []string{
	time.RFC1123Z,
	time.RFC1123,
	"Mon, 2 Jan 2006 15:04:05 -0700",
	"Mon, 2 Jan 2006 15:04:05 MST",
	"2 Jan 2006 15:04:05 -0700",
	time.RFC3339,
}

func parseFeedDate(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	for _, layout := range feedDateLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// resolveLink makes a link absolute and drops anything but http(s), so a
// feed can't put a javascript: link on the dashboard
func resolveLink(base *url.URL, link string) string {
	link = strings.TrimSpace(link)
	if link == "" {
		return ""
	}
	u, err := base.Parse(link)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return ""
	}
	return u.String()
}

func feedSource(title string, base *url.URL) string {
	if title = shortText(title); title != "" {
		return title
	}
	return base.Hostname()
}

// shortText collapses whitespace and cuts long text
func shortText(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if runes := []rune(s); len(runes) > maxRSSTitleRunes {
		return string(runes[:maxRSSTitleRunes]) + "…"
	}
	return s
}

func hostOf(rawURL string) string {
	if u, err := url.Parse(rawURL); err == nil && u.Host != "" {
		return u.Host
	}
	return rawURL
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
