package services

import (
	"testing"

	"github.com/nimbus/backend/internal/integrations"
	"github.com/nimbus/backend/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestGuessURL(t *testing.T) {
	tests := []struct {
		name       string
		baseURL    string
		socketHost string
		port       int
		want       string
	}{
		{"socket proxy host", "http://192.168.1.5:2375", "nimbus.lan", 8989, "http://192.168.1.5:8989"},
		{"portainer host, sub path dropped", "https://portainer.lan:9443/api/endpoints/1/docker", "", 8096, "http://portainer.lan:8096"},
		{"unix socket uses the host Nimbus was reached on", "unix:///var/run/docker.sock", "nimbus.lan", 8080, "http://nimbus.lan:8080"},
		{"unix socket without a known host", "unix:///var/run/docker.sock", "", 8080, "http://localhost:8080"},
		{"port 80", "http://docker.lan:2375", "", 80, "http://docker.lan"},
		{"port 443", "http://docker.lan:2375", "", 443, "https://docker.lan"},
		{"ipv6 host", "http://[fd00::5]:2375", "", 8080, "http://[fd00::5]:8080"},
		{"ipv6 host on port 80", "unix:///var/run/docker.sock", "fd00::5", 80, "http://[fd00::5]"},
		{"no published port", "http://docker.lan:2375", "nimbus.lan", 0, ""},
		{"broken base URL", "http://", "nimbus.lan", 8080, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := guessURL(tt.baseURL, tt.socketHost, tt.port)
			assert.Equal(t, tt.want, got)
			if got != "" {
				assert.True(t, validServiceURL(got), "a guess must pass the same check as a label")
			}
		})
	}
}

func TestDiscoveryCandidate(t *testing.T) {
	integration := &models.Integration{ID: "i-1", Name: "Homelab", BaseURL: "unix:///var/run/docker.sock"}
	withIcons := &DiscoveryService{iconBase: "https://icons.example/svg"}
	candidate := func(s *DiscoveryService, labels integrations.DiscoveryLabels) models.DiscoveredContainer {
		return s.candidate(integration, integrations.DiscoveredContainer{
			Name: "app", Image: "lscr.io/linuxserver/sonarr:latest", State: "running", Port: 8989, Labels: labels,
		}, "nimbus.lan")
	}

	guessed := candidate(withIcons, integrations.DiscoveryLabels{})
	assert.Equal(t, "i-1/app", guessed.Key)
	assert.Equal(t, "app", guessed.Name)
	assert.Equal(t, "http://nimbus.lan:8989", guessed.URL)
	assert.True(t, guessed.URLGuessed)
	assert.Equal(t, "https://icons.example/svg/sonarr.svg", guessed.IconURL)
	assert.Nil(t, guessed.Integration)

	for _, bad := range []string{"javascript:alert(1)", "javascript://x/%0aalert(1)", "ftp://files.lan", "//sonarr.lan", "sonarr.lan", "data:text/html,x"} {
		got := candidate(withIcons, integrations.DiscoveryLabels{URL: bad})
		assert.Equal(t, "http://nimbus.lan:8989", got.URL, "%s falls back to the guess", bad)
		assert.True(t, got.URLGuessed, bad)
	}

	icons := map[string]string{
		"radarr":                     "https://icons.example/svg/radarr.svg",
		"Radarr.png":                 "https://icons.example/svg/radarr.svg",
		"https://cdn.lan/radarr.png": "https://cdn.lan/radarr.png",
		"mdi-television":             "https://icons.example/svg/sonarr.svg", // not in dashboard-icons: guessed from the image
		"javascript:alert(1)":        "https://icons.example/svg/sonarr.svg",
		"../../x":                    "https://icons.example/svg/sonarr.svg",
	}
	for icon, want := range icons {
		assert.Equal(t, want, candidate(withIcons, integrations.DiscoveryLabels{Icon: icon}).IconURL, icon)
	}
	// DASHBOARD_ICONS_BASE_URL set to empty turns curated icons off
	withoutIcons := &DiscoveryService{}
	assert.Empty(t, candidate(withoutIcons, integrations.DiscoveryLabels{Icon: "radarr"}).IconURL)
	assert.Equal(t, "https://cdn.lan/radarr.png", candidate(withoutIcons, integrations.DiscoveryLabels{Icon: "https://cdn.lan/radarr.png"}).IconURL)

	// nimbus.widget suggests an integration only for a kind the server knows
	known := candidate(withIcons, integrations.DiscoveryLabels{Widget: "sonarr", URL: "https://sonarr.lan"})
	assert.Equal(t, &models.DiscoveredIntegration{Kind: "sonarr", Name: "Sonarr", BaseURL: "https://sonarr.lan"}, known.Integration)
	assert.Nil(t, candidate(withIcons, integrations.DiscoveryLabels{Widget: "no-such-kind", URL: "https://sonarr.lan"}).Integration)
}
