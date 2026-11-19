package importers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nimbus/backend/internal/dashdoc"
	"github.com/nimbus/backend/internal/integrations"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The fixtures under testdata are sample configs from each project's own
// documentation; every file names its source. homarr_default.json is
// https://github.com/ajnart/homarr/blob/dev/data/configs/default.json (the
// board Homarr 0.x ships with), and homarr_board.json is a board in that same
// schema with categories and integrations.

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return data
}

// convert converts a fixture, detecting its format, and checks the result is
// a valid document
func convert(t *testing.T, name, format string) *Result {
	t.Helper()
	result, err := Convert("auto", fixture(t, name), Options{Host: "nas.lan"})
	require.NoError(t, err)
	assert.Equal(t, format, result.Format, "detected format")
	require.NoError(t, dashdoc.Validate(&result.Document))
	assertImportable(t, &result.Document)
	return result
}

// assertImportable checks what the import would refuse an entity for
func assertImportable(t *testing.T, doc *dashdoc.Document) {
	t.Helper()
	raw, err := json.Marshal(doc)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), `\u0000`, "control characters are refused")
	assert.NotContains(t, string(raw), `\u0007`)
	for _, group := range doc.Groups {
		assert.NotEmpty(t, group.Name)
		assert.LessOrEqual(t, len(group.Name), models.MaxGroupNameLen)
	}
	for _, service := range doc.Services {
		assert.NotEmpty(t, service.Name)
		assert.NoError(t, utils.ValidateServiceURL(service.URL), "the rule the import applies")
		if service.IconType == models.IconTypeImageURL {
			_, ok := webURL(service.IconURL)
			assert.True(t, ok, "icon URL %q", service.IconURL)
		}
	}
	for _, integration := range doc.Integrations {
		_, known := integrations.Get(integration.Kind)
		assert.True(t, known, "kind %q", integration.Kind)
		_, ok := webURL(integration.BaseURL)
		assert.True(t, ok, "base URL %q", integration.BaseURL)
		assert.NotContains(t, integration.BaseURL, "?")
		assert.Nil(t, integration.Credentials)
		assert.Empty(t, integration.AuthType, "the server picks the kind's default")
		assert.Empty(t, integration.Options, "options a kind doesn't declare are refused")
	}
}

func serviceNames(doc dashdoc.Document) []string {
	names := make([]string, len(doc.Services))
	for i, service := range doc.Services {
		names[i] = service.Name
	}
	return names
}

func groupNames(doc dashdoc.Document) []string {
	names := make([]string, len(doc.Groups))
	for i, group := range doc.Groups {
		names[i] = group.Name
	}
	return names
}

func find(t *testing.T, doc dashdoc.Document, name string) dashdoc.Service {
	t.Helper()
	for _, service := range doc.Services {
		if service.Name == name {
			return service
		}
	}
	require.Failf(t, "service not found", "%s in %v", name, serviceNames(doc))
	return dashdoc.Service{}
}

func groupOf(doc dashdoc.Document, service dashdoc.Service) string {
	for _, group := range doc.Groups {
		if group.Ref == service.GroupRef {
			return group.Name
		}
	}
	return ""
}

func integrationOf(doc dashdoc.Document, service dashdoc.Service) *dashdoc.Integration {
	for i, integration := range doc.Integrations {
		if integration.Ref == service.IntegrationRef && service.IntegrationRef != "" {
			return &doc.Integrations[i]
		}
	}
	return nil
}

func warned(t *testing.T, warnings []string, parts ...string) {
	t.Helper()
	for _, warning := range warnings {
		all := true
		for _, part := range parts {
			all = all && strings.Contains(warning, part)
		}
		if all {
			return
		}
	}
	assert.Failf(t, "missing warning", "%v in:\n%s", parts, strings.Join(warnings, "\n"))
}

func TestKindsExist(t *testing.T) {
	for app, kind := range kinds {
		_, ok := integrations.Get(kind)
		assert.True(t, ok, "%s maps to %q, which is not a registered kind", app, kind)
	}
	assert.Equal(t, "adguard", kindOf("AdGuardHome"))
	assert.Equal(t, "home_assistant", kindOf("homeassistant"))
	assert.Equal(t, "uptime_kuma", kindOf("Uptime-Kuma"))
	assert.Empty(t, kindOf("emby"))
}

func TestHomepageServices(t *testing.T) {
	result := convert(t, "homepage_services.yaml", FormatHomepage)
	doc := result.Document

	assert.Equal(t, []string{"Media", "Downloads", "Network", "Home"}, groupNames(doc), "a nested group becomes its own group")
	assert.Equal(t, []string{
		"Emby", "Sonarr", "Radarr", "Jellyfin", "qBittorrent", "Pi-hole", "AdGuard Home", "UptimeRobot",
		"Proxmox", "Home Assistant", "Service", "Local icon",
	}, serviceNames(doc))

	sonarr := find(t, doc, "Sonarr")
	assert.Equal(t, "Media", groupOf(doc, sonarr))
	assert.Equal(t, "Series management", sonarr.Description)
	assert.Equal(t, "https://cdn.jsdelivr.net/gh/homarr-labs/dashboard-icons/png/sonarr.png", sonarr.IconURL)
	assert.Equal(t, &dashdoc.Integration{
		Ref: sonarr.IntegrationRef, Kind: "sonarr", Name: "Sonarr", BaseURL: "http://sonarr.host.or.ip", NeedsCredentials: true,
	}, integrationOf(doc, sonarr))

	kindsByService := map[string]string{}
	for _, service := range doc.Services {
		if integration := integrationOf(doc, service); integration != nil {
			kindsByService[service.Name] = integration.Kind
		}
	}
	assert.Equal(t, map[string]string{
		"Sonarr": "sonarr", "Radarr": "radarr", "Jellyfin": "jellyfin", "qBittorrent": "qbittorrent",
		"Pi-hole": "pihole", "AdGuard Home": "adguard", "Proxmox": "proxmox", "Home Assistant": "home_assistant",
	}, kindsByService)

	// Icons: prefixed sets, a colour, a format, a name with a dash
	assert.Equal(t, "https://cdn.simpleicons.org/jellyfin/a712a2", find(t, doc, "Jellyfin").IconURL)
	assert.Equal(t, "https://cdn.jsdelivr.net/gh/selfhst/icons@main/svg/qbittorrent.svg", find(t, doc, "qBittorrent").IconURL)
	assert.Equal(t, "https://cdn.jsdelivr.net/gh/homarr-labs/dashboard-icons/png/adguard-home.png", find(t, doc, "AdGuard Home").IconURL)
	assert.Equal(t, "https://cdn.jsdelivr.net/npm/@mdi/svg@latest/svg/flask-outline.svg", find(t, doc, "Service").IconURL)
	assert.Empty(t, find(t, doc, "Local icon").IconType)

	// A service without href links to its widget's address
	assert.Equal(t, "http://homeassistant.host.or.ip:8123", find(t, doc, "Home Assistant").URL)

	// Of Emby's two widgets Nimbus knows the second, whose address is a placeholder
	warned(t, result.Warnings, "Emby", "uptime_kuma integration is skipped")
	warned(t, result.Warnings, "UptimeRobot", `no "uptimerobot" integration`)
	warned(t, result.Warnings, "Uptime Kuma", "skipped", "host.or.ip:port")
	warned(t, result.Warnings, "Local icon", "/icons/myicon.png")

	// No secret of the config ends up anywhere
	all := strings.Join(result.Warnings, "\n") + toJSON(t, doc)
	for _, secret := range []string{"apikeyapikey", "SONARR_API_KEY", "yourpiholeapikey", "api_token_secret", "access_token", "your-secret-here", "password"} {
		assert.NotContains(t, all, secret)
	}
}

func TestHomepageSkeletonAndBookmarks(t *testing.T) {
	skeleton := convert(t, "homepage_skeleton.yaml", FormatHomepage)
	assert.Equal(t, []string{"My First Group", "My Second Group", "My Third Group"}, groupNames(skeleton.Document))
	assert.Equal(t, []string{"My First Service", "My Second Service", "My Third Service"}, serviceNames(skeleton.Document))
	assert.Equal(t, "Homepage is 😎", find(t, skeleton.Document, "My Third Service").Description)
	assert.Empty(t, skeleton.Warnings)

	bookmarks := convert(t, "homepage_bookmarks.yaml", FormatHomepage)
	assert.Equal(t, []string{"Developer", "Social", "Entertainment"}, groupNames(bookmarks.Document))
	assert.Equal(t, []string{"Github", "Reddit", "YouTube"}, serviceNames(bookmarks.Document))
	reddit := find(t, bookmarks.Document, "Reddit")
	assert.Equal(t, "https://reddit.com/", reddit.URL)
	assert.Equal(t, "The front page of the internet", reddit.Description)
	assert.Contains(t, reddit.IconURL, "/png/reddit.png")
}

func TestHomarr(t *testing.T) {
	shipped := convert(t, "homarr_default.json", FormatHomarr)
	assert.Equal(t, []string{"Contribute", "Discord", "Donate", "Documentation"}, serviceNames(shipped.Document), "by row, then column")
	assert.Empty(t, shipped.Document.Groups)
	assert.Equal(t, "https://homarr.dev", find(t, shipped.Document, "Documentation").URL)
	warned(t, shipped.Warnings, "4 Homarr widgets", "date, notebook, weather")
	warned(t, shipped.Warnings, "Documentation", "/imgs/logo/logo.png")

	board := convert(t, "homarr_board.json", FormatHomarr)
	doc := board.Document
	assert.Equal(t, []string{"Media", "Network"}, groupNames(doc), "categories by position")
	assert.Equal(t, []string{"Pi-hole", "Sonarr", "Radarr", "Plex"}, serviceNames(doc))

	sonarr := find(t, doc, "Sonarr")
	assert.Equal(t, "https://sonarr.example.com", sonarr.URL, "the external URL is the link")
	assert.Equal(t, "Media", groupOf(doc, sonarr))
	require.NotNil(t, integrationOf(doc, sonarr))
	assert.Equal(t, "http://192.168.1.10:8989", integrationOf(doc, sonarr).BaseURL, "the internal URL reaches the app")
	assert.Equal(t, "pihole", integrationOf(doc, find(t, doc, "Pi-hole")).Kind)
	assert.Empty(t, find(t, doc, "Plex").GroupRef)
	assert.Contains(t, find(t, doc, "Radarr").IconURL, "walkxcode/dashboard-icons/png/radarr.png")

	warned(t, board.Warnings, "Plex", `no "plex" integration`)
	warned(t, board.Warnings, "2 Homarr widgets", "calendar, torrents-status")
	assert.NotContains(t, toJSON(t, doc), "0123456789abcdef")
}

func TestHomer(t *testing.T) {
	result := convert(t, "homer_config.yml", FormatHomer)
	doc := result.Document
	assert.Equal(t, []string{"Application", "Other group"}, groupNames(doc))
	assert.Equal(t, []string{"Awesome app", "Pi-hole"}, serviceNames(doc))

	app := find(t, doc, "Awesome app")
	assert.Equal(t, "https://www.reddit.com/r/selfhosted/", app.URL)
	assert.Equal(t, "Bookmark example", app.Description)

	pihole := find(t, doc, "Pi-hole")
	require.NotNil(t, integrationOf(doc, pihole))
	assert.Equal(t, "pihole", integrationOf(doc, pihole).Kind)
	assert.Equal(t, "http://192.168.0.151/admin", integrationOf(doc, pihole).BaseURL)

	warned(t, result.Warnings, "Another one", "skipped", `"#"`)
	warned(t, result.Warnings, "Awesome app", "assets/tools/sample.png")
	warned(t, result.Warnings, "3 navbar links")
}

func TestDashy(t *testing.T) {
	result := convert(t, "dashy_conf.yml", FormatDashy)
	doc := result.Document
	assert.Equal(t, []string{"Getting Started", "Support Dashy"}, groupNames(doc))
	assert.Equal(t, []string{"GitHub", "Docs", "Sponsor"}, serviceNames(doc))

	github := find(t, doc, "GitHub")
	assert.Equal(t, "https://cdn.simpleicons.org/github", github.IconURL)
	assert.Equal(t, "Source Code, Issues and Pull Requests", github.Description)
	assert.Equal(t, "🩷", find(t, doc, "Sponsor").Icon, "an emoji stays an emoji")

	warned(t, result.Warnings, "Docs", "far fa-book")
	warned(t, result.Warnings, "2 Dashy widgets")
}

func TestCompose(t *testing.T) {
	result := convert(t, "compose_media.yml", FormatCompose)
	doc := result.Document
	assert.Equal(t, []string{"Home Assistant", "Jellyfin", "Pi-hole", "Sonarr", "Traefik"}, serviceNames(doc))

	sonarr := find(t, doc, "Sonarr")
	assert.Equal(t, "http://nas.lan:8989", sonarr.URL)
	assert.Equal(t, "https://cdn.jsdelivr.net/gh/homarr-labs/dashboard-icons/svg/sonarr.svg", sonarr.IconURL, "registry, organisation and tag are gone")
	assert.Equal(t, "http://nas.lan:8123", find(t, doc, "Home Assistant").URL)
	assert.Contains(t, find(t, doc, "Home Assistant").IconURL, "/svg/home-assistant.svg")
	assert.Equal(t, "http://nas.lan:8096", find(t, doc, "Jellyfin").URL, "the long port syntax")
	assert.Contains(t, find(t, doc, "Jellyfin").IconURL, "/svg/jellyfin.svg", "a digest is gone too")

	// Homepage's labels are used; the UDP port is not a link
	pihole := find(t, doc, "Pi-hole")
	assert.Equal(t, "https://nas.lan:8443", pihole.URL)
	assert.Equal(t, "Network", groupOf(doc, pihole))
	assert.Equal(t, "DNS", pihole.Description)
	traefik := find(t, doc, "Traefik")
	assert.Equal(t, "https://proxy.example.com", traefik.URL)
	assert.Contains(t, traefik.IconURL, "@mdi/svg@latest/svg/router.svg")

	warned(t, result.Warnings, "db", "publishes no TCP port")
	warned(t, result.Warnings, "whoami", "publishes no TCP port")

	// Without a host the placeholder is localhost
	result, err := Convert(FormatCompose, fixture(t, "compose_media.yml"), Options{})
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:8989", find(t, result.Document, "Sonarr").URL)
}

func TestConvertChecksItsInput(t *testing.T) {
	compose := fixture(t, "compose_media.yml")
	for _, host := range []string{"http://nas.lan", "nas.lan:8080", "nas lan", "a/b", "-nas", "nas.lan?x=1", "user@nas", strings.Repeat("a", 300)} {
		_, err := Convert(FormatCompose, compose, Options{Host: host})
		assert.Error(t, err, "host %q", host)
	}
	for _, host := range []string{"nas.lan", "192.168.1.5", "[fe80::1]", "NAS"} {
		_, err := Convert(FormatCompose, compose, Options{Host: host})
		assert.NoError(t, err, "host %q", host)
	}

	_, err := Convert("glance", compose, Options{})
	assert.ErrorContains(t, err, "unknown format")
	_, err = Convert("", []byte("just some text"), Options{})
	assert.ErrorContains(t, err, "not recognised")
	_, err = Convert(FormatHomepage, []byte(strings.Repeat("a", MaxSize+1)), Options{})
	assert.ErrorContains(t, err, "larger than")
	// The format given wins over what the config looks like
	_, err = Convert(FormatDashy, compose, Options{})
	assert.ErrorContains(t, err, "not a Dashy")
}

func TestIconURL(t *testing.T) {
	cases := []struct {
		icon  string
		plain bool
		want  string
	}{
		{"https://example.com/a.png", false, "https://example.com/a.png"},
		{"sonarr", true, dashboardIconsURL + "png/sonarr.png"},
		{"Sonarr.SVG", true, dashboardIconsURL + "svg/sonarr.svg"},
		{"home-assistant.webp", true, dashboardIconsURL + "webp/home-assistant.webp"},
		{"hl-adguardhome", false, dashboardIconsURL + "svg/adguardhome.svg"},
		{"mdi-flask-outline", false, materialIconsURL + "flask-outline.svg"},
		{"mdi-flask-outline-#f0d453", false, materialIconsURL + "flask-outline.svg"},
		{"si-github", false, simpleIconsURL + "github"},
		{"si-github-#fff", false, simpleIconsURL + "github/fff"},
		{"sh-home-assistant", false, selfhstIconsURL + "png/home-assistant.png"},
		{"sh-home-assistant.svg", false, selfhstIconsURL + "svg/home-assistant.svg"},
		// Not something Nimbus can show
		{"sonarr.png", false, ""},
		{"/icons/a.png", true, ""},
		{"assets/tools/sample.png", true, ""},
		{"fas fa-rocket", true, ""},
		{"../../etc/passwd", true, ""},
		{"javascript:alert(1)", true, ""},
		{"data:image/png;base64,AAAA", true, ""},
		{"si-<script>", true, ""},
		{"", true, ""},
	}
	for _, tc := range cases {
		got, ok := iconURL(tc.icon, tc.plain)
		assert.Equal(t, tc.want, got, tc.icon)
		assert.Equal(t, tc.want != "", ok, tc.icon)
	}
}

func TestPublishedPortAndImageSlug(t *testing.T) {
	ports := map[string]int{
		"8080:80": 8080, "127.0.0.1:8081:80": 8081, "[::1]:8082:80/tcp": 8082, "9000-9005:80-85": 9000,
		"80": 0, "53:53/udp": 0, "${PORT}:80": 0, "0:80": 0, "99999:80": 0, "": 0, ":": 0, "a:b:c:d": 0,
	}
	for port, want := range ports {
		assert.Equal(t, want, publishedPort([]any{port}), port)
	}
	assert.Equal(t, 8443, publishedPort([]any{nil, 5, "53:53/udp", map[string]any{"published": 8443, "target": 443}}))

	images := map[string]string{
		"sonarr": "sonarr", "postgres:16": "postgres", "lscr.io/linuxserver/sonarr:latest": "sonarr",
		"localhost:5000/team/app:1.2@sha256:abc": "app", "ghcr.io/Home-Assistant/Home-Assistant": "home-assistant",
		"": "", "weird name": "", "${IMAGE}": "",
	}
	for image, want := range images {
		assert.Equal(t, want, imageSlug(image), image)
	}
}

func TestOddURLsAreSkipped(t *testing.T) {
	config := `
- Links:
    - Script:
        href: "javascript:alert(1)"
    - File:
        href: file:///etc/passwd
    - FTP:
        href: ftp://files.lan/
    - Relative:
        href: /admin
    - Variable:
        href: "{{HOMEPAGE_VAR_URL}}"
    - Login:
        href: https://user:pass@example.com/path?token=abc#frag
        widget:
          type: sonarr
          url: https://user:pass@example.com/sonarr/?apikey=abc#frag
    - Metadata:
        href: http://169.254.169.254/latest/meta-data/
    - Good:
        href: https://example.com
`
	result, err := Convert(FormatHomepage, []byte(config), Options{})
	require.NoError(t, err)
	assert.Equal(t, []string{"Login", "Good"}, serviceNames(result.Document))
	require.Len(t, result.Document.Integrations, 1)
	assert.Equal(t, "https://example.com/sonarr", result.Document.Integrations[0].BaseURL, "no login, query or fragment")
	assert.Len(t, result.Warnings, 6)
	assert.Contains(t, strings.Join(result.Warnings, "\n"), "Metadata: skipped, \"http://169.254.169.254/latest/meta-data/\" is not a URL Nimbus can link to: cloud metadata endpoints are not allowed")
	assertImportable(t, &result.Document)
}

func TestLimitsAndDuplicates(t *testing.T) {
	var config strings.Builder
	config.WriteString("- " + strings.Repeat("Long ", 20) + ":\n")
	for i := 0; i < dashdoc.MaxServices+5; i++ {
		// Twice the same service and the same app: refs must stay unique
		config.WriteString("    - Sonarr:\n        href: http://sonarr.lan\n        icon: fas fa-x\n        widget:\n          type: sonarr\n          url: http://sonarr.lan\n")
	}
	result, err := Convert(FormatHomepage, []byte(config.String()), Options{})
	require.NoError(t, err)
	require.NoError(t, dashdoc.Validate(&result.Document))
	assertImportable(t, &result.Document)
	assert.Len(t, result.Document.Services, dashdoc.MaxServices)
	assert.Len(t, result.Document.Integrations, 1, "one integration per app")
	assert.LessOrEqual(t, len(result.Warnings), maxWarnings+1)
	assert.Contains(t, result.Warnings[len(result.Warnings)-1], "more warnings")
}

// garbage is input no transformer should choke on
var garbage = map[string]string{
	"empty":               "",
	"spaces":              "  \n\t ",
	"binary":              "\x00\x01\xff\xfe\x80",
	"a scalar":            "42",
	"null":                "null",
	"a string":            `"text"`,
	"broken yaml":         "a: [1, 2\nb: }",
	"broken json":         `{"apps": [{"name": `,
	"tabs":                "\t- a:\n\t\t- b",
	"list of nulls":       "- ~\n- ~\n",
	"list of lists":       "- - - - a\n",
	"scalars inside":      "- Group: text\n- 5\n- true\n- Other:\n    - 7\n    - Service: 12\n    - Service: [1, 2]\n",
	"wrong types":         "services: 5\nsections: text\napps: {}\ncategories: 9\nwidgets: x\nlinks: 1\n",
	"services of scalars": "services:\n  - 1\n  - text\n  - [a]\n  - name: G\n    items: 5\n  - name: H\n    items: [1, {name: 5, url: [x]}]\n",
	"compose of scalars":  "services:\n  a: 1\n  b: [x]\n  c:\n    image: [1]\n    ports: 80\n    labels: 5\n  d:\n    ports: [[1], {published: {x: 1}}]\n    labels: [5, =, a=]\n  '': {ports: ['1:1']}\n",
	"sections of scalars": "sections:\n  - 1\n  - items: x\n    widgets: 5\n  - name: [a]\n    items: [1, {title: {a: 1}, url: 5, icon: [x]}]\n",
	"homarr of scalars":   `{"apps": [1, "x", null, {"name": 5, "url": [1], "behaviour": 3, "area": "x", "shape": {"lg": 5}, "integration": [], "appearance": 1}], "categories": [1, {"id": 5}], "widgets": [1, {"type": 2}]}`,
	"odd keys":            "? [a, b]\n: c\n1: 2\ntrue: 3\n~: 4\n",
	"deep":                strings.Repeat("- a:\n"+strings.Repeat(" ", 2), 1) + strings.Repeat("[", 5000) + strings.Repeat("]", 5000),
	"deep groups":         deepGroups(200),
	"alias bomb":          aliasBomb(),
	"self alias":          "a: &a\n  b: *a\n",
	"long name":           "- " + strings.Repeat("G", 5000) + ":\n    - " + strings.Repeat("é", 5000) + ":\n        href: http://a.lan\n",
	"control characters":  "- \"G\\u0000\\u202e\":\n    - \"S\\u0007\":\n        href: \"http://a.lan/\\u0000\"\n        icon: \"\\u0000\"\n",
}

func deepGroups(depth int) string {
	var out strings.Builder
	for i := 0; i < depth; i++ {
		out.WriteString(strings.Repeat("    ", i) + "- G:\n")
	}
	out.WriteString(strings.Repeat("    ", depth) + "- S:\n" + strings.Repeat("    ", depth+1) + "    href: http://a.lan\n")
	return out.String()
}

func aliasBomb() string {
	bomb := "a: &a [x,x,x,x,x,x,x,x,x]\n"
	previous := "a"
	for _, name := range []string{"b", "c", "d", "e", "f", "g", "h", "i"} {
		bomb += name + ": &" + name + " [" + strings.TrimSuffix(strings.Repeat("*"+previous+",", 9), ",") + "]\n"
		previous = name
	}
	return bomb + "services: *i\nsections: *i\napps: *i\n"
}

var transformers = map[string]func([]byte) (dashdoc.Document, []string, error){
	FormatHomepage: Homepage,
	FormatHomarr:   Homarr,
	FormatHomer:    Homer,
	FormatDashy:    Dashy,
	FormatCompose:  func(data []byte) (dashdoc.Document, []string, error) { return Compose(data, "nas.lan") },
}

// run feeds data to every transformer and to the detection; whatever comes
// out without an error must be a document the import can take
func run(t *testing.T, data []byte) {
	t.Helper()
	for format, transform := range transformers {
		doc, _, err := transform(data)
		if err == nil {
			assert.NoError(t, dashdoc.Validate(&doc), format)
			assertImportable(t, &doc)
		}
	}
	if result, err := Convert("auto", data, Options{}); err == nil {
		assert.NoError(t, dashdoc.Validate(&result.Document))
		assert.NotNil(t, result.Warnings)
	}
}

func TestGarbageNeverPanics(t *testing.T) {
	for name, input := range garbage {
		t.Run(name, func(t *testing.T) { run(t, []byte(input)) })
	}
}

func TestTruncatedAndMangledConfigsNeverPanic(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("testdata", "*"))
	require.NoError(t, err)
	require.NotEmpty(t, files)
	for _, file := range files {
		data, err := os.ReadFile(file)
		require.NoError(t, err)
		// Cut off anywhere, and with every kind of bracket and colon broken
		for cut := 0; cut < len(data); cut += max(1, len(data)/150) {
			run(t, data[:cut])
			run(t, data[cut:])
		}
		for _, swap := range [][2]string{{":", ""}, {"-", "?"}, {"{", "["}, {"\"", "'"}, {"\n", "\n\t"}, {" ", ""}, {"http", "javascript"}} {
			run(t, []byte(strings.ReplaceAll(string(data), swap[0], swap[1])))
		}
	}
}

func toJSON(t *testing.T, v any) string {
	t.Helper()
	raw, err := jsonMarshal(v)
	require.NoError(t, err)
	return string(raw)
}

var jsonMarshal = json.Marshal
