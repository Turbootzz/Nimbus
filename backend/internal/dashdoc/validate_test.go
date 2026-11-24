package dashdoc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validDocument() *Document {
	return &Document{
		Version:      Version,
		Groups:       []Group{{Ref: "media", Name: "Media", Default: true}, {Name: "No ref"}},
		Integrations: []Integration{{Ref: "sonarr", Kind: "sonarr", Name: "Sonarr", BaseURL: "http://sonarr.lan"}},
		Services:     []Service{{Ref: "sonarr", Name: "Sonarr", URL: "http://sonarr.lan", GroupRef: "media", IntegrationRef: "sonarr"}},
		Widgets:      []Widget{{Ref: "queue", Type: "markdown", Position: 1, GroupRef: "media"}},
		Preferences: &Preferences{StatusStrip: &StatusStrip{Enabled: true, Chips: []Chip{
			{Source: ChipSourceIntegration, Ref: "sonarr", KPI: "queue"},
			{Source: ChipSourceWidget, Ref: "queue", KPI: "Temp"},
		}}},
	}
}

func TestValidateAcceptsAValidDocument(t *testing.T) {
	// The same ref may name a service and an integration: refs are per kind
	assert.NoError(t, Validate(validDocument()))
	assert.NoError(t, Validate(&Document{Version: Version}))
}

// Groups and services have no per-user limit, so whatever an import can hold
// is valid: 10 MiB of the smallest exported groups (49 bytes) and services (114)
func TestValidateTakesWhatFitsInAnImport(t *testing.T) {
	const size = 10 * 1024 * 1024
	assert.NoError(t, Validate(&Document{Version: Version, Groups: make([]Group, size/49), Services: make([]Service, size/114)}))
}

func TestValidateReportsWhereTheProblemIs(t *testing.T) {
	tests := []struct {
		name   string
		change func(*Document)
		paths  []string
	}{
		{"future version", func(d *Document) { d.Version = 2 }, []string{"version"}},
		{"no version", func(d *Document) { d.Version = 0 }, []string{"version"}},
		{"duplicate ref", func(d *Document) { d.Groups[1].Ref = "media" }, []string{"groups[1].ref"}},
		{"long ref", func(d *Document) { d.Widgets[0].Ref = string(make([]byte, MaxRefLen+1)) }, []string{"widgets[0].ref", "preferences.status_strip.chips[1].ref"}},
		{"two defaults", func(d *Document) { d.Groups[1].Default = true }, []string{"groups[1].default"}},
		{"dangling group", func(d *Document) { d.Widgets[0].GroupRef = "nope" }, []string{"widgets[0].group_ref"}},
		{"dangling integration", func(d *Document) { d.Services[0].IntegrationRef = "nope" }, []string{"services[0].integration_ref"}},
		{"negative position", func(d *Document) { d.Services[0].Position = -1 }, []string{"services[0].position"}},
		{"huge position", func(d *Document) { d.Widgets[0].Position = MaxPosition + 1 }, []string{"widgets[0].position"}},
		{"chip source", func(d *Document) { d.Preferences.StatusStrip.Chips[0].Source = "service" }, []string{"preferences.status_strip.chips[0].source"}},
		{"chip of the wrong kind", func(d *Document) { d.Preferences.StatusStrip.Chips[0].Ref = "queue" }, []string{"preferences.status_strip.chips[0].ref"}},
		{"chip without ref", func(d *Document) { d.Preferences.StatusStrip.Chips[1].Ref = "" }, []string{"preferences.status_strip.chips[1].ref"}},
		{"background twice", func(d *Document) {
			url := "https://example.com/a.png"
			d.Preferences.ThemeBackground, d.Preferences.WallpaperData = &url, "aGk="
		}, []string{"preferences.wallpaper_data"}},
		{"too many widgets", func(d *Document) { d.Widgets = make([]Widget, MaxWidgets+1) }, []string{"widgets"}},
		{"several problems", func(d *Document) {
			d.Services[0].GroupRef, d.Widgets[0].Position = "nope", -5
		}, []string{"services[0].group_ref", "widgets[0].position"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := validDocument()
			tt.change(doc)

			err := Validate(doc)

			var invalid *ValidationError
			require.ErrorAs(t, err, &invalid)
			paths := make([]string, len(invalid.Issues))
			for i, issue := range invalid.Issues {
				paths[i] = issue.Path
				assert.NotEmpty(t, issue.Message)
			}
			assert.Equal(t, tt.paths, paths)
			assert.Contains(t, err.Error(), tt.paths[0])
		})
	}
}

func TestParseMode(t *testing.T) {
	for _, mode := range []Mode{ModeMerge, ModeReplace} {
		parsed, ok := ParseMode(string(mode))
		assert.True(t, ok)
		assert.Equal(t, mode, parsed)
	}
	_, ok := ParseMode("")
	assert.False(t, ok)
	_, ok = ParseMode("REPLACE")
	assert.False(t, ok)
}
