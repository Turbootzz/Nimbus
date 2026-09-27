package widgets

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type configCase struct {
	name    string
	config  string
	want    string // normalised config; empty when an error is expected
	wantErr string
}

func runConfigCases(t *testing.T, typ string, cases []configCase) {
	t.Helper()
	w, ok := Get(typ)
	require.True(t, ok)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := w.Validate(json.RawMessage(tc.config))
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.JSONEq(t, tc.want, string(got))
		})
	}
}

func TestNotAnObject(t *testing.T) {
	for _, meta := range Types() {
		w, _ := Get(meta.Type)
		for _, config := range []string{``, `null`, `[]`, `"x"`, `{bad`} {
			_, err := w.Validate(json.RawMessage(config))
			assert.Error(t, err, "%s with %q", meta.Type, config)
		}
	}
}

func TestClock(t *testing.T) {
	runConfigCases(t, "clock", []configCase{
		{"defaults", `{}`, `{"timezone":"","hour12":false,"show_seconds":false,"date_format":"short"}`, ""},
		{"full", ` {"timezone":" Europe/Amsterdam ","hour12":true,"show_seconds":true,"date_format":"long","extra":1}`,
			`{"timezone":"Europe/Amsterdam","hour12":true,"show_seconds":true,"date_format":"long"}`, ""},
		{"unknown timezone", `{"timezone":"Mars/Olympus"}`, "", "unknown timezone"},
		{"server local zone", `{"timezone":"Local"}`, "", "unknown timezone"},
		{"path in timezone", `{"timezone":"../../etc/passwd"}`, "", "unknown timezone"},
		{"bad date format", `{"date_format":"iso"}`, "", "date format"},
	})
}

func TestMarkdown(t *testing.T) {
	runConfigCases(t, "markdown", []configCase{
		{"empty note", `{}`, `{"content":""}`, ""},
		{"content", `{"content":"# Hi\n- item"}`, `{"content":"# Hi\n- item"}`, ""},
		{"multibyte at the limit", `{"content":"` + strings.Repeat("é", maxMarkdownRunes) + `"}`,
			`{"content":"` + strings.Repeat("é", maxMarkdownRunes) + `"}`, ""},
		{"too long", `{"content":"` + strings.Repeat("a", maxMarkdownRunes+1) + `"}`, "", "10000 characters"},
	})
}

func TestBookmarks(t *testing.T) {
	many := make([]string, maxBookmarks+1)
	for i := range many {
		many[i] = `{"name":"x","url":"https://x.test"}`
	}
	runConfigCases(t, "bookmarks", []configCase{
		{"empty list", `{}`, `{"items":[]}`, ""},
		{"trimmed", `{"items":[{"name":" Router ","url":" http://192.168.1.1 ","icon":" 📡 "}]}`,
			`{"items":[{"name":"Router","url":"http://192.168.1.1","icon":"📡"}]}`, ""},
		{"image icon", `{"items":[{"name":"A","url":"https://a.test","icon":"https://a.test/i.png"}]}`,
			`{"items":[{"name":"A","url":"https://a.test","icon":"https://a.test/i.png"}]}`, ""},
		{"missing name", `{"items":[{"url":"https://a.test"}]}`, "", "bookmark 1 needs a name"},
		{"javascript url", `{"items":[{"name":"x","url":"javascript:alert(1)"}]}`, "", "bookmark 1: invalid URL"},
		{"relative url", `{"items":[{"name":"x","url":"/admin"}]}`, "", "invalid URL"},
		{"icon too long", `{"items":[{"name":"x","url":"https://a.test","icon":"` + strings.Repeat("a", 33) + `"}]}`, "", "emoji or an image URL"},
		{"too many", `{"items":[` + strings.Join(many, ",") + `]}`, "", "at most 50"},
	})
}

func TestIframe(t *testing.T) {
	runConfigCases(t, "iframe", []configCase{
		{"default height", `{"url":"http://grafana.lan:3000/d/abc"}`, `{"url":"http://grafana.lan:3000/d/abc","height":300}`, ""},
		{"custom height", `{"url":"https://a.test","height":600}`, `{"url":"https://a.test","height":600}`, ""},
		{"missing url", `{}`, "", "invalid URL"},
		{"javascript url", `{"url":"javascript:alert(1)"}`, "", "invalid URL"},
		{"data url", `{"url":"data:text/html,<script>alert(1)</script>"}`, "", "invalid URL"},
		{"too low", `{"url":"https://a.test","height":50}`, "", "height must be between"},
		{"too high", `{"url":"https://a.test","height":5000}`, "", "height must be between"},
	})
}
