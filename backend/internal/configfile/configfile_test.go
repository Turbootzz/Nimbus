package configfile

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nimbus/backend/internal/dashdoc"
	"github.com/nimbus/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func env(values map[string]string) func(string) (string, bool) {
	return func(name string) (string, bool) {
		value, ok := values[name]
		return value, ok
	}
}

const yamlFile = `
owner: admin@example.com
version: 1
groups:
  - ref: media
    name: Media
integrations:
  - ref: sonarr
    kind: sonarr
    name: Sonarr
    base_url: http://sonarr:8989
    credentials:
      api_key: ${SONARR_API_KEY}
services:
  - name: Sonarr
    url: http://sonarr.lan
    group_ref: media
    integration_ref: sonarr
    position: 0
widgets:
  - type: custom_api
    title: Power
    position: 1
    config:
      url: http://meter.lan/api
      headers:
        - name: X-Key
          value: "Bearer ${METER_TOKEN}"
  - type: markdown
    position: 2
    config:
      content: "costs $${PRICE} and $5"
`

func TestParseYAMLInterpolatesValues(t *testing.T) {
	file, err := Parse([]byte(yamlFile), env(map[string]string{"SONARR_API_KEY": "s3cret", "METER_TOKEN": "tok"}))
	require.NoError(t, err)
	assert.Equal(t, "admin@example.com", file.Owner)
	assert.Equal(t, 1, file.Version)
	require.Len(t, file.Integrations, 1)
	assert.Equal(t, &dashdoc.Credentials{APIKey: "s3cret"}, file.Integrations[0].Credentials)
	assert.Equal(t, "media", file.Services[0].GroupRef)
	assert.JSONEq(t, `{"url":"http://meter.lan/api","headers":[{"name":"X-Key","value":"Bearer tok"}]}`, string(file.Widgets[0].Config))
	assert.JSONEq(t, `{"content":"costs ${PRICE} and $5"}`, string(file.Widgets[1].Config), "$${ is the escape for ${")
	require.NoError(t, dashdoc.Validate(&file.Document))
}

func TestParseJSON(t *testing.T) {
	file, err := Parse([]byte(`{"owner": " admin@example.com ", "version": 1, "services": [{"name": "${NAME}", "url": "http://a.lan", "position": 0}]}`),
		env(map[string]string{"NAME": "Plex"}))
	require.NoError(t, err)
	assert.Equal(t, "admin@example.com", file.Owner)
	assert.Equal(t, "Plex", file.Services[0].Name)
}

func TestAValueCannotChangeTheStructure(t *testing.T) {
	// Replaced in the raw text, this value would add a second service
	injection := "x\"\n  - name: Evil\n    url: http://evil.example\n    position: 0\n#"
	file, err := Parse([]byte("owner: a@b.c\nversion: 1\nservices:\n  - name: \"${NAME}\"\n    url: http://a.lan\n    position: 0\n"),
		env(map[string]string{"NAME": injection}))
	require.NoError(t, err)
	require.Len(t, file.Services, 1)
	assert.Equal(t, injection, file.Services[0].Name)

	// And a value is not interpolated again
	file, err = Parse([]byte(`{"owner": "a@b.c", "version": 1, "services": [{"name": "${A}", "url": "http://a.lan", "position": 0}]}`),
		env(map[string]string{"A": "${B}", "B": "secret"}))
	require.NoError(t, err)
	assert.Equal(t, "${B}", file.Services[0].Name)
}

func TestParseNamesTheMissingVariables(t *testing.T) {
	_, err := Parse([]byte(yamlFile), env(map[string]string{"SONARR_API_KEY": ""}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "METER_TOKEN, SONARR_API_KEY", "an empty variable is as good as none")
}

func TestParseRefusesANULFromAVariable(t *testing.T) {
	_, err := Parse([]byte("owner: a@b.c\nversion: 1\ngroups:\n  - name: ${NAME}\n"), env(map[string]string{"NAME": "a\x00b"}))
	assert.ErrorContains(t, err, "can not be stored")
}

func TestParseRefuses(t *testing.T) {
	level := func(name, previous string) string {
		return name + ": &" + name + " [" + strings.TrimSuffix(strings.Repeat("*"+previous+",", 9), ",") + "]\n"
	}
	bomb := "a: &a [x,x,x,x,x,x,x,x,x]\n"
	for i, name := range []string{"b", "c", "d", "e", "f", "g", "h", "i"} {
		bomb += level(name, string(rune('a'+i)))
	}
	cases := map[string]string{
		"an unknown field":    "owner: a@b.c\nversion: 1\nservcies: []\n",
		"no owner":            "version: 1\n",
		"a list":              "- a\n- b\n",
		"a scalar":            "just text",
		"nothing":             "",
		"broken YAML":         "owner: [a\n",
		"broken JSON":         `{"owner": `,
		"a wrong type":        "owner: a@b.c\nversion: one\n",
		"keys of other types": "owner: a@b.c\nversion: 1\n1: 2\n",
		"an alias bomb":       bomb,
		"binary":              "\x00\xff\xfe\x01",
		"a NUL character":     `{"owner": "a@b.c", "version": 1, "groups": [{"name": "a\u0000b"}]}`,
		"too much":            strings.Repeat("a", MaxSize+1),
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := Parse([]byte(content), env(nil))
			assert.Error(t, err)
		})
	}
}

// fakeSyncer records what it is asked to sync
type fakeSyncer struct {
	calls []string
	prune bool
	err   error
}

func (f *fakeSyncer) Sync(_ context.Context, userID string, _ *dashdoc.Document, prune bool) (*dashdoc.Summary, error) {
	f.calls, f.prune = append(f.calls, userID), prune
	if f.err != nil {
		return nil, f.err
	}
	return &dashdoc.Summary{Mode: dashdoc.ModeSync}, nil
}

type fakeUsers map[string]*models.User

func (f fakeUsers) GetByEmail(email string) (*models.User, error) {
	if user, ok := f[email]; ok {
		return user, nil
	}
	return nil, errors.New("user not found")
}

const goodFile = "owner: admin@example.com\nversion: 1\npreferences:\n  theme_mode: dark\n"

func newTestWatcher(t *testing.T, content string, opts Options) (*Watcher, *fakeSyncer, fakeUsers, func(string)) {
	t.Helper()
	opts.Path = filepath.Join(t.TempDir(), "nimbus.yaml")
	write := func(content string) { require.NoError(t, os.WriteFile(opts.Path, []byte(content), 0o600)) }
	write(content)
	syncer, users := &fakeSyncer{}, fakeUsers{"admin@example.com": {ID: "admin-1", Role: "admin"}}
	return NewWatcher(opts, syncer, users), syncer, users, write
}

func TestWatcherAppliesOnceUntilTheFileChanges(t *testing.T) {
	w, syncer, _, write := newTestWatcher(t, goodFile, Options{Prune: true})
	ctx := context.Background()
	w.Check(ctx)
	w.Check(ctx)
	assert.Equal(t, []string{"admin-1"}, syncer.calls, "an unchanged file is not applied again")
	assert.True(t, syncer.prune)

	write(goodFile + "services: []\n")
	w.Check(ctx)
	assert.Len(t, syncer.calls, 2)
}

func TestWatcherSurvivesABadFile(t *testing.T) {
	w, syncer, _, write := newTestWatcher(t, "owner: [broken", Options{})
	ctx := context.Background()
	w.Check(ctx)
	w.Check(ctx)
	assert.Empty(t, syncer.calls)
	assert.False(t, w.Manages("admin-1"))

	write(goodFile)
	w.Check(ctx)
	assert.Len(t, syncer.calls, 1, "the fixed file is applied")

	// A later mistake keeps the last good state and its owner
	write("owner: admin@example.com\nversion: 1\nwidgets: 5\n")
	w.Check(ctx)
	assert.Len(t, syncer.calls, 1)
	assert.True(t, w.Manages("admin-1"))

	// A file that is gone changes nothing either
	w.opts.Path = filepath.Join(t.TempDir(), "missing.yaml")
	w.Check(ctx)
	assert.Len(t, syncer.calls, 1)
}

func TestWatcherWaitsForTheOwner(t *testing.T) {
	w, syncer, users, _ := newTestWatcher(t, goodFile, Options{})
	ctx := context.Background()
	delete(users, "admin@example.com")
	w.Check(ctx)
	users["admin@example.com"] = &models.User{ID: "user-1", Role: "user"}
	w.Check(ctx)
	assert.Empty(t, syncer.calls, "the owner must be an admin")
	assert.False(t, w.Manages("user-1"))

	// The setup creates the admin; the same file is then applied
	users["admin@example.com"] = &models.User{ID: "admin-1", Role: "admin"}
	w.Check(ctx)
	w.Check(ctx)
	assert.Equal(t, []string{"admin-1"}, syncer.calls)
}

func TestWatcherRetriesWhatMayPass(t *testing.T) {
	w, syncer, _, _ := newTestWatcher(t, goodFile, Options{})
	ctx := context.Background()

	syncer.err = errors.New("database is down")
	w.Check(ctx)
	w.Check(ctx)
	assert.Len(t, syncer.calls, 2, "tried again at the next check")

	syncer.err = &dashdoc.ValidationError{Issues: []dashdoc.Issue{{Path: "services[0]", Message: "Name and URL are required"}}}
	w.Check(ctx)
	w.Check(ctx)
	assert.Len(t, syncer.calls, 3, "an invalid file waits for a change")
}

func TestWatcherLocks(t *testing.T) {
	ctx := context.Background()
	w, _, _, write := newTestWatcher(t, goodFile, Options{Prune: true, ReadOnly: true})
	assert.False(t, w.Locks("admin-1"), "nothing is locked before the owner is known")
	w.Check(ctx)
	assert.True(t, w.Locks("admin-1"))
	assert.True(t, w.LocksPreferences("admin-1"))
	assert.False(t, w.Locks("user-2"))
	assert.False(t, w.Locks(""))

	write("owner: admin@example.com\nversion: 1\n")
	w.Check(ctx)
	assert.False(t, w.LocksPreferences("admin-1"), "the file no longer sets the preferences")

	w, _, _, _ = newTestWatcher(t, goodFile, Options{Prune: true})
	w.Check(ctx)
	assert.True(t, w.Manages("admin-1"))
	assert.False(t, w.Locks("admin-1"), "not read-only")

	w, _, _, _ = newTestWatcher(t, goodFile, Options{ReadOnly: true})
	w.Check(ctx)
	assert.False(t, w.LocksPreferences("admin-1"), "a merge leaves the preferences alone")
}

func TestWatcherStartsAndStops(t *testing.T) {
	w, syncer, _, _ := newTestWatcher(t, goodFile, Options{})
	w.Start()
	w.Stop()
	assert.Len(t, syncer.calls, 1, "applied at start")
}
