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
		},
		"content": "[NAS](http://root:hunter2@nas.lan) and ftp://me@files.lan, mail me@example.com, see https://host/path@x",
		"n":       1,
	}
	assert.Equal(t, map[string]any{
		"url": "http://nas.lan:5000/x?q=1#top",
		"items": []any{
			map[string]any{"url": "https://host/?a=b", "icon": "https://icons.lan/a.png"},
			map[string]any{"url": "http://host?mail=a@b.c"},
		},
		"content": "[NAS](http://nas.lan) and ftp://files.lan, mail me@example.com, see https://host/path@x",
		"n":       1,
	}, StripURLUserInfo(config))
}
