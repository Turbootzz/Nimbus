package widgets

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRSSValidate(t *testing.T) {
	runConfigCases(t, "rss", []configCase{
		{"defaults", `{"feeds":[" http://10.0.0.2/feed ",""]}`, `{"feeds":["http://10.0.0.2/feed"],"limit":10,"verify_tls":true}`, ""},
		{"limit, no TLS check", `{"feeds":["http://10.0.0.2/a","http://10.0.0.2/b"],"limit":5,"verify_tls":false}`,
			`{"feeds":["http://10.0.0.2/a","http://10.0.0.2/b"],"limit":5,"verify_tls":false}`, ""},
		{"no feeds", `{"feeds":[" "]}`, "", "add 1 to 3 feeds"},
		{"too many feeds", `{"feeds":["http://a.test","http://b.test","http://c.test","http://d.test"]}`, "", "add 1 to 3 feeds"},
		{"not http", `{"feeds":["javascript:alert(1)"]}`, "", "invalid feed URL"},
		{"limit too high", `{"feeds":["http://10.0.0.2/a"],"limit":51}`, "", "between 1 and 50"},
		{"negative limit", `{"feeds":["http://10.0.0.2/a"],"limit":-1}`, "", "between 1 and 50"},
	})
}

const rss2Fixture = `<?xml version="1.0"?>
<rss version="2.0" xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd" xmlns:atom="http://www.w3.org/2005/Atom"><channel>
	<title>Homelab   News</title>
	<itunes:title>Podcast name</itunes:title>
	<image><title>logo</title></image>
	<item>
		<title>Older &amp; wiser</title><itunes:title></itunes:title>
		<link>https://news.test/older</link><atom:link href="https://news.test/feed" rel="self"/>
		<pubDate>Mon, 28 Sep 2026 08:00:00 +0000</pubDate>
	</item>
	<item><title>Relative</title><link>/posts/relative</link><pubDate>Tue, 29 Sep 2026 04:00:00 EDT</pubDate></item>
	<item><title>Sneaky</title><link>javascript:alert(1)</link></item>
</channel></rss>`

const atomFixture = `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
	<title>Release notes</title>
	<entry>
		<title>v2.0</title>
		<link rel="edit" href="https://atom.test/edit/2"/>
		<link href="https://atom.test/v2"/>
		<updated>2026-09-30T10:00:00Z</updated>
	</entry>
</feed>`

const rdfFixture = `<?xml version="1.0"?>
<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns="http://purl.org/rss/1.0/" xmlns:dc="http://purl.org/dc/elements/1.1/">
	<channel><title>RDF feed</title></channel>
	<item><title>From RDF</title><link>https://rdf.test/1</link><dc:date>2026-09-27T12:00:00+02:00</dc:date></item>
</rdf:RDF>`

const jsonFeedFixture = "\xef\xbb\xbf" + `{
	"version": "https://jsonfeed.org/version/1.1",
	"title": "Micro",
	"items": [{"id": "1", "content_text": "A post without a title", "url": "https://micro.test/1", "date_published": "2026-10-01T09:00:00Z"}]
}`

// latin1Fixture is "Café" in ISO-8859-1
var latin1Fixture = "<?xml version=\"1.0\" encoding=\"ISO-8859-1\"?><rss><channel><title>Caf\xe9</title><item><title>Caf\xe9 &nbsp;open</title></item></channel></rss>"

func feedServer(t *testing.T, feeds map[string]string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := feeds[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server
}

func fetchRSS(t *testing.T, config string) (rssPayload, error) {
	t.Helper()
	w, _ := Get("rss")
	got, err := w.(Fetcher).Fetch(context.Background(), &FetchRequest{Config: json.RawMessage(config), Client: http.DefaultClient})
	if err != nil {
		return rssPayload{}, err
	}
	return got.(rssPayload), nil
}

func titles(items []rssItem) []string {
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = item.Title
	}
	return out
}

func TestRSSFetch(t *testing.T) {
	server := feedServer(t, map[string]string{"/rss": rss2Fixture, "/atom": atomFixture, "/rdf": rdfFixture})
	got, err := fetchRSS(t, `{"feeds":["`+server.URL+`/rss","`+server.URL+`/atom","`+server.URL+`/rdf"],"limit":10}`)
	require.NoError(t, err)

	assert.Equal(t, []string{"v2.0", "Relative", "Older & wiser", "From RDF", "Sneaky"}, titles(got.Items), "newest first, undated last")
	assert.Empty(t, got.Failed)

	byTitle := map[string]rssItem{}
	for _, item := range got.Items {
		byTitle[item.Title] = item
	}
	assert.Equal(t, "https://atom.test/v2", byTitle["v2.0"].Link, "alternate link, not edit")
	assert.Equal(t, "Release notes", byTitle["v2.0"].Source)
	assert.Equal(t, server.URL+"/posts/relative", byTitle["Relative"].Link)
	assert.Equal(t, "Homelab News", byTitle["Relative"].Source)
	assert.Equal(t, "2026-09-29T08:00:00Z", byTitle["Relative"].Date)
	assert.Equal(t, "2026-09-27T10:00:00Z", byTitle["From RDF"].Date)
	assert.Empty(t, byTitle["Sneaky"].Link, "javascript: links are dropped")
	assert.Empty(t, byTitle["Sneaky"].Date)

	got, err = fetchRSS(t, `{"feeds":["`+server.URL+`/rss"],"limit":1}`)
	require.NoError(t, err)
	assert.Equal(t, []string{"Relative"}, titles(got.Items))

	// Credentials in the feed URL stay out of the item links
	withUser := strings.Replace(server.URL, "http://", "http://user:pass@", 1)
	got, err = fetchRSS(t, `{"feeds":["`+withUser+`/rss"],"limit":1}`)
	require.NoError(t, err)
	assert.Equal(t, server.URL+"/posts/relative", got.Items[0].Link)
}

func TestParseFeedDate(t *testing.T) {
	cases := map[string]string{
		"Tue, 29 Sep 2026 08:00:00 +0000": "2026-09-29T08:00:00Z",
		"Tue, 29 Sep 2026 08:00:00 GMT":   "2026-09-29T08:00:00Z",
		"Tue, 29 Sep 2026 08:00:00 EST":   "2026-09-29T13:00:00Z",
		"Tue, 29 Sep 2026 08:00:00 cest":  "2026-09-29T06:00:00Z",
		"29 Sep 2026 08:00:00 GMT":        "2026-09-29T08:00:00Z",
		"Tue, 29 Sep 2026 08:00 +0000":    "2026-09-29T08:00:00Z",
		"Tue, 9 Sep 26 08:00:00 +0200":    "2026-09-09T06:00:00Z",
		"2026-09-29T08:00:00.123+02:00":   "2026-09-29T06:00:00Z",
	}
	for in, want := range cases {
		got, ok := parseFeedDate(in)
		require.True(t, ok, in)
		assert.Equal(t, want, got.UTC().Truncate(time.Second).Format(time.RFC3339), in)
	}
	for _, in := range []string{"", "yesterday", "Tue, 29 Sep 2026 08:00:00 XYZ"} {
		_, ok := parseFeedDate(in)
		assert.False(t, ok, in)
	}
}

func TestRSSFetchFormats(t *testing.T) {
	server := feedServer(t, map[string]string{"/json": jsonFeedFixture, "/latin1": latin1Fixture})

	got, err := fetchRSS(t, `{"feeds":["`+server.URL+`/json"],"limit":10}`)
	require.NoError(t, err)
	require.Len(t, got.Items, 1)
	assert.Equal(t, rssItem{Title: "A post without a title", Link: "https://micro.test/1", Date: "2026-10-01T09:00:00Z", Source: "Micro"},
		rssItem{Title: got.Items[0].Title, Link: got.Items[0].Link, Date: got.Items[0].Date, Source: got.Items[0].Source})

	got, err = fetchRSS(t, `{"feeds":["`+server.URL+`/latin1"],"limit":10}`)
	require.NoError(t, err)
	require.Len(t, got.Items, 1)
	assert.Equal(t, "Café open", got.Items[0].Title)
	assert.Equal(t, "Café", got.Items[0].Source)
}

func TestRSSFetchFailures(t *testing.T) {
	server := feedServer(t, map[string]string{"/rss": rss2Fixture, "/html": "<html><body>hi</body></html>", "/json": `{"items":[]}`})

	got, err := fetchRSS(t, `{"feeds":["`+server.URL+`/rss","`+server.URL+`/missing"],"limit":10}`)
	require.NoError(t, err, "one working feed is enough")
	assert.Len(t, got.Items, 3)
	require.Len(t, got.Failed, 1)
	assert.Contains(t, got.Failed[0], "unexpected status 404")

	for _, path := range []string{"/html", "/json", "/missing"} {
		_, err = fetchRSS(t, `{"feeds":["`+server.URL+path+`"],"limit":10}`)
		require.Error(t, err, path)
	}
	_, err = fetchRSS(t, `{"feeds":["`+server.URL+`/html"],"limit":10}`)
	assert.Contains(t, err.Error(), "not an RSS, Atom or JSON feed")
}
