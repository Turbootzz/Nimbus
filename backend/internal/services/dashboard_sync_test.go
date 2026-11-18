package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/nimbus/backend/internal/configfile"
	"github.com/nimbus/backend/internal/dashdoc"
	"github.com/nimbus/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dump is everything the user's dashboard tables hold, timestamps included
func (e *transferEnv) dump(t *testing.T, userID string) string {
	t.Helper()
	out := ""
	for _, table := range []string{"groups", "integrations", "services", "widgets", "user_preferences"} {
		rows, err := e.db.Query(`SELECT * FROM `+table+` WHERE user_id = ? ORDER BY id`, userID)
		require.NoError(t, err)
		columns, err := rows.Columns()
		require.NoError(t, err)
		for rows.Next() {
			values := make([]any, len(columns))
			targets := make([]any, len(columns))
			for i := range values {
				targets[i] = &values[i]
			}
			require.NoError(t, rows.Scan(targets...))
			out += fmt.Sprintf("%s %v\n", table, values)
		}
		require.NoError(t, rows.Err())
		rows.Close()
	}
	return out
}

func (e *transferEnv) sync(t *testing.T, userID string, doc *dashdoc.Document, prune bool) *dashdoc.Summary {
	t.Helper()
	summary, err := e.svc.Sync(context.Background(), userID, doc, prune)
	require.NoError(t, err)
	return summary
}

// id returns the ID of the user's only row with that name (or title)
func (e *transferEnv) id(t *testing.T, table, column, value string) string {
	t.Helper()
	var id string
	require.NoError(t, e.db.QueryRow(`SELECT id FROM `+table+` WHERE user_id = ? AND `+column+` = ?`, transferAdmin, value).Scan(&id))
	return id
}

func (e *transferEnv) credentials(t *testing.T, id string) models.IntegrationCredentials {
	t.Helper()
	blob, err := e.svc.Integrations.GetCredentials(context.Background(), id, transferAdmin)
	require.NoError(t, err)
	creds, err := e.integrations.decryptCredentials(blob, id)
	require.NoError(t, err)
	return creds
}

func changed(c dashdoc.Count) int { return c.Created + c.Updated + c.Removed }

func assertNothingChanged(t *testing.T, summary *dashdoc.Summary) {
	t.Helper()
	assert.Zero(t, changed(summary.Groups)+changed(summary.Integrations)+changed(summary.Services)+changed(summary.Widgets), "%+v", summary)
	assert.False(t, summary.Preferences)
}

func TestSyncOfAnUnchangedDocumentWritesNothing(t *testing.T) {
	env := newTransferEnv(t)
	env.seedDashboard(t)
	doc := env.export(t, transferAdmin)

	// The export of a dashboard describes it: only the group order, which a
	// document has no positions for, is normalised
	first := env.sync(t, transferAdmin, reparse(t, doc), true)
	assert.Zero(t, first.Groups.Created+first.Groups.Removed+changed(first.Integrations)+changed(first.Services)+changed(first.Widgets), "%+v", first)
	assert.Zero(t, first.NeedsCredentials, "stored credentials are kept")
	assert.Zero(t, first.NeedsSecrets, "masked widget secrets keep their stored value")

	before := env.dump(t, transferAdmin)
	files := len(env.images.files)
	second := env.sync(t, transferAdmin, reparse(t, doc), true)
	assertNothingChanged(t, second)
	assert.Equal(t, before, env.dump(t, transferAdmin), "not a single column may change")
	assert.Len(t, env.images.files, files, "no image is stored again")
	assert.Equal(t, "sonarr-secret-key", env.credentials(t, env.id(t, "integrations", "name", "Sonarr")).APIKey)
}

func TestSyncCreatesUpdatesAndRemovesKeepingIDs(t *testing.T) {
	env := newTransferEnv(t)
	env.seedDashboard(t)
	doc := reparse(t, env.export(t, transferAdmin))
	env.sync(t, transferAdmin, doc, true)

	sonarrService := env.id(t, "services", "name", "Sonarr")
	plex := env.id(t, "services", "name", "Plex")
	sonarr := env.id(t, "integrations", "name", "Sonarr")
	notes := env.id(t, "widgets", "title", "Notes")
	media := env.id(t, "groups", "name", "Media")
	otherUsers := env.dump(t, transferUser)

	for i := range doc.Services {
		switch doc.Services[i].Name {
		case "Sonarr":
			doc.Services[i].Description = "TV"
		case "Plex":
			// Only its name still identifies it
			doc.Services[i].URL = "https://plex.example.org"
		}
	}
	// Docs goes, Grafana comes
	doc.Services = append(doc.Services[:2:2], dashdoc.Service{Name: "Grafana", URL: "https://grafana.example.com", Position: 9, GroupRef: "media"})
	require.Equal(t, "Sonarr", doc.Integrations[1].Name)
	doc.Integrations[1].Credentials = &dashdoc.Credentials{APIKey: "rotated-key"}
	doc.Integrations[1].RefreshSeconds = new(120)
	doc.Integrations = append(doc.Integrations, dashdoc.Integration{
		Ref: "radarr", Kind: "radarr", Name: "Radarr", BaseURL: "http://192.168.1.12:7878",
		Credentials: &dashdoc.Credentials{APIKey: "radarr-key"},
	})
	for i := range doc.Widgets {
		if doc.Widgets[i].Title == "Notes" {
			doc.Widgets[i].Config = json.RawMessage(`{"content":"changed"}`)
		}
	}
	doc.Groups[1].Color = "#00ff00"
	doc.Preferences.ThemeMode = new("light")

	summary := env.sync(t, transferAdmin, doc, true)
	assert.Equal(t, dashdoc.Count{Created: 1, Updated: 2, Removed: 1}, summary.Services)
	assert.Equal(t, dashdoc.Count{Created: 1, Updated: 1}, summary.Integrations)
	assert.Equal(t, dashdoc.Count{Updated: 1}, summary.Widgets)
	assert.Equal(t, dashdoc.Count{Updated: 1}, summary.Groups)
	assert.True(t, summary.Preferences)
	assert.Zero(t, summary.NeedsCredentials)

	// Same rows, so status history, snapshots and credentials stay attached
	assert.Equal(t, sonarrService, env.id(t, "services", "name", "Sonarr"))
	assert.Equal(t, plex, env.id(t, "services", "url", "https://plex.example.org"))
	assert.Equal(t, sonarr, env.id(t, "integrations", "name", "Sonarr"))
	assert.Equal(t, notes, env.id(t, "widgets", "title", "Notes"))
	assert.Equal(t, media, env.id(t, "groups", "color", "#00ff00"))
	assert.Equal(t, 3, env.count(t, "services", transferAdmin))

	var status, group string
	require.NoError(t, env.db.QueryRow(`SELECT status FROM services WHERE id = ?`, plex).Scan(&status))
	assert.Equal(t, models.StatusOnline, status, "an update keeps the status")
	require.NoError(t, env.db.QueryRow(`SELECT group_id FROM services WHERE name = 'Grafana'`).Scan(&group))
	assert.Equal(t, media, group)

	assert.Equal(t, "rotated-key", env.credentials(t, sonarr).APIKey)
	assert.Equal(t, "radarr-key", env.credentials(t, env.id(t, "integrations", "name", "Radarr")).APIKey)
	assert.Equal(t, otherUsers, env.dump(t, transferUser))

	assertNothingChanged(t, env.sync(t, transferAdmin, doc, true))
}

func TestSyncWithoutPruneOnlyAddsAndUpdates(t *testing.T) {
	env := newTransferEnv(t)
	env.seedDashboard(t)
	doc := &dashdoc.Document{
		Version:     dashdoc.Version,
		Groups:      []dashdoc.Group{{Ref: "media", Name: "Media", Color: "#123456"}, {Ref: "new", Name: "New"}},
		Services:    []dashdoc.Service{{Name: "Plex", URL: "https://plex.example.com", Description: "Films", GroupRef: "new"}},
		Preferences: &dashdoc.Preferences{ThemeMode: new("light")},
	}

	summary := env.sync(t, transferAdmin, doc, false)
	assert.Equal(t, dashdoc.Count{Created: 1, Updated: 1}, summary.Groups)
	assert.Equal(t, dashdoc.Count{Updated: 1}, summary.Services)
	assert.False(t, summary.Preferences, "preferences are only applied by a replace")
	assert.Equal(t, 3, env.count(t, "services", transferAdmin))
	assert.Equal(t, 3, env.count(t, "widgets", transferAdmin))
	assert.Equal(t, 2, env.count(t, "integrations", transferAdmin))

	var position int
	require.NoError(t, env.db.QueryRow(`SELECT position FROM groups WHERE name = 'New'`).Scan(&position))
	assert.Equal(t, 4, position, "behind the groups the user has")
	assertNothingChanged(t, env.sync(t, transferAdmin, doc, false))
}

func TestSyncNeverRemovesTheDefaultGroup(t *testing.T) {
	env := newTransferEnv(t)
	defaultGroup := env.addGroup(t, transferAdmin, "Services", 0, true)
	env.addGroup(t, transferAdmin, "Old", 1, false)

	doc := &dashdoc.Document{Version: dashdoc.Version, Groups: []dashdoc.Group{{Name: "Services"}}}
	summary := env.sync(t, transferAdmin, doc, true)
	assert.Equal(t, dashdoc.Count{Created: 1, Removed: 1}, summary.Groups, "a group is only the default group when the document says so")
	assert.Equal(t, defaultGroup, env.id(t, "groups", "is_default", "1"))
	assertNothingChanged(t, env.sync(t, transferAdmin, doc, true))

	// A default group in the document is the one the user has
	doc.Groups = []dashdoc.Group{{Name: "Home", Default: true}}
	summary = env.sync(t, transferAdmin, doc, true)
	assert.Equal(t, dashdoc.Count{Updated: 1, Removed: 1}, summary.Groups)
	assert.Equal(t, defaultGroup, env.id(t, "groups", "name", "Home"))
}

func TestSyncIntoAnEmptyAccountStoresCredentials(t *testing.T) {
	env := newTransferEnv(t)
	doc := &dashdoc.Document{
		Version: dashdoc.Version,
		Integrations: []dashdoc.Integration{
			{Ref: "sonarr", Kind: "sonarr", Name: "Sonarr", BaseURL: "http://192.168.1.10:8989", Credentials: &dashdoc.Credentials{APIKey: "from-env"}},
			{Ref: "radarr", Kind: "radarr", Name: "Radarr", BaseURL: "http://192.168.1.12:7878"},
		},
	}
	summary, err := env.svc.Sync(context.Background(), transferFresh, doc, true)
	require.NoError(t, err)
	assert.Equal(t, dashdoc.Count{Created: 2}, summary.Integrations)
	assert.Equal(t, 1, summary.NeedsCredentials, "Radarr has none")

	var id string
	require.NoError(t, env.db.QueryRow(`SELECT id FROM integrations WHERE name = 'Sonarr'`).Scan(&id))
	blob, err := env.svc.Integrations.GetCredentials(context.Background(), id, transferFresh)
	require.NoError(t, err)
	creds, err := env.integrations.decryptCredentials(blob, id)
	require.NoError(t, err)
	assert.Equal(t, "from-env", creds.APIKey)

	// The wrong field for the auth type is an error, never an empty secret
	doc.Integrations[0].Credentials = &dashdoc.Credentials{Token: "x"}
	_, err = env.svc.Sync(context.Background(), transferFresh, doc, true)
	assert.Equal(t, []string{"integrations[0].credentials"}, issuePaths(t, err))
}

func TestSyncOfAnInvalidDocumentChangesNothing(t *testing.T) {
	env := newTransferEnv(t)
	env.seedDashboard(t)
	before := env.dump(t, transferAdmin)

	doc := reparse(t, env.export(t, transferAdmin))
	doc.Services = append(doc.Services[:1:1], dashdoc.Service{Name: "Broken", URL: "not a url"})
	_, err := env.svc.Sync(context.Background(), transferAdmin, doc, true)
	assert.Equal(t, []string{"services[1]"}, issuePaths(t, err))
	assert.Equal(t, before, env.dump(t, transferAdmin))
}

func TestImportRefusesCredentials(t *testing.T) {
	env := newTransferEnv(t)
	doc := &dashdoc.Document{Version: dashdoc.Version, Integrations: []dashdoc.Integration{{
		Kind: "sonarr", Name: "Sonarr", BaseURL: "http://192.168.1.10:8989", Credentials: &dashdoc.Credentials{APIKey: "k"},
	}}}
	for _, run := range []func(context.Context, string, *dashdoc.Document, dashdoc.Mode) (*dashdoc.Summary, error){env.svc.Apply, env.svc.Preview} {
		_, err := run(context.Background(), transferAdmin, doc, dashdoc.ModeMerge)
		assert.Equal(t, []string{"integrations[0].credentials"}, issuePaths(t, err))
	}
	assert.Zero(t, env.count(t, "integrations", transferAdmin))

	_, err := env.svc.Apply(context.Background(), transferAdmin, doc, dashdoc.ModeSync)
	assert.Error(t, err, "sync is not an import mode")
}

func TestExportNeverCarriesCredentials(t *testing.T) {
	env := newTransferEnv(t)
	env.seedDashboard(t)
	for _, integration := range env.export(t, transferAdmin).Integrations {
		assert.Nil(t, integration.Credentials)
	}
}

func TestMergeMatchingGroupsUsesTheGroupsTheUserHas(t *testing.T) {
	env := newTransferEnv(t)
	media := env.addGroup(t, transferAdmin, "Media", 0, false)
	env.addGroup(t, transferUser, "Tools", 0, false)
	doc := &dashdoc.Document{
		Version: dashdoc.Version,
		Groups:  []dashdoc.Group{{Ref: "a", Name: "Media"}, {Ref: "b", Name: "Tools"}},
		Services: []dashdoc.Service{
			{Name: "Plex", URL: "https://plex.example.com", GroupRef: "a"},
			{Name: "Gitea", URL: "https://git.example.com", GroupRef: "b"},
		},
	}
	opts := ImportOptions{Mode: dashdoc.ModeMerge, MatchGroups: true, DryRun: true}
	preview, err := env.svc.Import(context.Background(), transferAdmin, doc, opts)
	require.NoError(t, err)
	assert.Equal(t, 1, preview.Groups.Created, "only Tools is new: another user's group is not matched")

	opts.DryRun = false
	summary, err := env.svc.Import(context.Background(), transferAdmin, doc, opts)
	require.NoError(t, err)
	assert.Equal(t, 1, summary.Groups.Created)
	assert.Equal(t, 2, env.count(t, "groups", transferAdmin))

	var group string
	require.NoError(t, env.db.QueryRow(`SELECT group_id FROM services WHERE name = 'Plex'`).Scan(&group))
	assert.Equal(t, media, group)

	// Without the option a merge adds a second group, as before
	_, err = env.svc.Apply(context.Background(), transferAdmin, doc, dashdoc.ModeMerge)
	require.NoError(t, err)
	assert.Equal(t, 4, env.count(t, "groups", transferAdmin))
}

// A config file kept in Git plus the secrets in the environment are all a
// fresh volume needs to get the same dashboard again
func TestConfigFileReproducesTheDashboardOnAFreshVolume(t *testing.T) {
	const file = `
owner: admin@example.com
version: 1
groups:
  - ref: main
    name: Home
    default: true
  - ref: media
    name: Media
    color: "#ff8800"
integrations:
  - ref: sonarr
    kind: sonarr
    name: Sonarr
    base_url: http://192.168.1.10:8989
    credentials:
      api_key: ${SONARR_API_KEY}
services:
  - name: Sonarr
    url: http://192.168.1.10:8989
    icon_type: image_url
    icon_url: https://cdn.jsdelivr.net/gh/homarr-labs/dashboard-icons/svg/sonarr.svg
    position: 0
    group_ref: media
    integration_ref: sonarr
  - name: Router
    url: http://192.168.1.1
    icon: "🌐"
    position: 1
    group_ref: main
    monitoring_enabled: false
widgets:
  - ref: power
    type: custom_api
    title: Power
    position: 2
    config:
      url: http://192.168.1.30/api
      headers:
        - name: Authorization
          value: Bearer ${METER_TOKEN}
      fields:
        - label: Now
          path: power
          unit: W
  - type: calendar
    title: Upcoming
    position: 3
    config:
      integrations: [sonarr]
      days: 7
preferences:
  theme_mode: dark
  layout_mode: classic
  status_strip:
    enabled: true
    chips:
      - source: integration
        ref: sonarr
        kpi: queue
      - source: widget
        ref: power
        kpi: Now
`
	lookup := func(name string) (string, bool) {
		value, ok := map[string]string{"SONARR_API_KEY": "key-from-env", "METER_TOKEN": "token-from-env"}[name]
		return value, ok
	}

	// boot is a server starting on an empty database with the file mounted
	boot := func() (*transferEnv, *dashdoc.Summary) {
		env := newTransferEnv(t)
		parsed, err := configfile.Parse([]byte(file), lookup)
		require.NoError(t, err)
		summary, err := env.svc.Sync(context.Background(), transferFresh, &parsed.Document, true)
		require.NoError(t, err)
		return env, summary
	}

	first, summary := boot()
	assert.Equal(t, dashdoc.Count{Created: 2}, summary.Groups)
	assert.Equal(t, dashdoc.Count{Created: 2}, summary.Services)
	assert.Equal(t, dashdoc.Count{Created: 2}, summary.Widgets)
	assert.Equal(t, dashdoc.Count{Created: 1}, summary.Integrations)
	assert.True(t, summary.Preferences)
	assert.Zero(t, summary.NeedsCredentials, "the credentials came from the environment")
	assert.Zero(t, summary.NeedsSecrets, "and so did the widget's secret")

	// The integration can connect and the widget has its header
	var id, config string
	require.NoError(t, first.db.QueryRow(`SELECT id FROM integrations`).Scan(&id))
	integration, err := first.svc.Integrations.GetByID(context.Background(), id, transferFresh)
	require.NoError(t, err)
	conn, err := first.integrations.Conn(context.Background(), integration)
	require.NoError(t, err)
	assert.Equal(t, "key-from-env", conn.Creds.APIKey)
	require.NoError(t, first.db.QueryRow(`SELECT config FROM widgets WHERE type = 'custom_api'`).Scan(&config))
	assert.Contains(t, config, "Bearer token-from-env")

	// A second fresh volume gives the same dashboard, and a restart on the
	// first changes nothing
	second, _ := boot()
	assert.Equal(t, toJSON(t, first.export(t, transferFresh)), toJSON(t, second.export(t, transferFresh)))

	before := first.dump(t, transferFresh)
	parsed, err := configfile.Parse([]byte(file), lookup)
	require.NoError(t, err)
	restart, err := first.svc.Sync(context.Background(), transferFresh, &parsed.Document, true)
	require.NoError(t, err)
	assertNothingChanged(t, restart)
	assert.Equal(t, before, first.dump(t, transferFresh))

	// The export of it holds neither secret
	exported := toJSON(t, first.export(t, transferFresh))
	assert.NotContains(t, exported, "key-from-env")
	assert.NotContains(t, exported, "token-from-env")
}

// The example in docs/CONFIGURATION.md must stay a file Nimbus applies
func TestDocumentedConfigFileApplies(t *testing.T) {
	docs, err := os.ReadFile("../../../docs/CONFIGURATION.md")
	require.NoError(t, err)
	_, example, found := strings.Cut(string(docs), "```yaml\n# nimbus.yaml\n")
	require.True(t, found, "the example is gone from the docs")
	example, _, _ = strings.Cut(example, "```")

	parsed, err := configfile.Parse([]byte(example), func(name string) (string, bool) { return "value-of-" + name, true })
	require.NoError(t, err)
	assert.Equal(t, "admin@example.com", parsed.Owner)

	env := newTransferEnv(t)
	summary, err := env.svc.Sync(context.Background(), transferFresh, &parsed.Document, true)
	require.NoError(t, err)
	assert.Equal(t, 2, summary.Integrations.Created)
	assert.Equal(t, 2, summary.Services.Created)
	assert.Equal(t, 2, summary.Widgets.Created)
	assert.Zero(t, summary.NeedsCredentials+summary.NeedsSecrets)
}
