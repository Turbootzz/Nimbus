package integrations

import (
	"context"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/nimbus/backend/internal/models"
)

// Longest label values discovery keeps; longer text is cut, a longer URL or
// icon is dropped
const (
	discoveryNameMax        = 100
	discoveryDescriptionMax = 500
	discoveryURLMax         = 2048
)

// DiscoveryLabels are a container's nimbus.* labels, with homepage.* as
// fallback. They are set by whoever can start containers: treat every field
// as untrusted text.
type DiscoveryLabels struct {
	Enabled     bool   // nimbus.enable=true
	Name        string // nimbus.name, homepage.name
	URL         string // nimbus.url, homepage.href (not validated here)
	Icon        string // nimbus.icon, homepage.icon: a slug or a URL
	Group       string // nimbus.group, homepage.group
	Description string // nimbus.description, homepage.description
	Widget      string // nimbus.widget: an integration kind
}

// DiscoveredContainer is a container as discovery sees it
type DiscoveredContainer struct {
	Name  string
	Image string
	State string
	// Port is the lowest published TCP port, 0 without one
	Port   int
	Labels DiscoveryLabels
}

// DockerDiscover lists the containers that may become services. Containers
// with nimbus.enable=false, or without a name, are left out.
func DockerDiscover(ctx context.Context, conn *Conn) ([]DiscoveredContainer, error) {
	containers := []DiscoveredContainer{}
	err := eachJSON(ctx, conn, dockerContainersPath, dockerAuth(conn), func(c apiContainer) {
		labels, hidden := ParseDiscoveryLabels(c.Labels)
		if hidden || c.name() == "" {
			return
		}
		port := 0
		for _, p := range c.Ports {
			// Docker lists a port once per address family, in no fixed order
			if p.Type == "tcp" && p.PublicPort > 0 && (port == 0 || p.PublicPort < port) {
				port = p.PublicPort
			}
		}
		containers = append(containers, DiscoveredContainer{Name: c.name(), Image: c.Image, State: c.State, Port: port, Labels: labels})
	})
	if err != nil {
		return nil, dockerErr(conn, err)
	}
	return containers, nil
}

// ParseDiscoveryLabels reads the nimbus.* labels, falling back to homepage.*
// for the fields Homepage has too. hidden reports nimbus.enable=false. The
// docker-compose importer reads a compose file's labels with it too.
func ParseDiscoveryLabels(labels map[string]string) (parsed DiscoveryLabels, hidden bool) {
	// first returns the first label that is set, trimmed
	first := func(keys ...string) string {
		for _, key := range keys {
			if value := strings.TrimSpace(labels[key]); value != "" {
				return value
			}
		}
		return ""
	}
	// capped drops a value that is too long to be a URL
	capped := func(value string) string {
		if len(value) > discoveryURLMax {
			return ""
		}
		return value
	}

	enable := strings.ToLower(first("nimbus.enable"))
	return DiscoveryLabels{
		Enabled:     enable == "true",
		Name:        truncate(first("nimbus.name", "homepage.name"), discoveryNameMax),
		URL:         capped(first("nimbus.url", "homepage.href")),
		Icon:        capped(first("nimbus.icon", "homepage.icon")),
		Group:       truncate(first("nimbus.group", "homepage.group"), models.MaxGroupNameLen),
		Description: truncate(first("nimbus.description", "homepage.description"), discoveryDescriptionMax),
		Widget:      strings.ToLower(first("nimbus.widget")),
	}, enable == "false"
}

// truncate cuts s to at most max bytes without splitting a character
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	for max > 0 && !utf8.RuneStart(s[max]) {
		max--
	}
	return strings.TrimSpace(s[:max])
}

var iconSlugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// IconSlug turns an icon label ("sonarr", "Sonarr.png") into a
// dashboard-icons slug; "" when it is not one. Homepage's other icon sets
// (mdi-, si-, sh-) are not in dashboard-icons.
func IconSlug(icon string) string {
	slug := strings.ToLower(strings.TrimSpace(icon))
	for _, ext := range []string{".svg", ".png", ".webp"} {
		slug = strings.TrimSuffix(slug, ext)
	}
	for _, prefix := range []string{"mdi-", "si-", "sh-"} {
		if strings.HasPrefix(slug, prefix) {
			return ""
		}
	}
	if !iconSlugPattern.MatchString(slug) {
		return ""
	}
	return slug
}

// ImageIconSlug guesses a dashboard-icons slug from an image name, without
// its registry, organisation, tag and digest:
// lscr.io/linuxserver/sonarr:latest -> sonarr
func ImageIconSlug(image string) string {
	image, _, _ = strings.Cut(image, "@")
	image = image[strings.LastIndex(image, "/")+1:]
	image, _, _ = strings.Cut(image, ":")
	return IconSlug(strings.ReplaceAll(image, "_", "-"))
}
