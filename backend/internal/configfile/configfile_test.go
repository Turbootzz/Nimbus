package configfile

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nimbus/backend/internal/dashdoc"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
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
	// wait makes a sync wait for its context, like one held up by a row lock
	wait bool
}

func (f *fakeSyncer) Sync(ctx context.Context, userID string, _ *dashdoc.Document, prune bool) (*dashdoc.Summary, error) {
	f.calls, f.prune = append(f.calls, userID), prune
	if f.wait {
		<-ctx.Done()
		return nil, ctx.Err()
	}
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
	return nil, repository.ErrUserNotFound
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

	// As services.DashboardTransferService.Sync reports a limit
	syncer.err = fmt.Errorf("%w: %w", dashdoc.Issues{{Path: "widgets", Message: "the import would go over the limit of 50 widgets"}}.Err(),
		repository.ErrWidgetLimitReached)
	w.Check(ctx)
	w.Check(ctx)
	assert.Len(t, syncer.calls, 4, "the owner may have removed widgets by the next check")

	syncer.err = &dashdoc.ValidationError{Issues: []dashdoc.Issue{{Path: "services[0]", Message: "Name and URL are required"}}}
	w.Check(ctx)
	w.Check(ctx)
	assert.Len(t, syncer.calls, 5, "an invalid file waits for a change")
}

func TestWatcherKeepsTheOwnerOfTheLastAppliedFile(t *testing.T) {
	ctx := context.Background()
	w, syncer, users, write := newTestWatcher(t, goodFile, Options{Prune: true, ReadOnly: true})
	w.Check(ctx)
	require.True(t, w.Locks("admin-1"))
	require.True(t, w.LocksPreferences("admin-1"))

	// A file the sync refuses moves no lock to another admin
	users["other@example.com"] = &models.User{ID: "admin-2", Role: "admin"}
	syncer.err = &dashdoc.ValidationError{Issues: []dashdoc.Issue{{Path: "services[0]", Message: "Name and URL are required"}}}
	write("owner: other@example.com\nversion: 1\n")
	w.Check(ctx)
	assert.True(t, w.Locks("admin-1"))
	assert.False(t, w.Manages("admin-2"))

	// Nor does it lift the lock of the preferences
	write("owner: admin@example.com\nversion: 1\n")
	w.Check(ctx)
	assert.True(t, w.LocksPreferences("admin-1"))

	// Applied, it does both
	syncer.err = nil
	write("owner: other@example.com\nversion: 1\n")
	w.Check(ctx)
	assert.False(t, w.Manages("admin-1"))
	assert.True(t, w.Locks("admin-2"))
	assert.False(t, w.LocksPreferences("admin-2"))
}

func TestWatcherStopEndsACheckThatWaitsOnTheDatabase(t *testing.T) {
	w, syncer, _, _ := newTestWatcher(t, goodFile, Options{})
	syncer.wait = true
	w.Start()
	stopped := make(chan struct{})
	go func() {
		w.Stop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("Stop waits for a sync that waits on the database")
	}
	assert.Len(t, syncer.calls, 1)
	assert.False(t, w.Manages("admin-1"), "nothing was applied")
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
	assert.True(t, w.OwnsServices("admin-1"), "a replace removes what the file doesn't list")
	assert.False(t, w.OwnsServices("user-2"))

	w, _, _, _ = newTestWatcher(t, goodFile, Options{})
	w.Check(ctx)
	assert.False(t, w.OwnsServices("admin-1"), "a merge leaves other services alone")

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

func TestWatcherLetsGoOfAnOwnerWhoIsNoLongerThatAdmin(t *testing.T) {
	ctx := context.Background()
	w, syncer, users, _ := newTestWatcher(t, goodFile, Options{Prune: true, ReadOnly: true})
	w.Check(ctx)
	require.True(t, w.Locks("admin-1"))

	// Demoted: the lock lifts and the unchanged file is no longer applied
	users["admin@example.com"] = &models.User{ID: "admin-1", Role: "user"}
	w.Check(ctx)
	w.Check(ctx)
	assert.False(t, w.Manages("admin-1"))
	assert.False(t, w.Locks("admin-1"))
	assert.Len(t, syncer.calls, 1)

	// An admin again: the same file is applied once more
	users["admin@example.com"] = &models.User{ID: "admin-1", Role: "admin"}
	w.Check(ctx)
	w.Check(ctx)
	assert.True(t, w.Locks("admin-1"))
	assert.Len(t, syncer.calls, 2)

	// Deleted and created again: the new account is the owner
	delete(users, "admin@example.com")
	w.Check(ctx)
	assert.False(t, w.Manages("admin-1"))
	users["admin@example.com"] = &models.User{ID: "admin-2", Role: "admin"}
	w.Check(ctx)
	assert.False(t, w.Manages("admin-1"))
	assert.True(t, w.Manages("admin-2"))
	assert.Equal(t, []string{"admin-1", "admin-1", "admin-2"}, syncer.calls)

	// A file that now names somebody who is no admin manages nobody
	users["other@example.com"] = &models.User{ID: "user-9", Role: "user"}
	require.NoError(t, os.WriteFile(w.opts.Path, []byte("owner: other@example.com\nversion: 1\n"), 0o600))
	w.Check(ctx)
	assert.False(t, w.Manages("admin-2"))
	assert.False(t, w.Manages("user-9"))
}

func TestWatcherWaitsForAFileThatIsStillBeingWritten(t *testing.T) {
	w, syncer, _, write := newTestWatcher(t, goodFile, Options{Settle: time.Millisecond})
	ctx := context.Background()
	assert.True(t, w.settled(ctx, sha256.Sum256([]byte(goodFile))))
	assert.False(t, w.settled(ctx, sha256.Sum256([]byte("owner: half"))), "the file changed since it was read")

	// A file that keeps its content is applied, also after a change
	w.Check(context.Background())
	write(goodFile + "services: []\n")
	w.Check(context.Background())
	assert.Len(t, syncer.calls, 2)

	// Stopping does not wait for the file
	w.opts.Settle = time.Hour
	w.cancel()
	assert.False(t, w.settled(w.ctx, sha256.Sum256([]byte(goodFile+"services: []\n"))))
}

func TestWatcherLogsNoValuesOfTheFile(t *testing.T) {
	var logged bytes.Buffer
	log.SetOutput(&logged)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	lookup := env(map[string]string{"KEY": "SENTINEL"})

	// A file that can't be parsed, in every way a value can be misplaced
	w, _, _, write := newTestWatcher(t, "", Options{Lookup: lookup})
	// Neighbours differ, as a message is only logged when it is not the last one
	broken := []string{
		"owner: admin@example.com\nversion: 1\nintegrations:\n  - credentials:\n      api_key: SENTINEL-secret\n   bad: [\n",
		// The YAML decoder quotes an alias, a tagged value and a key
		"owner: admin@example.com\nversion: 1\nintegrations:\n  - credentials:\n      api_key: *SENTINEL-secret\n",
		"owner: admin@example.com\nversion: 1\nintegrations:\n  - credentials:\n      api_key: SENTINEL-secret: x\n",
		"owner: admin@example.com\nversion: 1\nintegrations:\n  - credentials:\n      api_key: !!int SENTINEL-secret\n",
		"owner: admin@example.com\nversion: 1\nintegrations:\n  - credentials: SENTINEL-secret\n",
		"owner: admin@example.com\nversion: 1\n? [SENTINEL-secret]\n: x\n",
		"owner: admin@example.com\nversion: 1\nintegrations:\n  - credentials:\n      api_key: [\"${KEY}\"]\n",
		"owner: admin@example.com\nversion: 1\nSENTINEL-secret: 1\nSENTINEL-secret: 2\n",
		"owner: admin@example.com\nversion: SENTINEL-secret\n",
		`{"owner": "admin@example.com", "version": 1, "integrations": [{"credentials": {"api_key": "SENTINEL-secret"}]}`,
		`{"owner": "admin@example.com", "version": 1, "preferences": {"theme_background": ["${KEY}"]}}`,
	}
	for _, content := range broken {
		write(content)
		w.Check(context.Background())
	}
	assert.Equal(t, len(broken), strings.Count(logged.String(), "not applied"), logged.String())
	assert.NotContains(t, logged.String(), "SENTINEL")
}

func TestParseErrorsTellWhereWithoutQuotingTheFile(t *testing.T) {
	cases := map[string]string{
		"owner: a@b.c\nservices: [a\n":             "invalid YAML: line 1: did not find expected ',' or ']'",
		"owner: a@b.c\nkey: *SECRET\n":             "invalid YAML: the parser's description is not shown, it quotes the file",
		"owner: a@b.c\nSECRET: 1\nSECRET: 2\n":     "invalid YAML: line 3: the parser's description is not shown, it quotes the file",
		"{\"owner\": \"a@b.c\",\n\"key\": SECRET}": "invalid JSON: line 2: the parser's description is not shown, it quotes the file",
		"{\"owner\": \"a@b.c\",\n\"key\": ":        "invalid JSON: line 2: unexpected end of JSON input",
	}
	for content, want := range cases {
		_, err := Parse([]byte(content), env(nil))
		assert.EqualError(t, err, want, content)
	}
}

func TestReasonLeavesOutWhatQuotesTheFile(t *testing.T) {
	file, err := Parse([]byte("owner: a@b.c\nversion: 1\nwidgets:\n  - type: rss\n    title: W\n    config:\n      feeds: [\"https://feeds.example.com/private\", \"https://x.lan/?key=${KEY}\"]\n"),
		env(map[string]string{"KEY": "abc"}))
	require.NoError(t, err)
	reason := file.Reason(&dashdoc.ValidationError{Issues: []dashdoc.Issue{
		{Path: "services[0]", Message: "Name and URL are required"},
		{Path: "widgets[0]", Message: `Invalid config: invalid feed URL "https://feeds.example.com/private": no such host`},
		{Path: "widgets[1]", Message: "the key abc is not valid"},
	}})
	assert.Equal(t, "invalid: services[0]: Name and URL are required; "+
		"widgets[0]: (not shown: the message quotes a value of the file); "+
		"widgets[1]: (not shown: the message quotes a value of the file)", reason,
		"what a variable held is never shown, however short; a short literal like the title W is no secret")
	assert.Equal(t, "database is down", file.Reason(errors.New("database is down")))
	assert.NotContains(t, file.Reason(errors.New(`pq: invalid input: "https://feeds.example.com/private"`)), "feeds.example.com")
}
