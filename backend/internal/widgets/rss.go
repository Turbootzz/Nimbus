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
	tlsOption
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

	// Count first: checking a URL looks up its host
	feeds := []string{}
	for _, feed := range cfg.Feeds {
		if feed = strings.TrimSpace(feed); feed != "" {
			feeds = append(feeds, feed)
		}
	}
	if len(feeds) == 0 || len(feeds) > maxRSSFeeds {
		return nil, fmt.Errorf("add 1 to %d feeds", maxRSSFeeds)
	}
	for _, feed := range feeds {
		if err := utils.ValidateWebhookURL(feed); err != nil {
			return nil, fmt.Errorf("invalid feed URL %q: %s", feed, err.Error())
		}
	}
	cfg.Feeds = feeds
	cfg.tlsOption.normalise()

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
		wg.Go(func() {
			defer recoverInto(&errs[i])
			items[i], errs[i] = fetchFeed(ctx, req.Client, feed)
		})
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
	if bytes.HasPrefix(bytes.TrimLeft(body, " \t\r\n"), []byte("{")) {
		return parseJSONFeed(body, base)
	}
	return parseXMLFeed(body, base)
}

// xmlFeed covers RSS 2.0, RSS 1.0 (RDF) and Atom
type xmlFeed struct {
	XMLName xml.Name
	Channel struct {
		Titles []xmlText  `xml:"title"`
		Items  []xmlEntry `xml:"item"`
	} `xml:"channel"`
	// RSS 1.0 keeps the items next to the channel
	Items []xmlEntry `xml:"item"`
	// Atom
	Titles  []xmlText  `xml:"title"`
	Entries []xmlEntry `xml:"entry"`
}

// xmlEntry is an RSS item or an Atom entry. The fields are lists because
// extensions add elements with the same local name, like itunes:title and
// atom:link, and Go matches those too.
type xmlEntry struct {
	Titles    []xmlText `xml:"title"`
	Links     []xmlLink `xml:"link"`
	PubDate   string    `xml:"pubDate"`
	DCDate    string    `xml:"http://purl.org/dc/elements/1.1/ date"`
	Published string    `xml:"published"`
	Updated   string    `xml:"updated"`
}

type xmlText struct {
	XMLName xml.Name
	Text    string `xml:",chardata"`
}

// xmlLink is an RSS link (text) or an Atom link (href)
type xmlLink struct {
	XMLName xml.Name
	Text    string `xml:",chardata"`
	Href    string `xml:"href,attr"`
	Rel     string `xml:"rel,attr"`
}

// Namespaces of the feed formats themselves; RSS 2.0 has none
var feedNamespaces = map[string]bool{
	"":                                 true,
	"http://purl.org/rss/1.0/":         true,
	"http://www.w3.org/2005/Atom":      true,
	"http://backend.userland.com/rss2": true,
}

// title returns the first title of the feed format itself
func title(titles []xmlText) string {
	for _, t := range titles {
		if feedNamespaces[t.XMLName.Space] {
			return t.Text
		}
	}
	return ""
}

// link returns an RSS text link, or else an Atom alternate link
func (e xmlEntry) link() string {
	for _, l := range e.Links {
		if feedNamespaces[l.XMLName.Space] && strings.TrimSpace(l.Text) != "" {
			return l.Text
		}
	}
	for _, l := range e.Links {
		if l.Href != "" && (l.Rel == "" || l.Rel == "alternate") {
			return l.Href
		}
	}
	return ""
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

	var source string
	var entries []xmlEntry
	switch feed.XMLName.Local {
	case "rss", "RDF":
		source, entries = title(feed.Channel.Titles), append(feed.Channel.Items, feed.Items...)
	case "feed":
		source, entries = title(feed.Titles), feed.Entries
	default:
		return nil, errors.New("not an RSS, Atom or JSON feed")
	}

	source = feedSource(source, base)
	items := make([]rssItem, 0, len(entries))
	for _, e := range entries {
		date := firstNonEmpty(e.PubDate, e.DCDate, e.Published, e.Updated)
		items = append(items, newRSSItem(title(e.Titles), resolveLink(base, e.link()), date, source))
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

// rfc822Zones are the named zones RFC 822 allows, plus the European ones
// that feeds use. time.Parse would read an unknown name as UTC.
var rfc822Zones = map[string]string{
	"UT": "+0000", "UTC": "+0000", "GMT": "+0000", "Z": "+0000",
	"EST": "-0500", "EDT": "-0400", "CST": "-0600", "CDT": "-0500",
	"MST": "-0700", "MDT": "-0600", "PST": "-0800", "PDT": "-0700",
	"CET": "+0100", "CEST": "+0200",
}

// parseFeedDate reads RFC 3339 (Atom, JSON Feed) and RFC 822 (RSS) dates.
// RFC 822 dates may leave out the weekday and the seconds.
func parseFeedDate(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, true
	}
	if _, rest, ok := strings.Cut(s, ", "); ok {
		s = rest
	}
	if i := strings.LastIndexByte(s, ' '); i > 0 {
		if offset, ok := rfc822Zones[strings.ToUpper(s[i+1:])]; ok {
			s = s[:i+1] + offset
		}
	}
	for _, layout := range []string{"2 Jan 2006 15:04:05 -0700", "2 Jan 2006 15:04 -0700", "2 Jan 06 15:04:05 -0700"} {
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
	// A relative link would copy credentials in the feed URL
	u.User = nil
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
