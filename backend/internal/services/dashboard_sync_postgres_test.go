package services

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/nimbus/backend/internal/configfile"
	"github.com/nimbus/backend/internal/dashdoc"
	"github.com/nimbus/backend/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/lib/pq"
)

// TestSyncOnPostgres runs a config file's life on a real PostgreSQL: create,
// unchanged, update in place, prune. The other sync tests run on SQLite; this
// one checks the SQL, the foreign keys and JSONB against the real schema.
//
// It only runs when NIMBUS_TEST_POSTGRES_DSN points at an EMPTY, throwaway
// database, which it migrates and writes to:
//
//	docker run -d --name nimbus-pg-test -e POSTGRES_PASSWORD=test -p 127.0.0.1:55432:5432 postgres:18
//	NIMBUS_TEST_POSTGRES_DSN='postgres://postgres:test@127.0.0.1:55432/postgres?sslmode=disable' \
//	  go test ./internal/services/ -run TestSyncOnPostgres -v
func TestSyncOnPostgres(t *testing.T) {
	dsn := os.Getenv("NIMBUS_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("NIMBUS_TEST_POSTGRES_DSN is not set")
	}
	database, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { database.Close() })
	// A container that was just started takes a few seconds to accept
	// connections on its published port
	for deadline := time.Now().Add(30 * time.Second); ; time.Sleep(500 * time.Millisecond) {
		if err = database.Ping(); err == nil || time.Now().After(deadline) {
			break
		}
	}
	require.NoError(t, err, "PostgreSQL did not accept connections within 30 seconds")
	// The migration runner reads its files relative to the backend directory
	t.Chdir("../..")
	require.NoError(t, db.RunMigrations(database))
	_, err = database.Exec(`INSERT INTO users (id, email, name, password, role) VALUES ($1, 'sync-admin@example.com', 'Admin', 'x', 'admin')
		ON CONFLICT (id) DO NOTHING`, transferFresh)
	require.NoError(t, err)

	env := newTransferEnvOn(t, database)
	ctx := context.Background()
	lookup := func(name string) (string, bool) { return "value-of-" + name, true }
	sync := func(file string) *dashdoc.Summary {
		t.Helper()
		parsed, err := configfile.Parse([]byte(file), lookup)
		require.NoError(t, err)
		summary, err := env.svc.Sync(ctx, transferFresh, &parsed.Document, true)
		require.NoError(t, err)
		return summary
	}
	// snapshot is every column of the user's dashboard, timestamps included
	snapshot := func() string {
		t.Helper()
		out := ""
		for _, table := range []string{"groups", "integrations", "services", "widgets", "user_preferences"} {
			var rows sql.NullString
			require.NoError(t, database.QueryRow(
				`SELECT string_agg(t::text, E'\n' ORDER BY t.id::text) FROM `+table+` t WHERE user_id = $1`, transferFresh).Scan(&rows))
			out += table + "\n" + rows.String + "\n"
		}
		return out
	}
	one := func(query string, args ...any) string {
		t.Helper()
		var value string
		require.NoError(t, database.QueryRow(query, args...).Scan(&value))
		return value
	}

	const file = `
owner: sync-admin@example.com
version: 1
groups:
  - {ref: home, name: Home, default: true}
  - {ref: media, name: Media, color: "#ff8800", tile_style: chips}
  - {ref: old, name: Old}
integrations:
  - ref: sonarr
    kind: sonarr
    name: Sonarr
    base_url: http://192.168.1.10:8989
    credentials: {api_key: "${SONARR_API_KEY}"}
  - ref: radarr
    kind: radarr
    name: Radarr
    base_url: http://192.168.1.11:7878
services:
  - {name: Sonarr, url: "http://192.168.1.10:8989", position: 0, group_ref: media, integration_ref: sonarr}
  - {name: Router, url: "http://192.168.1.1", icon: "🌐", position: 1, group_ref: home}
  - {name: Gone, url: "http://192.168.1.9", position: 2, group_ref: old, integration_ref: radarr}
widgets:
  - {type: markdown, title: Notes, position: 3, config: {content: hello}}
  - type: calendar
    title: Upcoming
    position: 4
    config: {integrations: [sonarr], days: 7}
preferences:
  theme_mode: dark
  group_display: sections
  theme_preset: nord
  kpi_mono: true
  custom_css: ".card { border-radius: 0; }"
  status_strip:
    enabled: true
    chips:
      - {source: integration, ref: sonarr, kpi: queued}
`

	// Create
	summary := sync(file)
	assert.Equal(t, dashdoc.Count{Created: 3}, summary.Groups)
	assert.Equal(t, dashdoc.Count{Created: 3}, summary.Services)
	assert.Equal(t, dashdoc.Count{Created: 2}, summary.Widgets)
	assert.Equal(t, dashdoc.Count{Created: 2}, summary.Integrations)
	assert.True(t, summary.Preferences)
	assert.Equal(t, 1, summary.NeedsCredentials, "Radarr has none")
	assert.Equal(t, "chips", one(`SELECT tile_style FROM groups WHERE user_id = $1 AND name = 'Media'`, transferFresh))
	assert.Equal(t, "sections", one(`SELECT group_display FROM user_preferences WHERE user_id = $1`, transferFresh))
	assert.Equal(t, "nord", one(`SELECT theme_preset FROM user_preferences WHERE user_id = $1`, transferFresh))

	sonarrService := one(`SELECT id FROM services WHERE user_id = $1 AND name = 'Sonarr'`, transferFresh)
	sonarr := one(`SELECT id FROM integrations WHERE user_id = $1 AND name = 'Sonarr'`, transferFresh)
	media := one(`SELECT id FROM groups WHERE user_id = $1 AND name = 'Media'`, transferFresh)
	gone := one(`SELECT id FROM services WHERE user_id = $1 AND name = 'Gone'`, transferFresh)
	credentials := one(`SELECT encode(credentials_enc, 'hex') FROM integrations WHERE id = $1`, sonarr)
	// History hangs on the service's ID
	for _, service := range []string{sonarrService, gone} {
		_, err = database.Exec(`INSERT INTO service_status_logs (service_id, status) VALUES ($1, 'online')`, service)
		require.NoError(t, err)
	}

	// Unchanged: nothing is written, whatever PostgreSQL made of the JSON
	before := snapshot()
	assertNothingChanged(t, sync(file))
	assert.Equal(t, before, snapshot())

	// Update in place, and prune what the file no longer has
	changed := `
owner: sync-admin@example.com
version: 1
groups:
  - {ref: home, name: Home, default: true}
  - {ref: media, name: Media, color: "#00ff00"}
integrations:
  - ref: sonarr
    kind: sonarr
    name: Sonarr
    base_url: http://192.168.1.10:9999
    credentials: {api_key: "${SONARR_API_KEY}"}
services:
  - {name: Sonarr, url: "http://192.168.1.10:9999", description: TV, position: 0, group_ref: home, integration_ref: sonarr}
  - {name: Router, url: "http://192.168.1.1", icon: "🌐", position: 1, group_ref: home}
widgets:
  - {type: markdown, title: Notes, position: 3, config: {content: changed}}
preferences:
  theme_mode: light
  theme_preset: dracula
`
	summary = sync(changed)
	assert.Equal(t, dashdoc.Count{Updated: 1, Removed: 1}, summary.Groups)
	assert.Equal(t, dashdoc.Count{Updated: 1, Removed: 1}, summary.Services)
	assert.Equal(t, dashdoc.Count{Updated: 1, Removed: 1}, summary.Widgets)
	assert.Equal(t, dashdoc.Count{Updated: 1, Removed: 1}, summary.Integrations)
	assert.True(t, summary.Preferences)

	assert.Equal(t, sonarrService, one(`SELECT id FROM services WHERE user_id = $1 AND url = 'http://192.168.1.10:9999'`, transferFresh))
	assert.Equal(t, sonarr, one(`SELECT id FROM integrations WHERE user_id = $1 AND base_url = 'http://192.168.1.10:9999'`, transferFresh))
	assert.Equal(t, media, one(`SELECT id FROM groups WHERE user_id = $1 AND color = '#00ff00' AND tile_style = 'cards'`, transferFresh))
	assert.Equal(t, credentials, one(`SELECT encode(credentials_enc, 'hex') FROM integrations WHERE id = $1`, sonarr), "the same key is not encrypted again")
	assert.Equal(t, "1", one(`SELECT COUNT(*) FROM service_status_logs WHERE service_id = $1`, sonarrService), "history survives the update")
	assert.Equal(t, "0", one(`SELECT COUNT(*) FROM service_status_logs WHERE service_id = $1`, gone), "and goes with a removed service")
	assert.Equal(t, "2", one(`SELECT COUNT(*) FROM services WHERE user_id = $1`, transferFresh))
	assert.Equal(t, "1", one(`SELECT COUNT(*) FROM integrations WHERE user_id = $1`, transferFresh))
	assert.Equal(t, "2", one(`SELECT COUNT(*) FROM groups WHERE user_id = $1`, transferFresh))
	assert.Equal(t, "light", one(`SELECT theme_mode FROM user_preferences WHERE user_id = $1`, transferFresh))
	assert.Equal(t, "dracula|true|.card { border-radius: 0; }", one(`SELECT theme_preset || '|' || kpi_mono::text || '|' || custom_css FROM user_preferences WHERE user_id = $1`, transferFresh),
		"the preset changed; the settings the file left out kept their value")
	assert.Equal(t, "", one(`SELECT COALESCE(group_display, '') FROM user_preferences WHERE user_id = $1`, transferFresh), "left out, it follows the layout")

	// And unchanged again
	before = snapshot()
	assertNothingChanged(t, sync(changed))
	assert.Equal(t, before, snapshot())
}
