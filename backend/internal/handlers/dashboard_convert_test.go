package handlers

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nimbus/backend/internal/dashdoc"
	"github.com/nimbus/backend/internal/importers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Every sample config of the other dashboards goes through the wizard's
// calls: convert, preview the import, import
func TestConvert_SampleConfigsImportWithoutErrors(t *testing.T) {
	files, err := filepath.Glob("../importers/testdata/*")
	require.NoError(t, err)
	require.Len(t, files, 8)
	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			app, db := setupTransferTestApp(t, integrationOtherID)
			config, err := os.ReadFile(file)
			require.NoError(t, err)

			status, _, body := transferRequest(t, app, http.MethodPost, "/import/convert?host=nas.lan", string(config))
			require.Equal(t, http.StatusOK, status, string(body))
			var result importers.Result
			require.NoError(t, json.Unmarshal(body, &result))
			require.NotEmpty(t, result.Document.Services)
			document, err := json.Marshal(result.Document)
			require.NoError(t, err)

			var preview, summary dashdoc.Summary
			status, _, body = transferRequest(t, app, http.MethodPost, "/import?mode=merge&match_groups=1&dry_run=1", string(document))
			require.Equal(t, http.StatusOK, status, string(body))
			require.NoError(t, json.Unmarshal(body, &preview))
			status, _, body = transferRequest(t, app, http.MethodPost, "/import?mode=merge&match_groups=1", string(document))
			require.Equal(t, http.StatusOK, status, string(body))
			require.NoError(t, json.Unmarshal(body, &summary))

			assert.Equal(t, len(result.Document.Services), summary.Services.Created)
			assert.Equal(t, len(result.Document.Groups), summary.Groups.Created)
			assert.Equal(t, len(result.Document.Integrations), summary.Integrations.Created)
			assert.Equal(t, len(result.Document.Integrations), summary.NeedsCredentials, "every imported integration waits for its credentials")
			preview.DryRun = false
			assert.Equal(t, summary, preview)

			var stored int
			require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM integrations WHERE credentials_enc IS NOT NULL`).Scan(&stored))
			assert.Zero(t, stored)
		})
	}
}

func TestConvert_Rejects(t *testing.T) {
	app, _ := setupTransferTestApp(t, integrationOwnerID)
	compose := "services:\n  sonarr:\n    image: sonarr\n    ports: ['8989:8989']\n"
	cases := map[string]struct {
		path, body string
		status     int
	}{
		"unknown format":      {"/import/convert?format=glance", compose, http.StatusBadRequest},
		"unrecognised config": {"/import/convert", "hello", http.StatusBadRequest},
		"wrong format":        {"/import/convert?format=homarr", compose, http.StatusBadRequest},
		"a host with a port":  {"/import/convert?host=" + url.QueryEscape("nas.lan:80/x"), compose, http.StatusBadRequest},
		"too large":           {"/import/convert", strings.Repeat("a", importers.MaxSize+1), http.StatusRequestEntityTooLarge},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			status, _, body := transferRequest(t, app, http.MethodPost, tc.path, tc.body)
			assert.Equal(t, tc.status, status, string(body))
			assert.Contains(t, string(body), `"error"`)
		})
	}

	status, _, body := transferRequest(t, app, http.MethodPost, "/import/convert?format=compose&host=10.0.0.5", compose)
	require.Equal(t, http.StatusOK, status, string(body))
	assert.Contains(t, string(body), "http://10.0.0.5:8989")
}

func TestConvert_NeverResolvesVariables(t *testing.T) {
	t.Setenv("SONARR_API_KEY", "server-side-secret")
	t.Setenv("HOMEPAGE_URL", "http://server-side.example")
	app, _ := setupTransferTestApp(t, integrationOwnerID)
	config := "- Media:\n    - Sonarr:\n        href: http://sonarr.lan\n        description: ${SONARR_API_KEY}\n        widget:\n          type: sonarr\n          url: http://sonarr.lan\n          key: ${SONARR_API_KEY}\n    - Other:\n        href: ${HOMEPAGE_URL}\n"
	status, _, body := transferRequest(t, app, http.MethodPost, "/import/convert", config)
	require.Equal(t, http.StatusOK, status, string(body))
	assert.NotContains(t, string(body), "server-side")
}

func TestTransfer_ImportRefusesCredentials(t *testing.T) {
	app, db := setupTransferTestApp(t, integrationOwnerID)
	doc := `{"version": 1, "integrations": [{"kind": "handler-fake", "name": "Sonarr", "base_url": "http://192.168.1.10:8989", "credentials": {"api_key": "k"}}]}`
	for _, query := range []string{"mode=merge", "mode=merge&dry_run=1", "mode=replace", "mode=replace&dry_run=1", "mode=merge&match_groups=1", "mode=merge&match_groups=1&dry_run=1"} {
		status, _, body := transferRequest(t, app, http.MethodPost, "/import?"+query, doc)
		assert.Equal(t, http.StatusBadRequest, status, string(body))
		assert.Contains(t, string(body), "integrations[0].credentials")
	}
	var stored int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM integrations`).Scan(&stored))
	assert.Zero(t, stored)

	status, _, _ := transferRequest(t, app, http.MethodPost, "/import?mode=sync", `{"version": 1}`)
	assert.Equal(t, http.StatusBadRequest, status, "sync is not a mode of the endpoint")
}

func TestTransfer_MatchGroups(t *testing.T) {
	app, db := setupTransferTestApp(t, integrationOwnerID)
	_, err := db.Exec(`INSERT INTO groups (id, user_id, name, color) VALUES ('g-media', ?, 'Media', '#ff0000'), ('g-other', ?, 'Tools', '#ff0000')`,
		integrationOwnerID, integrationOtherID)
	require.NoError(t, err)
	doc := `{"version": 1, "groups": [{"ref": "a", "name": "Media"}, {"ref": "b", "name": "Tools"}],
		"services": [{"name": "Plex", "url": "https://plex.example.com", "position": 0, "group_ref": "a"}]}`

	status, _, body := transferRequest(t, app, http.MethodPost, "/import?mode=merge&match_groups=1", doc)
	require.Equal(t, http.StatusOK, status, string(body))
	var summary dashdoc.Summary
	require.NoError(t, json.Unmarshal(body, &summary))
	assert.Equal(t, 1, summary.Groups.Created, "Tools is new: the other user's group is not the user's")

	var group string
	require.NoError(t, db.QueryRow(`SELECT group_id FROM services WHERE name = 'Plex'`).Scan(&group))
	assert.Equal(t, "g-media", group)
}
