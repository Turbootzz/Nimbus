package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/services"
	"github.com/nimbus/backend/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	discoveryAdminID = "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeee1"
	discoveryOtherID = "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeee2" // another admin
	discoveryUserID  = "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeee3" // not an admin
	discoverySecret  = "portainer-t0ken/secret"
	discoveryIcons   = "https://icons.example/svg"
)

// discoveryContainers is what the fake Docker API lists. The traefik label
// stands for the secrets labels often hold.
const discoveryContainers = `[
	{"Id":"1","Names":["/sonarr"],"Image":"lscr.io/linuxserver/sonarr:latest","State":"running",
	 "Ports":[{"PrivatePort":8989,"PublicPort":8989,"Type":"tcp"}],
	 "Labels":{"nimbus.enable":"true","nimbus.name":"Sonarr","nimbus.url":"https://sonarr.lan","nimbus.group":"media",
	           "nimbus.description":"<b>Series</b>","nimbus.widget":"sonarr","traefik.auth":"admin:label-s3cret"}},
	{"Id":"2","Names":["/jellyfin"],"Image":"jellyfin/jellyfin:10","State":"running",
	 "Labels":{"homepage.name":"Jellyfin","homepage.href":"http://jellyfin.lan:8096","homepage.icon":"jellyfin.png","homepage.group":"Media"}},
	{"Id":"3","Names":["/web"],"Image":"nginx:1.27","State":"running",
	 "Ports":[{"PrivatePort":80,"PublicPort":8080,"Type":"tcp"}]},
	{"Id":"4","Names":["/evil"],"Image":"evil/<script>","State":"running",
	 "Labels":{"nimbus.enable":"true","nimbus.url":"javascript:alert(1)","nimbus.icon":"file:///etc/passwd"}},
	{"Id":"5","Names":["/hidden"],"Image":"redis","State":"running","Labels":{"nimbus.enable":"false"}},
	{"Id":"6","Names":["/backup"],"Image":"restic","State":"exited"}
]`

// discoverySchema mirrors the tables discovery touches, for SQLite
const discoverySchema = `
	CREATE TABLE users (id TEXT PRIMARY KEY, role TEXT NOT NULL DEFAULT 'user');
	CREATE TABLE integrations (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		kind TEXT NOT NULL,
		name TEXT NOT NULL,
		base_url TEXT NOT NULL,
		auth_type TEXT NOT NULL DEFAULT 'none',
		credentials_enc BLOB,
		verify_tls INTEGER NOT NULL DEFAULT 1,
		options TEXT NOT NULL DEFAULT '{}',
		refresh_seconds INTEGER NOT NULL DEFAULT 60,
		last_test_at TIMESTAMP,
		last_test_ok INTEGER,
		last_error TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE groups (
		id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
		user_id TEXT NOT NULL,
		name TEXT NOT NULL,
		color TEXT,
		position INTEGER DEFAULT 0,
		is_default INTEGER DEFAULT 0,
		monitoring_enabled INTEGER DEFAULT 1,
		tile_style TEXT NOT NULL DEFAULT 'cards',
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	);
	CREATE TABLE services (
		id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
		integration_id TEXT,
		user_id TEXT NOT NULL,
		name TEXT NOT NULL,
		url TEXT NOT NULL,
		icon TEXT,
		icon_type TEXT,
		icon_image_path TEXT,
		description TEXT,
		status TEXT NOT NULL,
		response_time INTEGER,
		position INTEGER DEFAULT 0,
		card_size TEXT,
		group_id TEXT,
		monitoring_enabled INTEGER NOT NULL DEFAULT 1,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	);
	CREATE TABLE widgets (id TEXT PRIMARY KEY, user_id TEXT NOT NULL, position INTEGER DEFAULT 0);
	CREATE TABLE system_settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_by TEXT
	);
	-- Mirrors migration 000037
	CREATE TABLE discovered_containers (
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		discovery_key TEXT NOT NULL,
		service_id TEXT REFERENCES services(id) ON DELETE SET NULL,
		pending BOOLEAN NOT NULL DEFAULT FALSE,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (user_id, discovery_key)
	);
	INSERT INTO system_settings (key, value) VALUES ('discovery_auto_add', 'false');
	INSERT INTO users (id, role) VALUES
		('eeeeeeee-eeee-eeee-eeee-eeeeeeeeeee1', 'admin'),
		('eeeeeeee-eeee-eeee-eeee-eeeeeeeeeee2', 'admin'),
		('eeeeeeee-eeee-eeee-eeee-eeeeeeeeeee3', 'user');
`

// discoveryFixture is the discovery stack on one SQLite database. Building
// it again on the same database is what a server restart does.
type discoveryFixture struct {
	db           *sql.DB
	integrations *services.IntegrationService
	service      *services.DiscoveryService
}

func setupDiscoveryDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", "file::memory:?_foreign_keys=on")
	require.NoError(t, err)
	db.SetMaxOpenConns(1) // every connection to :memory: is its own database
	t.Cleanup(func() { db.Close() })
	_, err = db.Exec(discoverySchema)
	require.NoError(t, err)
	return db
}

func newDiscoveryFixture(t *testing.T, db *sql.DB) *discoveryFixture {
	t.Helper()
	cipher, err := utils.NewCipher(bytes.Repeat([]byte{7}, 32))
	require.NoError(t, err)
	integrationService := services.NewIntegrationService(repository.NewIntegrationRepository(db), cipher, "")
	return &discoveryFixture{
		db:           db,
		integrations: integrationService,
		service: services.NewDiscoveryService(integrationService, repository.NewDiscoveryRepository(db),
			repository.NewServiceRepository(db), repository.NewGroupRepository(db), repository.NewSettingsRepository(db), discoveryIcons),
	}
}

// app mirrors the routes in main.go with a stub auth middleware
func (f *discoveryFixture) app(userID, role string) *fiber.App {
	handler := NewDiscoveryHandler(f.service)
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", userID)
		c.Locals("role", role)
		return c.Next()
	})
	app.Get("/discovery/docker", handler.DiscoverDocker)
	app.Post("/discovery/docker/services", handler.AddDiscovered)
	app.Post("/discovery/docker/ignored", handler.IgnoreDiscovered)
	app.Delete("/discovery/docker/ignored", handler.UnignoreDiscovered)
	return app
}

// fakeDocker serves the container list, counts the requests and fails the
// test on anything else: discovery must request nothing but the list
func fakeDocker(t *testing.T, containers string) (*httptest.Server, *int) {
	t.Helper()
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodGet || r.URL.Path != "/containers/json" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		fmt.Fprint(w, containers)
	}))
	t.Cleanup(server.Close)
	return server, &requests
}

func (f *discoveryFixture) addDocker(t *testing.T, userID, name, baseURL string) *models.Integration {
	t.Helper()
	integration, err := f.integrations.Create(context.Background(), userID, true, &models.IntegrationRequest{
		Kind: "docker", Name: name, BaseURL: baseURL, AuthType: models.IntegrationAuthNone,
	})
	require.NoError(t, err)
	return integration
}

// discoveryRequest sends a request and returns status and body. No response
// may ever carry a raw label or a credential.
func discoveryRequest(t *testing.T, app *fiber.App, method, path, body string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Host = "nimbus.lan:3000"
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	text := string(raw)
	assert.NotContains(t, text, "label-s3cret", "%s %s leaked a label", method, path)
	assert.NotContains(t, text, "traefik", "%s %s leaked a label", method, path)
	assert.NotContains(t, text, discoverySecret, "%s %s leaked a credential", method, path)
	return resp.StatusCode, text
}

func discover(t *testing.T, app *fiber.App) models.DiscoveryResult {
	t.Helper()
	status, body := discoveryRequest(t, app, http.MethodGet, "/discovery/docker", "")
	require.Equal(t, fiber.StatusOK, status, body)
	var result models.DiscoveryResult
	require.NoError(t, json.Unmarshal([]byte(body), &result))
	return result
}

func containerNames(result models.DiscoveryResult, ignored bool) []string {
	names := []string{}
	for _, c := range result.Containers {
		if c.Ignored == ignored {
			names = append(names, c.Container)
		}
	}
	return names
}

func addBody(t *testing.T, c models.DiscoveredContainer) string {
	t.Helper()
	body, err := json.Marshal(models.DiscoveryAddRequest{
		Key: c.Key, Name: c.Name, URL: c.URL, IconURL: c.IconURL, Group: c.Group, Description: c.Description,
	})
	require.NoError(t, err)
	return string(body)
}

func keyBody(key string) string {
	return fmt.Sprintf(`{"key":%q}`, key)
}

func TestDiscoveryHandler_ListsParsedContainers(t *testing.T) {
	f := newDiscoveryFixture(t, setupDiscoveryDB(t))
	docker, requests := fakeDocker(t, discoveryContainers)
	integration := f.addDocker(t, discoveryAdminID, "Homelab", docker.URL)
	dockerHost, _ := url.Parse(docker.URL)

	result := discover(t, f.app(discoveryAdminID, "admin"))
	assert.Equal(t, 1, *requests, "one Docker API call, nothing else is fetched")
	assert.Equal(t, 1, result.Integrations)
	assert.Empty(t, result.Errors)
	assert.Equal(t, []string{"evil", "sonarr", "jellyfin", "web", "backup"}, containerNames(result, false),
		"labelled first, then running, then by name; nimbus.enable=false is left out")

	byName := map[string]models.DiscoveredContainer{}
	for _, c := range result.Containers {
		byName[c.Container] = c
	}
	assert.Equal(t, models.DiscoveredContainer{
		Key: integration.ID + "/sonarr", IntegrationID: integration.ID, IntegrationName: "Homelab",
		Container: "sonarr", Image: "lscr.io/linuxserver/sonarr:latest", State: "running", Enabled: true,
		Name: "Sonarr", URL: "https://sonarr.lan", IconURL: discoveryIcons + "/sonarr.svg",
		Group: "media", Description: "<b>Series</b>",
		Integration: &models.DiscoveredIntegration{Kind: "sonarr", Name: "Sonarr", BaseURL: "https://sonarr.lan"},
	}, byName["sonarr"])

	jellyfin := byName["jellyfin"]
	assert.False(t, jellyfin.Enabled, "homepage labels fill fields but do not enable auto-add")
	assert.Equal(t, []string{"Jellyfin", "http://jellyfin.lan:8096", discoveryIcons + "/jellyfin.svg", "Media"},
		[]string{jellyfin.Name, jellyfin.URL, jellyfin.IconURL, jellyfin.Group})
	assert.False(t, jellyfin.URLGuessed)

	web := byName["web"]
	assert.Equal(t, "web", web.Name, "falls back to the container name")
	assert.Equal(t, "http://"+dockerHost.Hostname()+":8080", web.URL, "published port on the integration's host")
	assert.True(t, web.URLGuessed)
	assert.Equal(t, discoveryIcons+"/nginx.svg", web.IconURL)

	evil := byName["evil"]
	assert.Empty(t, evil.URL, "a javascript: URL is dropped; without a port there is no guess")
	assert.Empty(t, evil.IconURL)
	assert.False(t, evil.URLGuessed)

	assert.Empty(t, byName["backup"].URL, "no published port, no guess")
}

// Behind a unix socket the address is guessed from the host the admin
// opened Nimbus on (nginx forwards it in the Host header)
func TestDiscoveryHandler_UnixSocketGuessUsesRequestHost(t *testing.T) {
	for host, want := range map[string]string{
		"nimbus.lan:3000":   "nimbus.lan",
		"dash.example.com":  "dash.example.com",
		"192.168.1.10:3000": "192.168.1.10",
		"[fd00::5]:3000":    "fd00::5",
		"":                  "",
		"bad host":          "",
		"evil.lan/path":     "",
		"evil.lan?x=1":      "",
		"user@evil.lan":     "",
		"evil<lan":          "",
		`evil"lan`:          "",
	} {
		assert.Equal(t, want, hostName(host), host)
	}

	// Not t.TempDir(): its path is too long for a socket on macOS
	dir, err := os.MkdirTemp("", "nimbus")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	socket := filepath.Join(dir, "docker.sock")
	listener, err := net.Listen("unix", socket)
	require.NoError(t, err)
	docker := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"Id":"1","Names":["/web"],"Image":"nginx","State":"running","Ports":[{"PublicPort":8080,"Type":"tcp"}]}]`)
	})}
	go func() { _ = docker.Serve(listener) }()
	t.Cleanup(func() { _ = docker.Close() })

	db := setupDiscoveryDB(t)
	cipher, err := utils.NewCipher(bytes.Repeat([]byte{7}, 32))
	require.NoError(t, err)
	f := newDiscoveryFixture(t, db)
	f.integrations = services.NewIntegrationService(repository.NewIntegrationRepository(db), cipher, socket)
	f.service = services.NewDiscoveryService(f.integrations, repository.NewDiscoveryRepository(db),
		repository.NewServiceRepository(db), repository.NewGroupRepository(db), repository.NewSettingsRepository(db), discoveryIcons)
	f.addDocker(t, discoveryAdminID, "Socket", "unix://"+socket)

	result := discover(t, f.app(discoveryAdminID, "admin")) // sent to nimbus.lan:3000
	require.Empty(t, result.Errors)
	require.Len(t, result.Containers, 1)
	assert.Equal(t, "http://nimbus.lan:8080", result.Containers[0].URL)
	assert.True(t, result.Containers[0].URLGuessed)
}

func TestDiscoveryHandler_AdminOnly(t *testing.T) {
	f := newDiscoveryFixture(t, setupDiscoveryDB(t))
	docker, requests := fakeDocker(t, discoveryContainers)
	integration := f.addDocker(t, discoveryAdminID, "Homelab", docker.URL)
	app := f.app(discoveryUserID, "user")

	for _, request := range [][3]string{
		{http.MethodGet, "/discovery/docker", ""},
		{http.MethodPost, "/discovery/docker/services", `{"key":"` + integration.ID + `/web","name":"Web","url":"http://web.lan"}`},
		{http.MethodPost, "/discovery/docker/ignored", keyBody(integration.ID + "/web")},
		{http.MethodDelete, "/discovery/docker/ignored?key=" + url.QueryEscape(integration.ID+"/web"), ""},
	} {
		status, _ := discoveryRequest(t, app, request[0], request[1], request[2])
		assert.Equal(t, fiber.StatusForbidden, status, "%s %s", request[0], request[1])
	}
	assert.Zero(t, *requests)
}

func TestDiscoveryHandler_OnlyOwnIntegrations(t *testing.T) {
	f := newDiscoveryFixture(t, setupDiscoveryDB(t))
	docker, requests := fakeDocker(t, discoveryContainers)
	integration := f.addDocker(t, discoveryAdminID, "Homelab", docker.URL)
	other := f.app(discoveryOtherID, "admin")

	result := discover(t, other)
	assert.Zero(t, result.Integrations)
	assert.Empty(t, result.Containers)
	assert.Zero(t, *requests, "another admin's Docker is never asked")

	// Nor can its containers be added or ignored
	status, body := discoveryRequest(t, other, http.MethodPost, "/discovery/docker/services",
		`{"key":"`+integration.ID+`/web","name":"Web","url":"http://web.lan"}`)
	assert.Equal(t, fiber.StatusBadRequest, status, body)
	status, _ = discoveryRequest(t, other, http.MethodPost, "/discovery/docker/ignored", keyBody(integration.ID+"/web"))
	assert.Equal(t, fiber.StatusBadRequest, status)

	var rows int
	require.NoError(t, f.db.QueryRow(`SELECT (SELECT COUNT(*) FROM services) + (SELECT COUNT(*) FROM discovered_containers)`).Scan(&rows))
	assert.Zero(t, rows)
}

func TestDiscoveryHandler_AddCreatesServiceWithIconAndGroup(t *testing.T) {
	f := newDiscoveryFixture(t, setupDiscoveryDB(t))
	docker, _ := fakeDocker(t, discoveryContainers)
	f.addDocker(t, discoveryAdminID, "Homelab", docker.URL)
	app := f.app(discoveryAdminID, "admin")
	_, err := f.db.Exec(`INSERT INTO groups (id, user_id, name, color, created_at, updated_at) VALUES ('g-media', $1, 'Media', '#0ea5e9', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, discoveryAdminID)
	require.NoError(t, err)

	result := discover(t, app)
	byName := map[string]models.DiscoveredContainer{}
	for _, c := range result.Containers {
		byName[c.Container] = c
	}

	// "media" matches the existing group "Media"
	status, body := discoveryRequest(t, app, http.MethodPost, "/discovery/docker/services", addBody(t, byName["sonarr"]))
	require.Equal(t, fiber.StatusCreated, status, body)
	var sonarr models.ServiceResponse
	require.NoError(t, json.Unmarshal([]byte(body), &sonarr))
	assert.Equal(t, "Sonarr", sonarr.Name)
	assert.Equal(t, "https://sonarr.lan", sonarr.URL)
	assert.Equal(t, models.IconTypeImageURL, sonarr.IconType)
	assert.Equal(t, discoveryIcons+"/sonarr.svg", sonarr.IconImagePath)
	assert.Equal(t, "<b>Series</b>", sonarr.Description, "label text is stored as it is; the UI renders it as text")
	require.NotNil(t, sonarr.GroupID)
	assert.Equal(t, "g-media", *sonarr.GroupID)

	// A missing group is created; the admin may edit the values first
	web := byName["web"]
	web.Name, web.URL, web.Group = "Website", "https://www.lan", "Web stuff"
	status, body = discoveryRequest(t, app, http.MethodPost, "/discovery/docker/services", addBody(t, web))
	require.Equal(t, fiber.StatusCreated, status, body)
	var groupName string
	require.NoError(t, f.db.QueryRow(`SELECT g.name FROM services s JOIN groups g ON g.id = s.group_id WHERE s.name = 'Website' AND s.user_id = $1`, discoveryAdminID).Scan(&groupName))
	assert.Equal(t, "Web stuff", groupName)

	// Added containers are not offered again, and can't be added twice
	assert.Equal(t, []string{"evil", "jellyfin", "backup"}, containerNames(discover(t, app), false))
	status, _ = discoveryRequest(t, app, http.MethodPost, "/discovery/docker/services", addBody(t, web))
	assert.Equal(t, fiber.StatusConflict, status)

	// Renaming the service or changing its URL does not bring it back
	_, err = f.db.Exec(`UPDATE services SET name = 'Renamed', url = 'http://elsewhere.lan' WHERE id = $1`, sonarr.ID)
	require.NoError(t, err)
	assert.NotContains(t, containerNames(discover(t, app), false), "sonarr")

	// Deleting it moves the container to the ignored ones
	_, err = f.db.Exec(`DELETE FROM services WHERE id = $1`, sonarr.ID)
	require.NoError(t, err)
	result = discover(t, app)
	assert.NotContains(t, containerNames(result, false), "sonarr")
	assert.Equal(t, []string{"sonarr"}, containerNames(result, true))
}

func TestDiscoveryHandler_AddValidates(t *testing.T) {
	f := newDiscoveryFixture(t, setupDiscoveryDB(t))
	docker, _ := fakeDocker(t, discoveryContainers)
	integration := f.addDocker(t, discoveryAdminID, "Homelab", docker.URL)
	sonarrKind, err := f.integrations.Create(context.Background(), discoveryAdminID, true, &models.IntegrationRequest{
		Kind: "handler-fake", Name: "Not Docker", BaseURL: "http://sonarr.lan", AuthType: models.IntegrationAuthNone,
	})
	require.NoError(t, err)
	app := f.app(discoveryAdminID, "admin")
	key := integration.ID + "/web"

	valid := models.DiscoveryAddRequest{Key: key, Name: "Web", URL: "http://web.lan"}
	tests := map[string]func(r *models.DiscoveryAddRequest){
		"no name":                  func(r *models.DiscoveryAddRequest) { r.Name = " " },
		"long name":                func(r *models.DiscoveryAddRequest) { r.Name = strings.Repeat("a", 256) },
		"no url":                   func(r *models.DiscoveryAddRequest) { r.URL = "" },
		"javascript url":           func(r *models.DiscoveryAddRequest) { r.URL = "javascript:alert(1)" },
		"javascript url with host": func(r *models.DiscoveryAddRequest) { r.URL = "javascript://web.lan/%0aalert(1)" },
		"file url":                 func(r *models.DiscoveryAddRequest) { r.URL = "file:///etc/passwd" },
		"cloud metadata url":       func(r *models.DiscoveryAddRequest) { r.URL = "http://169.254.169.254/latest/meta-data/" },
		"no host":                  func(r *models.DiscoveryAddRequest) { r.URL = "http://" },
		"data icon":                func(r *models.DiscoveryAddRequest) { r.IconURL = "data:image/svg+xml,<svg/>" },
		"long group":               func(r *models.DiscoveryAddRequest) { r.Group = strings.Repeat("g", 36) },
		"long description":         func(r *models.DiscoveryAddRequest) { r.Description = strings.Repeat("d", 501) },
		"no key":                   func(r *models.DiscoveryAddRequest) { r.Key = "" },
		"key without container":    func(r *models.DiscoveryAddRequest) { r.Key = integration.ID + "/" },
		"key with a path":          func(r *models.DiscoveryAddRequest) { r.Key = integration.ID + "/../web" },
		"unknown integration":      func(r *models.DiscoveryAddRequest) { r.Key = "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeee9/web" },
		"not a docker integration": func(r *models.DiscoveryAddRequest) { r.Key = sonarrKind.ID + "/web" },
	}
	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			req := valid
			change(&req)
			body, err := json.Marshal(req)
			require.NoError(t, err)
			status, text := discoveryRequest(t, app, http.MethodPost, "/discovery/docker/services", string(body))
			assert.Equal(t, fiber.StatusBadRequest, status, text)
		})
	}
	var rows int
	require.NoError(t, f.db.QueryRow(`SELECT (SELECT COUNT(*) FROM services) + (SELECT COUNT(*) FROM discovered_containers) + (SELECT COUNT(*) FROM groups)`).Scan(&rows))
	assert.Zero(t, rows, "a refused add leaves nothing behind")
}

func TestDiscoveryHandler_IgnoreSurvivesRestart(t *testing.T) {
	db := setupDiscoveryDB(t)
	f := newDiscoveryFixture(t, db)
	docker, _ := fakeDocker(t, discoveryContainers)
	integration := f.addDocker(t, discoveryAdminID, "Homelab", docker.URL)
	app := f.app(discoveryAdminID, "admin")
	key := integration.ID + "/web"

	for range 2 { // ignoring twice is fine
		status, body := discoveryRequest(t, app, http.MethodPost, "/discovery/docker/ignored", keyBody(key))
		require.Equal(t, fiber.StatusNoContent, status, body)
	}
	status, _ := discoveryRequest(t, app, http.MethodPost, "/discovery/docker/services", `{"key":"`+key+`","name":"Web","url":"http://web.lan"}`)
	assert.Equal(t, fiber.StatusConflict, status, "an ignored container must be un-ignored first")

	// A new server on the same database, and a re-created container (new id, same name)
	restarted := newDiscoveryFixture(t, db).app(discoveryAdminID, "admin")
	result := discover(t, restarted)
	assert.Equal(t, []string{"web"}, containerNames(result, true))
	assert.NotContains(t, containerNames(result, false), "web")

	status, body := discoveryRequest(t, restarted, http.MethodDelete, "/discovery/docker/ignored?key="+url.QueryEscape(key), "")
	require.Equal(t, fiber.StatusNoContent, status, body)
	result = discover(t, restarted)
	assert.Empty(t, containerNames(result, true))
	assert.Contains(t, containerNames(result, false), "web")
}

func TestDiscoveryHandler_SeveralIntegrationsAndFailures(t *testing.T) {
	f := newDiscoveryFixture(t, setupDiscoveryDB(t))
	docker, _ := fakeDocker(t, `[{"Id":"1","Names":["/web"],"Image":"nginx","State":"running"}]`)
	second, _ := fakeDocker(t, `[{"Id":"9","Names":["/web"],"Image":"caddy","State":"running"}]`)
	// Portainer that refuses the key, echoing it like some proxies do
	refusing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad key "+r.Header.Get("X-API-Key"), http.StatusUnauthorized)
	}))
	t.Cleanup(refusing.Close)

	a := f.addDocker(t, discoveryAdminID, "A", docker.URL)
	b := f.addDocker(t, discoveryAdminID, "B", second.URL)
	_, err := f.integrations.Create(context.Background(), discoveryAdminID, true, &models.IntegrationRequest{
		Kind: "docker", Name: "Portainer", BaseURL: refusing.URL + "/api/endpoints/1/docker", AuthType: models.IntegrationAuthAPIKey,
		Credentials: &models.IntegrationCredentials{APIKey: discoverySecret},
	})
	require.NoError(t, err)

	result := discover(t, f.app(discoveryAdminID, "admin"))
	assert.Equal(t, 3, result.Integrations)
	require.Len(t, result.Containers, 2, "the same container name on two hosts is two containers")
	assert.ElementsMatch(t, []string{a.ID + "/web", b.ID + "/web"}, []string{result.Containers[0].Key, result.Containers[1].Key})
	require.Len(t, result.Errors, 1, "a failing integration does not hide the others")
	assert.Equal(t, "Portainer", result.Errors[0].IntegrationName)
	assert.NotEmpty(t, result.Errors[0].Error)
}

func TestDiscoveryService_AutoAdd(t *testing.T) {
	db := setupDiscoveryDB(t)
	f := newDiscoveryFixture(t, db)
	docker, requests := fakeDocker(t, discoveryContainers)
	integration := f.addDocker(t, discoveryAdminID, "Homelab", docker.URL)
	f.addDocker(t, discoveryUserID, "Demoted admin", docker.URL) // owner is no admin (any more)
	ctx := context.Background()
	serviceNames := func() []string {
		rows, err := db.Query(`SELECT name FROM services WHERE user_id = $1 ORDER BY name`, discoveryAdminID)
		require.NoError(t, err)
		defer rows.Close()
		names := []string{}
		for rows.Next() {
			var name string
			require.NoError(t, rows.Scan(&name))
			names = append(names, name)
		}
		return names
	}

	// Off by default: nothing is added and Docker is not even asked
	added, err := f.service.AutoAdd(ctx)
	require.NoError(t, err)
	assert.Zero(t, added)
	assert.Zero(t, *requests)
	assert.Empty(t, serviceNames())

	_, err = db.Exec(`UPDATE system_settings SET value = 'true' WHERE key = 'discovery_auto_add'`)
	require.NoError(t, err)

	// Only nimbus.enable=true with a valid URL label: not jellyfin (homepage
	// labels only), not web (unlabelled), not evil (enabled, but no usable URL)
	added, err = f.service.AutoAdd(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, added)
	assert.Equal(t, []string{"Sonarr"}, serviceNames())
	assert.Equal(t, 1, *requests, "the demoted admin's integration is not read")
	var total int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM services`).Scan(&total))
	assert.Equal(t, 1, total, "nothing is added for other users")

	// Idempotent, also across a restart
	for _, service := range []*services.DiscoveryService{f.service, newDiscoveryFixture(t, db).service} {
		added, err = service.AutoAdd(ctx)
		require.NoError(t, err)
		assert.Zero(t, added)
	}
	assert.Equal(t, []string{"Sonarr"}, serviceNames())

	// A deleted service is not brought back
	_, err = db.Exec(`DELETE FROM services`)
	require.NoError(t, err)
	added, err = f.service.AutoAdd(ctx)
	require.NoError(t, err)
	assert.Zero(t, added)
	assert.Empty(t, serviceNames())

	// Until the admin un-ignores it; an ignored container is never added
	require.NoError(t, f.service.Unignore(ctx, discoveryAdminID, integration.ID+"/sonarr"))
	added, err = f.service.AutoAdd(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, added)
	assert.Equal(t, []string{"Sonarr"}, serviceNames())
}

func TestDiscoveryService_AutoAddSkipsIgnored(t *testing.T) {
	db := setupDiscoveryDB(t)
	f := newDiscoveryFixture(t, db)
	docker, _ := fakeDocker(t, discoveryContainers)
	integration := f.addDocker(t, discoveryAdminID, "Homelab", docker.URL)
	_, err := db.Exec(`UPDATE system_settings SET value = 'true' WHERE key = 'discovery_auto_add'`)
	require.NoError(t, err)
	ctx := context.Background()

	require.NoError(t, f.service.Ignore(ctx, discoveryAdminID, integration.ID+"/sonarr"))
	added, err := f.service.AutoAdd(ctx)
	require.NoError(t, err)
	assert.Zero(t, added)
}

func TestSettingsHandler_DiscoveryAutoAddIsBoolean(t *testing.T) {
	db := setupDiscoveryDB(t)
	handler := NewSettingsHandler(repository.NewSettingsRepository(db), nil)
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", discoveryAdminID)
		return c.Next()
	})
	app.Put("/settings/:key", handler.UpdateSetting)

	status, _ := discoveryRequest(t, app, http.MethodPut, "/settings/discovery_auto_add", `{"value":"yes"}`)
	assert.Equal(t, fiber.StatusBadRequest, status)
	status, body := discoveryRequest(t, app, http.MethodPut, "/settings/discovery_auto_add", `{"value":"true"}`)
	assert.Equal(t, fiber.StatusOK, status, body)
	enabled, err := newDiscoveryFixture(t, db).service.AutoAddEnabled(context.Background())
	require.NoError(t, err)
	assert.True(t, enabled)
}

// The container list is reused for a moment, so the dashboard pill does not
// ask Docker on every page load; Refresh asks again
func TestDiscoveryHandler_CachesTheContainerList(t *testing.T) {
	f := newDiscoveryFixture(t, setupDiscoveryDB(t))
	docker, requests := fakeDocker(t, discoveryContainers)
	integration := f.addDocker(t, discoveryAdminID, "Homelab", docker.URL)
	app := f.app(discoveryAdminID, "admin")

	first := discover(t, app)
	status, body := discoveryRequest(t, app, http.MethodPost, "/discovery/docker/ignored", keyBody(integration.ID+"/web"))
	require.Equal(t, fiber.StatusNoContent, status, body)
	second := discover(t, app)
	assert.Equal(t, 1, *requests, "Docker is asked once")
	assert.Len(t, second.Containers, len(first.Containers))
	assert.Equal(t, []string{"web"}, containerNames(second, true), "added and ignored state is never cached")

	status, body = discoveryRequest(t, app, http.MethodGet, "/discovery/docker?refresh=true", "")
	require.Equal(t, fiber.StatusOK, status, body)
	assert.Equal(t, 2, *requests)
}

// A label can't point the health checker at a cloud metadata endpoint
func TestDiscoveryService_DropsMetadataURLLabel(t *testing.T) {
	db := setupDiscoveryDB(t)
	f := newDiscoveryFixture(t, db)
	docker, _ := fakeDocker(t, `[{"Id":"1","Names":["/meta"],"Image":"x","State":"running",
		"Labels":{"nimbus.enable":"true","nimbus.url":"http://169.254.169.254/latest/meta-data/"}}]`)
	f.addDocker(t, discoveryAdminID, "Homelab", docker.URL)
	_, err := db.Exec(`UPDATE system_settings SET value = 'true' WHERE key = 'discovery_auto_add'`)
	require.NoError(t, err)

	result := discover(t, f.app(discoveryAdminID, "admin"))
	require.Len(t, result.Containers, 1)
	assert.Empty(t, result.Containers[0].URL)

	added, err := f.service.AutoAdd(context.Background())
	require.NoError(t, err)
	assert.Zero(t, added)
}

// When the service can't be created the claim is given back: the container
// must not end up hidden as "ignored"
func TestDiscoveryHandler_FailedAddReleasesTheContainer(t *testing.T) {
	db := setupDiscoveryDB(t)
	f := newDiscoveryFixture(t, db)
	docker, _ := fakeDocker(t, discoveryContainers)
	integration := f.addDocker(t, discoveryAdminID, "Homelab", docker.URL)
	_, err := db.Exec(`CREATE TRIGGER services_fail BEFORE INSERT ON services BEGIN SELECT RAISE(ABORT, 'boom'); END`)
	require.NoError(t, err)

	_, err = f.service.Add(context.Background(), discoveryAdminID, &models.DiscoveryAddRequest{
		Key: integration.ID + "/web", Name: "Web", URL: "http://web.lan",
	})
	require.Error(t, err)
	var claims int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM discovered_containers`).Scan(&claims))
	assert.Zero(t, claims)
}

// Adds running at once (two clicks, or a click during an auto-add run) share
// one new group instead of each creating it
func TestDiscoveryService_ConcurrentAddsCreateOneGroup(t *testing.T) {
	db := setupDiscoveryDB(t)
	f := newDiscoveryFixture(t, db)
	docker, _ := fakeDocker(t, discoveryContainers)
	integration := f.addDocker(t, discoveryAdminID, "Homelab", docker.URL)

	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := f.service.Add(context.Background(), discoveryAdminID, &models.DiscoveryAddRequest{
				Key: fmt.Sprintf("%s/app%d", integration.ID, i), Name: "App", URL: "http://app.lan", Group: "Media",
			})
			assert.NoError(t, err)
		}()
	}
	wg.Wait()
	var groups, linked int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM groups`).Scan(&groups))
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM discovered_containers WHERE service_id IS NOT NULL`).Scan(&linked))
	assert.Equal(t, 1, groups)
	assert.Equal(t, 8, linked)
}

// One run adds a bounded number of containers; the rest follow on later runs
func TestDiscoveryService_AutoAddIsBoundedPerRun(t *testing.T) {
	db := setupDiscoveryDB(t)
	f := newDiscoveryFixture(t, db)
	containers := []string{}
	for i := range 25 {
		containers = append(containers, fmt.Sprintf(
			`{"Id":"%d","Names":["/app%d"],"Image":"x","State":"running","Labels":{"nimbus.enable":"true","nimbus.url":"http://app%d.lan"}}`, i, i, i))
	}
	docker, _ := fakeDocker(t, "["+strings.Join(containers, ",")+"]")
	f.addDocker(t, discoveryAdminID, "Homelab", docker.URL)
	_, err := db.Exec(`UPDATE system_settings SET value = 'true' WHERE key = 'discovery_auto_add'`)
	require.NoError(t, err)

	for _, want := range []int{20, 5, 0} {
		added, err := f.service.AutoAdd(context.Background())
		require.NoError(t, err)
		assert.Equal(t, want, added)
	}
}

// Auto-add takes only an address from nimbus.url; homepage.href is offered in
// the tray, but someone checks it before it is added
func TestDiscoveryService_AutoAddNeedsNimbusURL(t *testing.T) {
	db := setupDiscoveryDB(t)
	f := newDiscoveryFixture(t, db)
	docker, _ := fakeDocker(t, `[{"Id":"1","Names":["/jellyfin"],"Image":"jellyfin/jellyfin","State":"running",
		"Labels":{"nimbus.enable":"true","homepage.href":"http://jellyfin.lan:8096"}}]`)
	f.addDocker(t, discoveryAdminID, "Homelab", docker.URL)
	_, err := db.Exec(`UPDATE system_settings SET value = 'true' WHERE key = 'discovery_auto_add'`)
	require.NoError(t, err)

	result := discover(t, f.app(discoveryAdminID, "admin"))
	require.Len(t, result.Containers, 1)
	assert.Equal(t, "http://jellyfin.lan:8096", result.Containers[0].URL)
	assert.False(t, result.Containers[0].URLGuessed)

	added, err := f.service.AutoAdd(context.Background())
	require.NoError(t, err)
	assert.Zero(t, added)
}

// Un-ignoring while an add is in progress leaves its claim, so a second add
// can't add the same container again
func TestDiscoveryService_UnignoreDuringAnAddKeepsItsClaim(t *testing.T) {
	db := setupDiscoveryDB(t)
	f := newDiscoveryFixture(t, db)
	docker, _ := fakeDocker(t, discoveryContainers)
	integration := f.addDocker(t, discoveryAdminID, "Homelab", docker.URL)
	repo := repository.NewDiscoveryRepository(db)
	ctx := context.Background()
	key := integration.ID + "/web"

	// The first add has claimed the container, its service is not created yet
	claimed, err := repo.Claim(ctx, discoveryAdminID, key, true)
	require.NoError(t, err)
	require.True(t, claimed)

	require.NoError(t, f.service.Unignore(ctx, discoveryAdminID, key))
	_, err = f.service.Add(ctx, discoveryAdminID, &models.DiscoveryAddRequest{Key: key, Name: "Web", URL: "http://web.lan"})
	assert.ErrorIs(t, err, services.ErrDiscoveryHandled)

	// The first add finishes
	_, err = db.Exec(`INSERT INTO services (id, user_id, name, url, status, created_at, updated_at)
		VALUES ('s-1', $1, 'Web', 'http://web.lan', 'unknown', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, discoveryAdminID)
	require.NoError(t, err)
	require.NoError(t, repo.SetService(ctx, discoveryAdminID, key, "s-1"))

	var count, linked int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM services`).Scan(&count))
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM discovered_containers WHERE service_id = 's-1' AND NOT pending`).Scan(&linked))
	assert.Equal(t, 1, count, "the container became one service")
	assert.Equal(t, 1, linked)
}
