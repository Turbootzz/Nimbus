package integrations

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/nimbus/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDiscoveryLabels(t *testing.T) {
	long := strings.Repeat("é", 300)
	tests := []struct {
		name   string
		labels map[string]string
		want   DiscoveryLabels
		hidden bool
	}{
		{name: "no labels"},
		{
			name: "all nimbus labels",
			labels: map[string]string{
				"nimbus.enable": "true", "nimbus.name": " Sonarr ", "nimbus.icon": "sonarr",
				"nimbus.url": "https://sonarr.lan", "nimbus.group": "Media",
				"nimbus.description": "Series", "nimbus.widget": "Sonarr",
			},
			want: DiscoveryLabels{Enabled: true, Name: "Sonarr", Icon: "sonarr", URL: "https://sonarr.lan", Group: "Media", Description: "Series", Widget: "sonarr"},
		},
		{
			name: "homepage fallback",
			labels: map[string]string{
				"homepage.name": "Jellyfin", "homepage.href": "http://jellyfin.lan:8096", "homepage.icon": "jellyfin.png",
				"homepage.group": "Media", "homepage.description": "Films",
			},
			want: DiscoveryLabels{Name: "Jellyfin", URL: "http://jellyfin.lan:8096", Icon: "jellyfin.png", Group: "Media", Description: "Films"},
		},
		{
			name: "nimbus wins over homepage, field by field",
			labels: map[string]string{
				"nimbus.name": "Mine", "homepage.name": "Theirs", "nimbus.url": " ", "homepage.href": "http://a.lan",
			},
			want: DiscoveryLabels{Name: "Mine", URL: "http://a.lan"},
		},
		{name: "enable is case-insensitive", labels: map[string]string{"nimbus.enable": "TRUE"}, want: DiscoveryLabels{Enabled: true}},
		{name: "enable=false hides", labels: map[string]string{"nimbus.enable": "False", "nimbus.name": "x"}, want: DiscoveryLabels{Name: "x"}, hidden: true},
		{name: "enable=yes is not true", labels: map[string]string{"nimbus.enable": "yes"}},
		{
			name: "other labels are ignored",
			labels: map[string]string{
				"traefik.http.middlewares.auth.basicauth.users": "admin:$apr1$secret", "nimbus.secret": "s3cret", "homepage.widget.key": "apikey",
			},
		},
		{
			name: "long values are cut or dropped",
			labels: map[string]string{
				"nimbus.name": long, "nimbus.group": long, "nimbus.description": long,
				"nimbus.url": "http://a.lan/" + strings.Repeat("a", 2048), "nimbus.icon": "https://a.lan/" + strings.Repeat("a", 2048),
			},
			want: DiscoveryLabels{Name: strings.Repeat("é", 50), Group: strings.Repeat("é", 17), Description: strings.Repeat("é", 250)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, hidden := parseDiscoveryLabels(tt.labels)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.hidden, hidden)
		})
	}
}

func TestIconSlugs(t *testing.T) {
	images := map[string]string{
		"lscr.io/linuxserver/sonarr:latest": "sonarr",
		"nginx":                             "nginx",
		"nginx:1.27":                        "nginx",
		"ghcr.io/home-assistant/home-assistant:stable":     "home-assistant",
		"registry.lan:5000/team/app@sha256:abcdef":         "app",
		"registry.lan:5000/app":                            "app",
		"postgres:18@sha256:abcdef":                        "postgres",
		"docker.io/library/My_App:1":                       "my-app",
		"":                                                 "",
		"weird/../..":                                      "",
		"evil/<script>:1":                                  "",
		"sha256:0123456789abcdef":                          "sha256",
		"ghcr.io/org/app-with-dashes.and.dots:v1":          "",
		"lscr.io/linuxserver/qbittorrent:version-4.6.0-r0": "qbittorrent",
	}
	for image, want := range images {
		assert.Equal(t, want, ImageIconSlug(image), image)
	}

	icons := map[string]string{
		"sonarr": "sonarr", " Sonarr.PNG ": "sonarr", "home-assistant.svg": "home-assistant", "plex.webp": "plex",
		"mdi-home": "", "si-github": "", "sh-plex": "", "../../etc/passwd": "", "a b": "", "": "", "-a": "",
		"https://cdn.lan/icon.png": "",
	}
	for icon, want := range icons {
		assert.Equal(t, want, IconSlug(icon), icon)
	}
}

const dockerDiscoveryFixture = `[
	{"Id":"1","Names":["/sonarr"],"Image":"lscr.io/linuxserver/sonarr:latest","State":"running",
	 "Ports":[{"IP":"::","PrivatePort":8989,"PublicPort":8989,"Type":"tcp"},{"IP":"0.0.0.0","PrivatePort":8989,"PublicPort":8989,"Type":"tcp"}],
	 "Labels":{"nimbus.enable":"true","nimbus.name":"Sonarr","nimbus.url":"https://sonarr.lan","traefik.auth":"admin:hunter2"}},
	{"Id":"2","Names":["/web"],"Image":"nginx:1.27","State":"running",
	 "Ports":[{"PrivatePort":443,"PublicPort":8443,"Type":"tcp"},{"PrivatePort":53,"PublicPort":53,"Type":"udp"},{"PrivatePort":80,"PublicPort":8080,"Type":"tcp"},{"PrivatePort":9000,"Type":"tcp"}]},
	{"Id":"3","Names":["/hidden"],"Image":"redis","State":"running","Labels":{"nimbus.enable":"false"}},
	{"Id":"4","Names":[],"Image":"busybox","State":"paused"},
	{"Id":"5","Names":["/backup"],"Image":"restic","State":"exited","Ports":[]}
]`

func TestDockerDiscover(t *testing.T) {
	server := fakeApp(t, map[string]http.HandlerFunc{
		"GET /containers/json": func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "1", r.URL.Query().Get("all"))
			fmt.Fprint(w, dockerDiscoveryFixture)
		},
	})
	conn := newTestConn(server.URL, models.IntegrationCredentials{})

	containers, err := DockerDiscover(context.Background(), conn)
	require.NoError(t, err)
	assert.Equal(t, []DiscoveredContainer{
		{Name: "sonarr", Image: "lscr.io/linuxserver/sonarr:latest", State: "running", Port: 8989,
			Labels: DiscoveryLabels{Enabled: true, Name: "Sonarr", URL: "https://sonarr.lan"}},
		{Name: "web", Image: "nginx:1.27", State: "running", Port: 8080},
		{Name: "backup", Image: "restic", State: "exited"},
	}, containers, "hidden and nameless containers are left out; the lowest published TCP port wins")
	assert.NotContains(t, fmt.Sprintf("%+v", containers), "hunter2", "other labels must not be kept")

	// The widget payload never carries labels or ports
	listed, err := DockerContainers(context.Background(), conn)
	require.NoError(t, err)
	assert.NotContains(t, fmt.Sprintf("%+v", listed), "hunter2")
}
