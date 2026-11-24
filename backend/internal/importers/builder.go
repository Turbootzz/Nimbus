package importers

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/nimbus/backend/internal/dashdoc"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/utils"
)

// A converted config keeps at most this many groups and services: a
// dashboard of another tool that is larger is not a realistic source
const (
	MaxGroups   = 100
	MaxServices = 500
)

const (
	maxServiceNameLen     = 255 // the column's size
	maxIntegrationNameLen = 100
)

// kinds maps what other dashboards call an app's widget or integration to
// the Nimbus integration kind (see internal/integrations). Keys are lower
// case without punctuation: "adGuardHome" and "adguard-home" are the same.
var kinds = map[string]string{
	"adguard":       "adguard",
	"adguardhome":   "adguard",
	"homeassistant": "home_assistant",
	"jellyfin":      "jellyfin",
	"pihole":        "pihole",
	"proxmox":       "proxmox",
	"qbittorrent":   "qbittorrent",
	"radarr":        "radarr",
	"sonarr":        "sonarr",
	"uptimekuma":    "uptime_kuma",
}

// kindOf returns the Nimbus integration kind for what another dashboard
// calls an app, or ""
func kindOf(app string) string {
	return kinds[strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, app)]
}

// builder collects a document and the warnings about what didn't fit in it.
// It only lets through what the import accepts, so one odd entry in a config
// never fails the whole import.
type builder struct {
	doc      dashdoc.Document
	warnings []string
	dropped  int
	// refs are the refs handed out, so no two entities share one
	refs map[string]bool
	// groups and integrations by what identifies them, to reuse them
	groups       map[string]string
	integrations map[string]string
	// plainIcons: an icon that is just a name is a dashboard-icons name
	// (Homepage); otherwise it is a file of the other dashboard
	plainIcons bool
}

func newBuilder() *builder {
	return &builder{
		doc: dashdoc.Document{
			Version: dashdoc.Version, Groups: []dashdoc.Group{}, Integrations: []dashdoc.Integration{},
			Services: []dashdoc.Service{}, Widgets: []dashdoc.Widget{},
		},
		refs: map[string]bool{}, groups: map[string]string{}, integrations: map[string]string{},
	}
}

func (b *builder) warn(format string, args ...any) {
	if len(b.warnings) < maxWarnings {
		b.warnings = append(b.warnings, fmt.Sprintf(format, args...))
		return
	}
	b.dropped++
}

// result returns the document and its warnings
func (b *builder) result() (dashdoc.Document, []string, error) {
	if b.dropped > 0 {
		b.warnings = append(b.warnings, fmt.Sprintf("%d more warnings are not shown", b.dropped))
	}
	return b.doc, b.warnings, nil
}

// ref hands out a readable ref that nothing else has
func (b *builder) ref(prefix, name string) string {
	var slug strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			slug.WriteRune(r)
		case slug.Len() > 0 && !strings.HasSuffix(slug.String(), "-"):
			slug.WriteByte('-')
		}
		if slug.Len() >= 40 {
			break
		}
	}
	base := prefix + "-" + strings.Trim(slug.String(), "-")
	ref := base
	for i := 2; b.refs[ref]; i++ {
		ref = fmt.Sprintf("%s-%d", base, i)
	}
	b.refs[ref] = true
	return ref
}

// truncate cuts s to at most max bytes without splitting a character
func truncate(s string, max int) string {
	for len(s) > max {
		_, size := utf8.DecodeLastRuneInString(s)
		s = s[:len(s)-size]
	}
	return strings.TrimSpace(s)
}

// group returns the ref of the group with that name, adding it when it is
// new. An empty name gives "": no group.
func (b *builder) group(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	if short := truncate(name, models.MaxGroupNameLen); short != name {
		b.warn("Group %q: the name is shortened to %q", name, short)
		name = short
	}
	if ref, ok := b.groups[name]; ok {
		return ref
	}
	if len(b.doc.Groups) >= MaxGroups {
		b.warn("Group %q: skipped, at most %d groups can be imported", name, MaxGroups)
		return ""
	}
	ref := b.ref("group", name)
	b.groups[name] = ref
	b.doc.Groups = append(b.doc.Groups, dashdoc.Group{Ref: ref, Name: name})
	return ref
}

// webURL returns raw when it is a URL the import takes for a service: http
// or https, with a host that is not a cloud metadata endpoint. It is the
// rule of utils.ValidateServiceURL, which looks nothing up.
func webURL(raw string) (*url.URL, bool) {
	if utils.ValidateServiceURL(raw) != nil {
		return nil, false
	}
	parsed, err := url.ParseRequestURI(raw)
	return parsed, err == nil
}

// withoutSecrets returns raw without its login and query, which can hold a
// password or key, and whether it had either; "" when raw doesn't parse
func withoutSecrets(raw string) (string, bool) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", false
	}
	if parsed.User == nil && parsed.RawQuery == "" {
		return raw, false
	}
	parsed.User, parsed.RawQuery, parsed.ForceQuery = nil, "", false
	return parsed.String(), true
}

// quoteURL quotes a URL for a warning, without its login and query. What
// still has an @ or ? after that (a login without a scheme like
// "admin:pw@nas.lan", which parses as something else) is not shown at all.
func quoteURL(raw string) string {
	clean := raw
	if strings.ContainsAny(raw, "@?") {
		clean, _ = withoutSecrets(raw)
	}
	if clean == "" || strings.ContainsAny(clean, "@?") {
		return "(not shown)"
	}
	return strconv.Quote(clean)
}

// entry is a link as another dashboard describes it
type entry struct {
	name, url, description, icon, group string
	// app is the other dashboard's widget or integration type, and appURL
	// where that reaches the app (defaults to url)
	app, appURL string
}

// service adds a link tile, and the integration of its app when Nimbus has
// that kind. A link the import would refuse is skipped with a warning.
func (b *builder) service(e entry) {
	name := truncate(strings.TrimSpace(e.name), maxServiceNameLen)
	if name == "" {
		b.warn("An entry without a name is skipped")
		return
	}
	link, secret := withoutSecrets(e.url)
	if _, ok := webURL(link); !ok {
		if e.url == "" {
			b.warn("%s: skipped, it has no URL", name)
		} else {
			b.warn("%s: skipped, %s is not an http(s) URL Nimbus can link to", name, quoteURL(e.url))
		}
		return
	}
	if len(b.doc.Services) >= MaxServices {
		b.warn("%s: skipped, at most %d services can be imported", name, MaxServices)
		return
	}
	if secret {
		b.warn("%s: the login and query of its URL are left out, they can hold a secret", name)
	}
	service := dashdoc.Service{
		Ref:         b.ref("service", name),
		Name:        name,
		URL:         link,
		Description: e.description,
		Position:    len(b.doc.Services),
		GroupRef:    b.group(e.group),
	}
	b.icon(&service, e.icon)
	if e.app != "" {
		appURL := e.appURL
		if appURL == "" {
			appURL = e.url
		}
		service.IntegrationRef = b.integration(name, e.app, appURL)
	}
	b.doc.Services = append(b.doc.Services, service)
}

// integration returns the ref of the integration for an app of another
// dashboard, adding it when it is new; "" when Nimbus has no such kind. It
// carries no credentials: the user enters them after the import.
func (b *builder) integration(name, app, rawURL string) string {
	kind := kindOf(app)
	if kind == "" {
		b.warn("%s: Nimbus has no %q integration, it is imported as a link only", name, app)
		return ""
	}
	parsed, ok := webURL(rawURL)
	if !ok {
		b.warn("%s: the %s integration is skipped, %s is not an http(s) URL", name, kind, quoteURL(rawURL))
		return ""
	}
	// A base URL holds no login, query or fragment
	parsed.User, parsed.RawQuery, parsed.Fragment = nil, "", ""
	if kind == "pihole" {
		// Dashboards link to Pi-hole's web UI under /admin; its API is beside it
		parsed.Path, parsed.RawPath = strings.TrimSuffix(strings.TrimRight(parsed.Path, "/"), "/admin"), ""
	}
	baseURL := strings.TrimRight(parsed.String(), "/")

	id := kind + " " + baseURL
	if ref, ok := b.integrations[id]; ok {
		return ref
	}
	if len(b.doc.Integrations) >= dashdoc.MaxIntegrations {
		b.warn("%s: the %s integration is skipped, at most %d integrations can be imported", name, kind, dashdoc.MaxIntegrations)
		return ""
	}
	ref := b.ref("integration", name)
	b.integrations[id] = ref
	b.doc.Integrations = append(b.doc.Integrations, dashdoc.Integration{
		Ref: ref, Kind: kind, Name: truncate(name, maxIntegrationNameLen), BaseURL: baseURL, NeedsCredentials: true,
	})
	return ref
}
