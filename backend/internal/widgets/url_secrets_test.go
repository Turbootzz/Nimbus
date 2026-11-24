package widgets

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStripURLUserInfo(t *testing.T) {
	config := map[string]any{
		"url": "http://admin:pw@nas.lan:5000/x?q=1#top",
		"items": []any{
			map[string]any{"url": "https://user:p@ss@host/?a=b", "icon": "https://u:p@icons.lan/a.png"},
			map[string]any{"url": "http://host?mail=a@b.c"},
			map[string]any{"url": "//admin:pw@nas.lan/", "icon": "https://host/a//b@c"},
		},
		"content": "[NAS](http://root:hunter2@nas.lan) and ftp://me@files.lan, mail me@example.com, see https://host/path@x, [UI](//root:pw@ui.lan/x)",
		"n":       1,
	}
	assert.Equal(t, map[string]any{
		"url": "http://nas.lan:5000/x?q=1#top",
		"items": []any{
			map[string]any{"url": "https://host/?a=b", "icon": "https://icons.lan/a.png"},
			map[string]any{"url": "http://host?mail=a@b.c"},
			map[string]any{"url": "//nas.lan/", "icon": "https://host/a//b@c"},
		},
		"content": "[NAS](http://nas.lan) and ftp://files.lan, mail me@example.com, see https://host/path@x, [UI](//ui.lan/x)",
		"n":       1,
	}, StripURLUserInfo(config))
}

func TestStripURLSecrets(t *testing.T) {
	config := map[string]any{
		"url":      "https://u:pw@feeds.lan/rss?apikey=s3cret#top",
		"relative": "//u:pw@feeds.lan/rss?apikey=s3cret",
		"plain":    "https://feeds.lan/rss",
		// Passes a save that checks with url.ParseRequestURI, fails url.Parse
		"bad":    []any{"https://u:pw@feeds.lan/rss?apikey=s3cret#%zz", " https://u:pw@feeds.lan/?apikey=s3cret"},
		"note":   "100% done? See [NAS](http://root:pw@nas.lan/?tab=1)",
		"number": 2,
	}
	assert.Equal(t, map[string]any{
		"url":      "https://feeds.lan/rss",
		"relative": "//feeds.lan/rss",
		"plain":    "https://feeds.lan/rss",
		"bad":      []any{"", ""},
		"note":     "100% done? See [NAS](http://nas.lan/?tab=1)",
		"number":   2,
	}, StripURLSecrets(config))
}
